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
		require.Equal(t, 0, len(a.Allocate(0)))
		require.Equal(t, 1, len(a.Allocate(1)))
		require.Equal(t, 1<<20, len(a.Allocate(1<<20)))
		require.Equal(t, 256<<20, len(a.Allocate(256<<20)))
		require.Panics(t, func() { a.Allocate(1 << 30) })
	}

	check()
	prev := a.Allocated()
	a.Reset()
	check()
	require.Equal(t, prev, a.Allocated())
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
	for i, buf := range a.buffers {
		t.Logf("Allocated %d: %d\n", i, len(buf))
	}
	require.Equal(t, prev, a.Allocated())
}

func TestAllocateFreeList(t *testing.T) {
	a := NewAllocator(1024)
	defer a.Release()

	for i := 1; i <= 100; i++ {
		b := a.Allocate(i)
		a.Return(b)
	}
	a.Allocate(65)
	a.Allocate(65)
	a.Allocate(65)
	a.Allocate(33)
	a.Allocate(33)
	a.Allocate(33)
	a.Allocate(17)
	a.Allocate(17)
	a.Allocate(17)

	for i := 0; i < 100; i++ {
		a.Allocate(16)
	}
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
