/*
 * SPDX-FileCopyrightText: © 2017-2025 Istari Digital, Inc.
 * SPDX-License-Identifier: Apache-2.0
 */

package ristretto

import (
	"math"
	"sync"
	"sync/atomic"

	"github.com/dgraph-io/ristretto/v2/z"
)

const (
	// lfuSample is the number of items to sample when looking at eviction
	// candidates. 5 seems to be the most optimal number [citation needed].
	lfuSample = 5
)

func newPolicy[V any](numCounters, maxCost int64) *defaultPolicy[V] {
	return newDefaultPolicy[V](numCounters, maxCost)
}

type defaultPolicy[V any] struct {
	admitMu  sync.Mutex
	evictMu  sync.RWMutex
	admit    *tinyLFU
	evict    *sampledLFU
	itemsCh  chan []uint64
	stop     chan struct{}
	done     chan struct{}
	isClosed bool
	metrics  *Metrics
}

func newDefaultPolicy[V any](numCounters, maxCost int64) *defaultPolicy[V] {
	p := &defaultPolicy[V]{
		admit:   newTinyLFU(numCounters),
		evict:   newSampledLFU(maxCost, numCounters),
		itemsCh: make(chan []uint64, 3),
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
	go p.processItems()
	return p
}

func (p *defaultPolicy[V]) CollectMetrics(metrics *Metrics) {
	p.metrics = metrics
	p.evict.metrics = metrics
}

type policyPair struct {
	key  uint64
	cost int64
}

func (p *defaultPolicy[V]) processItems() {
	for {
		select {
		case items := <-p.itemsCh:
			p.admitMu.Lock()
			p.admit.Push(items)
			p.admitMu.Unlock()
		case <-p.stop:
			p.done <- struct{}{}
			return
		}
	}
}

func (p *defaultPolicy[V]) Push(keys []uint64) bool {
	if p.isClosed {
		return false
	}

	if len(keys) == 0 {
		return true
	}

	select {
	case p.itemsCh <- keys:
		p.metrics.add(keepGets, keys[0], uint64(len(keys)))
		return true
	default:
		p.metrics.add(dropGets, keys[0], uint64(len(keys)))
		return false
	}
}

// Add decides whether the item with the given key and cost should be accepted by
// the policy. It returns the list of victims that have been evicted and a boolean
// indicating whether the incoming item should be accepted.
func (p *defaultPolicy[V]) Add(key uint64, cost int64) ([]*Item[V], bool) {
	// Cannot add an item bigger than entire cache.
	if cost > p.evict.getMaxCost() {
		return nil, false
	}

	var victims []*Item[V]
	incHits, hasIncHits := int64(0), false

	for {
		p.evictMu.Lock()
		// No need to go any further if the item is already in the cache.
		if has := p.evict.updateIfHas(key, cost); has {
			p.evictMu.Unlock()
			// An update does not count as an addition, so return false.
			return victims, false
		}

		// If the execution reaches this point, the key doesn't exist in the cache.
		// Calculate the remaining room in the cache (usually bytes).
		room := p.evict.roomLeft(cost)
		if room >= 0 {
			// There's enough room in the cache to store the new item without
			// overflowing. Do that now and stop here.
			p.evict.add(key, cost)
			p.metrics.add(costAdd, key, uint64(cost))
			p.evictMu.Unlock()
			return victims, true
		}

		if victims == nil {
			victims = make([]*Item[V], 0)
		}

		// sample is the eviction candidate pool to be filled via random sampling.
		// TODO: perhaps we should use a min heap here. Right now our time
		// complexity is N for finding the min. Min heap should bring it down to
		// O(lg N).

		// Fill up empty slots in sample.
		sample := p.evict.fillSample(make([]*policyPair, 0, lfuSample))
		p.evictMu.Unlock()

		if !hasIncHits {
			incHits = p.estimate(key)
			hasIncHits = true
		}

		// Find minimally used item in sample.
		minPair, minHits, ok := p.minSample(sample)

		// If the incoming item isn't worth keeping in the policy, reject.
		if !ok || incHits < minHits {
			p.metrics.add(rejectSets, key, 1)
			return victims, false
		}

		// Delete the victim from metadata.
		p.evictMu.Lock()
		minCost, evicted := p.evict.del(minPair.key)
		p.evictMu.Unlock()
		if evicted {
			// Store victim in evicted victims slice.
			victims = append(victims, &Item[V]{
				Key:      minPair.key,
				Conflict: 0,
				Cost:     minCost,
			})
		}
	}
}

func (p *defaultPolicy[V]) estimate(key uint64) int64 {
	p.admitMu.Lock()
	hits := p.admit.Estimate(key)
	p.admitMu.Unlock()
	return hits
}

func (p *defaultPolicy[V]) minSample(sample []*policyPair) (*policyPair, int64, bool) {
	if len(sample) == 0 {
		return nil, 0, false
	}

	p.admitMu.Lock()
	var minPair *policyPair
	minHits := int64(math.MaxInt64)
	for _, pair := range sample {
		if hits := p.admit.Estimate(pair.key); hits < minHits {
			minPair, minHits = pair, hits
		}
	}
	p.admitMu.Unlock()
	return minPair, minHits, true
}

func (p *defaultPolicy[V]) Has(key uint64) bool {
	p.evictMu.RLock()
	_, exists := p.evict.keyCosts[key]
	p.evictMu.RUnlock()
	return exists
}

func (p *defaultPolicy[V]) Del(key uint64) {
	p.evictMu.Lock()
	p.evict.del(key)
	p.evictMu.Unlock()
}

func (p *defaultPolicy[V]) Cap() int64 {
	p.evictMu.RLock()
	capacity := p.evict.getMaxCost() - p.evict.used
	p.evictMu.RUnlock()
	return capacity
}

func (p *defaultPolicy[V]) Update(key uint64, cost int64) {
	p.evictMu.Lock()
	p.evict.updateIfHas(key, cost)
	p.evictMu.Unlock()
}

func (p *defaultPolicy[V]) Cost(key uint64) int64 {
	p.evictMu.RLock()
	if cost, found := p.evict.keyCosts[key]; found {
		p.evictMu.RUnlock()
		return cost
	}
	p.evictMu.RUnlock()
	return -1
}

func (p *defaultPolicy[V]) Clear() {
	p.admitMu.Lock()
	p.evictMu.Lock()
	p.admit.clear()
	p.evict.clear()
	p.evictMu.Unlock()
	p.admitMu.Unlock()
}

func (p *defaultPolicy[V]) Close() {
	if p.isClosed {
		return
	}

	// Block until the p.processItems goroutine returns.
	p.stop <- struct{}{}
	<-p.done
	close(p.stop)
	close(p.done)
	close(p.itemsCh)
	p.isClosed = true
}

func (p *defaultPolicy[V]) MaxCost() int64 {
	if p == nil || p.evict == nil {
		return 0
	}
	return p.evict.getMaxCost()
}

func (p *defaultPolicy[V]) UpdateMaxCost(maxCost int64) {
	if p == nil || p.evict == nil {
		return
	}
	p.evict.updateMaxCost(maxCost)
}

// sampledLFU is an eviction helper storing key-cost pairs.
type sampledLFU struct {
	// NOTE: align maxCost to 64-bit boundary for use with atomic.
	// As per https://golang.org/pkg/sync/atomic/: "On ARM, x86-32,
	// and 32-bit MIPS, it is the caller’s responsibility to arrange
	// for 64-bit alignment of 64-bit words accessed atomically.
	// The first word in a variable or in an allocated struct, array,
	// or slice can be relied upon to be 64-bit aligned."
	maxCost  int64
	used     int64
	keyCap   int64
	metrics  *Metrics
	keyCosts map[uint64]int64
}

func newSampledLFU(maxCost, numCounters int64) *sampledLFU {
	// Pre-allocate keyCosts assuming the recommended 10:1 counter-to-item ratio
	// to minimize map resizing.
	capacity := numCounters / 10
	return &sampledLFU{
		keyCosts: make(map[uint64]int64, capacity),
		maxCost:  maxCost,
		keyCap:   capacity,
	}
}

func (p *sampledLFU) getMaxCost() int64 {
	return atomic.LoadInt64(&p.maxCost)
}

func (p *sampledLFU) updateMaxCost(maxCost int64) {
	atomic.StoreInt64(&p.maxCost, maxCost)
}

func (p *sampledLFU) roomLeft(cost int64) int64 {
	return p.getMaxCost() - (p.used + cost)
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

func (p *sampledLFU) del(key uint64) (int64, bool) {
	cost, ok := p.keyCosts[key]
	if !ok {
		return 0, false
	}
	p.used -= cost
	delete(p.keyCosts, key)
	p.metrics.add(costEvict, key, uint64(cost))
	p.metrics.add(keyEvict, key, 1)
	return cost, true
}

func (p *sampledLFU) add(key uint64, cost int64) {
	p.keyCosts[key] = cost
	p.used += cost
}

func (p *sampledLFU) updateIfHas(key uint64, cost int64) bool {
	if prev, found := p.keyCosts[key]; found {
		// Update the cost of an existing key, but don't worry about evicting.
		// Evictions will be handled the next time a new item is added.
		p.metrics.add(keyUpdate, key, 1)
		if prev > cost {
			diff := prev - cost
			p.metrics.add(costAdd, key, ^(uint64(diff) - 1))
		} else if cost > prev {
			diff := cost - prev
			p.metrics.add(costAdd, key, uint64(diff))
		}
		p.used += cost - prev
		p.keyCosts[key] = cost
		return true
	}
	return false
}

func (p *sampledLFU) clear() {
	p.used = 0
	p.keyCosts = make(map[uint64]int64, p.keyCap)
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

func (p *tinyLFU) Push(keys []uint64) {
	for _, key := range keys {
		p.Increment(key)
	}
}

func (p *tinyLFU) Estimate(key uint64) int64 {
	hits := p.freq.Estimate(key)
	if p.door.Has(key) {
		hits++
	}
	return hits
}

func (p *tinyLFU) Increment(key uint64) {
	// Flip doorkeeper bit if not already done.
	if added := p.door.AddIfNotHas(key); !added {
		// Increment count-min counter if doorkeeper bit is already set.
		p.freq.Increment(key)
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

func (p *tinyLFU) clear() {
	p.incrs = 0
	p.door.Clear()
	p.freq.Clear()
}
