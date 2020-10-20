package z

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"strings"
)

// Tree represents the structure for mmaped B+ tree
type Tree struct {
	pageSize int
	maxKeys  int
	mf       *MmapFile
	nextPage uint64
}

// NewTree returns a memory mapped B+ tree
func NewTree(mf *MmapFile, pageSize int) *Tree {
	t := &Tree{
		pageSize: pageSize,
		maxKeys:  (pageSize / 16) - 1,
		mf:       mf,
		nextPage: 1,
	}
	t.newNode(0)
	return t
}

func (t *Tree) newNode(bit uint64) node {
	offset := int(t.nextPage) * t.pageSize
	t.nextPage++
	n := node(t.mf.Data[offset : offset+t.pageSize])
	ZeroOut(n, 0, len(n))
	n.setBit(bitUsed|bit, t.maxKeys)
	n.setAt(keyOffset(t.maxKeys), t.nextPage-1)
	return n
}
func (t *Tree) node(pid uint64) node {
	if pid == 0 {
		return nil
	}
	start := t.pageSize * int(pid)
	return node(t.mf.Data[start : start+t.pageSize])
}

// Set sets the key-value pair in the tree
func (t *Tree) Set(k, v uint64) {
	if k == math.MaxUint64 {
		panic("Does not support setting MaxUint64")
	}
	root := t.node(1)
	t.set(root, k, v)
	if root.isFull(t.maxKeys) {
		right := t.split(root)
		left := t.newNode(root.bits(t.maxKeys))
		copy(left[:keyOffset(t.maxKeys)], root)

		part := root[:keyOffset(t.maxKeys)]
		ZeroOut(part, 0, len(part))

		root.set(left.maxKey(t.maxKeys), left.pageID(t.maxKeys), t.maxKeys)
		root.set(right.maxKey(t.maxKeys), right.pageID(t.maxKeys), t.maxKeys)
	}
}

// Get looks for key and returns the corresponding value.
// If key is not found, 0 is returned.
func (t *Tree) Get(k uint64) uint64 {
	if k == math.MaxUint64 {
		panic("Does not support getting MaxUint64")
	}
	root := t.node(1)
	return t.get(root, k)
}

func (t *Tree) get(n node, k uint64) uint64 {
	if n.isLeaf(t.maxKeys) {
		return n.get(k, t.maxKeys)
	}
	// This is internal node
	idx := n.search(k, t.maxKeys)
	if idx == t.maxKeys || n.key(idx) == 0 {
		return 0
	}
	child := t.node(n.uint64(valOffset(idx)))
	assert(child != nil)
	return t.get(child, k)
}

// DeleteBelow sets value 0, for all the keys which have value below ts
func (t *Tree) DeleteBelow(ts uint64) {
	fn := func(n node) {
		// Set the values to 0.
		n.iterate(func(n node, idx int) {
			if n.val(idx) < ts {
				n.setAt(valOffset(idx), 0)
			}
		}, t.maxKeys)
		// remove the deleted entries from the node.
		n.compact(t.maxKeys)
	}
	t.Iterate(fn)
}

func (t *Tree) iterate(n node, fn func(node)) {
	if n.isLeaf(t.maxKeys) {
		fn(n)
		return
	}
	for i := 0; i < t.maxKeys; i++ {
		if n.key(i) == 0 {
			return
		}
		childID := n.uint64(valOffset(i))
		child := t.node(childID)
		t.iterate(child, fn)
	}
}

// Iterate iterates over the tree and executes the fn on each leaf node. It is
// the responsibility of caller to iterate over all the kvs in a leaf node.
func (t *Tree) Iterate(fn func(node)) {
	root := t.node(1)
	t.iterate(root, fn)
	fmt.Println("Done iterating")
}

