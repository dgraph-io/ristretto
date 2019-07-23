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
	"math"
	"sync"
	"sync/atomic"
)

const (
	// lfuSample is the number of items to sample when looking at eviction
	// candidates. 5 seems to be the most optimal number [citation needed].
	lfuSample = 5
)

// Policy is the interface encapsulating eviction/admission behavior.
type Policy interface {
	Consumer
	// Add attempts to Add the key-cost pair to the Policy. It returns a slice
	// of evicted keys and a bool denoting whether or not the key-cost pair
	// was added.
	Add(string, uint64) ([]string, bool)
	// Has returns true if the key exists in the Policy.
	Has(string) bool
	// Del deletes the key from the Policy.
	Del(string)
	// Res is a reset operation and maintains metadata freshness.
	Res()
	// Cap returns the available capacity.
	Cap() int64
	Log() *PolicyLog
}

func NewPolicy(maxBytes, maxItems uint64, cost bool) Policy {
	return &policy{
		admi: newTinyLFU(maxItems),
		evic: newSampledLFU(maxBytes, maxItems),
		cost: cost,
		mean: maxBytes / maxItems,
	}
}

// policy is the default policy, which is currently TinyLFU admission with
// sampledLFU eviction.
type policy struct {
	sync.Mutex
	admi *tinyLFU
	evic *sampledLFU
	cost bool
	mean uint64
}

type policyPair struct {
	key  string
	cost uint64
}

func (p *policy) Push(keys []Element) {
	p.Lock()
	defer p.Unlock()
	p.admi.Push(keys)
}

func (p *policy) Add(key string, cost uint64) ([]string, bool) {
	p.Lock()
	defer p.Unlock()
	// can't add an item bigger than entire cache
	if cost > p.evic.size {
		return nil, false
	}
	if _, exists := p.evic.data[key]; exists {
		p.admi.Increment(key)
		return nil, true
	}
	over := p.evic.over(cost)
	if over <= 0 {
		// there's room in the cache
		p.evic.add(key, cost)
		return nil, true
	}
	victims := make([]string, 0)
	// delete victims until there's enough space
	for ; over > 0; over = p.evic.over(cost) {
		sample := p.evic.sample(lfuSample)
		// find minimally used item in sample
		minKey, minHits := "", uint64(math.MaxUint64)
		for _, pair := range sample {
			if hits := p.admi.Estimate(pair.key); hits < minHits {
				minKey, minHits = pair.key, hits
			}
		}
		// evic updates the over
		p.evic.del(minKey)
		victims = append(victims, minKey)
		// only run this loop once if ignoring cost
		if p.cost {
			break
		}
	}
	p.evic.add(key, cost)
	return victims, true
}

func (p *policy) Has(key string) bool {
	p.Lock()
	defer p.Unlock()
	_, exists := p.evic.data[key]
	return exists
}

func (p *policy) Del(key string) {
	p.Lock()
	defer p.Unlock()
	p.evic.del(key)
}

func (p *policy) Res() {
	p.Lock()
	defer p.Unlock()
	p.admi.Reset()
}

func (p *policy) Cap() int64 {
	p.Lock()
	defer p.Unlock()
	return int64(p.evic.size - p.evic.used)
}

func (p *policy) Log() *PolicyLog {
	return nil
}

// sampledLFU is an eviction helper storing key-cost pairs.
type sampledLFU struct {
	data map[string]uint64
	size uint64
	used uint64
}

func newSampledLFU(maxBytes, maxItems uint64) *sampledLFU {
	return &sampledLFU{
		data: make(map[string]uint64, maxItems),
		size: maxBytes,
	}
}

func (p *sampledLFU) over(cost uint64) int64 {
	return int64((p.used + cost) - p.size)
}

func (p *sampledLFU) sample(n uint64) []*policyPair {
	pairs := make([]*policyPair, 0, n)
	for key, cost := range p.data {
		pairs = append(pairs, &policyPair{key, cost})
		if len(pairs) == cap(pairs) {
			break
		}
	}
	return pairs
}

func (p *sampledLFU) del(key string) {
	p.used -= p.data[key]
	delete(p.data, key)
}

func (p *sampledLFU) add(key string, cost uint64) {
	p.data[key] = cost
	p.used += cost
}

// tinyLFU is an admission helper that keeps track of access frequency using
// tiny (4-bit) counters in the form of a count-min sketch.
type tinyLFU struct {
	freq Sketch
	door *Filter
}

func newTinyLFU(maxItems uint64) *tinyLFU {
	return &tinyLFU{
		freq: NewCM(maxItems),
		door: NewFilter(maxItems, 0.01),
	}
}

func (p *tinyLFU) Push(keys []Element) {
	for _, key := range keys {
		p.Increment(string(key))
	}
}

func (p *tinyLFU) Estimate(key string) uint64 {
	hits := p.freq.Estimate(key)
	if p.door.Has(key) {
		hits += 1
	}
	return hits
}

