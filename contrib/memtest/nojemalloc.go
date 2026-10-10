//go:build !jemalloc
// +build !jemalloc

package main

// #include <stdlib.h>
import "C"
import (
	"log"
	"sync/atomic"
	"unsafe"
)

// Calloc allocates zero-initialized C memory and tracks its size until Free is called.
func Calloc(size int) []byte {
	if size == 0 {
		return make([]byte, 0)
	}
	ptr := C.calloc(C.size_t(size), 1)
	if ptr == nil {
		panic("OOM")
	}
	atomic.AddInt64(&numbytes, int64(size))
	return unsafe.Slice((*byte)(ptr), size)
}

// Free releases a Calloc allocation and subtracts its capacity from byte accounting.
func Free(bs []byte) {
	if len(bs) == 0 {
		return
	}

	if sz := cap(bs); sz != 0 {
		bs = bs[:cap(bs)]
		C.free(unsafe.Pointer(&bs[0]))
		atomic.AddInt64(&numbytes, -int64(sz))
	}
}

// NumAllocBytes returns the number of C-allocated bytes not yet freed.
func NumAllocBytes() int64 { return atomic.LoadInt64(&numbytes) }

// check needs no allocator setup for the standard C allocation mode.
func check() {}

// init identifies the allocator mode in diagnostic output.
func init() {
	log.Println("USING CALLOC")
}
