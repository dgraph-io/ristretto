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

	"github.com/dgraph-io/ristretto/z"
)

const (
	// lfuSample is the number of items to sample when looking at eviction
	// candidates. 5 seems to be the most optimal number [citation needed].
	lfuSample = 5
)

// Policy is the interface encapsulating eviction/admission behavior.
type Policy interface {
	ringConsumer
	// Add attempts to Add the key-cost pair to the Policy. It returns a slice
	// of evicted keys and a bool denoting whether or not the key-cost pair
	// was added. If it returns true, the key should be stored in cache.
	Add(string, int64) ([]string, bool)
	// Has returns true if the key exists in the Policy.
	Has(string) bool
	// Del deletes the key from the Policy.
	Del(string)
	// Cap returns the available capacity.
	Cap() int64
	Log() *PolicyLog
}

func newPolicy(numCounters, maxCost int64) Policy {
	return &defaultPolicy{
		admit: newTinyLFU(numCounters),
		evict: newSampledLFU(maxCost, numCounters),
	}
}

// defaultPolicy is the default defaultPolicy, which is currently TinyLFU admission with
// sampledLFU eviction.
type defaultPolicy struct {
	sync.Mutex
	admit *tinyLFU
	evict *sampledLFU
}

type policyPair struct {
	key  string
	cost int64
}

func (p *defaultPolicy) Push(keys []string) {
	p.Lock()
	defer p.Unlock()
	p.admit.Push(keys)
}

func (p *defaultPolicy) Add(key string, cost int64) ([]string, bool) {
	p.Lock()
	defer p.Unlock()

	// can't add an item bigger than entire cache
	if cost > p.evict.size {
		return nil, false
	}
	if _, exists := p.evict.keyCosts[key]; exists {
		return nil, true
	}
	// Calculate how much room do we have in the cache.
	room := p.evict.roomLeft(cost)
	if room >= 0 {
		// There's room in the cache.
		p.evict.add(key, cost)
		return nil, true
	}
	// incHits is the hit count for the incoming item
	incHits := p.admit.Estimate(key)
	// sample is the eviction candidate pool to be filled via random sampling
	// TODO: perhaps we should use a min heap here. Right now our time complexity is N for finding
	// the min. Min heap should bring it down to O(lg N).
	sample := make([]*policyPair, 0, lfuSample)

	// Victims contains keys that have already been evicted
	var victims []string
	// Delete victims until there's enough space or a minKey is found that has
	// more hits than incoming item.
	for ; room < 0; room = p.evict.roomLeft(cost) {
		// fill up empty slots in sample
		sample = p.evict.fillSample(sample)
		// find minimally used item in sample
		minKey, minHits, minId := "", int64(math.MaxInt64), 0
		for i, pair := range sample {
			// look up hit count for sample key
			if hits := p.admit.Estimate(pair.key); hits < minHits {
				minKey, minHits, minId = pair.key, hits, i
			}
		}
		// if the incoming item isn't worth keeping in the policy, stop
		if incHits < minHits {
			return victims, false
		}
		// delete the victim from metadata
		p.evict.del(minKey)
		// delete the victim from sample
		sample[minId] = sample[len(sample)-1]
		sample = sample[:len(sample)-1]
		// store victim in evicted victims slice
		victims = append(victims, minKey)
	}
	p.evict.add(key, cost)
	return victims, true
}

func (p *defaultPolicy) Has(key string) bool {
	p.Lock()
	defer p.Unlock()
	_, exists := p.evict.keyCosts[key]
	return exists
}

func (p *defaultPolicy) Del(key string) {
	p.Lock()
	defer p.Unlock()
	p.evict.del(key)
}

func (p *defaultPolicy) Cap() int64 {
	p.Lock()
	defer p.Unlock()
	return int64(p.evict.size - p.evict.used)
}

func (p *defaultPolicy) Log() *PolicyLog {
	return nil
}

// sampledLFU is an eviction helper storing key-cost pairs.
type sampledLFU struct {
	keyCosts map[string]int64
	size     int64
	used     int64
}

func newSampledLFU(numCounters, maxCost int64) *sampledLFU {
	return &sampledLFU{
		keyCosts: make(map[string]int64, numCounters),
		size:     maxCost,
	}
}

func (p *sampledLFU) roomLeft(cost int64) int64 {
	return p.size - (p.used + cost)
}

func (p *sampledLFU) fillSample(in []*policyPair) []*policyPair {
	if len(in) >= lfuSample {
		return in
	}
	for key, cost := range p.keyCosts {
		in = append(in, &policyPair{key, cost})
		if len(in) >= lfuSample {
			return in
		}
	}
	return in
}

