/*
 * Copyright 2020 Dgraph Labs, Inc. and Contributors
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

	"github.com/tushar-zomato/ristretto/z"
)

type policy interface {
	CollectMetrics(metrics *Metrics)
	Push(keys []uint64) bool
	Add(key uint64, cost int64) ([]*Item, bool)
	Has(key uint64) bool
	Del(key uint64)
	Cap() int64
	Update(key uint64, cost int64)
	Cost(key uint64) int64
	Clear()
	Close()
	MaxCost() int64
	UpdateMaxCost(maxCost int64)
}

// keyCosts stores key-cost pairs.
type keyCosts struct {
	// NOTE: align maxCost to 64-bit boundary for use with atomic.
	// As per https://golang.org/pkg/sync/atomic/: "On ARM, x86-32,
	// and 32-bit MIPS, it is the caller’s responsibility to arrange
	// for 64-bit alignment of 64-bit words accessed atomically.
	// The first word in a variable or in an allocated struct, array,
	// or slice can be relied upon to be 64-bit aligned."
	maxCost  int64
	used     int64
	metrics  *Metrics
	keyCosts map[uint64]int64
}

func newSampledLFU(maxCost int64) *keyCosts {
	return &keyCosts{
		keyCosts: make(map[uint64]int64),
		maxCost:  maxCost,
	}
}

func (p *keyCosts) getMaxCost() int64 {
	return atomic.LoadInt64(&p.maxCost)
}

func (p *keyCosts) updateMaxCost(maxCost int64) {
	atomic.StoreInt64(&p.maxCost, maxCost)
}

func (p *keyCosts) roomLeft(cost int64) int64 {
	return p.getMaxCost() - (p.used + cost)
}

func (p *keyCosts) fillSample(in []*policyPair, evictionSampleSize int) []*policyPair {
	if len(in) >= evictionSampleSize {
		return in
	}

	for key, cost := range p.keyCosts {
		in = append(in, &policyPair{key, cost})
		if len(in) >= evictionSampleSize {
			return in
		}
	}

	return in
}

func (p *keyCosts) del(key uint64) {
	cost, ok := p.keyCosts[key]
	if !ok {
		return
	}
	p.used -= cost
	delete(p.keyCosts, key)
	p.metrics.add(costEvict, key, uint64(cost))
	p.metrics.add(keyEvict, key, 1)
}

func (p *keyCosts) add(key uint64, cost int64) {
	p.keyCosts[key] = cost
	p.used += cost
}

func (p *keyCosts) updateIfHas(key uint64, cost int64) bool {
	if prev, found := p.keyCosts[key]; found {
		// Update the cost of an existing key, but don't worry about evicting.
		// Evictions will be handled the next time a new item is added.
		p.metrics.add(keyUpdate, key, 1)
		if prev > cost {
			diff := prev - cost
			p.metrics.add(costAdd, key, ^uint64(uint64(diff)-1))
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

func (p *keyCosts) clear() {
	p.used = 0
	p.keyCosts = make(map[uint64]int64)
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
