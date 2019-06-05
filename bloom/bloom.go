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
	"hash"
	"hash/fnv"
	"sync"

	"github.com/dgraph-io/ristretto/ring"
)

const (
	COUNTER_BITS = 4
	COUNTER_MAX  = 16
	COUNTER_SEGS = 64 / COUNTER_BITS
	COUNTER_ROWS = 3
)

type Counter struct {
	sync.Mutex
	capa uint64
	mask uint64
	data []uint64
	algs []hash.Hash64
}

func NewCounter(samples uint64) *Counter {
	capa := samples / COUNTER_MAX
	data := make([]uint64, (capa/COUNTER_SEGS)*COUNTER_ROWS)
	algs := make([]hash.Hash64, COUNTER_ROWS)
	for i := range algs {
		algs[i] = fnv.New64a()
		seed := make([]byte, 32, 32)
		rand.Read(seed)
		algs[i].Write(seed)
	}
	return &Counter{
		capa: capa,
		mask: capa - 1,
		data: data,
		algs: algs,
	}
}

func (c *Counter) hash(key string, row uint64) uint64 {
	c.algs[row].Write([]byte(key))
	return c.algs[row].Sum64() & c.mask
}

func (c *Counter) estimate(key string) uint64 {
	min := uint64(0)

	for row := uint64(0); row < COUNTER_ROWS; row++ {
		i := c.hash(key, row)

		// get value of the counter on this row
		count := (c.data[index(i)+(row*(c.capa/COUNTER_SEGS))] << shift(i)) >>
			(shift(i) + (offset(i) * COUNTER_BITS))

		// adjust the minimum value
		if row == 0 || count < min {
			min = count
		}
	}

	return min
}

// Push fulfills the ring.Consumer interface for BP-Wrapper batched updates.
func (c *Counter) Push(keys []ring.Element) {
	c.Lock()
	defer c.Unlock()
	for _, key := range keys {
		k := string(key)
		count := c.estimate(k)

		// if the counter is already at the maximum value, do nothing
		if count == COUNTER_MAX-1 {
			continue
		}

		full := uint64(0xffffffffffffffff)

		for row := uint64(0); row < COUNTER_ROWS; row++ {
			// get counter index (0 through s.capa)
			i := c.hash(k, row)

			// calculate mask for isolation
			mask := (full << (64 - COUNTER_BITS)) >> shift(i)

			// isolate the counter segment
			isol := (c.data[index(i)+(row*(c.capa/COUNTER_SEGS))] | mask) ^ mask

			// mutate counter
			c.data[index(i)+(row*(c.capa/COUNTER_SEGS))] = isol |
				((count + 1) << (offset(i) * COUNTER_BITS))
		}
	}
}

func (c *Counter) Evict() string {
	c.Lock()
	defer c.Unlock()
	return ""
}

func shift(i uint64) uint64 {
	return (64 - COUNTER_BITS) - (offset(i) * COUNTER_BITS)
}

func index(i uint64) uint64 {
	return i / COUNTER_SEGS
}

func offset(i uint64) uint64 {
	return i & (COUNTER_SEGS - 1)
}

////////////////////////////////////////////////////////////////////////////////

type Clock struct{}
