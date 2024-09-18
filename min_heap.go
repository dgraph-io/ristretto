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

package ristretto

// Item interface for heap elements
type Comparable[T any] interface {
	Less(other *T) bool
}

// MinHeap represents a min heap data structure
type MinHeap[T Comparable[T]] struct {
	items []*T
}

// NewMinHeap creates a new min heap
func NewMinHeap[T Comparable[T]]() *MinHeap[T] {
	return &MinHeap[T]{}
}

// Insert adds a new element to the heap
func (h *MinHeap[T]) Insert(item *T) {
	h.items = append(h.items, item)
	h.heapifyUp(len(h.items) - 1)
}

// Extract removes and returns the minimum element from the heap
func (h *MinHeap[T]) Extract() (*T, bool) {
	if len(h.items) == 0 {
		return nil, false
	}

	min := h.items[0]
	last := len(h.items) - 1
	h.items[0] = h.items[last]
	h.items = h.items[:last]

	if len(h.items) > 0 {
		h.heapifyDown(0)
	}

	return min, true
}

// heapifyUp maintains the heap property by moving a node up
func (h *MinHeap[T]) heapifyUp(index int) {
	for index > 0 {
		parentIndex := (index - 1) / 2
		if !(*h.items[index]).Less(h.items[parentIndex]) {
			break
		}
		h.items[parentIndex], h.items[index] = h.items[index], h.items[parentIndex]
		index = parentIndex
	}
}

// heapifyDown maintains the heap property by moving a node down
func (h *MinHeap[T]) heapifyDown(index int) {
	for {
		smallest := index
		leftChild := 2*index + 1
		rightChild := 2*index + 2

		if leftChild < len(h.items) && (*h.items[leftChild]).Less(h.items[smallest]) {
			smallest = leftChild
		}

		if rightChild < len(h.items) && (*h.items[rightChild]).Less(h.items[smallest]) {
			smallest = rightChild
		}

		if smallest == index {
			break
		}

		h.items[index], h.items[smallest] = h.items[smallest], h.items[index]
		index = smallest
	}
}

// Peek returns the minimum element without removing it
func (h *MinHeap[T]) Peek() (*T, bool) {
	if len(h.items) == 0 {
		return nil, false
	}
	return h.items[0], true
}

// Size returns the number of elements in the heap
func (h *MinHeap[T]) Size() int {
	return len(h.items)
}
