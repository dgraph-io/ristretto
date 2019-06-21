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
	"container/list"
	"sync"

	"github.com/dgraph-io/ristretto/bloom"
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
	// Log returns a PolicyLog with performance statistics.
	Log() PolicyLog
}

// PolicyLog is maintained by all Policys and is useful for tracking statistics
// about Policy efficiency.
type PolicyLog struct {
	Hits      uint64
	Requests  uint64
	Evictions uint64
}

func (p *PolicyLog) hit() {
	p.Hits++
	p.Requests++
}

func (p *PolicyLog) miss() {
	p.Requests++
}

func (p *PolicyLog) evict() {
	p.Evictions++
}

////////////////////////////////////////////////////////////////////////////////

// Clairvoyant is a Policy meant to be theoretically optimal (see [1]). Normal
// Push and Add operations just maintain a history log. The real work is done
// when Log is called, and it "looks into the future" to evict the best
// candidates (the furthest away).
//
// [1]: https://bit.ly/2WTPdJ9
//
// This Policy is primarily for benchmarking purposes (as a baseline).
type Clairvoyant struct {
	sync.Mutex
	time     uint64
	log      *PolicyLog
	access   map[string][]uint64
	capacity uint64
	future   []string
}

func NewClairvoyant(capacity uint64) *Clairvoyant {
	return &Clairvoyant{
		log:      &PolicyLog{},
		capacity: capacity,
		access:   make(map[string][]uint64, capacity),
	}
}

// distance finds the "time distance" from the start position to the minimum
// time value - this is used to judge eviction candidates.
func (p *Clairvoyant) distance(start uint64, times []uint64) (uint64, bool) {
	good, min := false, uint64(0)
	for i := range times {
		if times[i] > start {
			good = true
		}
		if i == 0 || times[i] < min {
			min = times[i] - start
		}
	}
	return min, good
}

func (p *Clairvoyant) record(key string) {
	p.time++
	if p.access[key] == nil {
		p.access[key] = make([]uint64, 0)
	}
	p.access[key] = append(p.access[key], p.time)
	p.future = append(p.future, key)
}

func (p *Clairvoyant) Push(keys []ring.Element) {
	p.Lock()
	defer p.Unlock()
	for _, key := range keys {
		p.record(string(key))
	}
}

func (p *Clairvoyant) Add(key string) (victim string, added bool) {
	p.Lock()
	defer p.Unlock()
	p.record(key)
	added = true
	return
}

func (p *Clairvoyant) Log() PolicyLog {
	p.Lock()
	defer p.Unlock()
	// data serves as the "pseudocache" with the ability to see into the future
	data := make(map[string]struct{}, p.capacity)
	size := uint64(0)
	for i, key := range p.future {
		// check if already exists
		if _, exists := data[key]; exists {
			p.log.hit()
			continue
		}
		p.log.miss()
		// check if eviction is needed
		if size == p.capacity {
			// eviction is needed
			//
			// collect item distances
			good := false
			distance := make(map[string]uint64, p.capacity)
			for k, _ := range data {
				distance[k], good = p.distance(uint64(i), p.access[k])
				if !good {
					// there's no good distances because the key isn't used
					// again in the future, so we can just stop here and delete
					// it, and skip over the rest
					p.log.evict()
					delete(data, k)
					size--
					goto add
				}
			}
			// find the largest distance
			maxDistance, maxKey, c := uint64(0), "", 0
			for k, d := range distance {
				if c == 0 || d > maxDistance {
					maxKey = k
					maxDistance = d
				}
				c++
			}
			// delete the item furthest away
			p.log.evict()
			delete(data, maxKey)
			size--
		}
	add:
		// add the item
		data[key] = struct{}{}
		size++
	}
	return *p.log
}

////////////////////////////////////////////////////////////////////////////////

// LFU is a Policy with no admission policy and a sampled LFU eviction policy.
type LFU struct {
	sync.Mutex
	log      *PolicyLog
	data     map[string]uint64
	size     uint64
	capacity uint64
}

func NewLFU(capacity uint64) *LFU {
	return &LFU{
		log:      &PolicyLog{},
		data:     make(map[string]uint64, capacity),
		capacity: capacity,
	}
}

