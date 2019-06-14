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

package ring

import (
	"testing"
)

const (
	// LOSSLESS number of stripes to test with
	STRIPES = 16
	// LOSSY/LOSSLESS size of individual stripes
	CAPACITY = 128
)

type BaseConsumer struct{}

func (c *BaseConsumer) Push(elements []Element) {}

type TestConsumer struct {
	push func([]Element)
}

func (c *TestConsumer) Push(elements []Element) { c.push(elements) }

func TestLossy(t *testing.T) {
	drainCount := 0
	buffer := NewBuffer(LOSSY, &Config{
		Consumer: &TestConsumer{
			push: func(elements []Element) {
				drainCount++
			},
		},
		Capacity: 4,
	})

	buffer.Push(Element("1"))
	buffer.Push(Element("2"))
	buffer.Push(Element("3"))
	buffer.Push(Element("4"))

	if drainCount != 1 {
		t.Fatal("drain error")
	}
}

func BenchmarkLossy(b *testing.B) {
	buffer := NewBuffer(LOSSY, &Config{
		Consumer: &BaseConsumer{},
		Capacity: CAPACITY,
	})
	elem := Element("1")
	b.Run("Singular", func(b *testing.B) {
		b.SetBytes(1)
		for n := 0; n < b.N; n++ {
			buffer.Push(elem)
		}
	})
	b.Run("Parallel", func(b *testing.B) {
		b.SetBytes(1)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buffer.Push(elem)
			}
		})
	})
}

func BenchmarkLossless(b *testing.B) {
	buffer := NewBuffer(LOSSLESS, &Config{
		Consumer: &BaseConsumer{},
		Stripes:  STRIPES,
		Capacity: CAPACITY,
	})
	elem := Element("1")
	b.Run("Singular", func(b *testing.B) {
		b.SetBytes(1)
		for n := 0; n < b.N; n++ {
			buffer.Push(elem)
		}
	})
	b.Run("Parallel", func(b *testing.B) {
		b.SetBytes(1)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buffer.Push(elem)
			}
		})
	})
}
