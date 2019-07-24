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

const (
	// LOSSLESS number of stripes to test with
	RING_STRIPES = 16
	// LOSSY/LOSSLESS size of individual stripes
	RING_CAPACITY = 128
)

type BaseConsumer struct{}

func (c *BaseConsumer) Push(items []ringItem) {}

type TestConsumer struct {
	push func([]ringItem)
}

func (c *TestConsumer) Push(items []ringItem) { c.push(items) }

func TestRingLossy(t *testing.T) {
	drainCount := 0
	buffer := newRingBuffer(ringLossy, &ringConfig{
		Consumer: &TestConsumer{
			push: func(items []ringItem) {
				drainCount++
			},
		},
		Capacity: 4,
	})
	buffer.Push(ringItem("1"))
	buffer.Push(ringItem("2"))
	buffer.Push(ringItem("3"))
	buffer.Push(ringItem("4"))
	if drainCount != 1 {
		t.Fatal("drain error")
	}
}

func BenchmarkRingLossy(b *testing.B) {
	buffer := newRingBuffer(ringLossy, &ringConfig{
		Consumer: &BaseConsumer{},
		Capacity: RING_CAPACITY,
	})
	item := ringItem("1")
	b.Run("single", func(b *testing.B) {
		b.SetBytes(1)
		for n := 0; n < b.N; n++ {
			buffer.Push(item)
		}
	})
	b.Run("multiple", func(b *testing.B) {
		b.SetBytes(1)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buffer.Push(item)
			}
		})
	})
}

func BenchmarkRingLossless(b *testing.B) {
	buffer := newRingBuffer(ringLossless, &ringConfig{
		Consumer: &BaseConsumer{},
		Stripes:  RING_STRIPES,
		Capacity: RING_CAPACITY,
	})
	item := ringItem("1")
	b.Run("single", func(b *testing.B) {
		b.SetBytes(1)
		for n := 0; n < b.N; n++ {
			buffer.Push(item)
		}
	})
	b.Run("multiple", func(b *testing.B) {
		b.SetBytes(1)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buffer.Push(item)
			}
		})
	})
}
