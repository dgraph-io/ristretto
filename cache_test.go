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
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/dgraph-io/ristretto/bench/sim"
)

// TestCache is used to pass instances of Ristretto and Clairvoyant around and
// compare their performance.
type TestCache interface {
	Get(interface{}) (interface{}, bool)
	Set(interface{}, interface{}, int64) bool
	metrics() *metrics
}

// capacity is the cache capacity to be used across all tests and benchmarks.
const capacity = 1000

// newCache should be used for all Ristretto instances in local tests.
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

// newBenchmark should be used for all local benchmarks to ensure consistency
// across comparisons.
func newBenchmark(bencher func(uint64)) func(b *testing.B) {
	return func(b *testing.B) {
		b.SetParallelism(1)
		b.SetBytes(1)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for i := uint64(0); pb.Next(); i++ {
				bencher(i)
			}
		})
	}
}

// BenchmarkCacheGetOne Gets the same key-value item over and over.
func BenchmarkCacheGetOne(b *testing.B) {
	cache := newCache(false)
	cache.Set(1, nil, 1)
	newBenchmark(func(i uint64) { cache.Get(1) })(b)
}

// BenchmarkCacheSetOne Sets the same key-value item over and over.
func BenchmarkCacheSetOne(b *testing.B) {
	cache := newCache(false)
	newBenchmark(func(i uint64) { cache.Set(1, nil, 1) })(b)
}

// BenchmarkCacheSetUni Sets keys incrementing by 1.
func BenchmarkCacheSetUni(b *testing.B) {
	cache := newCache(false)
	newBenchmark(func(i uint64) { cache.Set(i, nil, 1) })(b)
}

// newRatioTest simulates a workload for a TestCache so you can just run the
// returned test and call cache.metrics() to get a basic idea of performance.
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

// TestCacheRatios gives us a rough idea of the hit ratio relative to the
// theoretical optimum. Useful for quickly seeing the effects of changes.
func TestCacheRatios(t *testing.T) {
	cache := newCache(true)
	optimal := NewClairvoyant(capacity)
	newRatioTest(cache)(t)
	newRatioTest(optimal)(t)
	t.Logf("ristretto: %.2f\n", cache.metrics().Ratio())
	t.Logf("- optimal: %.2f\n", optimal.metrics().Ratio())
}

func TestCacheSetGet(t *testing.T) {
	cache := newCache(true)
	// fill the cache with data
	for key := 0; key < capacity; key++ {
		cache.Set(key, key, 1)
	}
	// wait for the Sets to be processed so that all values are in the cache
	// before we begin Gets, otherwise the hit ratio would be bad
	time.Sleep(time.Second / 100)
	wg := &sync.WaitGroup{}
	// launch goroutines to concurrently Get random keys
	for r := 0; r < 8; r++ {
		wg.Add(1)
		go func() {
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			// it's not too important that we iterate through the whole capacity
			// here, but we want all the goroutines to be Getting in parallel,
			// so it should iterate long enough that it will continue until the
			// other goroutines are done spinning up
			for i := 0; i < capacity; i++ {
				key := r.Int() % capacity
				if val, ok := cache.Get(key); ok {
					if val.(int) != key {
						t.Fatalf("expected %d but got %d\n", key, val.(int))
					}
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
	if ratio := cache.metrics().Ratio(); ratio != 1.0 {
		t.Fatalf("expected 1.00 but got %.2f\n", ratio)
	}
}

// TestCacheSetNil makes sure nil values are working properly.
func TestCacheSetNil(t *testing.T) {
	cache := newCache(false)
	cache.Set(1, nil, 1)
	// must wait for the set buffer
	time.Sleep(time.Second / 1000)
	if value, ok := cache.Get(1); !ok || value != nil {
		t.Fatal("cache value should exist and be nil")
	}
}

// TestCacheSetDrops simulates a period of high contention and reports the
// percentage of Sets that are dropped. For most use cases, it would be rare to
// have more than 4 goroutines calling Set in parallel. Nevertheless, this is a
// useful stress test.
func TestCacheSetDrops(t *testing.T) {
	for goroutines := 1; goroutines <= 50; goroutines++ {
		n, size := goroutines, capacity*10
		sample := uint64(n * size)
		cache := newCache(true)
		keys := sim.Collection(sim.NewUniform(sample), sample)
		start, finish := &sync.WaitGroup{}, &sync.WaitGroup{}
		for i := 0; i < n; i++ {
			start.Add(1)
			finish.Add(1)
			go func(i int) {
				start.Done()
				// wait for all goroutines to be ready
				start.Wait()
				for j := i * size; j < (i*size)+size; j++ {
					cache.Set(keys[j], 0, 1)
				}
				finish.Done()
			}(i)
		}
		finish.Wait()
		dropped := cache.metrics().Get(dropSets)
		t.Logf("%d goroutines: %.2f%% dropped \n",
			goroutines, float64(float64(dropped)/float64(sample))*100)
		runtime.GC()
	}
}

// Clairvoyant is a mock cache providing us with optimal hit ratios to compare
// with Ristretto's. It looks ahead and evicts the absolute least valuable item,
// which we try to approximate in a real cache.
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

// Get just records the cache access so that we can later take this event into
// consideration when calculating the absolute least valuable item to evict.
func (c *Clairvoyant) Get(key interface{}) (interface{}, bool) {
	c.hits[key.(uint64)]++
	c.access = append(c.access, key.(uint64))
	return nil, false
}

// Set isn't important because it is only called after a Get (in the case of our
// hit ratio benchmarks, at least).
func (c *Clairvoyant) Set(key, value interface{}, cost int64) bool {
	return false
}

func (c *Clairvoyant) metrics() *metrics {
	stat := newMetrics()
	look := make(map[uint64]struct{}, c.capacity)
	data := &clairvoyantHeap{}
	heap.Init(data)
	for _, key := range c.access {
		if _, has := look[key]; has {
			stat.Add(hit, 0, 1)
			continue
		}
		if uint64(data.Len()) >= c.capacity {
			victim := heap.Pop(data)
			delete(look, victim.(*clairvoyantItem).key)
		}
		stat.Add(miss, 0, 1)
		look[key] = struct{}{}
		heap.Push(data, &clairvoyantItem{key, c.hits[key]})
	}
	return stat
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
