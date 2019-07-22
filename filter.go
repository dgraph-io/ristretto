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
	"hash"
	"hash/fnv"
	"math"
)

// Filter is a simple Bloom Filter implementation to be used as the Doorkeeper
// mechanism described in the TinyLFU paper [1] in section 3.4.2.
//
// [1]: https://arxiv.org/abs/1512.00727
type Filter struct {
	keys uint64
	data []byte
	mask uint64
	algo hash.Hash64
}

// NewFilter creates a new Bloom Filter with the size being the max number of
// items to be stored (bits), and rate being the acceptable false positive rate.
func NewFilter(size uint64, rate float64) *Filter {
	m := -1 * float64(size) * math.Log(rate) / math.Pow(math.Log(2), 2)
	b := uint64(math.Ceil(m / 8))
	return &Filter{
		keys: uint64(math.Ceil(math.Log(2) * m / float64(size))),
		data: make([]byte, b),
		mask: b - 1,
		algo: fnv.New64a(),
	}
}

// Set returns true if the key didn't exist in the Filter and the bits were set.
// Set returns false if the key did exist in the Filter and nothing was changed.
func (f *Filter) Set(key string) bool {
	changed := false
	for i := uint64(0); i < f.keys; i++ {
		block, bit := f.index(f.hash(key, i))
		if !f.has(block, bit) {
			changed = true
			f.data[block] |= 1 << bit
		}
	}
	return changed
}

// Has returns whether or not key is in the Filter. If false, it's definitely
// not. If true, it probably is.
func (f *Filter) Has(key string) bool {
	for i := uint64(0); i < f.keys; i++ {
		if !f.has(f.index(f.hash(key, i))) {
			return false
		}
	}
	return true
}

// Reset sets all bits to 0.
func (f *Filter) Reset() {
	for i := range f.data {
		f.data[i] = 0
	}
}

// has returns true if the bit value is 1, and false if 0. The bit value denotes
// the bit *within* the block (i.e. max value is 7).
func (f *Filter) has(block, bit uint64) bool {
	return f.data[block]<<(7-bit)>>7 == 1
}

// index returns the block and bit values for the hashed key param.
func (f *Filter) index(hashed uint64) (uint64, uint64) {
	return hashed & f.mask, hashed & 7
}

// hash appends i to the key and returns the hashed result.
func (f *Filter) hash(key string, i uint64) uint64 {
	if _, err := f.algo.Write(append([]byte(key), byte(i))); err != nil {
		panic(err)
	}
	hashed := f.algo.Sum64()
	f.algo.Reset()
	return hashed
}
