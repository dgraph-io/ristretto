/*
 * Copyright 2019 Dgraph Labs, Inc. and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package ristretto

import (
	"errors"

	"github.com/dgraph-io/ristretto/z"
)

// Cache ties everything together. The three main components are:
//
//     1) The hash map: this is the Map interface.
//     2) The admission and eviction policy: this is the Policy interface.
//     3) The bp-wrapper buffer: this is the Buffer struct.
//
// All three of these components work together to try and keep the most valuable
// key-value pairs in the hash map. Value is determined by the Policy, and
// BP-Wrapper keeps the Policy fast (by batching metadata updates).
type Cache struct {
	data   store
	policy Policy
	buffer *ringBuffer
	setCh  chan *item
}

type item struct {
	key  uint64
	val  interface{}
	cost int64
}

type Config struct {
	// NumCounters is the number of keys whose frequency of access should be
	// tracked via TinyLFU.
	// Each counter is worth 4-bits, so TinyLFU can store 2 counters per byte.
	NumCounters int64
	// MaxCost is the cost capacity of the Cache.
	MaxCost int64
	// BufferItems is max number of items in access batches (BP-Wrapper).
	BufferItems int64
	// Log is whether or not to Log hit ratio statistics (with some overhead).
	// TODO: With lossy setup, the policy might not be invoked, in which case the numbers from this
	// are going to be incorrect. Instead, we should be doing log within Cache itself.
	Log bool
}

func NewCache(config *Config) (*Cache, error) {
	switch {
	case config.NumCounters == 0:
		return nil, errors.New("NumCounters can't be zero.")
	case config.MaxCost == 0:
		return nil, errors.New("MaxCost can't be zero.")
	case config.BufferItems == 0:
		return nil, errors.New("BufferItems can't be zero.")
	}

	// Data is the hash map for the entire cache, it's initialized outside of
	// the cache struct declaration because it may need to be passed to the
	// policy in some cases
	data := newStore()
	// initialize the policy (with a recorder wrapping if logging is enabled)
	policy := newPolicy(config.NumCounters, config.MaxCost)
	if config.Log {
		policy = NewRecorder(policy, data)
	}
	cache := &Cache{
		data:   data,
		policy: policy,
		buffer: newRingBuffer(ringLossy, &ringConfig{
			Consumer: policy,
			Capacity: config.BufferItems,
			Stripes:  0, // Don't care about the stripes in ringLossy.
		}),
		setCh: make(chan *item, 32*1024),
	}
	// We can possibly make this configurable. But having 2 goroutines processing this seems
	// sufficient for now.
	// TODO: Allow a way to stop these goroutines.
	for i := 0; i < 2; i++ {
		go cache.processItems()
	}
	return cache, nil
}

func (c *Cache) Get(key interface{}) (interface{}, bool) {
	hash := z.KeyToHash(key)
	c.buffer.Push(hash)
	return c.data.Get(hash)
}

func (c *Cache) processItems() {
	for item := range c.setCh {
		victims, added := c.policy.Add(item.key, item.cost)
		if added {
			c.data.Set(item.key, item.val)
		}
		for _, victim := range victims {
			c.data.Del(victim)
		}
	}
}

func (c *Cache) Set(key interface{}, val interface{}, cost int64) bool {
	hash := z.KeyToHash(key)
	// We should not set the key value to c.data here. Otherwise, if we drop the item on the floor,
	// we would have an orphan key whose cost is not being tracked.
	select {
	case c.setCh <- &item{key: hash, val: val, cost: cost}:
		return true
	default:
		// Drop the set on the floor to avoid blocking.
		return false
	}
}

func (c *Cache) Del(key interface{}) {
	hash := z.KeyToHash(key)
	c.policy.Del(hash)
	c.data.Del(hash)
}

func (c *Cache) Log() *PolicyLog {
	return c.policy.Log()
}
