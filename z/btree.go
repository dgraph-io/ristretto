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
	"reflect"
	"strings"
	"unsafe"
)

var (
	pageSize    = os.Getpagesize()
	maxKeys     = (pageSize / 16) - 1
	oneThird    = int(float64(maxKeys) / 3)
	absoluteMax = uint64(math.MaxUint64 - 1)
)

// Tree represents the structure for custom mmaped B+ tree.
// It supports keys in range [1, math.MaxUint64-1] and values [1, math.Uint64].
type Tree struct {
	mf       *MmapFile
	nextPage uint64
	freePage uint64
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
	t.Set(absoluteMax, 0)
	return t
}

type TreeStats struct {
	NextPage    int
	NumPages    int
	NumLeafKeys int
	Bytes       int
	Occupancy   float64
	FreePages   int
}

// Stats returns stats about the tree.
func (t *Tree) Stats() TreeStats {
	var totalKeys, maxPossible, numPages, numKeys int
	fn := func(n node) {
		numPages++
		nk := n.numKeys()
		totalKeys += nk
		maxPossible += maxKeys
		if n.isLeaf() {
			numKeys += nk
		}
	}
	t.Iterate(fn)
	occ := float64(totalKeys) / float64(maxPossible)

	freePage, numFree := t.freePage, 0
	for freePage > 0 {
		numFree++
		n := t.node(freePage)
		freePage = n.uint64(0)
	}
	assert(int(t.nextPage-1) == numFree+numPages)

	return TreeStats{
		NextPage:    int(t.nextPage),
		NumPages:    numPages,
		NumLeafKeys: numKeys,
		Bytes:       int(t.nextPage-1) * pageSize,
		Occupancy:   occ * 100,
		FreePages:   numFree,
	}
}

// BytesToU32Slice converts the given byte slice to uint32 slice
func BytesToUint64Slice(b []byte) []uint64 {
	if len(b) == 0 {
		return nil
	}
	var u64s []uint64
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&u64s))
	hdr.Len = len(b) / 8
	hdr.Cap = hdr.Len
	hdr.Data = uintptr(unsafe.Pointer(&b[0]))
	return u64s
}

func (t *Tree) newNode(bit uint64) node {
	var pageId uint64
	if t.freePage > 0 {
		pageId = t.freePage
	} else {
		pageId = t.nextPage
		t.nextPage++
		offset := int(pageId) * pageSize
		sz := len(t.mf.Data)
		// Double the size of file if current buffer is insufficient.
		if offset+pageSize > sz {
			check(t.mf.Truncate(int64(2 * sz)))
		}
	}
	n := t.node(pageId)
	if t.freePage > 0 {
		t.freePage = n.uint64(0)
	}
	zeroOut(n)
	n.setBit(bit)
	n.setAt(keyOffset(maxKeys), pageId)
	return n
}

func getNode(data []byte) node {
	return node(BytesToUint64Slice(data))
}

func zeroOut(data []uint64) {
	for i := 0; i < len(data); i++ {
		data[i] = 0
	}
}

func (t *Tree) node(pid uint64) node {
	// page does not exist
	if pid == 0 {
		return nil
	}
	start := pageSize * int(pid)
	return getNode(t.mf.Data[start : start+pageSize])
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
		zeroOut(root[:keyOffset(maxKeys)])
		root.setNumKeys(0)

		// set the pointers for left and right child in the root node.
		root.set(left.maxKey(), left.pageID())
		root.set(right.maxKey(), right.pageID())
	}
}

