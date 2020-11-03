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
