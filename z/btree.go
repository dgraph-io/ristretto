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
	"io/ioutil"
	"math"
	"os"
	"strings"
	"unsafe"
)

var (
	pageSize = os.Getpagesize()
	maxKeys  = (pageSize / 16) - 1
)

// Tree represents the structure for custom mmaped B+ tree.
// It supports keys in range [1, math.MaxUint64-1] and values [1, math.Uint64].
type Tree struct {
	mf       *MmapFile
	nextPage uint64
}

// Release the memory allocated to tree.
func (t *Tree) Release() {
	if t != nil && t.mf != nil {
		check(t.mf.Delete())
	}
}

// NewTree returns a memory mapped B+ tree.
func NewTree(maxSz int) *Tree {
	// Tell kernel that we'd be reading pages in random order, so don't do read ahead.
	fd, err := ioutil.TempFile("", "btree")
	check(err)

	mf, err := OpenMmapFileUsing(fd, maxSz, true)
	if err != NewFile {
		check(err)
	}
	check(Madvise(mf.Data, false))

	t := &Tree{
		mf:       mf,
		nextPage: 1,
	}
	// This is the root node.
	t.newNode(0)

	// This acts as the rightmost pointer (all the keys are <= this key).
	t.Set(math.MaxUint64-1, 0)
	return t
}

type TreeStats struct {
	NumPages  int
	Bytes     int
	Occupancy float64
}

// Stats returns stats about the tree.
func (t *Tree) Stats() TreeStats {
	var totalKeys, maxPossible int
	fn := func(n node) {
		totalKeys += n.numKeys()
		maxPossible += maxKeys
	}
	t.Iterate(fn)
	occ := float64(totalKeys) / float64(maxPossible)

	return TreeStats{
		NumPages:  int(t.nextPage - 1),
		Bytes:     int(t.nextPage-1) * pageSize,
		Occupancy: occ,
	}
}

func (t *Tree) newNode(bit byte) node {
	offset := int(t.nextPage) * pageSize
	t.nextPage++
	sz := len(t.mf.Data)
	// Double the size of file if current buffer is insufficient.
	if offset+pageSize > sz {
		check(t.mf.Truncate(int64(2 * sz)))
	}
	n := node(t.mf.Data[offset : offset+pageSize])
	ZeroOut(n, 0, len(n))
	n.setBit(bitUsed | bit)
	n.setAt(keyOffset(maxKeys), t.nextPage-1)
	return n
}

func (t *Tree) node(pid uint64) node {
	// page does not exist
	if pid == 0 {
		return nil
	}
	start := pageSize * int(pid)
	return node(t.mf.Data[start : start+pageSize])
}

// Set sets the key-value pair in the tree.
func (t *Tree) Set(k, v uint64) {
	if k == math.MaxUint64 || k == 0 {
		panic("Error setting zero or MaxUint64")
	}
	root := t.set(1, k, v)
	if root.isFull() {
		right := t.split(1)
		left := t.newNode(root.bits())
		// Re-read the root as the underlying buffer for tree might have changed during split.
		root = t.node(1)
		copy(left[:keyOffset(maxKeys)], root)
		left.setNumKeys(root.numKeys())

		// reset the root node.
		part := root[:keyOffset(maxKeys)]
		ZeroOut(part, 0, len(part))
		root.setNumKeys(0)

		// set the pointers for left and right child in the root node.
		root.set(left.maxKey(), left.pageID())
		root.set(right.maxKey(), right.pageID())
	}
}

// For internal nodes, they contain <key, ptr>.
// where all entries <= key are stored in the corresponding ptr.
func (t *Tree) set(offset, k, v uint64) node {
	n := t.node(offset)
	if n.isLeaf() {
		return n.set(k, v)
	}

	// This is an internal node.
	idx := n.search(k)
	if idx >= maxKeys {
		panic("search returned index >= maxKeys")
	}
	// If no key at idx.
	if n.key(idx) == 0 {
		n.setAt(keyOffset(idx), k)
		n.setNumKeys(n.numKeys() + 1)
	}
	child := t.node(n.uint64(valOffset(idx)))
	if child == nil {
		child = t.newNode(bitLeaf)
		n = t.node(offset)
		n.setAt(valOffset(idx), child.pageID())
	}
	child = t.set(child.pageID(), k, v)
	// Re-read n as the underlying buffer for tree might have changed during set.
	n = t.node(offset)
	if child.isFull() {
		nn := t.split(child.pageID())
		// Re-read n and child as the underlying buffer for tree might have changed during split.
		n = t.node(offset)
		child = t.node(n.uint64(valOffset(idx)))
		// Set child pointers in the node n.
		// Note that key for right node (nn) already exist in node n, but the
		// pointer is updated.
		n.set(child.maxKey(), child.pageID())
		n.set(nn.maxKey(), nn.pageID())
	}
	return n
}

