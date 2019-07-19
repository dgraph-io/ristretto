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
	"testing"

	"github.com/dgraph-io/ristretto/bench/sim"
)

const (
	CACHE_SIZE  = 256
	BUFFER_SIZE = CACHE_SIZE / 4
	SAMPLE_SIZE = CACHE_SIZE * 8
	ZIPF_V      = 1.01
	ZIPF_S      = 2
)

func GenerateCacheTest(p PolicyCreator, k sim.Simulator) func(*testing.T) {
	return func(t *testing.T) {
		// create the cache with the provided policy and constant params
		cache := NewCache(&Config{
			CacheSize:  CACHE_SIZE,
			BufferSize: BUFFER_SIZE,
			Policy:     p,
			Log:        true,
		})
		// must iterate through SAMPLE_SIZE because it's fixed and should be
		// much larger than the CACHE_SIZE
		for i := 0; i < SAMPLE_SIZE; i++ {
			// generate a key from the simulator
			key, err := k()
			if err != nil {
				panic(err)
			}
			// must be a set operation for hit ratio logging
			cache.Set(fmt.Sprintf("%d", key), i)
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
		{"clairvoyant", NewClairvoyant},
		{"        lfu", NewLFU},
		{"        lru", NewLRU},
		{"    tinylfu", NewTinyLFU},
		{"       wlfu", NewWLFU},
	}
	// accesses is a slice of all access distributions to test (see sim package)
	accesses := []accessTest{
		{"uniform    ", sim.NewUniform(SAMPLE_SIZE)},
		{"zipfian    ", sim.NewZipfian(ZIPF_V, ZIPF_S, SAMPLE_SIZE)},
	}
	for _, access := range accesses {
		for _, policy := range policies {
			t.Logf("%s-%s", policy.label, access.label)
			GenerateCacheTest(policy.creator, access.access)(t)
		}
	}
}

func TestSetGet(t *testing.T) {
	c := NewCache(&Config{
		CacheSize:  4,
		BufferSize: 4,
		Policy:     NewLFU,
		Log:        false,
	})
	for i := 0; i < 16; i++ {
		key := fmt.Sprintf("%d", i)
		vic := c.Set(key, i)
		val := c.Get(key)
		if val == nil || val.(int) != i {
			t.Fatal("set/get error")
		}
		if i > 4 && vic == "" {
			t.Fatal("no eviction")
		}
	}
}
