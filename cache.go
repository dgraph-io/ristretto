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
	"bytes"
	"errors"
	"fmt"
	"sync/atomic"

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
	stats  *metrics
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
	// If set to true, cache would collect useful metrics about usage.
	Metrics bool
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
	if config.Metrics {
		cache.collectMetrics()
	}
	// We can possibly make this configurable. But having 2 goroutines processing this seems
	// sufficient for now.
	// TODO: Allow a way to stop these goroutines.
	for i := 0; i < 2; i++ {
		go cache.processItems()
	}
	return cache, nil
}

func (c *Cache) collectMetrics() {
	c.stats = newMetrics()
	c.policy.CollectMetrics(c.stats)
}

func (c *Cache) Get(key interface{}) (interface{}, bool) {
	hash := z.KeyToHash(key)
	c.buffer.Push(hash)
	val, ok := c.data.Get(hash)
	if ok {
		c.stats.Add(hit, 1)
	} else {
		c.stats.Add(miss, 1)
	}
	return val, ok
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
		c.stats.Add(dropSets, 1)
		return false
	}
}

func (c *Cache) Del(key interface{}) {
	hash := z.KeyToHash(key)
	c.policy.Del(hash)
	c.data.Del(hash)
}

func (c *Cache) Metrics() *metrics {
	return c.stats
}

type metricType int

const (
	// The following 2 keep track of hits and misses.
	hit = iota
	miss

	// The following 3 keep track of number of keys added, updated and evicted.
	keyAdd
	keyUpdate
	keyEvict

	// The following 2 keep track of cost of keys added and evicted.
	costAdd
	costEvict

	// The following keep track of how many sets were dropped or rejected later.
	dropSets
	rejectSets

	// The following 2 keep track of how many gets were kept and dropped on the floor.
	dropGets
	keepGets

	// This should be the final enum. Other enums should be set before this.
	doNotUse
)

func stringFor(t metricType) string {
	switch t {
	case hit:
		return "hit"
	case miss:
		return "miss"
	case keyAdd:
		return "keys-added"
	case keyUpdate:
		return "keys-updated"
	case keyEvict:
		return "keys-evicted"
	case costAdd:
		return "cost-added"
	case costEvict:
		return "cost-evicted"
	case dropSets:
		return "sets-dropped"
	case rejectSets:
		return "sets-rejected" // by policy.
	case dropGets:
		return "gets-dropped"
	case keepGets:
		return "gets-kept"
	default:
		return "unidentified"
	}
}

// metrics is the struct for hit ratio statistics. Note that there is some
// cost to maintaining the counters, so it's best to wrap Policies via the
// Recorder type when hit ratio analysis is needed.
type metrics struct {
	all [doNotUse][256]uint64
}

func newMetrics() *metrics {
	s := &metrics{}
	// for i := 0; i < doNotUse; i++ {
	// 	v := new(uint64)
	// 	s.all = append(s.all, v)
	// }
	return s
}

func (p *metrics) Add(t metricType, delta uint64) {
	if p == nil {
		return
	}
	valp := p.all[t]
	idx := z.FastRand() % 256
	atomic.AddUint64(&valp[idx], delta)
}

func (p *metrics) Get(t metricType) uint64 {
	if p == nil {
		return 0
	}
	valp := p.all[t]
	idx := z.FastRand() % 256
	return atomic.LoadUint64(&valp[idx])
}

func (p *metrics) Ratio() float64 {
	if p == nil {
		return 0.0
	}
	hits, misses := p.Get(hit), p.Get(miss)
	if hits == 0 && misses == 0 {
		return 0.0
	}
	return float64(hits) / float64(hits+misses)
}

func (p *metrics) String() string {
	if p == nil {
		return ""
	}
	var buf bytes.Buffer
	for i := 0; i < doNotUse; i++ {
		t := metricType(i)
		fmt.Fprintf(&buf, "%s: %d ", stringFor(t), p.Get(t))
	}
	fmt.Fprintf(&buf, "gets-total: %d ", p.Get(hit)+p.Get(miss))
	fmt.Fprintf(&buf, "hit-ratio: %.2f", p.Ratio())
	return buf.String()
}
