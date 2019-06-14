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

type Policy interface {
	ring.Consumer
	Victim() string
}

type Cache struct {
	meta     Policy
	data     store.Map
	size     uint64
	capacity uint64
	buffer   *ring.Buffer
}

func NewCache(capacity uint64) *Cache {
	meta := NewSampledLFU(capacity)
	return &Cache{
		meta:     meta,
		data:     store.NewMap(),
		size:     0,
		capacity: capacity,
		buffer: ring.NewBuffer(ring.LOSSY, &ring.Config{
			Consumer: meta,
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
	// check if the cache is full and we need to evict
	if atomic.AddUint64(&c.size, 1) >= c.capacity {
		// delete the victim from data store
		c.data.Del(c.meta.Victim())
		// decrement size counter
		atomic.AddUint64(&c.size, ^uint64(0))
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

type SampledLFU struct {
	sync.Mutex
	data map[string]*uint64
}

func NewSampledLFU(capacity uint64) *SampledLFU {
	return &SampledLFU{
		data: make(map[string]*uint64, capacity),
	}
}

func (p *SampledLFU) Push(elements []ring.Element) {
	p.Lock()
	defer p.Unlock()
	for _, element := range elements {
		p.Record(string(element))
	}
}

func (p *SampledLFU) Record(key string) {
	if counter, exists := p.data[key]; exists {
		*counter++
		return
	}
	// make a new counter
	counter := uint64(1)
	p.data[key] = &counter
}

const SAMPLE = 5

func (p *SampledLFU) Victim() string {
	p.Lock()
	defer p.Unlock()
	m := struct {
		key  string
		hits uint64
	}{}
	i := 0
	for key, hits := range p.data {
		h := atomic.LoadUint64(hits)
		if i == 0 || h < m.hits {
			m.key, m.hits = key, h
		}
		if i++; i == SAMPLE {
			break
		}
	}
	delete(p.data, m.key)
	return m.key
}

/*
type Meta struct {
	sync.Mutex
	tracking *Stack
}

func NewMeta(capacity uint64) *Meta {
	return &Meta{
		tracking: NewStack(capacity),
	}
}

func (m *Meta) Push(elements []ring.Element) {
	m.Lock()
	defer m.Unlock()
	for i := range elements {
		// increment the current element's counter
		*(elements[i].Hits)++
		// add to the tracking stack
		m.tracking.Push(elements[i])
	}
}

func (m *Meta) Victim() string {
	m.Lock()
	defer m.Unlock()
	element := m.tracking.Pop()
	if element.Key == nil {
		return ""
	}
	return *(element.Key)
}

// Stack is a ring.Element stack that keeps track of the minimum element.
type Stack struct {
	size     uint64
	capacity uint64
	elements []ring.Element
}

func NewStack(capacity uint64) *Stack {
	return &Stack{
		capacity: capacity,
		elements: make([]ring.Element, capacity),
	}
}

func (s *Stack) Push(element ring.Element) {
	if s.size == 0 || *(element.Hits) < *(s.elements[s.size-1].Hits) {
		if s.size == s.capacity {
			s.elements = append(s.elements[1:], element)
		} else {
			s.elements[s.size] = element
			s.size++
		}
	}
}

func (s *Stack) Pop() ring.Element {
	if s.size > 0 {
		s.size--
	}
	return s.elements[s.size]
}
*/
