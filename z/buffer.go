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
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"sort"

	"github.com/pkg/errors"
)

// Buffer is equivalent of bytes.Buffer without the ability to read. It is NOT thread-safe.
//
// In UseCalloc mode, z.Calloc is used to allocate memory, which depending upon how the code is
// compiled could use jemalloc for allocations.
//
// In UseMmap mode, Buffer  uses file mmap to allocate memory. This allows us to store big data
// structures without using physical memory.
//
// MaxSize can be set to limit the memory usage.
type Buffer struct {
	buf     []byte
	offset  int
	curSz   int
	maxSz   int
	fd      *os.File
	bufType BufferType
}

type BufferType int

func (t BufferType) String() string {
	switch t {
	case UseCalloc:
		return "UseCalloc"
	case UseMmap:
		return "UseMmap"
	}
	return "invalid"
}

const (
	UseCalloc BufferType = iota
	UseMmap
	UseInvalid
)

// smallBufferSize is an initial allocation minimal capacity.
const smallBufferSize = 64

// Newbuffer is a helper utility, which creates a virtually unlimited Buffer in UseCalloc mode.
func NewBuffer(sz int) *Buffer {
	buf, err := NewBufferWith(sz, math.MaxInt64, UseCalloc)
	if err != nil {
		log.Fatalf("while creating buffer: %v", err)
	}
	return buf
}

// NewBufferWith would allocate a buffer of size sz upfront, with the total size of the buffer not
// exceeding maxSz. Both sz and maxSz can be set to zero, in which case reasonable defaults would be
// used. Buffer can't be used without initialization via NewBuffer.
func NewBufferWith(sz, maxSz int, bufType BufferType) (*Buffer, error) {
	var buf []byte
	var fd *os.File

	if sz == 0 {
		sz = smallBufferSize
	}
	if maxSz == 0 {
		maxSz = math.MaxInt32
	}

	switch bufType {
	case UseCalloc:
		buf = Calloc(sz)

	case UseMmap:
		var err error
		fd, err = ioutil.TempFile("", "buffer")
		if err != nil {
			return nil, err
		}
		if err := fd.Truncate(int64(sz)); err != nil {
			return nil, errors.Wrapf(err, "while truncating %s to size: %d", fd.Name(), sz)
		}

		buf, err = Mmap(fd, true, int64(maxSz)) // Mmap up to max size.
		if err != nil {
			return nil, errors.Wrapf(err, "while mmapping %s with size: %d", fd.Name(), maxSz)
		}

	default:
		log.Fatalf("Invalid bufType: %q\n", bufType)
	}

	buf[0] = 0x00
	return &Buffer{
		buf:     buf,
		offset:  1, // Always leave offset 0.
		curSz:   sz,
		maxSz:   maxSz,
		fd:      fd,
		bufType: bufType,
	}, nil
}

func (b *Buffer) IsEmpty() bool {
	return b.offset == 1
}

// Len would return the number of bytes written to the buffer so far.
func (b *Buffer) Len() int {
	return b.offset
}

// Bytes would return all the written bytes as a slice.
func (b *Buffer) Bytes() []byte {
	return b.buf[1:b.offset]
}

// Grow would grow the buffer to have at least n more bytes. In case the buffer is at capacity, it
// would reallocate twice the size of current capacity + n, to ensure n bytes can be written to the
// buffer without further allocation. In UseMmap mode, this might result in underlying file expansion.
func (b *Buffer) Grow(n int) {
	// In this case, len and cap are the same.
	if b.buf == nil {
		panic("z.Buffer needs to be initialized before using")
	}
	if b.maxSz-b.offset < n {
		panic(fmt.Sprintf("Buffer max size exceeded: %d."+
			" Offset: %d. Grow: %d", b.maxSz, b.offset, n))
	}
	if b.curSz-b.offset > n {
		return
	}

	growBy := b.curSz + n
	if growBy > 1<<30 {
		growBy = 1 << 30
	}
	b.curSz += growBy

	switch b.bufType {
	case UseCalloc:
		newBuf := Calloc(b.curSz)
		copy(newBuf, b.buf[:b.offset])
		Free(b.buf)
		b.buf = newBuf
	case UseMmap:
		if err := b.fd.Truncate(int64(b.curSz)); err != nil {
			log.Fatalf("While trying to truncate file %s to size: %d error: %v\n",
				b.fd.Name(), b.curSz, err)
		}
	}
}

