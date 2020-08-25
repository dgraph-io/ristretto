package z

import "sync/atomic"

var numBytes int64

// NumAllocBytes returns the number of bytes allocated using calls to z.Calloc. The allocations
// could be happening via either Go or jemalloc, depending upon the build flags.
func NumAllocBytes() int64 {
	return atomic.LoadInt64(&numBytes)
}
