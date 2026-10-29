/*
 * SPDX-FileCopyrightText: © 2017-2025 Istari Digital, Inc.
 * SPDX-License-Identifier: Apache-2.0
 */

package z

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// allocate sets the size of fd to size bytes and allocates their blocks, so that a write into a
// mapping of the file never meets a disk without room for it, which would end the process with
// SIGBUS. A filesystem without fallocate gets a sparse file, as from Truncate. When the allocation
// fails, the file gets its earlier size back: ext4 keeps the blocks and the size that a failed
// fallocate reached.
func allocate(fd *os.File, size int64) error {
	fi, err := fd.Stat()
	if err != nil {
		return err
	}
	for {
		err = unix.Fallocate(int(fd.Fd()), 0, 0, size)
		// tmpfs stops an allocation when a signal arrives.
		if !errors.Is(err, unix.EINTR) {
			break
		}
	}
	if errors.Is(err, unix.EOPNOTSUPP) {
		return fd.Truncate(size)
	}
	if err != nil {
		return errors.Join(err, fd.Truncate(fi.Size()))
	}
	return nil
}

// Truncate would truncate the mmapped file to the given size. On Linux, we resize
// the underlying file, allocating its blocks when it grows, and then call mremap,
// but on other systems, we unmap first, then truncate, then re-map.
func (m *MmapFile) Truncate(maxSz int64) error {
	if err := m.Sync(); err != nil {
		return fmt.Errorf("while sync file: %s, error: %v\n", m.Fd.Name(), err)
	}
	size := int64(len(m.Data))
	resize := m.Fd.Truncate
	if maxSz > size {
		resize = func(sz int64) error { return allocate(m.Fd, sz) }
	}
	if err := resize(maxSz); err != nil {
		return fmt.Errorf("while truncate file: %s, error: %w", m.Fd.Name(), err)
	}

	var err error
	m.Data, err = mremap(m.Data, int(maxSz)) // Mmap up to max size.
	if err != nil && maxSz > size {
		// The mapping did not grow. Truncating the file back frees the blocks allocated above.
		err = errors.Join(err, m.Fd.Truncate(size))
	}
	return err
}
