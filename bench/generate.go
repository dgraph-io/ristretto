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

package main

import (
	"compress/gzip"
	"fmt"
	"os"
	"sync/atomic"
	"testing"

	"github.com/dgraph-io/ristretto/bench/sim"
)

const (
	// CAPACITY is the cache size in number of elements
	capacity = 32000000
	// W is the number of elements in the "sample size" as mentioned in the
	// TinyLFU paper, where W/C = 16. W denotes the sample size, and C is the
	// cache size (denoted by *CAPA).
	w = capacity * 16
	// zipf generation variables (see https://golang.org/pkg/math/rand/#Zipf)
	//
	// ZIPF_S must be > 1, the larger the value the more spread out the
	// distribution is
	zipfS = 1.001
	zipfV = 10
)

func NewHits(bench *Benchmark, coll *LogCollection, keys sim.Simulator) func(b *testing.B) {
	return func(b *testing.B) {
		cache := bench.Create(true)
		b.SetBytes(1)
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			key, err := keys()
			if err != nil {
				if err == sim.ErrDone {
					break
				}
				panic(err)
			}
			cache.Set(fmt.Sprintf("%d", key), []byte("*"))
		}
		if stats := cache.Log(); stats != nil {
			coll.Append(stats)
		}
	}
}

// HitsZipf records the hit ratio using a Zipfian distribution.
func HitsZipf(bench *Benchmark, coll *LogCollection) func(b *testing.B) {
	return NewHits(bench, coll, sim.NewZipfian(zipfS, zipfV, w))
}

func HitsLIRS(pre string) func(*Benchmark, *LogCollection) func(b *testing.B) {
	return func(bench *Benchmark, coll *LogCollection) func(b *testing.B) {
		file, err := os.Open("./trace/" + pre + ".lirs.gz")
		if err != nil {
			panic(err)
		}
		trace, err := gzip.NewReader(file)
		if err != nil {
			panic(err)
		}
		return NewHits(bench, coll, sim.NewReader(sim.ParseLIRS, trace))
	}
}

func HitsARC(pre string) func(*Benchmark, *LogCollection) func(b *testing.B) {
	return func(bench *Benchmark, coll *LogCollection) func(b *testing.B) {
		file, err := os.Open("./trace/" + pre + ".arc.gz")
		if err != nil {
			panic(err)
		}
		trace, err := gzip.NewReader(file)
		if err != nil {
			panic(err)
		}
		return NewHits(bench, coll, sim.NewReader(sim.ParseARC, trace))
	}
}

func GetSame(bench *Benchmark, coll *LogCollection) func(b *testing.B) {
	return func(b *testing.B) {
		cache := bench.Create(false)
		cache.Set("*", []byte("*"))
		b.SetParallelism(bench.Para)
		b.SetBytes(1)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				cache.Get("*")
			}
		})
	}
}

func GetZipf(bench *Benchmark, coll *LogCollection) func(b *testing.B) {
	return func(b *testing.B) {
		cache := bench.Create(false)
		keys := sim.StringCollection(sim.NewZipfian(zipfS, zipfV, capacity), capacity)
		b.SetParallelism(bench.Para)
		b.SetBytes(1)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for i := uint64(0); pb.Next(); i++ {
				cache.Get(keys[i&(capacity-1)])
			}
		})
	}
}

func SetSame(bench *Benchmark, coll *LogCollection) func(b *testing.B) {
	return func(b *testing.B) {
		cache := bench.Create(false)
		data := []byte("*")
		b.SetParallelism(bench.Para)
		b.SetBytes(1)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				cache.Set("*", data)
			}
		})
	}
}

func SetZipf(bench *Benchmark, coll *LogCollection) func(b *testing.B) {
	return func(b *testing.B) {
		cache := bench.Create(false)
		keys := sim.StringCollection(sim.NewZipfian(zipfS, zipfV, capacity), capacity)
		vals := []byte("*")
		b.SetParallelism(bench.Para)
		b.SetBytes(1)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for i := uint64(0); pb.Next(); i++ {
				cache.Set(keys[i&(capacity-1)], vals)
			}
		})
	}
}

func SetGetZipf(bench *Benchmark, coll *LogCollection) func(b *testing.B) {
	return func(b *testing.B) {
		cache := bench.Create(false)
		keys := sim.StringCollection(sim.NewZipfian(zipfS, zipfV, capacity), capacity)
		vals := []byte("*")
		b.SetParallelism(bench.Para)
		b.SetBytes(1)
		b.ResetTimer()
		i := int32(0)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				ti := atomic.AddInt32(&i, 1)
				if _, ok := cache.Get(keys[ti&(capacity-1)]); !ok {
					cache.Set(keys[ti&(capacity-1)], vals)
				}
			}
		})
	}
}

func SetGet(bench *Benchmark, coll *LogCollection) func(b *testing.B) {
	return func(b *testing.B) {
		cache := bench.Create(false)
		vals := []byte("*")
		b.SetParallelism(bench.Para)
		b.SetBytes(1)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for i := 0; pb.Next(); i++ {
				// alternate between setting and getting
				if i&1 == 0 {
					cache.Set("*", vals)
				} else {
					cache.Get("*")
				}
			}
		})
	}
}
