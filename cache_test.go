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
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/dgraph-io/ristretto/bench/sim"
)

const (
	NUM_COUNTERS = 256
	BUFFER_ITEMS = NUM_COUNTERS / 4
	SAMPLE_ITEMS = NUM_COUNTERS * 8
	ZIPF_V       = 1.01
	ZIPF_S       = 2
)

func newCache(config *Config, p PolicyCreator) *Cache {
	if config.MaxCost == 0 && config.NumCounters != 0 {
		config.MaxCost = config.NumCounters
	}
	data := newStore()
	policy := p(config.NumCounters, config.MaxCost)
	if config.Log {
		policy = NewRecorder(policy, data)
	}
	return &Cache{
		data:   data,
		policy: policy,
		buffer: newRingBuffer(ringLossy, &ringConfig{
			Consumer: policy,
			Capacity: config.BufferItems,
		}),
	}
}

func BenchmarkCacheOneGet(b *testing.B) {
	c := newCache(&Config{
		NumCounters: NUM_COUNTERS,
		BufferItems: BUFFER_ITEMS,
		Log:         false,
	}, newPolicy)
	c.Set("1", 1, 1)
	b.SetBytes(1)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Get("1")
		}
	})
}

func BenchmarkCacheLong(b *testing.B) {
	cache, err := NewCache(&Config{
		NumCounters: 64 << 20,
		BufferItems: 1000,
		Log:         true,
		MaxCost:     256 << 20,
	})
	if err != nil {
		b.Fatal(err)
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			idx := r.Int31() % (512 << 20)
			if out, _ := cache.Get(idx); out != nil {
				if out.(int32) != idx {
					b.Fatalf("Wanted: %d. Got: %d\n", idx, out)
				}
			} else {
				cache.Set(idx, idx, int64(idx>>10)+1)
			}
		}
	})
	b.Logf("Got Hit Ratio: %.2f\n", cache.Log().Ratio())
}

func GenerateCacheTest(p PolicyCreator, k sim.Simulator) func(*testing.T) {
	return func(t *testing.T) {
		// create the cache with the provided policy and constant params
		cache := newCache(&Config{
			NumCounters: NUM_COUNTERS,
			BufferItems: BUFFER_ITEMS,
			Log:         true,
		}, p)
		// must iterate through SAMPLE_ITEMS because it's fixed and should be
		// much larger than the MAX_ITEMS
		for i := 0; i < SAMPLE_ITEMS; i++ {
			// generate a key from the simulator
			key, err := k()
			if err != nil {
				panic(err)
			}
			// must be a set operation for hit ratio logging
			cache.Set(fmt.Sprintf("%d", key), i, 1)
		}
		// stats is the hit ratio stats for the cache instance
		stats := cache.Log()
		// log the hit ratio
		t.Logf("------------------- %d%%\n", uint64(stats.Ratio()*100))
	}
}

type (
	policyTest struct {
		label   string
		creator PolicyCreator
	}
	accessTest struct {
		label  string
		access sim.Simulator
	}
)

func TestCache(t *testing.T) {
	// policies is a slice of all policies to test (see policy.go)
	policies := []policyTest{
		{"clairvoyant", newClairvoyant},
		{"    default", newPolicy},
	}
	// accesses is a slice of all access distributions to test (see sim package)
	accesses := []accessTest{
		{"uniform    ", sim.NewUniform(SAMPLE_ITEMS)},
		{"zipfian    ", sim.NewZipfian(ZIPF_V, ZIPF_S, SAMPLE_ITEMS)},
	}
	for _, access := range accesses {
		for _, policy := range policies {
			t.Logf("%s-%s", policy.label, access.label)
			GenerateCacheTest(policy.creator, access.access)(t)
		}
	}
}

func TestCacheBasic(t *testing.T) {
	c, err := NewCache(&Config{
		NumCounters: 100,
		MaxCost:     10,
		BufferItems: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	if added := c.Set("1", 1, 1); !added {
		t.Fatal("set error")
	}
	if value, found := c.Get("1"); found && value.(int) != 1 {
		t.Fatal("get error")
	}
}

func TestCacheSetGet(t *testing.T) {
	c, err := NewCache(&Config{
		NumCounters: 100,
		MaxCost:     4,
		BufferItems: 4,
	})
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 16; i++ {
		key := fmt.Sprintf("%d", i)
		if added := c.Set(key, i, 1); added {
			value, found := c.Get(key)
			if !found || value == nil || value.(int) != i {
				t.Fatal("set/get error")
			}
		}
	}
}

func TestCacheSize(t *testing.T) {
	c, err := NewCache(&Config{
		NumCounters: 16,
		MaxCost:     16 * 4,
		BufferItems: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 8; i++ {
		c.Set(fmt.Sprintf("%d", i), i, 4)
		if c.policy.Cap() < 0 {
			t.Fatal("size overflow")
		}
	}
}