// Get looks for key and returns the corresponding value.
// If key is not found, 0 is returned.
func (t *Tree) Get(k uint64) uint64 {
	if k == math.MaxUint64 || k == 0 {
		panic("Does not support getting MaxUint64/Zero")
	}
	root := t.node(1)
	return t.get(root, k)
}

func (t *Tree) get(n node, k uint64) uint64 {
	if n.isLeaf() {
		return n.get(k)
	}
	// This is internal node
	idx := n.search(k)
	if idx == n.numKeys() || n.key(idx) == 0 {
		return 0
	}
	child := t.node(n.uint64(valOffset(idx)))
	assert(child != nil)
	return t.get(child, k)
}

// DeleteBelow deletes all keys with value under ts.
func (t *Tree) DeleteBelow(ts uint64) {
	fn := func(n node) {
		// We want to compact only the leaf nodes. The internal nodes aren't compacted.
		if !n.isLeaf() {
			return
		}
		n.compact(ts)
	}
	t.Iterate(fn)
}

func (t *Tree) iterate(n node, fn func(node)) {
	fn(n)
	if n.isLeaf() {
		return
	}
	// Explore children.
	for i := 0; i < maxKeys; i++ {
		if n.key(i) == 0 {
			return
		}
		childID := n.uint64(valOffset(i))
		child := t.node(childID)
		t.iterate(child, fn)
	}
}

// Iterate iterates over the tree and executes the fn on each node.
func (t *Tree) Iterate(fn func(node)) {
	root := t.node(1)
	t.iterate(root, fn)
}

func (t *Tree) print(n node, parentID uint64) {
	n.print(parentID)
	if n.isLeaf() {
		return
	}
	pid := n.pageID()
	for i := 0; i < maxKeys; i++ {
		if n.key(i) == 0 {
			return
		}
		childID := n.uint64(valOffset(i))
		child := t.node(childID)
		t.print(child, pid)
	}
}

// Print iterates over the tree and prints all valid KVs.
func (t *Tree) Print() {
	root := t.node(1)
	t.print(root, 0)
}

// Splits the node into two. It moves right half of the keys from the original node to a newly
// created right node. It returns the right node.
func (t *Tree) split(offset uint64) node {
	n := t.node(offset)
	if !n.isFull() {
		panic("This should be called only when n is full")
	}

	// Create a new node nn, copy over half the keys from n, and set the parent to n's parent.
	nn := t.newNode(n.bits())
	// Re-read n as the underlying buffer for tree might have changed during newNode.
	n = t.node(offset)
	rightHalf := n[keyOffset(maxKeys/2):keyOffset(maxKeys)]
	copy(nn, rightHalf)
	nn.setNumKeys(maxKeys - maxKeys/2)

	// Remove entries from node n.
	ZeroOut(rightHalf, 0, len(rightHalf))
	n.setNumKeys(maxKeys / 2)
	return nn
}

// Each node in the node is of size pageSize. Two kinds of nodes. Leaf nodes and internal nodes.
// Leaf nodes only contain the data. Internal nodes would contain the key and the offset to the
// child node.
// Internal node would have first entry as
// <0 offset to child>, <1000 offset>, <5000 offset>, and so on...
// Leaf nodes would just have: <key, value>, <key, value>, and so on...
// Last 16 bytes of the node are off limits.
// | pageID (8 bytes) | metaBits (1 byte) | 3 free bytes | numKeys (4 bytes) |
type node []byte

func (n node) uint64(start int) uint64 { return *(*uint64)(unsafe.Pointer(&n[start])) }
func (n node) uint32(start int) uint32 { return *(*uint32)(unsafe.Pointer(&n[start])) }

func keyOffset(i int) int        { return 16 * i }
func valOffset(i int) int        { return 16*i + 8 }
func (n node) numKeys() int      { return int(n.uint32(valOffset(maxKeys) + 4)) }
func (n node) pageID() uint64    { return n.uint64(keyOffset(maxKeys)) }
func (n node) key(i int) uint64  { return n.uint64(keyOffset(i)) }
func (n node) val(i int) uint64  { return n.uint64(valOffset(i)) }
func (n node) data(i int) []byte { return n[keyOffset(i):keyOffset(i+1)] }

func (n node) setAt(start int, k uint64) {
	v := (*uint64)(unsafe.Pointer(&n[start]))
	*v = k
}

