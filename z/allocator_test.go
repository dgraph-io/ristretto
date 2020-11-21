/*
 * Copyright 2020 Dgraph Labs, Inc. and Contributors
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

package z

import (
	"math/rand"
	"sort"
	"sync"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

func TestAllocate(t *testing.T) {
	a := NewAllocator(1024)
	defer a.Release()

	check := func() {
		t.Logf("Running checks\n")
		require.Equal(t, 0, len(a.Allocate(0)))
		require.Equal(t, 1, len(a.Allocate(1)))
		require.Equal(t, 1<<20+1, len(a.Allocate(1<<20+1)))
		require.Equal(t, 256<<20, len(a.Allocate(256<<20)))
		require.Panics(t, func() { a.Allocate(maxAlloc + 1) })
	}

	check()
	t.Logf("%s", a)
	prev := a.Allocated()
	t.Logf("Resetting\n")
	a.Reset()
	check()
	t.Logf("%s", a)
	require.Equal(t, int(prev), int(a.Allocated()))
	t.Logf("Allocated: %d\n", prev)
}

func TestAllocateReset(t *testing.T) {
	a := NewAllocator(16)
	defer a.Release()

	buf := make([]byte, 128)
	rand.Read(buf)
	for i := 0; i < 1000; i++ {
		a.Copy(buf)
	}

	prev := a.Allocated()
	a.Reset()
	for i := 0; i < 100; i++ {
		a.Copy(buf)
	}
	t.Logf("%s", a)
	require.Equal(t, prev, a.Allocated())
}

func TestAllocateTrim(t *testing.T) {
	a := NewAllocator(16)
	defer a.Release()

	buf := make([]byte, 128)
	rand.Read(buf)
	for i := 0; i < 1000; i++ {
		a.Copy(buf)
	}

	N := 2048
	a.TrimTo(N)
	require.LessOrEqual(t, int(a.Allocated()), N)
}

func TestPowTwo(t *testing.T) {
	require.Equal(t, 2, log2(4))
	require.Equal(t, 2, log2(7))
	require.Equal(t, 3, log2(8))
	require.Equal(t, 3, log2(15))
	require.Equal(t, 4, log2(16))
	require.Equal(t, 4, log2(31))
	require.Equal(t, 10, log2(1024))
	require.Equal(t, 10, log2(1025))
	require.Equal(t, 10, log2(2047))
	require.Equal(t, 11, log2(2048))
}

func TestAllocateAligned(t *testing.T) {
	a := NewAllocator(1024)
	defer a.Release()

	a.Allocate(1)
	out := a.Allocate(1)
	ptr := uintptr(unsafe.Pointer(&out[0]))
	require.True(t, ptr%8 == 1)

	out = a.AllocateAligned(5)
	ptr = uintptr(unsafe.Pointer(&out[0]))
	require.True(t, ptr%8 == 0)

	out = a.AllocateAligned(3)
	ptr = uintptr(unsafe.Pointer(&out[0]))
	require.True(t, ptr%8 == 0)
}

func TestAllocatorCorrupt(t *testing.T) {
	a := NewAllocator(1023)
	defer a.Release()

	type KV struct {
		key []byte
		val []byte
	}
	kvsz := int(unsafe.Sizeof(KV{}))

	big := make([]byte, 1024)
	rand.Read(big)

	N := 1000

	go func() {
		for i := 0; i < N; i++ {
			copy(a.Allocate(rand.Intn(10)), big)
		}
	}()
	for i := 0; i < N; i++ {
		var kvs []*KV
		for j := 0; j < N; j++ {
			copy(a.Allocate(rand.Intn(10)), big)
			kb := a.AllocateAligned(kvsz)
			kv := (*KV)(unsafe.Pointer(&kb[0]))
			kv.key = a.Allocate(rand.Intn(1024))
			copy(kv.key, big)

			kv.val = a.Allocate(rand.Intn(1024))
			copy(kv.val, big)
			kvs = append(kvs, kv)
		}
		for _, kv := range kvs {
			sz := len(kv.key) + len(kv.val)
			if sz > 2048 {
				t.Fatalf("Size is too high: %d\n", sz)
			}
		}
		a.Reset()
	}
}

func TestAllocateConcurrent(t *testing.T) {
	a := NewAllocator(63)
	defer a.Release()

	N := 10240
	M := 16
	var wg sync.WaitGroup

	m := make(map[uintptr]struct{})
	mu := new(sync.Mutex)
	for i := 0; i < M; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var bufs []uintptr
			for j := 0; j < N; j++ {
				buf := a.Allocate(16)
				require.Equal(t, 16, len(buf))
				bufs = append(bufs, uintptr(unsafe.Pointer(&buf[0])))
			}

			mu.Lock()
			for _, b := range bufs {
				if _, ok := m[b]; ok {
					t.Fatalf("Did not expect to see the same ptr")
				}
				m[b] = struct{}{}
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	t.Logf("Size of allocator: %v. Allocator: %s\n", a.Size(), a)

	require.Equal(t, N*M, len(m))
	var sorted []uintptr
	for ptr := range m {
		sorted = append(sorted, ptr)
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	var last uintptr
	for _, ptr := range sorted {
		if ptr-last < 16 {
			t.Fatalf("Should not have less than 16: %v %v\n", ptr, last)
		}
		// fmt.Printf("ptr [%d]: %x %d\n", i, ptr, ptr-last)
		last = ptr
	}
}

func BenchmarkAllocate(b *testing.B) {
	a := NewAllocator(15)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := a.Allocate(1)
			if len(buf) != 1 {
				b.FailNow()
			}
		}
	})
	b.StopTimer()
	b.Logf("%s", a)
}
