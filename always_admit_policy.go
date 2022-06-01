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
	"math"
	"sync"
)

// lfuAlwaysAdmitPolicy encapsulates eviction/admission behavior.
type lfuAlwaysAdmitPolicy struct {
	sync.Mutex
	admit         *tinyLFU
	costs         *keyCosts
	itemsCh       chan []uint64
	stop          chan struct{}
	isClosed      bool
	metrics       *Metrics
	lfuSampleSize int
}

func newAlwaysAdmitPolicy(numCounters, maxCost int64, lfuSampleSize int) *lfuAlwaysAdmitPolicy {
	p := &lfuAlwaysAdmitPolicy{
		admit:         newTinyLFU(numCounters),
		costs:         newSampledLFU(maxCost),
		itemsCh:       make(chan []uint64, 3),
		stop:          make(chan struct{}),
		lfuSampleSize: lfuSampleSize,
	}

	go p.processItems()
	return p
}

func (p *lfuAlwaysAdmitPolicy) CollectMetrics(metrics *Metrics) {
	p.metrics = metrics
	p.costs.metrics = metrics
}

func (p *lfuAlwaysAdmitPolicy) processItems() {
	for {
		select {
		case items := <-p.itemsCh:
			p.Lock()
			p.admit.Push(items)
			p.Unlock()
		case <-p.stop:
			return
		}
	}
}

func (p *lfuAlwaysAdmitPolicy) Push(keys []uint64) bool {
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
func (p *lfuAlwaysAdmitPolicy) Add(key uint64, cost int64) ([]*Item, bool) {
	p.Lock()
	defer p.Unlock()

	// Cannot add an item bigger than entire cache.
	if cost > p.costs.getMaxCost() {
		return nil, false
	}

	// No need to go any further if the item is already in the cache.
	if has := p.costs.updateIfHas(key, cost); has {
		// An update does not count as an addition, so return false.
		return nil, false
	}

	// If the execution reaches this point, the key doesn't exist in the cache.
	// Calculate the remaining room in the cache (usually bytes).
	room := p.costs.roomLeft(cost)
	if room >= 0 {
		// There's enough room in the cache to store the new item without
		// overflowing. Do that now and stop here.
		p.costs.add(key, cost)
		p.metrics.add(costAdd, key, uint64(cost))
		return nil, true
	}

	// sample is the eviction candidate pool to be filled via random sampling.
	// TODO: perhaps we should use a min heap here. Right now our time
	// complexity is N for finding the min. Min heap should bring it down to
	// O(lg N).
	sample := make([]*policyPair, 0, p.lfuSampleSize)
	// As items are evicted they will be appended to victims.
	victims := make([]*Item, 0)

	// Delete victims until there's enough space or a minKey is found that has
	// more hits than incoming item.
	for ; room < 0; room = p.costs.roomLeft(cost) {
		// Fill up empty slots in sample.
		sample = p.costs.fillSample(sample, p.lfuSampleSize)

		// Find minimally used item in sample.
		minKey, minHits, minId, minCost := uint64(0), int64(math.MaxInt64), 0, int64(0)
		for i, pair := range sample {
			// Look up hit count for sample key.
			//
			if hits := p.admit.Estimate(pair.key); hits < minHits {
				minKey, minHits, minId, minCost = pair.key, hits, i, pair.cost
			}
		}

		// Delete the victim from metadata.
		p.costs.del(minKey)

		// Delete the victim from sample.
		sample[minId] = sample[len(sample)-1]
		sample = sample[:len(sample)-1]
		// Store victim in evicted victims slice.
		victims = append(victims, &Item{
			Key:      minKey,
			Conflict: 0,
			Cost:     minCost,
		})
	}

	p.costs.add(key, cost)
	p.metrics.add(costAdd, key, uint64(cost))
	return victims, true
}

func (p *lfuAlwaysAdmitPolicy) Has(key uint64) bool {
	p.Lock()
	_, exists := p.costs.keyCosts[key]
	p.Unlock()
	return exists
}

func (p *lfuAlwaysAdmitPolicy) Del(key uint64) {
	p.Lock()
	p.costs.del(key)
	p.Unlock()
}

func (p *lfuAlwaysAdmitPolicy) Cap() int64 {
	p.Lock()
	capacity := int64(p.costs.getMaxCost() - p.costs.used)
	p.Unlock()
	return capacity
}

func (p *lfuAlwaysAdmitPolicy) Update(key uint64, cost int64) {
	p.Lock()
	p.costs.updateIfHas(key, cost)
	p.Unlock()
}

func (p *lfuAlwaysAdmitPolicy) Cost(key uint64) int64 {
	p.Lock()
	if cost, found := p.costs.keyCosts[key]; found {
		p.Unlock()
		return cost
	}
	p.Unlock()
	return -1
}

func (p *lfuAlwaysAdmitPolicy) Clear() {
	p.Lock()
	p.admit.clear()
	p.costs.clear()
	p.Unlock()
}

func (p *lfuAlwaysAdmitPolicy) Close() {
	if p.isClosed {
		return
	}

	// Block until the p.processItems goroutine returns.
	p.stop <- struct{}{}
	close(p.stop)
	close(p.itemsCh)
	p.isClosed = true
}

func (p *lfuAlwaysAdmitPolicy) MaxCost() int64 {
	if p == nil || p.costs == nil {
		return 0
	}
	return p.costs.getMaxCost()
}

func (p *lfuAlwaysAdmitPolicy) UpdateMaxCost(maxCost int64) {
	if p == nil || p.costs == nil {
		return
	}
	p.costs.updateMaxCost(maxCost)
}
