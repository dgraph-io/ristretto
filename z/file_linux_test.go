/*
 * SPDX-FileCopyrightText: © 2017-2025 Istari Digital, Inc.
 * SPDX-License-Identifier: Apache-2.0
 */

package z

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

// requireAllocated checks that fd is size bytes long and has a block on disk for each of them.
func requireAllocated(t *testing.T, fd *os.File, size int64) {
	t.Helper()
	var st unix.Stat_t
	require.NoError(t, unix.Fstat(int(fd.Fd()), &st))
	require.Equal(t, size, st.Size)
	require.GreaterOrEqual(t, st.Blocks*512, size, "the file has holes")
}

func TestOpenMmapFileAllocatesBlocks(t *testing.T) {
	dir := t.TempDir()
	probe, err := os.Create(filepath.Join(dir, "probe"))
	require.NoError(t, err)
	err = unix.Fallocate(int(probe.Fd()), 0, 0, 4096)
	require.NoError(t, probe.Close())
	if errors.Is(err, unix.EOPNOTSUPP) {
		t.Skip("the filesystem of the test directory has no fallocate")
	}
	require.NoError(t, err)

	mf, err := OpenMmapFile(filepath.Join(dir, "file"), os.O_RDWR|os.O_CREATE, 1<<20)
	require.ErrorIs(t, err, NewFile)
	defer func() { require.NoError(t, mf.Close(-1)) }()
	requireAllocated(t, mf.Fd, 1<<20)

	require.NoError(t, mf.Truncate(4<<20))
	requireAllocated(t, mf.Fd, 4<<20)
}

// When a new file cannot be mapped, OpenMmapFile frees the blocks that it allocated.
func TestOpenMmapFileFreesBlocksWhenMmapFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	// A file that is open for writing only cannot be mapped.
	_, err := OpenMmapFile(path, os.O_WRONLY|os.O_CREATE, 1<<20)
	require.ErrorIs(t, err, unix.EACCES)
	fi, err := os.Stat(path)
	require.NoError(t, err)
	require.Zero(t, fi.Size())
}

// When the mapping cannot grow, Truncate frees the blocks that it allocated.
func TestTruncateFreesBlocksWhenMremapFails(t *testing.T) {
	page := os.Getpagesize()
	mf, err := OpenMmapFile(filepath.Join(t.TempDir(), "file"), os.O_RDWR|os.O_CREATE, 2*page)
	require.ErrorIs(t, err, NewFile)
	data := mf.Data
	defer func() {
		// A failed mremap sets Data to nil and leaves the two pages mapped.
		mf.Data = data
		require.NoError(t, mf.Close(-1))
	}()

	// mremap cannot resize a range that mprotect has split into two mappings.
	require.NoError(t, unix.Mprotect(data[page:], unix.PROT_READ))
	require.ErrorIs(t, mf.Truncate(4<<20), unix.EFAULT)
	fi, err := mf.Fd.Stat()
	require.NoError(t, err)
	require.Equal(t, int64(2*page), fi.Size())
}

// mountTmpfs mounts a tmpfs of size bytes on a new directory. It needs root with CAP_SYS_ADMIN,
// and skips the test without them.
func mountTmpfs(t *testing.T, size int) string {
	t.Helper()
	if os.Geteuid() != 0 {
		t.Skip("mounting a small filesystem needs root")
	}
	dir := t.TempDir()
	err := unix.Mount("tmpfs", dir, "tmpfs", 0, fmt.Sprintf("size=%d", size))
	if errors.Is(err, unix.EPERM) {
		t.Skip("mounting a small filesystem needs CAP_SYS_ADMIN")
	}
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, unix.Unmount(dir, 0)) })
	return dir
}

// touch writes a byte into each page of data. A write the disk has no room for faults; under
// SetPanicOnFault the fault is a panic, which touch reports as a test failure.
func touch(t *testing.T, data []byte) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("a write into the mapping faulted: %v", r)
		}
	}()
	for off := 0; off < len(data); off += os.Getpagesize() {
		data[off] = 1
	}
}

// On a full disk, OpenMmapFile and Truncate return ENOSPC, instead of a mapping whose writes fault.
func TestMmapFileOnFullDisk(t *testing.T) {
	dir := mountTmpfs(t, 4<<20)
	defer debug.SetPanicOnFault(debug.SetPanicOnFault(true))

	var files []*MmapFile
	t.Cleanup(func() {
		for _, mf := range files {
			require.NoError(t, mf.Close(-1))
		}
	})
	var err error
	for i := 0; err == nil; i++ {
		require.Less(t, i, 8, "a 4 MiB filesystem took 8 files of 1 MiB")
		var mf *MmapFile
		mf, err = OpenMmapFile(filepath.Join(dir, fmt.Sprint(i)), os.O_RDWR|os.O_CREATE, 1<<20)
		if errors.Is(err, NewFile) {
			err = nil
			files = append(files, mf)
			touch(t, mf.Data)
		}
	}
	require.ErrorIs(t, err, syscall.ENOSPC)

	// Growing a file past the free space fails the same way, and keeps the file and its mapping.
	mf := files[0]
	require.ErrorIs(t, mf.Truncate(8<<20), syscall.ENOSPC)
	require.Len(t, mf.Data, 1<<20)
	requireAllocated(t, mf.Fd, 1<<20)
	touch(t, mf.Data)
}
