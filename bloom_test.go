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

type TestSketch interface {
	Sketch
	increment(uint64)
	estimate(uint64) uint64
	string() string
}

func GenerateBloomTest(create func() TestSketch) func(t *testing.T) {
	return func(t *testing.T) {
		s := create()
		s.increment(0)
		s.increment(0)
		s.increment(0)
		s.increment(0)
		if s.estimate(0) != 4 {
			t.Fatal("increment/estimate error")
		}
		if s.Estimate("*") != 0 {
			t.Fatal("neighbor corruption")
		}
		s.Reset()
		if s.estimate(0) != 2 {
			t.Fatal("reset error")
		}
		if s.estimate(9) != 0 {
			t.Fatal("neighbor corruption")
		}
	}
}

func TestCBF(t *testing.T) {
	// TODO: fix neighbor corruption
	//GenerateTest(func() TestSketch { return NewCBF(16) })(t)
}

func TestCM(t *testing.T) {
	GenerateBloomTest(func() TestSketch { return NewCM(16) })(t)
}

func GenerateBloomBenchmark(create func() TestSketch) func(b *testing.B) {
	return func(b *testing.B) {
		s := create()
		b.Run("increment", func(b *testing.B) {
			b.SetBytes(1)
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				s.increment(1)
			}
		})
		b.Run("estimate", func(b *testing.B) {
			b.SetBytes(1)
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				s.estimate(1)
			}
		})
	}
}

func BenchmarkCBF(b *testing.B) {
	GenerateBloomBenchmark(func() TestSketch { return NewCBF(16) })(b)
}

func BenchmarkCM(b *testing.B) {
	GenerateBloomBenchmark(func() TestSketch { return NewCM(16) })(b)
}
