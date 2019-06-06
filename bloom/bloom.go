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
	"crypto/rand"
	"encoding/binary"
	"hash"
	"hash/fnv"
	"sync"

	"github.com/dgraph-io/ristretto/ring"
)

// This package includes multiple probabalistic data structures needed for
// admission/eviction metadata. Most are Counting Bloom Filter variations, but
// a caching-specific feature that is also required is a "freshness" mechanism,
// which basically serves as a "lifetime" process. This freshness mechanism
// was described in the original TinyLFU paper [1], but other mechanisms may
// be better suited for certain data distributions.
//
// [1]: https://arxiv.org/abs/1512.00727

// CBF is a basic Counting Bloom Filter implementation that fulfills the
// ring.Consumer interface for maintaining admission/eviction statistics.
type CBF struct {
	sync.Mutex
	sample   uint64
	capacity uint64
	blocks   uint64
	data     []uint64
	seed     []uint64
	alg      hash.Hash64
}

const (
	CBF_BITS     = 4
	CBF_MAX      = 16
	CBF_ROWS     = 3
	CBF_COUNTERS = 64 / CBF_BITS
)

func NewCBF(capacity, sample uint64) *CBF {
	// initialize hash seeds for each row
	seed := make([]uint64, CBF_ROWS)
	for i := range seed {
		tmp := make([]byte, 8)
		rand.Read(tmp)
		seed[i] = binary.LittleEndian.Uint64(tmp)
	}
	return &CBF{
		sample:   sample,
		capacity: capacity,
		blocks:   capacity / CBF_MAX,
		data:     make([]uint64, (capacity/CBF_MAX)*CBF_ROWS),
		seed:     seed,
		alg:      fnv.New64a(),
	}
}

func (c *CBF) Push(keys []ring.Element) {
	c.Lock()
	defer c.Unlock()
	for _, key := range keys {
		c.alg.Write([]byte(key))
		c.increment(c.alg.Sum64())
		c.alg.Reset()
	}
}

func (c *CBF) Evict() string { return "" }

func (c *CBF) estimate(hashed uint64) uint64 {
	min := uint64(0)
	for row := uint64(0); row < CBF_ROWS; row++ {
		var (
			rowHashed uint64 = hashed ^ c.seed[row]
			blockId   uint64 = (row * c.blocks) + (rowHashed & (c.blocks - 1))
			counterId uint64 = (rowHashed & (CBF_COUNTERS - 1)) + 1
			left      uint64 = (CBF_BITS * counterId) - CBF_BITS
			right     uint64 = 64 - CBF_BITS
		)

		// current count
		count := c.data[blockId] << left >> right

		if row == 0 || count < min {
			min = count
		}
	}
	return min
}

func (c *CBF) increment(hashed uint64) {
	full := uint64(0xffffffffffffffff)
	for row := uint64(0); row < CBF_ROWS; row++ {
		var (
			rowHashed uint64 = hashed ^ c.seed[row]
			blockId   uint64 = (row * c.blocks) + (rowHashed & (c.blocks - 1))
			counterId uint64 = (rowHashed & (CBF_COUNTERS - 1)) + 1
			left      uint64 = (CBF_BITS * counterId) - CBF_BITS
			right     uint64 = 64 - CBF_BITS
		)

		// current count
		count := c.data[blockId] << left >> right

		// skip if max value already
		if count == CBF_MAX-1 {
			continue
		}

		// mask for isolating counter for mutation
		mask := full << right >> left

		// clear out the counter for mutation
		isol := (c.data[blockId] | mask) ^ mask

		// increment
		c.data[blockId] = isol | ((count + 1) << (64 - (counterId * CBF_BITS)))
	}
}

////////////////////////////////////////////////////////////////////////////////

// TODO
//
// Fingerprint Counting Bloom Filter (FP-CBF): lower false positive rates than
// basic CBF with little added complexity.
//
// https://doi.org/10.1016/j.ipl.2015.11.002
type FPCBF struct {
}

func (c *FPCBF) Push(keys []ring.Element) {}
func (c *FPCBF) Evict() string            { return "" }

////////////////////////////////////////////////////////////////////////////////

// TODO
//
// d-left Counting Bloom Filter: based on d-left hashing which allows for much
// better space efficiency (usually saving a factor of 2 or more).
//
// https://link.springer.com/chapter/10.1007/11841036_61
type DLCBF struct {
}

func (c *DLCBF) Push(keys []ring.Element) {}
func (c *DLCBF) Evict() string            { return "" }

////////////////////////////////////////////////////////////////////////////////

// TODO
//
// Bloom Clock: this might be a good route for keeping track of LRU information
// in a space efficient, probabilistic manner.
//
// https://arxiv.org/abs/1905.13064
type BC struct{}

func (c *BC) Push(keys []ring.Element) {}
func (c *BC) Evict() string            { return "" }
