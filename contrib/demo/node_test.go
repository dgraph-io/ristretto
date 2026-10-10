/*
 * SPDX-FileCopyrightText: Copyright 2026 Istari Digital, Inc.
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"testing"

	"github.com/dgraph-io/ristretto/v2/z"
)

func TestNodeLifecycle(t *testing.T) {
	if alloc != nil {
		original := alloc
		alloc = z.NewAllocator(1024, t.Name())
		t.Cleanup(func() {
			alloc.Release()
			alloc = original
		})
	}
	root := newNode(-1)
	tail := root
	for i := 0; i < 16; i++ {
		tail.next = newNode(i)
		tail = tail.next
	}
	printNode(root)
	for current, want := root, -1; current != nil; want++ {
		if current.val != want {
			t.Fatalf("node value = %d, want %d", current.val, want)
		}
		next := current.next
		freeNode(current)
		current = next
	}
}
