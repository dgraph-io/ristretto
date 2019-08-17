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
	"container/heap"
	"testing"

	"github.com/dgraph-io/ristretto/bench/sim"
)

type TestCache interface {
	Get(interface{}) (interface{}, bool)
	Set(interface{}, interface{}, int64) bool
}

const capacity = 1000

func newCache(metrics bool) *Cache {
	cache, err := NewCache(&Config{
		NumCounters: capacity * 10,
		MaxCost:     capacity,
		BufferItems: 64,
		Metrics:     metrics,
	})
	if err != nil {
		panic(err)
	}
	return cache
}

func newBenchmark(bencher func(uint64)) func(b *testing.B) {
	return func(b *testing.B) {
		b.SetParallelism(1)
		b.SetBytes(1)
		b.ResetTimer()
		/*
			b.RunParallel(func(pb *testing.PB) {
				for i := uint64(0); pb.Next(); i++ {
					bencher(i)
				}
			})
		*/
		for n := uint64(0); n < uint64(b.N); n++ {
			bencher(n)
		}
	}
}

func BenchmarkCacheGetOne(b *testing.B) {
	cache := newCache(false)
	cache.Set(1, nil, 1)
	newBenchmark(func(i uint64) { cache.Get(1) })(b)
}

func BenchmarkCacheSetOne(b *testing.B) {
	cache := newCache(false)
	newBenchmark(func(i uint64) { cache.Set(1, nil, 1) })(b)
}

func BenchmarkCacheSetUni(b *testing.B) {
	cache := newCache(false)
	newBenchmark(func(i uint64) { cache.Set(i, nil, 1) })(b)
}

func newRatioTest(cache TestCache) func(t *testing.T) {
	return func(t *testing.T) {
		keys := sim.NewZipfian(1.0001, 1, capacity*100)
		for i := 0; i < capacity*1000; i++ {
			key, err := keys()
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := cache.Get(key); !ok {
				cache.Set(key, nil, 1)
			}
		}
	}
}

func TestCacheHits(t *testing.T) {
	cache := newCache(true)
	newRatioTest(cache)(t)
	metrics := cache.Metrics()
	t.Logf(" ristretto: %.2f\n", metrics.Ratio())
	t.Logf("- d. sets: %.6f\n", float64(metrics.Get(dropSets))/float64(metrics.Get(dropSets)+metrics.Get(keyAdd)+metrics.Get(keyUpdate)))
	t.Logf("- d. gets: %.6f\n", float64(metrics.Get(dropGets))/float64(metrics.Get(dropGets)+metrics.Get(keepGets)))
	optimal := NewClairvoyant(capacity)
	newRatioTest(optimal)(t)
	t.Logf("  optimal: %.2f\n", optimal.Log())
}

type Clairvoyant struct {
	capacity uint64
	hits     map[uint64]uint64
	access   []uint64
}

func NewClairvoyant(capacity uint64) *Clairvoyant {
	return &Clairvoyant{
		capacity: capacity,
		hits:     make(map[uint64]uint64),
		access:   make([]uint64, 0),
	}
}

func (c *Clairvoyant) Get(key interface{}) (interface{}, bool) {
	c.hits[key.(uint64)]++
	c.access = append(c.access, key.(uint64))
	return nil, false
}

func (c *Clairvoyant) Set(key, value interface{}, cost int64) bool {
	return false
}

func (c *Clairvoyant) Log() float64 {
	hits, misses, evictions := uint64(0), uint64(0), uint64(0)
	look := make(map[uint64]struct{}, c.capacity)
	data := &clairvoyantHeap{}
	heap.Init(data)
	for _, key := range c.access {
		if _, has := look[key]; has {
			hits++
			continue
		}
		if uint64(data.Len()) >= c.capacity {
			victim := heap.Pop(data)
			delete(look, victim.(*clairvoyantItem).key)
			evictions++
		}
		misses++
		look[key] = struct{}{}
		heap.Push(data, &clairvoyantItem{key, c.hits[key]})
	}
	return float64(hits) / float64(hits+misses)
}

type clairvoyantItem struct {
	key  uint64
	hits uint64
}

type clairvoyantHeap []*clairvoyantItem

func (h clairvoyantHeap) Len() int           { return len(h) }
func (h clairvoyantHeap) Less(i, j int) bool { return h[i].hits < h[j].hits }
func (h clairvoyantHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *clairvoyantHeap) Push(x interface{}) {
	*h = append(*h, x.(*clairvoyantItem))
}

func (h *clairvoyantHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
