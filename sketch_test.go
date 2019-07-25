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
	"testing"
)

// sketch is a collection of approximate frequency counters.
type sketch interface {
	// Increment increments the count(ers) for the specified key.
	Increment(uint64)
	// Estimate returns the value of the specified key.
	Estimate(uint64) int64
	// Reset halves all counter values.
	Reset()
}

type TestSketch interface {
	sketch
	string() string
}

func GenerateSketchTest(create func() TestSketch) func(t *testing.T) {
	return func(t *testing.T) {
		s := create()
		s.Increment(0)
		s.Increment(0)
		s.Increment(0)
		s.Increment(0)
		if s.Estimate(0) != 4 {
			t.Fatal("increment/estimate error")
		}
		if s.Estimate(1) != 0 {
			t.Fatal("neighbor corruption")
		}
		s.Reset()
		if s.Estimate(0) != 2 {
			t.Fatal("reset error")
		}
		if s.Estimate(9) != 0 {
			t.Fatal("neighbor corruption")
		}
	}
}

func TestCM(t *testing.T) {
	GenerateSketchTest(func() TestSketch { return newCmSketch(16) })(t)
}

func GenerateSketchBenchmark(create func() TestSketch) func(b *testing.B) {
	return func(b *testing.B) {
		s := create()
		b.Run("increment", func(b *testing.B) {
			b.SetBytes(1)
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				s.Increment(1)
			}
		})
		b.Run("estimate", func(b *testing.B) {
			b.SetBytes(1)
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				s.Estimate(1)
			}
		})
	}
}

func BenchmarkCM(b *testing.B) {
	GenerateSketchBenchmark(func() TestSketch { return newCmSketch(16) })(b)
}
