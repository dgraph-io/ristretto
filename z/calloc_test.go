/*
 * SPDX-FileCopyrightText: © Hypermode Inc. <hello@hypermode.com>
 * SPDX-License-Identifier: Apache-2.0
 */

package z

import (
	"fmt"
	"github.com/dgraph-io/ristretto/v2/utils"
	"math/rand"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// $ go test -failfast -run xxx -bench . -benchmem  -count 10 > out.txt
// $ benchstat out.txt
// name                 time/op
// Allocation/Pool-8    200µs ± 5%
// Allocation/Calloc-8  100µs ±11%
//
// name                 alloc/op
// Allocation/Pool-8     477B ±29%
// Allocation/Calloc-8  4.00B ± 0%
//
// name                 allocs/op
// Allocation/Pool-8     1.00 ± 0%
// Allocation/Calloc-8   0.00
func BenchmarkAllocation(b *testing.B) {
	b.Run("Pool", func(b *testing.B) {
		pool := sync.Pool{
			New: func() interface{} {
				return make([]byte, 4<<10)
			},
		}
		b.RunParallel(func(pb *testing.PB) {
			source := rand.NewSource(utils.NowUnixNano())
			r := rand.New(source)
			for pb.Next() {
				x := pool.Get().([]byte)
				sz := r.Intn(100) << 10
				if len(x) < sz {
					x = make([]byte, sz)
				}
				r.Read(x)
				//nolint:staticcheck
				pool.Put(x)
			}
		})
	})

	b.Run("Calloc", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			source := rand.NewSource(utils.NowUnixNano())
			r := rand.New(source)
			for pb.Next() {
				sz := r.Intn(100) << 10
				x := Calloc(sz, "test")
				r.Read(x)
				Free(x)
			}
		})
	})
}

func TestCalloc(t *testing.T) {
	// Check if we're using jemalloc.
	// JE_MALLOC_CONF="abort:true,tcache:false"

	StatsPrint()
	buf := CallocNoRef(1, "test")
	if len(buf) == 0 {
		t.Skipf("Not using jemalloc. Skipping test.")
	}
	Free(buf)
	require.Equal(t, int64(0), NumAllocBytes())

	buf1 := Calloc(128, "test")
	require.Equal(t, int64(128), NumAllocBytes())
	buf2 := Calloc(128, "test")
	require.Equal(t, int64(256), NumAllocBytes())

	Free(buf1)
	require.Equal(t, int64(128), NumAllocBytes())

	// _ = buf2
	Free(buf2)
	require.Equal(t, int64(0), NumAllocBytes())
	fmt.Println(Leaks())

	// Double free would panic when debug mode is enabled in jemalloc.
	// Free(buf2)
	// require.Equal(t, int64(0), NumAllocBytes())
}
