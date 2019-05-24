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
	"sync"
	"sync/atomic"
	"time"
)

type BufferType byte

const (
	LOSSY BufferType = iota
	LOSSLESS
)

type Element string

// Consumer is the user-defined object responsible for receiving and processing
// elements in batches when buffers are drained.
type Consumer interface {
	Push([]Element)
}

// Stripe is a singular ring buffer that is not concurrent safe by itself.
type Stripe struct {
	Consumer Consumer
	data     []Element
	head     int
	capacity int
	busy     int32
}

func NewStripe(config *Config) *Stripe {
	return &Stripe{
		Consumer: config.Consumer,
		data:     make([]Element, config.Capacity),
		capacity: config.Capacity,
	}
}

// Push appends an element in the ring buffer and drains (copies elements and
// sends to Consumer) if full.
func (s *Stripe) Push(element Element) {
	s.data[s.head] = element
	s.head++

	// check if we should drain
	if s.head >= s.capacity {
		// copy elements and send to consumer
		s.Consumer.Push(append(s.data[:0:0], s.data...))
		s.head = 0
	}
}

type Config struct {
	Consumer Consumer
	Stripes  int
	Capacity int
}

// Buffer stores multiple buffers (stripes) and distributes Pushed elements
// between them to lower contention.
//
// This implements the "batching" process described in the BP-Wrapper paper
// (section III part A).
type Buffer struct {
	stripes []*Stripe
	pool    *sync.Pool
	push    func(*Buffer, Element)
	rand    int
	mask    int
}

// NewBuffer returns a striped ring buffer. The Type can be either LOSSY or
// LOSSLESS. LOSSY should provide better performance. The Consumer in Config
// will be called when individual stripes are full and need to drain their
// elements.
func NewBuffer(Type BufferType, config *Config) *Buffer {
	if Type == LOSSY {
		// LOSSY buffers use a very simple sync.Pool for concurrently reusing
		// stripes. We do lose some stripes due to GC (unheld items in sync.Pool
		// are cleared), but the performance gains generally outweigh the small
		// percentage of elements lost. The performance primarily comes from
		// low-level runtime functions used in the standard library that aren't
		// available to us (such as runtime_procPin()).
		return &Buffer{
			pool: &sync.Pool{
				New: func() interface{} { return NewStripe(config) },
			},
			push: pushLossy,
		}
	}

	// begin LOSSLESS buffer handling
	//
	// unlike lossy, lossless manually handles all stripes
	stripes := make([]*Stripe, config.Stripes)
	for i := range stripes {
		stripes[i] = NewStripe(config)
	}
	return &Buffer{
		stripes: stripes,
		mask:    config.Stripes - 1,
		rand:    int(time.Now().UnixNano()), // random seed for picking stripes
		push:    pushLossless,
	}
}

// Push adds an element to one of the internal stripes and possibly drains if
// the stripe becomes full.
func (b *Buffer) Push(element Element) { b.push(b, element) }

func pushLossy(b *Buffer, element Element) {
	// reuse or create a new stripe
	stripe := b.pool.Get().(*Stripe)
	stripe.Push(element)
	b.pool.Put(stripe)
}

func pushLossless(b *Buffer, element Element) {
	// xorshift random (racy but it's random enough)
	b.rand ^= b.rand << 13
	b.rand ^= b.rand >> 7
	b.rand ^= b.rand << 17
	// try to find an available stripe
	for i := b.rand & b.mask; ; i = (i + 1) & b.mask {
		// try to get exclusive lock on the stripe
		if atomic.CompareAndSwapInt32(&b.stripes[i].busy, 0, 1) {
			b.stripes[i].Push(element)
			// unlock
			atomic.StoreInt32(&b.stripes[i].busy, 0)
			return
		}
	}
}
