//go:build linux && !js

/*
 * SPDX-FileCopyrightText: © 2017-2026 Istari Digital, Inc.
 * SPDX-License-Identifier: Apache-2.0
 */

package z

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestMunmapRemovesRegistryEntry(t *testing.T) {
	fd, err := os.CreateTemp(t.TempDir(), "mmap")
	require.NoError(t, err)
	defer fd.Close()
	require.NoError(t, fd.Truncate(int64(os.Getpagesize())))
	data, err := Mmap(fd, true, int64(os.Getpagesize()))
	require.NoError(t, err)
	require.NoError(t, Munmap(data))
	// Do not map anything between these calls: Linux permits unmapping an already
	// unmapped range, so only the x/sys ownership check can reject the second call.
	err = unix.Munmap(data)
	require.ErrorIs(t, err, unix.EINVAL)
}

func TestMremapUpdatesRegistry(t *testing.T) {
	page := os.Getpagesize()
	for _, size := range []int{page, 2 * page, 4 * page} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			fd, err := os.CreateTemp(t.TempDir(), "mremap")
			require.NoError(t, err)
			defer fd.Close()
			require.NoError(t, fd.Truncate(int64(4*page)))
			data, err := Mmap(fd, true, int64(2*page))
			require.NoError(t, err)
			data[0] = 42
			data[page-1] = 24
			data, err = mremap(data, size)
			require.NoError(t, err)
			require.Len(t, data, size)
			require.Equal(t, size, cap(data))
			require.Equal(t, byte(42), data[0])
			require.Equal(t, byte(24), data[page-1])
			// The new full slice must be owned by x/sys, including after shrink/grow.
			require.NoError(t, unix.Munmap(data))
		})
	}
}

func TestMmapInvalidSlices(t *testing.T) {
	data, err := unix.Mmap(-1, 0, os.Getpagesize(), unix.PROT_READ|unix.PROT_WRITE, unix.MAP_PRIVATE|unix.MAP_ANON)
	require.NoError(t, err)
	defer func() { require.NoError(t, unix.Munmap(data)) }()
	for _, invalid := range [][]byte{nil, data[:0], data[:len(data)-1]} {
		require.ErrorIs(t, Munmap(invalid), unix.EINVAL)
		_, err := mremap(invalid, len(data)*2)
		require.ErrorIs(t, err, unix.EINVAL)
	}
	_, err = mremap(data, 0)
	require.ErrorIs(t, err, unix.EINVAL)
	data[0] = 42 // Failed operations must leave the original mapping usable.
}

func TestMremapMovedRegistry(t *testing.T) {
	page := os.Getpagesize()
	data, err := unix.Mmap(-1, 0, 3*page, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_PRIVATE|unix.MAP_ANON)
	require.NoError(t, err)
	guardAddr := unsafe.Pointer(&data[page])
	data, err = mremap(data, page)
	require.NoError(t, err)
	// Occupy the next page without replacing any concurrent mapping. Growth must move.
	guard, err := unix.MmapPtr(-1, 0, guardAddr, uintptr(page), unix.PROT_NONE,
		unix.MAP_PRIVATE|unix.MAP_ANON|unix.MAP_FIXED_NOREPLACE)
	require.NoError(t, err)
	defer func() { require.NoError(t, unix.MunmapPtr(guard, uintptr(page))) }()
	old := data
	data[0] = 42
	data, err = mremap(data, 2*page)
	require.NoError(t, err)
	require.True(t, &old[0] != &data[0], "mapping must move past the guard page")
	require.Equal(t, byte(42), data[0])
	// Reject the stale slice without touching the old address, and accept the new one.
	require.ErrorIs(t, unix.Munmap(old), unix.EINVAL)
	require.NoError(t, unix.Munmap(data))
}

func TestMmapFileTruncateRegistry(t *testing.T) {
	for _, remove := range []bool{false, true} {
		t.Run(map[bool]string{false: "close", true: "delete"}[remove], func(t *testing.T) {
			name := filepath.Join(t.TempDir(), "mapped")
			m, err := OpenMmapFile(name, os.O_CREATE|os.O_RDWR, os.Getpagesize())
			require.ErrorIs(t, err, NewFile)
			m.Data[0] = 42
			require.NoError(t, m.Truncate(int64(2*os.Getpagesize())))
			require.Equal(t, byte(42), m.Data[0])
			require.NoError(t, m.Truncate(int64(os.Getpagesize())))
			data := m.Data
			if remove {
				err = m.Delete()
			} else {
				err = m.Close(-1)
			}
			require.NoError(t, err)
			err = unix.Munmap(data)
			require.ErrorIs(t, err, unix.EINVAL)
			if remove {
				_, err = os.Stat(name)
				require.ErrorIs(t, err, os.ErrNotExist)
			} else {
				contents, err := os.ReadFile(name)
				require.NoError(t, err)
				require.Equal(t, byte(42), contents[0])
			}
		})
	}
}