// Allocate is a way to get a slice of size n back from the buffer. This slice can be directly
// written to. Warning: Allocate is not thread-safe. The byte slice returned MUST be used before
// further calls to Buffer.
func (b *Buffer) Allocate(n int) []byte {
	b.Grow(n)
	off := b.offset
	b.offset += n
	return b.buf[off:b.offset]
}

// AllocateOffset works the same way as allocate, but instead of returning a byte slice, it returns
// the offset of the allocation.
func (b *Buffer) AllocateOffset(n int) int {
	b.Grow(n)
	b.offset += n
	return b.offset - n
}

func (b *Buffer) writeLen(sz int) {
	buf := b.Allocate(4)
	binary.BigEndian.PutUint32(buf, uint32(sz))
}

// SliceAllocate would encode the size provided into the buffer, followed by a call to Allocate,
// hence returning the slice of size sz. This can be used to allocate a lot of small buffers into
// this big buffer.
// Note that SliceAllocate should NOT be mixed with normal calls to Write.
func (b *Buffer) SliceAllocate(sz int) []byte {
	b.Grow(4 + sz)
	b.writeLen(sz)
	return b.Allocate(sz)
}

type LessFunc func(a, b []byte) bool
type sortHelper struct {
	offsets []int
	b       *Buffer
	tmp     *Buffer
	less    LessFunc
	small   []int
}

func (s *sortHelper) sortSmall(start, end int) {
	fmt.Printf("sortSmall: %d %d\n", start, end)
	s.tmp.Reset()
	s.small = s.small[:0]
	next := start
	for next != 0 && next < end {
		s.small = append(s.small, next)
		_, next = s.b.Slice(next)
	}

	sort.Slice(s.small, func(i, j int) bool {
		left, _ := s.b.Slice(s.small[i])
		right, _ := s.b.Slice(s.small[j])
		return s.less(left, right)
	})
	for _, off := range s.small {
		s.tmp.Write(s.b.rawSlice(off))
	}
	assert(s.tmp.Len()-1 == end-start)
	copy(s.b.buf[start:end], s.tmp.Bytes())
}

func assert(b bool) {
	if !b {
		log.Fatalf("Assertion failure")
	}
}
func check(err error) {
	assert(err == nil)
}
func check2(_ interface{}, err error) {
	check(err)
}

func (s *sortHelper) merge(lo, m, hi int) {
	left := s.offsets[lo]
	mid := s.offsets[m]
	right := s.offsets[hi]
	fmt.Printf("merge: %d %d %d\n", left, mid, right)
	if left == mid {
		return
	}
	s.tmp.Reset()
	check2(s.tmp.Write(s.b.buf[left:mid]))

	ln, rn := 1, mid
	var ls, rs []byte

	copyLeft := func() {
		assert(len(ls) == copy(s.b.buf[left:], ls))
		ln += len(ls)
		left += len(ls)
	}
	copyRight := func() {
		assert(len(rs) == copy(s.b.buf[left:], rs))
		rn += len(rs)
		left += len(rs)
	}

	for left < right {
		if ln >= s.tmp.Len() {
			rs = s.b.rawSlice(rn)
			copyRight()

		} else if rn >= right {
			ls = s.tmp.rawSlice(ln)
			copyLeft()

		} else {
			ls = s.tmp.rawSlice(ln)
			rs = s.b.rawSlice(rn)

			if s.less(ls[4:], rs[4:]) {
				copyLeft()
			} else {
				copyRight()
			}
		}
	}
}

func (s *sortHelper) sort(lo, hi int) {
	fmt.Printf("sort: lo: %d hi: %d\n", lo, hi)
	if hi <= lo {
		return
	}
	mid := lo + (hi-lo)/2
	s.sort(lo, mid)
	s.sort(mid+1, hi)
	s.merge(lo, mid, hi)
}

