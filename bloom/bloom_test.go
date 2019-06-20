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

package bloom

import (
	"testing"
)

type TestSketch interface {
	Sketch
	increment(uint64)
	reset()
}

func GenerateTest(create func() TestSketch) func(t *testing.T) {
	return func(t *testing.T) {
		s := create()
		s.increment(1)
		s.increment(1)
		s.increment(1)
		s.increment(1)
		if s.Estimate(1) != 4 {
			t.Fatal("increment/estimate error")
		}
		s.reset()
		if s.Estimate(1) != 2 {
			t.Fatal("reset error")
		}
	}
}

func TestCBF(t *testing.T) {
	GenerateTest(func() TestSketch { return NewCBF(16) })(t)
}

////////////////////////////////////////////////////////////////////////////////

func GenerateBenchmark(create func() TestSketch) func(b *testing.B) {
	return func(b *testing.B) {
		s := create()
		b.Run("increment", func(b *testing.B) {
			b.SetBytes(1)
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				s.Increment("1")
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

func BenchmarkCBF(b *testing.B) {
	GenerateBenchmark(func() TestSketch { return NewCBF(16) })(b)
}