// hit is called for each key in a Push() operation or when the key is accessed
// during an Add() operation.
func (p *LFU) hit(key string) {
	if _, exists := p.data[key]; exists {
		p.data[key]++
		p.log.hit()
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
		p.data[key]++
		p.log.hit()
		return
	}
	p.log.miss()
	// check if eviction is needed
	if p.size >= p.capacity {
		// find a victim
		min, i := uint64(0), 0
		for k, v := range p.data {
			// stop once we reach the sample size
			if i == LFU_SAMPLE {
				break
			}
			if i == 0 || v < min {
				min = v
				victim = k
			}
			i++
		}
		// delete the victim
		delete(p.data, victim)
		p.log.evict()
	}
	// add the new item to the policy
	p.data[key] = 1
	added = true
	p.size++
	return
}

func (p *LFU) Log() PolicyLog {
	return *p.log
}

////////////////////////////////////////////////////////////////////////////////

// TinyLFU keeps track of frequency using tiny (4-bit) counters in the form of a
// counting bloom filter. For eviction, sampled LFU is done.
type TinyLFU struct {
	sync.Mutex
	data     store.Map
	size     uint64
	capacity uint64
	sketch   bloom.Sketch
}

func NewTinyLFU(capacity uint64, data store.Map) *TinyLFU {
	return &TinyLFU{
		data:     data,
		sketch:   bloom.NewCBF(capacity),
		capacity: capacity,
	}
}

func (p *TinyLFU) Push(keys []ring.Element) {
	p.Lock()
	defer p.Unlock()
	for _, key := range keys {
		p.sketch.Increment(string(key))
	}
}

func (p *TinyLFU) Add(key string) (victim string, added bool) {
	p.Lock()
	defer p.Unlock()
	// tinylfu doesn't have an "adding" mechanism because the structure is
	// probabilistic, so we can just assume it's already in the policy - but we
	// do need to keep track of the policy size and capacity for eviction
	// purposes
	//
	// check if eviction is needed
	if p.size >= p.capacity {
		// eviction is needed
		//
		// create a slice that will hold the random sample of keys from the map
		keys, i := make([]string, LFU_SAMPLE), 0
		// get the random sample
		p.data.Run(func(k, v interface{}) bool {
			keys[i] = k.(string)
			i++
			return !(i == LFU_SAMPLE)
		})
		// keep track of mins
		minKey, minHits := "", uint64(0)
		// find the minimally used item from the random sample
		for j, k := range keys {
			if k != "" {
				// lookup the key in the frequency sketch
				hits := p.sketch.Estimate(k)
				// keep track of minimally used item
				if j == 0 || hits < minHits {
					minKey = k
					minHits = hits
				}
			}
		}
		// set victim to minimally used item
		victim = minKey
	}
	// increment key counter
	p.sketch.Increment(key)
	added = true
	p.size++
	return
}

////////////////////////////////////////////////////////////////////////////////

// LRU is a Policy with no admission policy and a LRU eviction policy (using
// doubly linked list).
type LRU struct {
	sync.Mutex
	list     *list.List
	look     map[string]*list.Element
	capacity uint64
	size     uint64
}

func NewLRU(capacity uint64) *LRU {
	return &LRU{
		list:     list.New(),
		look:     make(map[string]*list.Element),
		capacity: capacity,
	}
}

func (p *LRU) Push(keys []ring.Element) {
	p.Lock()
	defer p.Unlock()
	for _, key := range keys {
		p.list.MoveToFront(p.look[string(key)])
	}
}

func (p *LRU) Add(key string) (victim string, added bool) {
	p.Lock()
	defer p.Unlock()
	// check if the element already exists in the policy
	if element, exists := p.look[key]; exists {
		p.list.MoveToFront(element)
		return
	}
	// check if eviction is needed
	if p.size >= p.capacity {
		// get the victim key
		victim = p.list.Back().Value.(string)
		// delete the victim from the list
		p.list.Remove(p.list.Back())
		// delete the victim from the lookup map
		delete(p.look, victim)
	}
	// add the new key to the list
	p.look[key] = p.list.PushFront(key)
	added = true
	p.size++
	return
}

func (p *LRU) String() string {
	out := "["
	for element := p.list.Front(); element != nil; element = element.Next() {
		out += element.Value.(string) + ", "
	}
	return out[:len(out)-2] + "]"
}
