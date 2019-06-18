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
	"sync"

	"github.com/dgraph-io/ristretto/ring"
	"github.com/dgraph-io/ristretto/store"
)

// Policy is the interface encapsulating eviction/admission behavior.
type Policy interface {
	ring.Consumer
	// Add is the most important function of a Policy. It determines what keys
	// are added and what keys are evicted. An important distinction is that
	// Add is an *attempt* to add a key to the policy (and later the cache as
	// a whole), but the return values indicate whether or not the attempt was
	// successful.
	//
	// For example, Add("1") may return false because the Policy determined that
	// "1" isn't valuable enough to be added. Or, it will return true with a
	// victim key because it determined that "1" was more valuable than the
	// victim and thus added it to the Policy. The caller of Add would then
	// delete the victim and store the key-value pair.
	Add(string) (string, bool)
}

// Cache ties everything together. The three main components are:
//
//     1) The hash map: this is the store.Map interface.
//     2) The admission and eviction policy: this is the Policy interface.
//     3) The bp-wrapper buffer: this is the ring.Buffer struct.
//
// All three of these components work together to try and keep the most valuable
// key-value pairs in the hash map. Value is determined by the Policy, and
// BP-Wrapper keeps the Policy fast (by batching metadata updates).
type Cache struct {
	data   store.Map
	policy Policy
	buffer *ring.Buffer
}

func NewCache(capacity uint64) *Cache {
	policy := NewLFU(capacity)
	return &Cache{
		data:   store.NewMap(),
		policy: policy,
		buffer: ring.NewBuffer(ring.LOSSY, &ring.Config{
			Consumer: policy,
			Capacity: 2048,
		}),
	}
}

func (c *Cache) Get(key string) interface{} {
	c.buffer.Push(ring.Element(key))
	return c.data.Get(key)
}

func (c *Cache) Set(key string, value interface{}) {
	// if already exists, just update the value
	if rawValue := c.data.Get(key); rawValue != nil {
		c.data.Set(key, value)
		return
	}
	// attempt to add and delete victim if needed
	if victim, added := c.policy.Add(key); added {
		// there is an eviction victim
		if victim != "" {
			c.data.Del(key)
		}
		// since the key was added to the policy, add it to the data store too
		c.data.Set(key, value)
	}
}

func (c *Cache) Del(key string) {
	c.data.Del(key)
}

////////////////////////////////////////////////////////////////////////////////

// LFU_SAMPLE is the number of items to sample when looking at eviction
// candidates.
const LFU_SAMPLE = 5

type LFU struct {
	sync.Mutex
	data     map[string]*uint64
	size     uint64
	capacity uint64
}

func NewLFU(capacity uint64) *LFU {
	return &LFU{
		data:     make(map[string]*uint64, capacity),
		capacity: capacity,
	}
}

// hit is called for each key in a Push() operation or when the key is accessed
// during an Add() operation.
//
// NOTE: this function needs to be wrapped in a mutex to be safe.
func (p *LFU) hit(key string) {
	counter := p.data[key]
	if counter != nil {
		*(counter)++
	}
}

// Push acquires a lock and calls hit() for each key (incrementing the counter).
func (p *LFU) Push(keys []ring.Element) {
	p.Lock()
	defer p.Unlock()
	for _, key := range keys {
		p.hit(string(key))
	}
}

// Add attempts to add the key param to the policy. If added, it may return the
// victim key so the caller can delete the victim from its data store.
func (p *LFU) Add(key string) (victim string, added bool) {
	p.Lock()
	defer p.Unlock()
	// if it's already in the policy, just increment the counter and return
	if _, exists := p.data[key]; exists {
		p.hit(key)
		return
	}
	// the key doesn't exist in our records yet, so create a new counter
	counter := uint64(0)
	// add the counter
	p.data[key] = &counter
	added = true
	p.size++
	// check if eviction is needed
	if p.size >= p.capacity {
		// find a victim
		min, i := uint64(0), 0
		for k, v := range p.data {
			// stop once we reach the sample size
			if i == LFU_SAMPLE {
				break
			}
			if i == 0 || *v < min {
				min = *v
				victim = k
			}
			i++
		}
		// delete the victim
		delete(p.data, victim)
	}
	return
}
