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

import "fmt"

// Allocator amortizes the cost of small allocations by allocating memory in bigger chunks.
// Internally it uses z.Calloc to allocate memory. Once allocated, the memory is not moved,
// so it is safe to use the allocated bytes to unsafe cast them to Go struct pointers.
type Allocator struct {
	pageSize int
	curBuf   int
	curIdx   int
	buffers  [][]byte
	size     uint64
}

// NewAllocator creates an allocator starting with the given size.
func NewAllocator(sz int) *Allocator {
	return &Allocator{pageSize: sz}
}

// Size returns the size of the allocations so far.
func (a *Allocator) Size() uint64 {
	return a.size
}

// Release would release the memory back. Remember to make this call to avoid memory leaks.
func (a *Allocator) Release() {
	for _, b := range a.buffers {
		Free(b)
	}
}

const maxAlloc = 1 << 30

func (a *Allocator) MaxAlloc() int {
	return maxAlloc
}

// Allocate would allocate a byte slice of length sz. It is safe to use this memory to unsafe cast
// to Go structs.
func (a *Allocator) Allocate(sz int) []byte {
	if len(a.buffers) == 0 {
		buf := Calloc(a.pageSize)
		a.buffers = append(a.buffers, buf)
	}

	if sz >= maxAlloc {
		panic(fmt.Sprintf("Allocate call exceeds max allocation possible."+
			" Requested: %d. Max Allowed: %d\n", sz, maxAlloc))
	}
	cb := a.buffers[a.curBuf]
	if len(cb) < a.curIdx+sz {
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
		a.curBuf++
		a.curIdx = 0
		cb = a.buffers[a.curBuf]
	}

	slice := cb[a.curIdx : a.curIdx+sz]
	a.curIdx += sz
	a.size += uint64(sz)
	return slice
}
