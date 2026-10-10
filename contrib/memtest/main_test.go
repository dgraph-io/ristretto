/*
 * SPDX-FileCopyrightText: Copyright 2026 Istari Digital, Inc.
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import "testing"

func TestAllocationLifecycle(t *testing.T) {
	before := NumAllocBytes()
	for _, size := range []int{0, 1, 1536} {
		buf := Calloc(size)
		if len(buf) != size || cap(buf) != size {
			t.Fatalf("allocation length/capacity = %d/%d, want %d", len(buf), cap(buf), size)
		}
		for i, value := range buf {
			if value != 0 {
				t.Fatalf("allocation byte %d is not zero", i)
			}
			buf[i] = 1
		}
		if got := NumAllocBytes(); got != before+int64(size) {
			t.Fatalf("allocated bytes = %d, want %d", got, before+int64(size))
		}
		Free(buf)
	}
	if got := NumAllocBytes(); got != before {
		t.Fatalf("allocated bytes after free = %d, want %d", got, before)
	}
}

func TestMemoryReporting(t *testing.T) {
	before := NumAllocBytes()
	n := newS(1536)
	n.allocateNext(1024)
	memory()
	n.deallocNext()
	freeS(n)
	if got := NumAllocBytes(); got != before {
		t.Fatalf("allocated bytes after list cleanup = %d, want %d", got, before)
	}
}