// For internal nodes, they contain <key, ptr>.
// where all entries <= key are stored in the corresponding ptr.
func (t *Tree) set(pid, k, v uint64) node {
	n := t.node(pid)
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
	child := t.node(n.val(idx))
	if child == nil {
		child = t.newNode(bitLeaf)
		n = t.node(pid)
		n.setAt(valOffset(idx), child.pageID())
	}
	child = t.set(child.pageID(), k, v)
	// Re-read n as the underlying buffer for tree might have changed during set.
	n = t.node(pid)
	if child.isFull() {
		// Just consider the left sibling for simplicity.
		// if t.shareWithSibling(n, idx) {
		// 	return n
		// }

		nn := t.split(child.pageID())
		// Re-read n and child as the underlying buffer for tree might have changed during split.
		n = t.node(pid)
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
	// fn := func(n node) {
	// 	// We want to compact only the leaf nodes. The internal nodes aren't compacted.
	// 	if !n.isLeaf() {
	// 		return
	// 	}
	// 	n.compact(ts)
	// }
	// t.Iterate(fn)
	root := t.node(1)
	t.compact(root, ts)
	assert(root.numKeys() >= 1)
}

func (t *Tree) compact(n node, ts uint64) int {
	if n.isLeaf() {
		return n.compact(ts)
	}
	// Not leaf.
	N := n.numKeys()
	for i := 0; i < N; i++ {
		assert(n.key(i) > 0)
		childID := n.uint64(valOffset(i))
		child := t.node(childID)
		if rem := t.compact(child, ts); rem == 0 && i < N-1 {
			// If no valid key is remaining we can drop this child. However, don't do that if this
			// is the max key.
			child.setAt(0, t.freePage)
			t.freePage = childID
			n.setAt(valOffset(i), 0)
		}
	}
	// We use ts=1 here because we want to delete all the keys whose value is 0, which means they no
	// longer have a valid page for that key.
	return n.compact(1)
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
		if childID <= 0 {
			n.print(0)
			fmt.Printf("n: %d key: %d num keys: %d\n", n.pageID(), n.key(i), absoluteMax)
			fmt.Println()
			os.Exit(1)
		}
		// assert(childID > 0)

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
func (t *Tree) split(pid uint64) node {
	n := t.node(pid)
	if !n.isFull() {
		panic("This should be called only when n is full")
	}

	// Create a new node nn, copy over half the keys from n, and set the parent to n's parent.
	nn := t.newNode(n.bits())
	// Re-read n as the underlying buffer for tree might have changed during newNode.
	n = t.node(pid)
	rightHalf := n[keyOffset(maxKeys/2):keyOffset(maxKeys)]
	copy(nn, rightHalf)
	nn.setNumKeys(maxKeys - maxKeys/2)

	// Remove entries from node n.
	zeroOut(rightHalf)
	n.setNumKeys(maxKeys / 2)
	return nn
}

// shareWithSiblingXXX is unused for now. The idea is to move some keys to
// sibling when a node is full. But, I don't see any special benefits in our
// access pattern. It doesn't result in better occupancy ratios.
func (t *Tree) shareWithSiblingXXX(n node, idx int) bool {
	if idx == 0 {
		return false
	}
	left := t.node(n.val(idx - 1))
	ns := left.numKeys()
	if ns >= maxKeys/2 {
		// Sibling is already getting full.
		return false
	}

	right := t.node(n.val(idx))
	// Copy over keys from right child to left child.
	copied := copy(left[keyOffset(ns):], right[:keyOffset(oneThird)])
	copied /= 2 // Considering that key-val constitute one key.
	left.setNumKeys(ns + copied)

	// Update the max key in parent node n for the left sibling.
	n.setAt(keyOffset(idx-1), left.maxKey())

	// Now move keys to left for the right sibling.
	until := copy(right, right[keyOffset(oneThird):keyOffset(maxKeys)])
	right.setNumKeys(until / 2)
	zeroOut(right[until:keyOffset(maxKeys)])
	return true
}

// Each node in the node is of size pageSize. Two kinds of nodes. Leaf nodes and internal nodes.
// Leaf nodes only contain the data. Internal nodes would contain the key and the offset to the
// child node.
// Internal node would have first entry as
// <0 offset to child>, <1000 offset>, <5000 offset>, and so on...
// Leaf nodes would just have: <key, value>, <key, value>, and so on...
// Last 16 bytes of the node are off limits.
// | pageID (8 bytes) | metaBits (1 byte) | 3 free bytes | numKeys (4 bytes) |
type node []uint64

func (n node) uint64(start int) uint64 { return n[start] }

// func (n node) uint32(start int) uint32 { return *(*uint32)(unsafe.Pointer(&n[start])) }

func keyOffset(i int) int          { return 2 * i }
func valOffset(i int) int          { return 2*i + 1 }
func (n node) numKeys() int        { return int(n.uint64(valOffset(maxKeys)) & 0xFFFFFFFF) }
func (n node) pageID() uint64      { return n.uint64(keyOffset(maxKeys)) }
func (n node) key(i int) uint64    { return n.uint64(keyOffset(i)) }
func (n node) val(i int) uint64    { return n.uint64(valOffset(i)) }
func (n node) data(i int) []uint64 { return n[keyOffset(i):keyOffset(i+1)] }

func (n node) setAt(start int, k uint64) {
	n[start] = k
}

func (n node) setNumKeys(num int) {
	idx := valOffset(maxKeys)
	val := n[idx]
	val &= 0xFFFFFFFF00000000
	val |= uint64(num)
	n[idx] = val
}

func (n node) moveRight(lo int) {
	hi := n.numKeys()
	assert(hi != maxKeys)
	// copy works despite of overlap in src and dst.
	// See https://golang.org/pkg/builtin/#copy
	copy(n[keyOffset(lo+1):keyOffset(hi+1)], n[keyOffset(lo):keyOffset(hi)])
}

const (
	bitLeaf = uint64(1 << 63)
)

func (n node) setBit(b uint64) {
	vo := valOffset(maxKeys)
	val := n[vo]
	val &= 0xFFFFFFFF
	val |= b
	n[vo] = val
}
func (n node) bits() uint64 {
	return n.val(maxKeys) & 0xFF00000000000000
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
	N := n.numKeys()
	mk := n.maxKey()
	var left, right int
	for right = 0; right < N; right++ {
		k := n.key(right)
		v := n.val(right)
		if v < lo {
			if k == mk {
				// Keep it.
			} else {
				// Don't copy this key if value is less than lo.
				// But, do keep the max key irrespective of the value.
				continue
			}
		}

		// Valid data. Copy it from right to left. Advance left.
		if left != right {
			copy(n.data(left), n.data(right))
		}
		left++
	}
	// zero out rest of the kv pairs.
	zeroOut(n[keyOffset(left):keyOffset(right)])
	n.setNumKeys(left)

	// If the only key we have is the max key, and its value is less than lo, then we can indicate
	// to the caller by returning a zero that it's OK to drop the node.
	if left == 1 && n.key(0) == mk && n.val(0) < lo {
		return 0
	}
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
	fmt.Printf("%d Child of: %d num keys: %d keys: %s\n",
		n.pageID(), parentID, n.numKeys(), strings.Join(keys, " "))
}
