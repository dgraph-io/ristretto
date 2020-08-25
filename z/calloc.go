// Copyright 2020 The LevelDB-Go and Pebble Authors. All rights reserved. Use
// of this source code is governed by a BSD-style license that can be found in
// the LICENSE file.

// +build !jemalloc

package z

import "fmt"

// Provides versions of New and Free when cgo is not available (e.g. cross
// compilation).

func NumAllocBytes() int64 {
	return 0
}

// New allocates a slice of size n.
func Calloc(n int) []byte {
	return make([]byte, n)
}

// Free frees the specified slice.
func Free(b []byte) {
}

func StatsPrint() {
	fmt.Printf("Using Go memory")
}
