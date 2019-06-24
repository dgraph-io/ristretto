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
	"testing"

	"github.com/dgraph-io/ristretto/bench/sim"
)

const (
	// CAPACITY is the cache size in number of elements
	CAPACITY = 16
	// W is the number of elements in the "sample size" as mentioned in the
	// TinyLFU paper, where W/C = 16. W denotes the sample size, and C is the
	// cache size (denoted by *CAPA).
	W = CAPACITY * 16
	// zipf generation variables (see https://golang.org/pkg/math/rand/#Zipf)
	//
	// ZIPF_S must be > 1, the larger the value the more spread out the
	// distribution is
	ZIPF_S = 1.01
	ZIPF_V = 2
)

// HitsUniform records the hit ratio using a uniformly random distribution.
func HitsUniform(bench *Benchmark, coll *LogCollection) func(b *testing.B) {
	return func(b *testing.B) {
		cache := bench.Create()
		keys := sim.StringCollection(sim.NewUniform(W), uint64(b.N))
		vals := []byte("*")
		b.SetParallelism(bench.Para)
		b.SetBytes(1)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for i := uint64(0); pb.Next(); i++ {
				cache.Set(keys[i&(uint64(b.N-1))], vals)
			}
		})
		// save hit ratio stats
		coll.Append(cache.Log())
	}
}

// HitsZipf records the hit ratio using a Zipfian distribution.
func HitsZipf(bench *Benchmark, coll *LogCollection) func(b *testing.B) {
	return func(b *testing.B) {
		cache := bench.Create()
		keys := sim.StringCollection(sim.NewZipfian(ZIPF_S, ZIPF_V, W), uint64(b.N))
		vals := []byte("*")
		b.SetParallelism(bench.Para)
		b.SetBytes(1)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for i := uint64(0); pb.Next(); i++ {
				cache.Set(keys[i&(uint64(b.N)-1)], vals)
			}
		})
		// save hit ratio stats
		coll.Append(cache.Log())
	}
}

func GetSame(bench *Benchmark, coll *LogCollection) func(b *testing.B) {
	return func(b *testing.B) {
		cache := bench.Create()
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
		cache := bench.Create()
		keys := sim.StringCollection(sim.NewZipfian(ZIPF_S, ZIPF_V, W), W)
		b.SetParallelism(bench.Para)
		b.SetBytes(1)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for i := uint64(0); pb.Next(); i++ {
				cache.Get(keys[i&(W-1)])
			}
		})
	}
}

func SetSame(bench *Benchmark, coll *LogCollection) func(b *testing.B) {
	return func(b *testing.B) {
		cache := bench.Create()
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
		cache := bench.Create()
		keys := sim.StringCollection(sim.NewZipfian(ZIPF_S, ZIPF_V, W), W)
		vals := []byte("*")
		b.SetParallelism(bench.Para)
		b.SetBytes(1)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for i := uint64(0); pb.Next(); i++ {
				cache.Set(keys[i&(W-1)], vals)
			}
		})
	}
}

func SetGet(bench *Benchmark, coll *LogCollection) func(b *testing.B) {
	return func(b *testing.B) {
		cache := bench.Create()
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