func (t *Tree) print(n node, parentID uint64) {
	n.print(parentID, t.maxKeys)
	if n.isLeaf(t.maxKeys) {
		return
	}
	pid := n.pageID(t.maxKeys)
	for i := 0; i < t.maxKeys; i++ {
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

// For internal nodes, they contain <key, ptr>.
// where all entries <= key are stored in the corresponding ptr.
func (t *Tree) set(n node, k, v uint64) {
	if n.isLeaf(t.maxKeys) {
		n.set(k, v, t.maxKeys)
		return
	}

	// This is an internal node.
	idx := n.search(k, t.maxKeys)
	if idx >= t.maxKeys {
		panic("this shouldn't happen")
	}
	// If no key at idx.
	if n.key(idx) == 0 {
		n.setAt(keyOffset(idx), k)
	}
	child := t.node(n.uint64(valOffset(idx)))
	if child == nil {
		child = t.newNode(bitLeaf)
		n.setAt(valOffset(idx), child.pageID(t.maxKeys))
	}
	t.set(child, k, v)

	if child.isFull(t.maxKeys) {
		// Split child.
		nn := t.split(child)

		// Set children.
		n.set(child.maxKey(t.maxKeys), child.pageID(t.maxKeys), t.maxKeys)
		n.set(nn.maxKey(t.maxKeys), nn.pageID(t.maxKeys), t.maxKeys)
	}
}

func (t *Tree) split(n node) node {
	if !n.isFull(t.maxKeys) {
		panic("This should be called only when n is full")
	}
	rightHalf := n[keyOffset(t.maxKeys/2):keyOffset(t.maxKeys)]

	// Create a new node nn, copy over half the keys from n, and set the parent to n's parent.
	nn := t.newNode(n.bits(t.maxKeys))
	copy(nn, rightHalf)

	// Remove entries from node n.
	ZeroOut(rightHalf, 0, len(rightHalf))
	return nn
}

// Each node in the node is of size pageSize. Two kinds of nodes. Leaf nodes and internal nodes.
// Leaf nodes only contain the data. Internal nodes would contain the key and the offset to the child node.
// Internal node would have first entry as
// <0 offset to child>, <1000 offset>, <5000 offset>, and so on...
// Leaf nodes would just have: <key, value>, <key, value>, and so on...
// Last 16 bytes of the node are off limits.
type node []byte

func (n node) uint64(start int) uint64 {
	return binary.BigEndian.Uint64(n[start : start+8])
}

func keyOffset(i int) int                { return 16 * i }   // Last 16 bytes are kept off limits.
func valOffset(i int) int                { return 16*i + 8 } // Last 16 bytes are kept off limits.
func (n node) pageID(maxKeys int) uint64 { return n.uint64(keyOffset(maxKeys)) }
func (n node) key(i int) uint64          { return n.uint64(keyOffset(i)) }
func (n node) val(i int) uint64          { return n.uint64(valOffset(i)) }
func (n node) id(maxKeys int) uint64     { return n.key(maxKeys) }
func (n node) data(i int) []byte         { return n[keyOffset(i):keyOffset(i+1)] }

func (n node) setAt(start int, k uint64) {
	binary.BigEndian.PutUint64(n[start:start+8], k)
}
func (n node) moveRight(lo int, maxKeys int) {
	hi := n.search(math.MaxUint64, maxKeys)
	if hi == maxKeys {
		panic("endIdx == maxKeys")
	}
	for i := hi; i > lo; i-- {
		copy(n.data(i), n.data(i-1))
	}
}

const (
	bitUsed = uint64(1)
	bitLeaf = uint64(2)
)

func (n node) setBit(b uint64, maxKeys int) {
	vo := valOffset(maxKeys)
	v := n.uint64(vo)
	n.setAt(vo, v|b)
}
func (n node) bits(maxKeys int) uint64 {
	vo := valOffset(maxKeys)
	return n.uint64(vo)
}

func (n node) isLeaf(maxKeys int) bool {
	return n.uint64(valOffset(maxKeys))&bitLeaf > 0
}

// isFull checks that the node is already full.
func (n node) isFull(maxKeys int) bool {
	return n.key(maxKeys-1) > 0
}
func (n node) search(k uint64, maxKeys int) int {
	return sort.Search(maxKeys, func(i int) bool {
		// ks := n.keyAt(i + 1)
		ks := n.key(i)
		if ks == 0 {
			return true
		}
		return ks >= k
	})
}
func (n node) maxKey(maxKeys int) uint64 {
	idx := n.search(math.MaxUint64, maxKeys)
	// idx points to the first key which is zero.
	if idx > 0 {
		idx--
	}
	return n.key(idx)
}

// compacts the node i.e., remove all the kvs with value 0. It returns the remaining number of keys.
func (n node) compact(maxKeys int) int {
	// compact should be called only on leaf nodes
	assert(n.isLeaf(maxKeys))
	var left, right int
	for right < maxKeys {
		if n.key(right) == 0 {
			break
		}
		// If the value is non-zero, move the kv to right.
		if n.val(right) != 0 {
			if left != right {
				copy(n.data(left), n.data(right))
			}
			left++
		}
		right++
	}
	// zero out rest of the kv pairs
	ZeroOut(n, keyOffset(left), keyOffset(right))
	return left
}

func (n node) get(k uint64, maxKeys int) uint64 {
	idx := n.search(k, maxKeys)
	// key is not found
	if idx == maxKeys {
		return 0
	}
	if ki := n.key(idx); ki == k {
		return n.val(idx)
	}
	return 0
}
func (n node) set(k, v uint64, maxKeys int) {
	idx := n.search(k, maxKeys)
	if idx == maxKeys {
		panic("node should not be full")
	}
	ki := n.key(idx)
	if ki == 0 || ki == k {
		n.setAt(keyOffset(idx), k)
		n.setAt(valOffset(idx), v)
		return
	}
	if ki > k {
		// Found the first entry which is greater than k. So, we need to fit k
		// just before it. For that, we should move the rest of the data in the
		// node to the right to make space for k.
		n.moveRight(idx, maxKeys)
		n.setAt(keyOffset(idx), k)
		n.setAt(valOffset(idx), v)
		return
	}
	panic("shouldn't reach here")
}

func (n node) iterate(fn func(node, int), maxKeys int) {
	for i := 0; i < maxKeys; i++ {
		if k := n.key(i); k > 0 {
			fn(n, i)
		} else {
			break
		}
	}
}

func (n node) print(parentID uint64, maxKeys int) {
	var keys []string
	n.iterate(func(n node, i int) {
		keys = append(keys, fmt.Sprintf("%d", n.key(i)))
	}, maxKeys)
	if len(keys) > 8 {
		copy(keys[4:], keys[len(keys)-4:])
		keys[3] = "..."
		keys = keys[:8]
	}
	numKeys := n.search(math.MaxUint64, maxKeys)
	fmt.Printf("%d Child of: %d bits: %04b num keys: %d, keys: %s\n",
		n.pageID(maxKeys), parentID, n.bits(maxKeys), numKeys, strings.Join(keys, " "))
}
