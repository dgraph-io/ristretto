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

// This package includes multiple probabalistic data structures needed for
// admission/eviction metadata. Most are Counting Bloom Filter variations, but
// a caching-specific feature that is also required is a "freshness" mechanism,
// which basically serves as a "lifetime" process. This freshness mechanism
// was described in the original TinyLFU paper [1], but other mechanisms may
// be better suited for certain data distributions.
//
// [1]: https://arxiv.org/abs/1512.00727
package ristretto

import (
	"fmt"
	"math/rand"
	"time"
)

// cmSketch is a Count-Min sketch implementation with 4-bit counters, heavily
// based on Damian Gryski's CM4 [1].
//
// [1]: https://github.com/dgryski/go-tinylfu/blob/master/cm4.go
type cmSketch struct {
	rows [cmDepth]cmRow
	seed [cmDepth]uint64
	mask uint64
}

const (
	// cmDepth is the number of counter copies to store (think of it as rows).
	// This value hasn't changed in years. The functions below using `fourIndexes`
	// use that fact to unwind the loops.
	cmDepth = 4
)

func newCmSketch(numCounters int64) *cmSketch {
	if numCounters == 0 {
		panic("cmSketch: bad numCounters")
	}
	// Get the next power of 2 for better cache performance.
	numCounters = next2Power(numCounters)
	sketch := &cmSketch{mask: uint64(numCounters - 1)}
	// Initialize rows of counters and seeds.
	// Cryptographic precision not needed
	source := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec
	for i := 0; i < cmDepth; i++ {
		sketch.seed[i] = spread(source.Uint64())
		sketch.rows[i] = newCmRow(numCounters)
	}
	return sketch
}

func circRightShift(x uint64, shift uint) uint64 {
	return (x << (64 - shift)) | (x >> shift)
}
func circLeftShift(x uint64, shift uint) uint64 {
	return (x << shift) | (x >> (64 - shift))
}

//  Applies a supplemental hash function to a given hashCode, which defends against poor quality
//  hash functions.
func spread(x uint64) uint64 {
	x = (circRightShift(x, 16) ^ x) * 0x45d9f3b
	x = (circRightShift(x, 16) ^ x) * 0x45d9f3b
	return circRightShift(x, 16) ^ x
}

func leftAndRight(x uint64) (l, r uint64) {
	l = x<<32 | (x & 0x00000000ffffffff)
	r = x>>32 | (x & 0xffffffff00000000)
	return
}

// fourIndexes returns the four indexes to use against the four rows for the given key.
// Because many 64 keys come in with low entropy in some portions of the 64 bits, some work is done
// to spread any entropy at all across all four index results, with the goal of minimizing the
// number of times similar keys results in similar row indexes. No tests against zero are needed so
// there are no branch predictions to stall the instruction pipeline.
func (s *cmSketch) fourIndexes(x uint64) (a, b, c, d uint64) {
	x = spread(x)
	l, r := leftAndRight(x)

	l = circLeftShift(l, 3)
	r = circRightShift(r, 5)

	l = circLeftShift(l, 8)
	a = (l ^ r)
	l = circLeftShift(l, 8)
	b = (l ^ (r >> 8))
	l = circLeftShift(l, 8)
	c = (l ^ (r >> 16))
	l = circLeftShift(l, 8)
	d = (l ^ (r >> 24))

	// Interestingly, the seeds don't do anything important if the incoming hash doesn't cause
	// different parts of them to be used. The seed is meant to further swivel the counter index
	// per row, but if a given row's seed is used each time, unaltered for a row, it serves no
	// purpose. An improvement below uses the seeds based on the hash low bits.
	// a ^= s.seed[0]
	// b ^= s.seed[1]
	// c ^= s.seed[2]
	// d ^= s.seed[3]

	// Use the hash's lowest 6 bits for an int in range [0,63] that is then used to shift right
	// each seed. The low bits of the hash are used to determine rotation amount of each row's seed
	// before the seed is applied to the four index computers. These computers are actually
	// swivelers, causing the index that is created via s.mask to be swiveled.
	a ^= circRightShift(s.seed[0], uint(x&63))
	b ^= circRightShift(s.seed[1], uint(x&63))
	c ^= circRightShift(s.seed[2], uint(x&63))
	d ^= circRightShift(s.seed[3], uint(x&63))
	return
}

// Increment increments the counters for the specified key.
func (s *cmSketch) Increment(hashed uint64) {
	a, b, c, d := s.fourIndexes(hashed)
	m := s.mask

	s.rows[0].increment(a & m)
	s.rows[1].increment(b & m)
	s.rows[2].increment(c & m)
	s.rows[3].increment(d & m)
}

// Estimate returns the value of the specified key.
// It does this by calculating the index for each row that `Increment` would have used for the
// specified key and returning the lowest of the four counters.
func (s *cmSketch) Estimate(hashed uint64) int64 {
	a, b, c, d := s.fourIndexes(hashed)
	m := s.mask

	// find the smallest counter value from all the rows
	v0 := s.rows[0].get(a & m)
	v1 := s.rows[1].get(b & m)
	v2 := s.rows[2].get(c & m)
	v3 := s.rows[3].get(d & m)
	if v1 < v0 {
		v0 = v1
	}
	if v3 < v2 {
		v2 = v3
	}
	if v2 < v0 {
		return int64(v2)
	}
	return int64(v0)
}

// Reset halves all counter values.
func (s *cmSketch) Reset() {
	for _, r := range s.rows {
		r.reset()
	}
}

// Clear zeroes all counters.
func (s *cmSketch) Clear() {
	for _, r := range s.rows {
		r.clear()
	}
}

// cmRow is a row of bytes, with each byte holding two counters.
type cmRow []byte

func newCmRow(numCounters int64) cmRow {
	return make(cmRow, numCounters/2)
}

func (r cmRow) get(n uint64) byte {
	return byte(r[n/2]>>((n&1)*4)) & 0x0f
}

func (r cmRow) increment(n uint64) {
	// Index of the counter.
	i := n / 2
	// Shift distance (even 0, odd 4).
	s := (n & 1) * 4
	// Counter value.
	v := (r[i] >> s) & 0x0f
	// Only increment if not max value (overflow wrap is bad for LFU).
	if v < 15 {
		r[i] += 1 << s
	}
}

func (r cmRow) reset() {
	// Halve each counter.
	for i := range r {
		r[i] = (r[i] >> 1) & 0x77
	}
}

func (r cmRow) clear() {
	// Zero each counter.
	for i := range r {
		r[i] = 0
	}
}

func (r cmRow) string() string {
	s := ""
	for i := uint64(0); i < uint64(len(r)*2); i++ {
		s += fmt.Sprintf("%02d ", (r[(i/2)]>>((i&1)*4))&0x0f)
	}
	s = s[:len(s)-1]
	return s
}

// next2Power rounds x up to the next power of 2, if it's not already one.
func next2Power(x int64) int64 {
	x--
	x |= x >> 1
	x |= x >> 2
	x |= x >> 4
	x |= x >> 8
	x |= x >> 16
	x |= x >> 32
	x++
	return x
}
