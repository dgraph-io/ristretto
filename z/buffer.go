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
)

// Buffer is equivalent of bytes.Buffer without the ability to read. It uses z.Calloc to allocate
// memory, which depending upon how the code is compiled could use jemalloc for allocations.
type Buffer struct {
	buf    []byte
	offset int
}

// NewBuffer would allocate a buffer of size sz upfront.
func NewBuffer(sz int) *Buffer {
	return &Buffer{
		buf:    Calloc(sz),
		offset: 0,
	}
}

// Len would return the number of bytes written to the buffer so far.
func (b *Buffer) Len() int {
	return b.offset
}

// Bytes would return all the written bytes as a slice.
func (b *Buffer) Bytes() []byte {
	return b.buf[0:b.offset]
}

// smallBufferSize is an initial allocation minimal capacity.
const smallBufferSize = 64

// Grow would grow the buffer to have at least n more bytes. In case the buffer is at capacity, it
// would reallocate twice the size of current capacity + n, to ensure n bytes can be written to the
// buffer without further allocation.
func (b *Buffer) Grow(n int) {
	// In this case, len and cap are the same.
	if len(b.buf) == 0 && n <= smallBufferSize {
		b.buf = Calloc(smallBufferSize)
		return
	} else if b.buf == nil {
		b.buf = Calloc(n)
		return
	}
	if b.offset+n < len(b.buf) {
		return
	}

	sz := 2*len(b.buf) + n
	newBuf := Calloc(sz)
	copy(newBuf, b.buf[:b.offset])
	Free(b.buf)
	b.buf = newBuf
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

func (b *Buffer) writeLen(sz int) {
	buf := b.Allocate(4)
	binary.BigEndian.PutUint32(buf, uint32(sz))
}

// SliceAllocate would encode the size provided into the buffer, followed by a call to Allocate,
// hence returning the slice of size sz. This can be used to allocate a lot of small buffers into
// this big buffer.
// Note that SliceAllocate should NOT be mixed with normal calls to Write. Otherwise, SliceOffsets
// won't work.
func (b *Buffer) SliceAllocate(sz int) []byte {
	b.Grow(4 + sz)
	b.writeLen(sz)
	return b.Allocate(sz)
}

// SliceOffsets would return the offsets of all slices written to the buffer.
// TODO: Perhaps keep the offsets separate in another buffer, and allow access to slices via index.
func (b *Buffer) SliceOffsets(offsets []int) []int {
	start := 0
	for start < b.offset {
		offsets = append(offsets, start)
		sz := binary.BigEndian.Uint32(b.buf[start:])
		start += 4 + int(sz)
	}
	return offsets
}

// Slice would return the slice written at offset.
func (b *Buffer) Slice(offset int) []byte {
	sz := binary.BigEndian.Uint32(b.buf[offset:])
	start := offset + 4
	return b.buf[start : start+int(sz)]
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
	b.offset = 0
}

// Release would free up the memory allocated by the buffer. Once the usage of buffer is done, it is
// important to call Release, otherwise a memory leak can happen.
func (b *Buffer) Release() {
	Free(b.buf)
}