func (p *tinyLFU) Increment(key string) {
	// doorkeeper (set if not already set)
	p.door.Set(key)
	// count-min
	p.freq.Increment(key)
}

func (p *tinyLFU) Reset() {
	p.door.Reset()
	p.freq.Reset()
}

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

func NewClairvoyant(maxBytes, maxItems uint64, cost bool) Policy {
	return newClairvoyant(maxItems, cost)
}

func newClairvoyant(maxItems uint64, cost bool) *Clairvoyant {
	return &Clairvoyant{
		log:      &PolicyLog{},
		capacity: maxItems,
		access:   make(map[string][]uint64, maxItems),
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

func (p *Clairvoyant) Push(keys []Element) {
	p.Lock()
	defer p.Unlock()
	for _, key := range keys {
		p.record(string(key))
	}
}

func (p *Clairvoyant) Add(key string, cost uint64) ([]string, bool) {
	p.Lock()
	defer p.Unlock()
	p.record(key)
	return nil, true
}

func (p *Clairvoyant) Has(key string) bool {
	p.Lock()
	defer p.Unlock()
	_, exists := p.access[key]
	return exists
}

func (p *Clairvoyant) Del(key string) {
	p.Lock()
	defer p.Unlock()
	delete(p.access, key)
}

func (p *Clairvoyant) Res() {
}

func (p *Clairvoyant) Cap() int64 {
	return 0
}

func (p *Clairvoyant) Log() *PolicyLog {
	p.Lock()
	defer p.Unlock()
	// data serves as the "pseudocache" with the ability to see into the future
	data := make(map[string]struct{}, p.capacity)
	size := uint64(0)
	for i, key := range p.future {
		// check if already exists
		if _, exists := data[key]; exists {
			p.log.Hit()
			continue
		}
		p.log.Miss()
		// check if eviction is needed
		if size == p.capacity {
			// eviction is needed
			//
			// collect item distances
			good := false
			distance := make(map[string]uint64, p.capacity)
			for k := range data {
				distance[k], good = p.distance(uint64(i), p.access[k])
				if !good {
					// there's no good distances because the key isn't used
					// again in the future, so we can just stop here and delete
					// it, and skip over the rest
					p.log.Evict()
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
			p.log.Evict()
			delete(data, maxKey)
			size--
		}
	add:
		// add the item
		data[key] = struct{}{}
		size++
	}
	return p.log
}

// recorder is a wrapper type useful for logging policy performance. Because hit
// ratio tracking adds substantial overhead (either by atomically incrementing
// counters or using policy-level mutexes), this struct allows us to only incur
// that overhead when we want to analyze the hit ratio performance.
type recorder struct {
	data   Map
	policy Policy
	log    *PolicyLog
}

func NewRecorder(policy Policy, data Map) Policy {
	return &recorder{
		data:   data,
		policy: policy,
		log:    &PolicyLog{},
	}
}

func (r *recorder) Push(keys []Element) {
	r.policy.Push(keys)
}

func (r *recorder) Add(key string, cost uint64) ([]string, bool) {
	if r.data.Get(key) != nil {
		r.log.Hit()
	} else {
		r.log.Miss()
	}
	victims, added := r.policy.Add(key, cost)
	if victims != nil {
		r.log.Evict()
	}
	return victims, added
}

func (r *recorder) Has(key string) bool {
	return r.policy.Has(key)
}

func (r *recorder) Del(key string) {
	r.policy.Del(key)
}

func (r *recorder) Res() {
	r.policy.Res()
}

func (r *recorder) Cap() int64 {
	return r.policy.Cap()
}

func (r *recorder) Log() *PolicyLog {
	return r.log
}

// PolicyLog is the struct for hit ratio statistics. Note that there is some
// cost to maintaining the counters, so it's best to wrap Policies via the
// Recorder type when hit ratio analysis is needed.
type PolicyLog struct {
	hits      uint64
	miss      uint64
	evictions uint64
}

func (p *PolicyLog) Hit() {
	atomic.AddUint64(&p.hits, 1)
}

func (p *PolicyLog) Miss() {
	atomic.AddUint64(&p.miss, 1)
}

func (p *PolicyLog) Evict() {
	atomic.AddUint64(&p.evictions, 1)
}

func (p *PolicyLog) GetHits() uint64 {
	return atomic.LoadUint64(&p.hits)
}

func (p *PolicyLog) GetMisses() uint64 {
	return atomic.LoadUint64(&p.miss)
}

func (p *PolicyLog) GetEvictions() uint64 {
	return atomic.LoadUint64(&p.evictions)
}

func (p *PolicyLog) Ratio() float64 {
	hits, misses := atomic.LoadUint64(&p.hits), atomic.LoadUint64(&p.miss)
	return float64(hits) / float64(hits+misses)
}
