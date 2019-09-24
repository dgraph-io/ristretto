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
	"time"
)

const (
	// LOSSLESS number of stripes to test with
	RING_STRIPES = 16
	// LOSSY/LOSSLESS size of individual stripes
	RING_CAPACITY = 128
)

type BaseConsumer struct{}

func (c *BaseConsumer) Push(items []uint64) bool { return true }

type TestConsumer struct {
	push func([]uint64)
}

func (c *TestConsumer) Push(items []uint64) bool { c.push(items); return true }

func TestRingLossy(t *testing.T) {
	drainCount := 0
	buffer := newRingBuffer(ringLossy, &ringConfig{
		Consumer: &TestConsumer{
			push: func(items []uint64) {
				drainCount++
			},
		},
		Capacity: 4,
	})
	for i := 0; i < 100; i++ {
		buffer.Push(uint64(i))
	}
	// ideally we'd be able to check for a certain "drop percentage" here, but
	// that may vary per platform and testing configuration. for example: if
	// drainCount == 20 then we have 100% accuracy, but it's most likely around
	// 13-20 due to dropping and unfilled rings.
	if drainCount == 0 {
		t.Fatal("drain error")
	}
}

func TestRingLossless(t *testing.T) {
	drainCount := 0
	found := make(map[uint64]struct{})
	buffer := newRingBuffer(ringLossless, &ringConfig{
		Consumer: &TestConsumer{
			push: func(items []uint64) {
				drainCount++
				for _, item := range items {
					found[item] = struct{}{}
				}
			},
		},
		Capacity: 4,
		Stripes:  2,
	})
	buffer.Push(1)
	buffer.Push(2)
	buffer.Push(3)
	buffer.Push(4)
	buffer.Push(5)
	buffer.Push(6)
	buffer.Push(7)
	buffer.Push(8)
	time.Sleep(5 * time.Millisecond)
	if drainCount != 2 || len(found) != 8 {
		t.Fatal("drain error")
	}
}

func BenchmarkRingLossy(b *testing.B) {
	buffer := newRingBuffer(ringLossy, &ringConfig{
		Consumer: &BaseConsumer{},
		Capacity: RING_CAPACITY,
	})
	item := uint64(1)
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
	item := uint64(1)
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
