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

package bench

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

const (
	GET_SAME_CAPA = 1
	GET_ZIPF_CAPA = 128
	GET_ZIPF_MULT = 1024

	SET_SAME_CAPA = 1
	SET_ZIPF_CAPA = 128

	SET_GET_CAPA = 1

	// zipf generation variables (see https://golang.org/pkg/math/rand/#Zipf)
	ZIPF_S = 1.1
	ZIPF_V = 1
)

func report(cache Cache, stats chan *Stats) {
	stats <- cache.Bench()
}

////////////////////////////////////////////////////////////////////////////////

func GetSame(benchmark *Benchmark, stats chan *Stats) func(b *testing.B) {
	return func(b *testing.B) {
		cache := benchmark.Create()
		cache.Set("*", []byte("*"))
		b.SetParallelism(benchmark.Para)
		b.SetBytes(1)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				cache.Get("*")
			}
		})
		report(cache, stats)
	}
}

// zipfKeys generates a string slice with repeating keys corresponding to a
// zipfian distribution meant to simulate realistic traffic
//
// see ZIPF_* constants for the parameters
func zipfKeys(size int) []string {
	// create zipf generator
	zipf := rand.NewZipf(rand.New(rand.NewSource(time.Now().UnixNano())),
		ZIPF_S, ZIPF_V, uint64(size))
	// keys with a zipf distribution
	keys := make([]string, size)
	// fill keys
	for i := range keys {
		keys[i] = fmt.Sprintf("%d", zipf.Uint64())
	}
	return keys
}

func GetZipf(benchmark *Benchmark, stats chan *Stats) func(b *testing.B) {
	return func(b *testing.B) {
		cache := benchmark.Create()
		keys := zipfKeys(GET_ZIPF_CAPA * GET_ZIPF_MULT)
		b.SetParallelism(benchmark.Para)
		b.SetBytes(1)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for i := 0; pb.Next(); i++ {
				cache.Get(keys[i&((GET_ZIPF_CAPA*GET_ZIPF_MULT)-1)])
			}
		})
		report(cache, stats)
	}
}

////////////////////////////////////////////////////////////////////////////////

func SetSame(benchmark *Benchmark, stats chan *Stats) func(b *testing.B) {
	return func(b *testing.B) {
		cache := benchmark.Create()
		data := []byte("*")
		b.SetParallelism(benchmark.Para)
		b.SetBytes(1)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				cache.Set("*", data)
			}
		})
		report(cache, stats)
	}
}

func SetZipf(benchmark *Benchmark, stats chan *Stats) func(b *testing.B) {
	return func(b *testing.B) {
		cache := benchmark.Create()
		keys := zipfKeys(SET_ZIPF_CAPA)
		data := []byte("*")
		b.SetParallelism(benchmark.Para)
		b.SetBytes(1)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for i := 0; pb.Next(); i++ {
				cache.Set(keys[i&(SET_ZIPF_CAPA-1)], data)
			}
		})
		report(cache, stats)
	}
}

////////////////////////////////////////////////////////////////////////////////

func SetGet(benchmark *Benchmark, stats chan *Stats) func(b *testing.B) {
	return func(b *testing.B) {
		cache := benchmark.Create()
		data := []byte("*")
		b.SetParallelism(benchmark.Para)
		b.SetBytes(1)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for i := 0; pb.Next(); i++ {
				// alternate between setting and getting
				if i&1 == 0 {
					cache.Set("*", data)
				} else {
					cache.Get("*")
				}
			}
		})
		report(cache, stats)
	}
}