func (p *sampledLFU) del(key string) {
	p.used -= p.keyCosts[key]
	delete(p.keyCosts, key)
}

func (p *sampledLFU) add(key string, cost int64) {
	p.keyCosts[key] = cost
	p.used += cost
}

// tinyLFU is an admission helper that keeps track of access frequency using
// tiny (4-bit) counters in the form of a count-min sketch.
// tinyLFU is NOT thread safe.
type tinyLFU struct {
	freq    *cmSketch
	door    *z.Bloom
	incrs   int64
	resetAt int64
}

func newTinyLFU(numCounters int64) *tinyLFU {
	return &tinyLFU{
		freq:    newCmSketch(numCounters),
		door:    z.NewBloomFilter(float64(numCounters), 0.01),
		resetAt: numCounters,
	}
}

func (p *tinyLFU) Push(keys []string) {
	for _, key := range keys {
		p.Increment(key)
	}
}

func (p *tinyLFU) Estimate(key string) int64 {
	hash := z.AESHashString(key)
	hits := p.freq.Estimate(hash)
	if p.door.Has(hash) {
		hits += 1
	}
	return hits
}

func (p *tinyLFU) Increment(key string) {
	// flip doorkeeper bit if not already
	hash := z.AESHashString(key)
	if added := p.door.AddIfNotHas(hash); !added {
		// increment count-min counter if doorkeeper bit is already set.
		p.freq.Increment(hash)
	}
	p.incrs++
	if p.incrs >= p.resetAt {
		p.reset()
	}
}

func (p *tinyLFU) reset() {
	// Zero out incrs.
	p.incrs = 0
	// clears doorkeeper bits
	p.door.Clear()
	// halves count-min counters
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
	time     int64
	log      *PolicyLog
	access   map[string][]int64
	capacity int64
	future   []string
}

func newClairvoyant(numCounters, maxCost int64) Policy {
	return &Clairvoyant{
		log:      &PolicyLog{},
		capacity: numCounters,
		access:   make(map[string][]int64, numCounters),
	}
}

// distance finds the "time distance" from the start position to the minimum
// time value - this is used to judge eviction candidates.
func (p *Clairvoyant) distance(start int64, times []int64) (int64, bool) {
	good, min := false, int64(0)
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
		p.access[key] = make([]int64, 0)
	}
	p.access[key] = append(p.access[key], p.time)
	p.future = append(p.future, key)
}

func (p *Clairvoyant) Push(keys []string) {
	p.Lock()
	defer p.Unlock()
	for _, key := range keys {
		p.record(string(key))
	}
}

func (p *Clairvoyant) Add(key string, cost int64) ([]string, bool) {
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

func (p *Clairvoyant) Cap() int64 {
	return 0
}

func (p *Clairvoyant) Log() *PolicyLog {
	p.Lock()
	defer p.Unlock()
	// data serves as the "pseudocache" with the ability to see into the future
	data := make(map[string]struct{}, p.capacity)
	size := int64(0)
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
			distance := make(map[string]int64, p.capacity)
			for k := range data {
				distance[k], good = p.distance(int64(i), p.access[k])
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
			maxDistance, maxKey, c := int64(0), "", 0
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
	data   store
	policy Policy
	log    *PolicyLog
}

func NewRecorder(policy Policy, data store) Policy {
	return &recorder{
		data:   data,
		policy: policy,
		log:    &PolicyLog{},
	}
}

func (r *recorder) Push(keys []string) {
	r.policy.Push(keys)
}

func (r *recorder) Add(key string, cost int64) ([]string, bool) {
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
	hits      int64
	miss      int64
	evictions int64
}

func (p *PolicyLog) Hit() {
	atomic.AddInt64(&p.hits, 1)
}

func (p *PolicyLog) Miss() {
	atomic.AddInt64(&p.miss, 1)
}

func (p *PolicyLog) Evict() {
	atomic.AddInt64(&p.evictions, 1)
}

func (p *PolicyLog) GetHits() int64 {
	return atomic.LoadInt64(&p.hits)
}

func (p *PolicyLog) GetMisses() int64 {
	return atomic.LoadInt64(&p.miss)
}

func (p *PolicyLog) GetEvictions() int64 {
	return atomic.LoadInt64(&p.evictions)
}

func (p *PolicyLog) Ratio() float64 {
	hits, misses := atomic.LoadInt64(&p.hits), atomic.LoadInt64(&p.miss)
	return float64(hits) / float64(hits+misses)
}
