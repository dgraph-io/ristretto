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
	"sync/atomic"

	"github.com/dgraph-io/ristretto/ring"
	"github.com/dgraph-io/ristretto/store"
)

type Cache struct {
	meta     *Meta
	data     store.Map
	size     uint64
	capacity uint64
	buffer   *ring.Buffer
}

func NewCache(capacity uint64) *Cache {
	meta := &Meta{data: make(map[string]*uint64, capacity)}
	return &Cache{
		meta:     meta,
		data:     store.NewMap(),
		size:     0,
		capacity: capacity,
		buffer: ring.NewBuffer(ring.LOSSY, &ring.Config{
			Consumer: meta,
			Capacity: 1024 * 2,
		}),
	}
}

func (c *Cache) Get(key string) interface{} {
	// record access for admission/eviction tracking
	c.buffer.Push(ring.Element(key))
	// return value from data store
	return c.data.Get(key)
}

func (c *Cache) Set(key string, value interface{}) {
	// if already exists, just update the value
	if c.data.Get(key) != nil {
		// TODO: test whether we should update the metadata on a set operation,
		//       my gut feeling is that this is better for hit ratio
		c.buffer.Push(ring.Element(key))
		c.data.Set(key, value)
		return
	}
	// check if the cache is full and we need to evict
	if atomic.AddUint64(&c.size, 1) >= c.capacity {
		// delete the victim from data store
		// c.data.Del(c.meta.Evict())
	}
	// record the access *after* possible eviction, so as we don't immediately
	// evict the item just added (in this function call, anyway - eviction
	// policies such as hyperbolic caching adjust for this)
	c.buffer.Push(ring.Element(key))
	// save new item to data store
	c.data.Set(key, value)
}

func (c *Cache) Del(key string) {
	// TODO
}

////////////////////////////////////////////////////////////////////////////////

// Meta is a naive metadata manager for admission/eviction statistics. Currently
// it's just a hash map with counters as values.
type Meta struct {
	sync.Mutex
	data map[string]*uint64
}

func (m *Meta) Push(keys []ring.Element) {
	m.Lock()
	defer m.Unlock()
	for _, key := range keys {
		m.record(string(key))
	}
}

func (m *Meta) record(key string) {
	if counter, exists := m.data[key]; exists {
		*counter++
		return
	}
	// make a new counter for this key
	counter := uint64(1)
	m.data[key] = &counter
}

// number of elements to check when looking for victims
const SAMPLE_SIZE = 5

func (m *Meta) Victim() string {
	m.Lock()
	defer m.Unlock()
	victim := struct {
		key   string
		count uint64
	}{}
	i := 0
	for key, counter := range m.data {
		count := atomic.LoadUint64(counter)
		if i == 0 || count < victim.count {
			victim.key, victim.count = key, count
		}
		if i == SAMPLE_SIZE {
			break
		}
		i++
	}
	delete(m.data, victim.key)
	return victim.key
}