// SortSlice is like SortSliceBetween but sorting over the entire buffer.
func (b *Buffer) SortSlice(less func(left, right []byte) bool) {
	b.SortSliceBetween(1, b.offset, less)
}
func (b *Buffer) SortSliceBetween(start, end int, less LessFunc) {
	offsets := make([]int, 0, 1024)
	next, count := start, 0
	for next != 0 && next < end {
		if count%1024 == 0 {
			offsets = append(offsets, next)
		}
		_, next = b.Slice(next)
		count++
	}
	if offsets[len(offsets)-1] != end {
		offsets = append(offsets, end)
	}

	s := &sortHelper{
		offsets: offsets,
		b:       b,
		less:    less,
		small:   make([]int, 0, 1024),
		tmp:     NewBuffer((end - start) / 2),
	}
	defer s.tmp.Release()

	left := offsets[0]
	for _, off := range offsets[1:] {
		s.sortSmall(left, off)
		left = off
	}
	s.sort(0, len(offsets)-1)
	fmt.Printf("tmp buffer size: %d. slice size: %d\n", s.tmp.curSz, end-start)
}

// SortSliceBetween accepts a less function, to allow in-place sorting of slices. It also accepts
// start and end offsets in the buffer, to only sort portion of the buffer. Ensure that the less
// function returns false on equals.
// func (b *Buffer) SortSliceBetweenX(start, end int, less func(a, b []byte) bool) {
// 	tmpBuf := make([]byte, 0, 1024)

// 	// We use bubble sort here, to only deal with two consecutive buffers at a time.
// 	didSwap := true
// 	cycles := 0
// 	for didSwap { // No swap happened, slice is sorted.
// 		didSwap = false
// 		lo := start
// 		count := 0
// 		for lo < end {
// 			count++
// 			left := b.rawSlice(lo)
// 			ro := lo + len(left)
// 			if ro >= end {
// 				break
// 			}
// 			right := b.rawSlice(ro)
// 			if less(right[4:], left[4:]) {
// 				// fmt.Printf("swapping: %x with %x\n", right[4:], left[4:])
// 				didSwap = true
// 				tmpBuf = append(tmpBuf[:0], left...)
// 				assert(copy(b.buf[lo:], right) == len(right))
// 				assert(copy(b.buf[lo+len(right):], tmpBuf) == len(left))
// 				lo += len(right)
// 			} else {
// 				lo += len(left)
// 			}
// 		}
// 		cycles++
// 		fmt.Printf("Count: %d. Cycles: %d\n", count, cycles)
// 	}
// }

func (b *Buffer) rawSlice(offset int) []byte {
	sz := binary.BigEndian.Uint32(b.buf[offset:])
	return b.buf[offset : offset+4+int(sz)]
}

// Slice would return the slice written at offset.
func (b *Buffer) Slice(offset int) ([]byte, int) {
	sz := binary.BigEndian.Uint32(b.buf[offset:])
	start := offset + 4
	next := start + int(sz)
	res := b.buf[start:next]
	if next >= b.offset {
		next = 0
	}
	return res, next
}

func (b *Buffer) Data(offset int) []byte {
	if offset > b.curSz {
		panic("offset beyond current size")
	}
	return b.buf[offset:b.curSz]
}

// Write would write p bytes to the buffer.
func (b *Buffer) Write(p []byte) (n int, err error) {
	b.Grow(len(p))
	n = copy(b.buf[b.offset:], p)
	b.offset += n
	return n, nil
}

// Reset would reset the buffer to be reused.
func (b *Buffer) Reset() {
	b.offset = 1
}

// Release would free up the memory allocated by the buffer. Once the usage of buffer is done, it is
// important to call Release, otherwise a memory leak can happen.
func (b *Buffer) Release() error {
	switch b.bufType {
	case UseCalloc:
		Free(b.buf)

	case UseMmap:
		fname := b.fd.Name()
		if err := Munmap(b.buf); err != nil {
			return errors.Wrapf(err, "while munmap file %s", fname)
		}
		if err := b.fd.Truncate(0); err != nil {
			return errors.Wrapf(err, "while truncating file %s", fname)
		}
		if err := b.fd.Close(); err != nil {
			return errors.Wrapf(err, "while closing file %s", fname)
		}
		if err := os.Remove(b.fd.Name()); err != nil {
			return errors.Wrapf(err, "while deleting file %s", fname)
		}
	}
	return nil
}
