/*
 * Copyright 2020 Dgraph Labs, Inc. and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package z

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/dustin/go-humanize"
)

// Allocator amortizes the cost of small allocations by allocating memory in bigger chunks.
// Internally it uses z.Calloc to allocate memory. Once allocated, the memory is not moved,
// so it is safe to use the allocated bytes to unsafe cast them to Go struct pointers.
type Allocator struct {
	sync.Mutex
	pageSize int
	curBuf   int
	curIdx   int
	buffers  [][]byte
	size     uint64
	Ref      uint64
	Tag      string
	reused   int

	freelist map[int]uint64 // Key is the size. Value is the first node of that size.
}

// allocs keeps references to all Allocators, so we can safely discard them later.
var allocsMu *sync.Mutex
var allocRef uint64
var allocs map[uint64]*Allocator
var calculatedLog2 []int
var allocatorPool chan *Allocator
var numGets int64

func init() {
	allocsMu = new(sync.Mutex)
	allocs = make(map[uint64]*Allocator)

	// Set up a unique Ref per process.
	rand.Seed(time.Now().UnixNano())
	allocRef = uint64(rand.Int63n(1<<16)) << 48

	calculatedLog2 = make([]int, 1025)
	for i := 1; i <= 1024; i++ {
		calculatedLog2[i] = int(math.Log2(float64(i)))
	}
	allocatorPool = make(chan *Allocator, 8)
	go freeupAllocators()
	// fmt.Printf("Using z.Allocator with starting ref: %x\n", allocRef)
}

func freeupAllocators() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var last int64
	for range ticker.C {
		gets := atomic.LoadInt64(&numGets)
		if gets != last {
			// Some retrievals were made since the last time. So, let's avoid doing a release.
			last = gets
			continue
		}
		select {
		case alloc := <-allocatorPool:
			alloc.Release()
		default:
		}
	}
}

func GetAllocatorFromPool(sz int) *Allocator {
	atomic.AddInt64(&numGets, 1)
	select {
	case alloc := <-allocatorPool:
		alloc.Reset()
		return alloc
	default:
		return NewAllocator(sz)
	}
}
func ReturnAllocator(a *Allocator) {
	a.TrimTo(400 << 20)

	select {
	case allocatorPool <- a:
		return
	default:
		a.Release()
	}
}

// NewAllocator creates an allocator starting with the given size.
func NewAllocator(sz int) *Allocator {
	ref := atomic.AddUint64(&allocRef, 1)
	// We should not allow a zero sized page because addBufferWithMinSize
	// will run into an infinite loop tyring to double the pagesize.
	if sz == 0 {
		sz = 1
	}
	a := &Allocator{
		pageSize: sz,
		Ref:      ref,
		freelist: make(map[int]uint64),
	}

	allocsMu.Lock()
	allocs[ref] = a
	allocsMu.Unlock()
	return a
}

func (a *Allocator) Reset() {
	a.curBuf, a.curIdx = 0, 0
	a.freelist = make(map[int]uint64)
}

func PrintAllocators() {
	allocsMu.Lock()
	tags := make(map[string]int)
	var total uint64
	for _, ac := range allocs {
		tags[ac.Tag]++
		total += ac.Allocated()
	}
	for tag, count := range tags {
		fmt.Printf("Allocator Tag: %s Count: %d\n", tag, count)
	}
	fmt.Printf("Total allocators: %d. Total Size: %s\n",
		len(allocs), humanize.IBytes(total))
	allocsMu.Unlock()
}

// AllocatorFrom would return the allocator corresponding to the ref.
func AllocatorFrom(ref uint64) *Allocator {
	allocsMu.Lock()
	a := allocs[ref]
	allocsMu.Unlock()
	return a
}

// Size returns the size of the allocations so far.
func (a *Allocator) Size() uint64 {
	a.Lock()
	defer a.Unlock()

	return a.size
}

func log2(sz int) int {
	if sz < len(calculatedLog2) {
		return calculatedLog2[sz]
	}
	pow := 10
	sz >>= 10
	for sz > 1 {
		sz >>= 1
		pow++
	}
	return pow
}

func (a *Allocator) addToFreelist(b []byte) {
	if len(b) < 32 {
		// Don't do anything.
		return
	}

	l2 := log2(len(b))
	root := a.freelist[l2]
	n := node(b)
	// Length would be the first 8 bytes.
	n.setAt(0, uint64(len(b)))
	// Followed by the pointer to the next byte array.
	n.setAt(8, root)
	a.freelist[l2] = uint64(uintptr(unsafe.Pointer(&b[0])))
}

func (a *Allocator) Return(b []byte) {
	// Turning this off for now.
	// a.Lock()
	// defer a.Unlock()
	// a.addToFreelist(b)
}

func getBuf(p uint64, sz int) []byte {
	return (*[MaxArrayLen]byte)(unsafe.Pointer(uintptr(p)))[:sz:sz]
}

func (a *Allocator) fromFreeList(need int) []byte {
	var last uint64
	span := log2(need)
	n := a.freelist[span]
	for n != 0 {
		curBuf := getBuf(n, 16)
		curNode := node(curBuf)
		sz := int(curNode.uint64(0))
		if sz < need {
			last, n = n, curNode.uint64(8)
			continue
		}
		curBuf = getBuf(n, sz)
		use := curBuf[:need]
		left := curBuf[need:]

		next := node(curBuf).uint64(8)
		if last == 0 {
			a.freelist[span] = next
		} else if last > 0 {
			lastBuf := getBuf(last, 16)
			node(lastBuf).setAt(8, next)
		}

		a.addToFreelist(left)
		ZeroOut(use, 0, 16) // just zero out the initial 16 bytes.
		return use
	}
	return nil
}

func (a *Allocator) Allocated() uint64 {
	a.Lock()
	defer a.Unlock()
	var alloc int
	for _, b := range a.buffers {
		alloc += cap(b)
	}
	return uint64(alloc)
}

func (a *Allocator) TrimTo(max int) {
	a.Lock()
	defer a.Unlock()

	var alloc int
	idx := -1
	for i, b := range a.buffers {
		alloc += len(b)
		if alloc >= max {
			idx = i
			break
		}
	}
	if idx == -1 {
		return
	}
	for _, b := range a.buffers[idx:] {
		fmt.Printf("Trim: Removing buffer of size: %d\n", len(b))
		Free(b)
	}
	a.buffers = a.buffers[:idx]
	alloc = 0
	for _, b := range a.buffers {
		alloc += len(b)
	}
	fmt.Printf("Trim: Final size: %d\n", alloc)
}

// Release would release the memory back. Remember to make this call to avoid memory leaks.
func (a *Allocator) Release() {
	if a == nil {
		return
	}

	a.Lock()
	defer a.Unlock()

	var alloc int
	for _, b := range a.buffers {
		alloc += len(b)
		Free(b)
	}
	// ratio := float64(a.reused) / float64(alloc)
	// if ratio == 0.0 || ratio > 0.5 {
	fmt.Printf("Releasing Allocator Size: %s\n",
		humanize.IBytes(uint64(alloc)))
	// }

	allocsMu.Lock()
	delete(allocs, a.Ref)
	allocsMu.Unlock()
}

const maxAlloc = 1 << 30

func (a *Allocator) MaxAlloc() int {
	return maxAlloc
}

const nodeAlign = int(unsafe.Sizeof(uint64(0))) - 1

func (a *Allocator) AllocateAligned(sz int) []byte {
	tsz := sz + nodeAlign
	out := a.Allocate(tsz)
	// TODO: We should align based on out's address.
	aligned := (a.curIdx - tsz + nodeAlign) & ^nodeAlign

	start := tsz - (a.curIdx - aligned)
	return out[start : start+sz]
}

func (a *Allocator) Copy(buf []byte) []byte {
	if a == nil {
		return append([]byte{}, buf...)
	}
	out := a.Allocate(len(buf))
	copy(out, buf)
	return out
}

func (a *Allocator) addBufferWithMinSize(sz int) {
	for {
		a.pageSize *= 2 // Do multiply by 2 here.
		if a.pageSize >= sz {
			break
		}
	}
	if a.pageSize > maxAlloc {
		a.pageSize = maxAlloc
	}

	buf := Calloc(a.pageSize)
	a.buffers = append(a.buffers, buf)
}

// Allocate would allocate a byte slice of length sz. It is safe to use this memory to unsafe cast
// to Go structs.
func (a *Allocator) Allocate(sz int) []byte {
	if a == nil {
		return make([]byte, sz)
	}
	a.Lock()
	defer a.Unlock()

	if len(a.buffers) == 0 {
		buf := Calloc(a.pageSize)
		a.buffers = append(a.buffers, buf)
	}
	if sz >= maxAlloc {
		panic(fmt.Sprintf("Allocate call exceeds max allocation possible."+
			" Requested: %d. Max Allowed: %d\n", sz, maxAlloc))
	}
	// Turning this off for now. This slows down allocations somewhat. Even though it shows a 50%
	// reuse ratio, we get a better bang for the buck by just reusing an entire Allocator for the
	// next cycle.
	//
	// if out := a.fromFreeList(sz); out != nil {
	// 	a.reused += len(out)
	// 	return out
	// }
	cb := a.buffers[a.curBuf]
	for len(cb) < a.curIdx+sz {
		a.curBuf++
		a.curIdx = 0
		if a.curBuf == len(a.buffers) {
			a.addBufferWithMinSize(sz)
		}
		cb = a.buffers[a.curBuf]
	}
	slice := cb[a.curIdx : a.curIdx+sz]
	a.curIdx += sz
	a.size += uint64(sz)
	return slice
}
