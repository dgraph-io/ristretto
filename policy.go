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
	// Has is just an existence check (whether the key exists in the Policy).
	Has(string) bool
	// Del deletes the key from the Policy (in some Policies this does nothing
	// because they don't store keys, but it's useful otherwise).
	Del(string)
	Log() *PolicyLog
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

func NewClairvoyant(capacity uint64, data Map) Policy {
	return newClairvoyant(capacity, data)
}

func newClairvoyant(capacity uint64, data Map) *Clairvoyant {
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

func (p *Clairvoyant) Del(key string) {
	p.Lock()
	defer p.Unlock()
	delete(p.access, key)
}

func (p *Clairvoyant) Has(key string) bool {
	p.Lock()
	defer p.Unlock()
	if _, exists := p.access[key]; exists {
		return true
	}
	return false
}

func (p *Clairvoyant) Push(keys []Element) {
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

type seg struct {
	data map[string]uint8
	size uint64
}

func newSeg(size uint64) *seg {
	return &seg{
		data: make(map[string]uint8, size),
		size: size,
	}
}

func (s *seg) add(key string) (string, uint8) {
	victimKey, victimCount := s.candidate()
	if victimKey != "" {
		delete(s.data, victimKey)
	}
	s.data[key]++
	return victimKey, victimCount
}

func (s *seg) candidate() (string, uint8) {
	// check if eviction is needed
	if uint64(len(s.data)) != s.size {
		return "", 0
	}
	// sample items and find the minimally used
	i, minKey, minCount := 0, "", uint8(math.MaxUint8)
	for key, count := range s.data {
		if count < minCount {
			minKey, minCount = key, count
		}
		if i++; i == lfuSample {
			break
		}
	}
	return minKey, minCount
}

// LFU is a Policy with no admission policy and a sampled LFU eviction policy.
type LFU struct {
	sync.Mutex
	freq *seg
}

func NewLFU(size uint64, data Map) Policy {
	return newLFU(size, data)
}

func newLFU(size uint64, data Map) *LFU {
	return &LFU{
		freq: newSeg(size),
	}
}

func (p *LFU) Push(keys []Element) {
	p.Lock()
	defer p.Unlock()
	for _, key := range keys {
		k := string(key)
		if _, exists := p.freq.data[k]; exists {
			p.freq.data[k]++
		}
	}
}

func (p *LFU) Add(key string) (victim string, added bool) {
	p.Lock()
	defer p.Unlock()
	if _, exists := p.freq.data[key]; exists {
		return
	}
	victim, _ = p.freq.add(key)
	added = true
	return
}

func (p *LFU) Del(key string) {
	p.Lock()
	defer p.Unlock()
	delete(p.freq.data, key)
	return
}

func (p *LFU) Has(key string) bool {
	p.Lock()
	defer p.Unlock()
	_, exists := p.freq.data[key]
	return exists
}

func (p *LFU) Log() *PolicyLog {
	return nil
}

type WLFU struct {
	sync.Mutex
	segs [2]*seg
}

func NewWLFU(size uint64, data Map) Policy {
	return newWLFU(size)
}

func newWLFU(size uint64) *WLFU {
	return &WLFU{
		segs: [2]*seg{
			// window segment (1% of total size as per TinyLFU paper)
			newSeg(uint64(math.Ceil(float64(size) * 0.01))),
			// main  segment (99% of total size as per TinyLFU paper)
			newSeg(uint64(math.Floor(float64(size) * 0.99))),
		},
	}
}

func (p *WLFU) Push(keys []Element) {
	p.Lock()
	defer p.Unlock()
	for _, key := range keys {
		k := string(key)
		s := p.seg(k)
		if s == -1 {
			continue
		}
		if _, exists := p.segs[s].data[k]; exists {
			p.segs[s].data[k]++
		}
	}
}

func (p *WLFU) Add(key string) (victim string, added bool) {
	p.Lock()
	defer p.Unlock()
	// do nothing if the item is already in the policy
	if p.seg(key) != -1 {
		return
	}
	// get the window victim and count if the window is full
	windowKey, windowCount := p.segs[0].add(key)
	if windowKey == "" {
		// window has room, so nothing else needs to be done
		added = true
		return
	}
	// get the main eviction candidate key and count if the main segment is full
	mainKey, mainCount := p.segs[1].candidate()
	if mainKey == "" {
		// main has room, so just move the window victim to there
		goto move
	}
	// compare the window victim with the main candidate, also note that window
	// victims are preferred (>=) over main candidates, as we can assume that
	// window victims have been used more recently than the main candidate
	if windowCount >= mainCount {
		// main candidate lost to the window victim, so actually evict the main
		// candidate
		victim = mainKey
		delete(p.segs[1].data, mainKey)
		// main now has room for one more, so move the window victim to there
		goto move
	} else {
		// window victim lost to the main candidate, and the window victim has
		// already been evicted from the window, so nothing else needs to be
		// done
		victim = windowKey
		added = true
	}
	return
move:
	// move places the window key-count pair in the main segment
	p.segs[1].data[windowKey] = windowCount
	added = true
	return
}

func (p *WLFU) Has(key string) bool {
	return p.seg(key) != -1
}

func (p *WLFU) Del(key string) {
	p.Lock()
	defer p.Unlock()
	if seg := p.seg(key); seg != -1 {
		delete(p.segs[seg].data, key)
	}
	return
}

func (p *WLFU) Log() *PolicyLog {
	return nil
}

func (p *WLFU) seg(key string) int {
	if p.segs[0].data[key] != 0 {
		return 0
	} else if p.segs[1].data[key] != 0 {
		return 1
	}
	return -1
}

// TinyLFU keeps track of frequency using tiny (4-bit) counters in the form of a
// counting bloom filter. For eviction, sampled LFU is done.
type TinyLFU struct {
	sync.Mutex
	data Map
	size uint64
	used uint64
	freq Sketch
}

func NewTinyLFU(size uint64, data Map) Policy {
	return newTinyLFU(size, data)
}

func newTinyLFU(size uint64, data Map) *TinyLFU {
	return &TinyLFU{
		data: data,
		size: size,
		freq: NewCM(size),
	}
}

func (p *TinyLFU) Del(key string) {
	p.Lock()
	defer p.Unlock()
	p.used--
}

func (p *TinyLFU) Has(key string) bool {
	p.Lock()
	defer p.Unlock()
	return p.data.Get(key) != nil
}

func (p *TinyLFU) Push(keys []Element) {
	p.Lock()
	defer p.Unlock()
	for _, key := range keys {
		p.freq.Increment(string(key))
	}
}

func (p *TinyLFU) Add(key string) (victim string, added bool) {
	p.Lock()
	defer p.Unlock()
	if p.used == p.size {
		victim = p.candidate()
		p.used--
	}
	p.freq.Increment(key)
	added = true
	p.used++
	return
}

func (p *TinyLFU) Log() *PolicyLog {
	return nil
}

func (p *TinyLFU) candidate() string {
	keys := p.sample()
	minKey, minHits := "", uint64(math.MaxUint64)
	for _, key := range keys {
		if key == "" {
			continue
		}
		if hits := p.freq.Estimate(key); hits < minHits {
			minKey, minHits = key, hits
		}
	}
	return minKey
}

func (p *TinyLFU) sample() []string {
	keys, i := make([]string, 0, lfuSample), 0
	p.data.Run(func(key, value interface{}) bool {
		keys = append(keys, key.(string))
		i++
		return !(i == lfuSample)
	})
	return keys
}

// LRU is a Policy with no admission policy and a LRU eviction policy (using
// doubly linked list).
type LRU struct {
	sync.Mutex
	list     *list.List
	look     map[string]*list.Element
	capacity uint64
	size     uint64
}

func NewLRU(capacity uint64, data Map) Policy {
	return newLRU(capacity, data)
}

func newLRU(capacity uint64, data Map) *LRU {
	return &LRU{
		list:     list.New(),
		look:     make(map[string]*list.Element, capacity),
		capacity: capacity,
	}
}

func (p *LRU) Del(key string) {
	p.Lock()
	defer p.Unlock()
	delete(p.look, key)
	p.size--
}

func (p *LRU) Has(key string) bool {
	p.Lock()
	defer p.Unlock()
	if _, exists := p.look[key]; exists {
		return true
	}
	return false
}

func (p *LRU) Push(keys []Element) {
	p.Lock()
	defer p.Unlock()
	for _, key := range keys {
		if element, exists := p.look[string(key)]; exists {
			p.list.MoveToFront(element)
		}
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
		p.size--
	}
	// add the new key to the list
	p.look[key] = p.list.PushFront(key)
	added = true
	p.size++
	return
}

func (p *LRU) Log() *PolicyLog {
	return nil
}

func (p *LRU) String() string {
	out := "["
	for element := p.list.Front(); element != nil; element = element.Next() {
		out += element.Value.(string) + ", "
	}
	return out[:len(out)-2] + "]"
}

// None is a policy that does nothing.
type None struct {
	log *PolicyLog
}

func NewNone(capacity uint64, data Map) Policy {
	return newNone(capacity, data)
}

func newNone(capacity uint64, data Map) *None {
	return &None{
		log: &PolicyLog{},
	}
}

func (p *None) Del(key string) {
}

func (p *None) Has(key string) bool {
	return false
}

func (p *None) Push(keys []Element) {
}

func (p *None) Add(key string) (victim string, added bool) {
	return
}

func (p *None) Log() *PolicyLog {
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

func (r *recorder) Del(key string) {
	r.policy.Del(key)
}

func (r *recorder) Has(key string) bool {
	return r.policy.Has(key)
}

func (r *recorder) Push(keys []Element) {
	r.policy.Push(keys)
}

func (r *recorder) Add(key string) (string, bool) {
	if r.data.Get(key) != nil {
		r.log.Hit()
	} else {
		r.log.Miss()
	}
	victim, added := r.policy.Add(key)
	if victim != "" {
		r.log.Evict()
	}
	return victim, added
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
