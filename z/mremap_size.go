//go:build linux && !arm64 && !arm && !s390x && !riscv64 && !js
// +build linux,!arm64,!arm,!s390x,!riscv64,!js

/*
 * SPDX-FileCopyrightText: © 2017-2025 Istari Digital, Inc.
 * SPDX-License-Identifier: Apache-2.0
 */

package z

import "golang.org/x/sys/unix"

// mremap is a Linux-specific system call to remap pages in memory. This can be used in place of munmap + mmap.
func mremap(data []byte, size int) ([]byte, error) {
	return unix.Mremap(data, size, unix.MREMAP_MAYMOVE)
}
