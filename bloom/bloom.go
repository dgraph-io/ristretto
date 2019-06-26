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
package bloom

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"hash"
	"hash/fnv"

	"github.com/dgraph-io/ristretto/ring"
)

type Sketch interface {
	Increment(string)
	Estimate(string) uint64
}

// CBF is a basic Counting Bloom Filter implementation that fulfills the
// ring.Consumer interface for maintaining admission/eviction statistics.
type CBF struct {
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

func NewCBF(capacity uint64) *CBF {
	// capacity must be above a certain level because we do division below
	if capacity < CBF_COUNTERS {
		capacity = CBF_COUNTERS
	}
	// initialize hash seeds for each row
	seed := make([]uint64, CBF_ROWS)
	for i := range seed {
		tmp := make([]byte, 8)
		if _, err := rand.Read(tmp); err != nil {
			panic(err)
		}
		seed[i] = binary.LittleEndian.Uint64(tmp)
	}
	return &CBF{
		capacity: capacity,
		blocks:   capacity / CBF_MAX,
		data:     make([]uint64, (capacity/CBF_MAX)*CBF_ROWS),
		seed:     seed,
		alg:      fnv.New64a(),
	}
}

func (c *CBF) Increment(key string) {
	if _, err := c.alg.Write([]byte(key)); err != nil {
		panic(err)
	}
	hashed := c.alg.Sum64()
	c.alg.Reset()
	c.increment(hashed)
}

func (c *CBF) Estimate(key string) uint64 {
	if _, err := c.alg.Write([]byte(key)); err != nil {
		panic(err)
	}
	hashed := c.alg.Sum64()
	c.alg.Reset()
	return c.estimate(hashed)
}

func (c *CBF) estimate(hashed uint64) uint64 {
	min := uint64(0)
	for row := uint64(0); row < CBF_ROWS; row++ {
		var (
			rowHashed uint64 = hashed ^ c.seed[row]
			blockId   uint64 = (row * c.blocks) + (rowHashed & (c.blocks - 1))
			counterId uint64 = ((rowHashed >> CBF_BITS) & (CBF_COUNTERS - 1))
			left      uint64 = (CBF_BITS * counterId)
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
			// counterId uses rowHashed >> CBF_BITS because otherwise it would
			// equal blockId and counters wouldn't be evenly distributed across
			// all available counters in the block
			//
			// TODO: clean this up
			counterId uint64 = ((rowHashed >> CBF_BITS) & (CBF_COUNTERS - 1))
			left      uint64 = (CBF_BITS * counterId)
			right     uint64 = 64 - CBF_BITS
		)
		// current count
		count := c.data[blockId] << left >> right
		// if the current count is max
		if count == (CBF_MAX-1) && row == 0 {
			c.reset()
		}
		// skip if max value already
		if count == CBF_MAX-1 {
			continue
		}
		// mask for isolating counter for mutation
		mask := full << right >> left
		// clear out the counter for mutation
		isol := (c.data[blockId] | mask) ^ mask
		// increment
		c.data[blockId] = isol | ((count + 1) <<
			(((CBF_COUNTERS - 1) - counterId) * CBF_BITS))
	}
}

// CBF.reset() serves as the freshness mechanism described in section 3.3 of the
// TinyLFU paper [1]. It is called by CBF.increment() when the number of items
// reaches the sample size (W).
//
// [1]: https://arxiv.org/abs/1512.00727
func (c *CBF) reset() {
	mask := uint64(0xf0f0f0f0f0f0f0f0)
	// divide the counters in each block by 2
	for blockId := 0; blockId < len(c.data); blockId++ {
		c.data[blockId] = (((c.data[blockId] & mask) >> 1) & mask) |
			(((c.data[blockId] & (mask >> 4)) >> 1) & (mask >> 4))
	}
}

func (c *CBF) string() string {
	var state string
	for i := 0; i < len(c.data); i++ {
		state += "  ["
		block := c.data[i]
		for j := uint64(0); j < CBF_COUNTERS; j++ {
			count := block << (j * CBF_BITS) >> 60
			if count > 0 {
				state += fmt.Sprintf("%2d ", count)
			} else {
				state += "   "
			}
		}
		state = state[:len(state)-1] + "]\n"
	}
	state += "\n"
	return state
}

// TODO
//
// Fingerprint Counting Bloom Filter (FP-CBF): lower false positive rates than
// basic CBF with little added complexity.
//
// https://doi.org/10.1016/j.ipl.2015.11.002
type FPCBF struct {
}

func (c *FPCBF) Push(keys []ring.Element)      {}
func (c *FPCBF) Estimtae(hashed uint64) uint64 { return 0 }

// TODO
//
// d-left Counting Bloom Filter: based on d-left hashing which allows for much
// better space efficiency (usually saving a factor of 2 or more).
//
// https://link.springer.com/chapter/10.1007/11841036_61
type DLCBF struct {
}

func (c *DLCBF) Push(keys []ring.Element)      {}
func (c *DLCBF) Estimtae(hashed uint64) uint64 { return 0 }

// TODO
//
// Bloom Clock: this might be a good route for keeping track of LRU information
// in a space efficient, probabilistic manner.
//
// https://arxiv.org/abs/1905.13064
type BC struct{}

func (c *BC) Push(keys []ring.Element)      {}
func (c *BC) Estimtae(hashed uint64) uint64 { return 0 }
