// Copyright 2020 The LevelDB-Go and Pebble Authors. All rights reserved. Use
// of this source code is governed by a BSD-style license that can be found in
// the LICENSE file.

// +build jemalloc

package z

/*
#cgo LDFLAGS: -L/usr/local/lib -Wl,-rpath,/usr/local/lib -ljemalloc -lm -lstdc++ -pthread -ldl
#include <stdlib.h>
#include <jemalloc/jemalloc.h>
*/
import "C"
import (
	"sync/atomic"
	"unsafe"
)

// The go:linkname directives provides backdoor access to private functions in
// the runtime. Below we're accessing the throw function.

//go:linkname throw runtime.throw
func throw(s string)

// New allocates a slice of size n. The returned slice is from manually managed
// memory and MUST be released by calling Free. Failure to do so will result in
// a memory leak.
//
// Compile jemalloc with ./configure --with-jemalloc-prefix="je_"
// https://android.googlesource.com/platform/external/jemalloc_new/+/6840b22e8e11cb68b493297a5cd757d6eaa0b406/TUNING.md
// These two config options seems useful for frequent allocations and deallocations in
// multi-threaded programs (like we have).
// JE_MALLOC_CONF="background_thread:true,metadata_thp:auto"
//
// Compile Go program with `go build -tags=jemalloc` to enable this.
func Calloc(n int) []byte {
	if n == 0 {
		return make([]byte, 0)
	}
	// We need to be conscious of the Cgo pointer passing rules:
	//
	//   https://golang.org/cmd/cgo/#hdr-Passing_pointers
	//
	//   ...
	//   Note: the current implementation has a bug. While Go code is permitted
	//   to write nil or a C pointer (but not a Go pointer) to C memory, the
	//   current implementation may sometimes cause a runtime error if the
	//   contents of the C memory appear to be a Go pointer. Therefore, avoid
	//   passing uninitialized C memory to Go code if the Go code is going to
	//   store pointer values in it. Zero out the memory in C before passing it
	//   to Go.

	ptr := C.je_calloc(C.size_t(n), 1)
	if ptr == nil {
		// NB: throw is like panic, except it guarantees the process will be
		// terminated. The call below is exactly what the Go runtime invokes when
		// it cannot allocate memory.
		throw("out of memory")
	}
	atomic.AddInt64(&numBytes, int64(n))
	// Interpret the C pointer as a pointer to a Go array, then slice.
	return (*[MaxArrayLen]byte)(unsafe.Pointer(ptr))[:n:n]
}

// CallocNoRef does the exact same thing as Calloc with jemalloc enabled.
func CallocNoRef(n int) []byte {
	return Calloc(n)
}

// Free frees the specified slice.
func Free(b []byte) {
	if sz := cap(b); sz != 0 {
		if len(b) == 0 {
			b = b[:cap(b)]
		}
		ptr := unsafe.Pointer(&b[0])
		C.je_free(ptr)
		atomic.AddInt64(&numBytes, -int64(sz))
	}
}

func StatsPrint() {
	opts := C.CString("mdablxe")
	C.je_malloc_stats_print(nil, nil, opts)
	C.free(unsafe.Pointer(opts))
}