func (n node) setNumKeys(num int) {
	start := valOffset(maxKeys) + 4
	v := (*uint32)(unsafe.Pointer(&n[start]))
	*v = uint32(num)
}

func (n node) moveRight(lo int) {
	hi := n.numKeys()
	assert(hi != maxKeys)
	// copy works despite of overlap in src and dst.
	// See https://golang.org/pkg/builtin/#copy
	copy(n[keyOffset(lo+1):keyOffset(hi+1)], n[keyOffset(lo):keyOffset(hi)])
}

const (
	bitUsed = byte(1)
	bitLeaf = byte(2)
)

func (n node) setBit(b byte) {
	vo := valOffset(maxKeys)
	n[vo] |= b
}
func (n node) bits() byte {
	vo := valOffset(maxKeys)
	return n[vo]
}
func (n node) isLeaf() bool {
	return n.bits()&bitLeaf > 0
}

// isFull checks that the node is already full.
func (n node) isFull() bool {
	return n.numKeys() == maxKeys
}

// Search returns the index of a smallest key >= k in a node.
func (n node) search(k uint64) int {
	N := n.numKeys()
	lo, hi := 0, N
	// Reduce the search space using binary seach and then do linear search.
	for hi-lo > 32 {
		mid := (hi + lo) / 2
		km := n.key(mid)
		if k == km {
			return mid
		}
		if k > km {
			// key is greater than the key at mid, so move right.
			lo = mid + 1
		} else {
			// else move left.
			hi = mid
		}
	}
	for i := lo; i <= hi; i++ {
		if ki := n.key(i); ki >= k {
			return i
		}
	}
	return N
}
func (n node) maxKey() uint64 {
	idx := n.numKeys()
	// idx points to the first key which is zero.
	if idx > 0 {
		idx--
	}
	return n.key(idx)
}

// compacts the node i.e., remove all the kvs with value < lo. It returns the remaining number of
// keys.
func (n node) compact(lo uint64) int {
	// compact should be called only on leaf nodes
	assert(n.isLeaf())
	N := n.numKeys()
	mk := n.maxKey()
	// Just zero-out the value of maxKey if value <= lo. Don't remove the key.
	if N > 0 && n.val(N-1) < lo {
		n.setAt(valOffset(N-1), 0)
	}
	var left, right int
	for right = 0; right < N; right++ {
		if n.val(right) < lo && n.key(right) < mk {
			// Skip over this key. Don't copy it.
			continue
		}
		// Valid data. Copy it from right to left. Advance left.
		if left != right {
			copy(n.data(left), n.data(right))
		}
		left++
	}
	// zero out rest of the kv pairs.
	ZeroOut(n, keyOffset(left), keyOffset(right))
	n.setNumKeys(left)
	return left
}

func (n node) get(k uint64) uint64 {
	idx := n.search(k)
	// key is not found
	if idx == n.numKeys() {
		return 0
	}
	if ki := n.key(idx); ki == k {
		return n.val(idx)
	}
	return 0
}

func (n node) set(k, v uint64) node {
	idx := n.search(k)
	ki := n.key(idx)
	if n.numKeys() == maxKeys {
		// This happens during split of non-root node, when we are updating the child pointer of
		// right node. Hence, the key should already exist.
		assert(ki == k)
	}
	if ki > k {
		// Found the first entry which is greater than k. So, we need to fit k
		// just before it. For that, we should move the rest of the data in the
		// node to the right to make space for k.
		n.moveRight(idx)
	}
	// If the k does not exist already, increment the number of keys.
	if ki != k {
		n.setNumKeys(n.numKeys() + 1)
	}
	if ki == 0 || ki >= k {
		n.setAt(keyOffset(idx), k)
		n.setAt(valOffset(idx), v)
		return n
	}
	panic("shouldn't reach here")
}

func (n node) iterate(fn func(node, int)) {
	for i := 0; i < maxKeys; i++ {
		if k := n.key(i); k > 0 {
			fn(n, i)
		} else {
			break
		}
	}
}

func (n node) print(parentID uint64) {
	var keys []string
	n.iterate(func(n node, i int) {
		keys = append(keys, fmt.Sprintf("%d", n.key(i)))
	})
	if len(keys) > 8 {
		copy(keys[4:], keys[len(keys)-4:])
		keys[3] = "..."
		keys = keys[:8]
	}
	fmt.Printf("%d Child of: %d bits: %04b num keys: %d keys: %s\n",
		n.pageID(), parentID, n.bits(), n.numKeys(), strings.Join(keys, " "))
}
