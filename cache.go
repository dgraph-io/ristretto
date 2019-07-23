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
	"sync/atomic"
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
	data   Map
	policy Policy
	buffer *Buffer
	notify func(string)
	used   uint64
	size   uint64
}

type Config struct {
	// MaxItems is the max number of items expected to be in the Cache.
	MaxItems uint64
	// MaxBytes is the Cache memory usage limit. If CostAware == false, this
	// field is ignored and MaxItems is used instead.
	MaxBytes uint64
	// BufferItems is max number of items in access batches (BP-Wrapper).
	BufferItems uint64
	// CostAware is true if item costs are non-uniform, and false if they're
	// uniform (won't be considered during evicion process).
	CostAware bool
	// Policy is the admission / eviction Policy creator.
	Policy func(uint64, uint64, bool) Policy
	// OnEvict is ran for each key evicted.
	OnEvict func(string)
	// Log is whether or not to Log hit ratio statistics (with some overhead).
	Log bool
}

func NewCache(config *Config) *Cache {
	if !config.CostAware {
		config.MaxBytes = config.MaxItems
	}
	// data is the hash map for the entire cache, it's initialized outside of
	// the cache struct declaration because it may need to be passed to the
	// policy in some cases
	data := NewMap()
	// initialize the policy (with a recorder wrapping if logging is enabled)
	policy := config.Policy(
		config.MaxItems,
		config.MaxBytes,
		config.CostAware,
	)
	if config.Log {
		policy = NewRecorder(policy, data)
	}
	return &Cache{
		data:   data,
		policy: policy,
		buffer: NewBuffer(LOSSY, &RingConfig{
			Consumer: policy,
			Capacity: config.BufferItems,
		}),
		notify: config.OnEvict,
		size:   config.MaxItems,
	}
}

func (c *Cache) Get(key string) interface{} {
	c.buffer.Push(Element(key))
	return c.data.Get(key)
}

func (c *Cache) Set(key string, val interface{}, cost uint64) ([]string, bool) {
	victims, added := c.policy.Add(key, cost)
	if !added {
		return nil, false
	}
	for _, victim := range victims {
		c.data.Del(victim)
		atomic.AddUint64(&c.used, ^uint64(0))
		if c.notify != nil {
			c.notify(victim)
		}
	}
	c.data.Set(key, val)
	if atomic.AddUint64(&c.used, 1) == c.size {
		atomic.StoreUint64(&c.used, c.size/2)
		c.policy.Res()
	}
	return victims, true
}

func (c *Cache) Del(key string) {
	c.policy.Del(key)
	c.data.Del(key)
}

func (c *Cache) Log() *PolicyLog {
	return c.policy.Log()
}
