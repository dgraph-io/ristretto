package z

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
)

var pageSize = os.Getpagesize()
var maxKeys = (pageSize / 16) - 1

type Tree struct {
	mf       *MmapFile
	nextPage uint64
}

func NewTree(mf *MmapFile, numRanges int) *Tree {
	t := &Tree{
		mf:       mf,
		nextPage: 1,
	}
	t.newNode(0)
	t.Set(math.MaxUint64-1, 0)
	// if numRanges > 0 {
	// 	jump := uint64(math.MaxUint64) / uint64(numRanges)
	// 	fmt.Printf("jump = %d\n", jump)
	// 	start := jump
	// 	for i := 0; i < numRanges; i++ {
	// 		t.Set(start, 0)
	// 		start += jump
	// 	}
	// }
	return t
}

func (t *Tree) newNode(bit uint64) node {
	offset := int(t.nextPage) * pageSize
	t.nextPage++
	n := node(t.mf.Data[offset : offset+pageSize])
	ZeroOut(n, 0, len(n))
	n.setBit(bitUsed | bit)
	n.setAt(keyOffset(maxKeys), t.nextPage-1)
	// fmt.Printf("Created page of id: %d at offset: %d\n", n.pageId(), offset)
	return n
}
func (t *Tree) node(pid uint64) node {
	if pid == 0 {
		return nil
	}
	start := pageSize * int(pid)
	return node(t.mf.Data[start : start+pageSize])
}

func (t *Tree) Set(k, v uint64) {
	if k == math.MaxUint64 {
		panic("Does not support setting MaxUint64")
	}
	root := t.node(1)
	t.set(root, k, v)
	if root.isFull() {
		right := t.split(root)
		left := t.newNode(root.bits())
		copy(left[:keyOffset(maxKeys)], root)

		part := root[:keyOffset(maxKeys)]
		ZeroOut(part, 0, len(part))

		root.set(left.maxKey(), left.pageId())
		root.set(right.maxKey(), right.pageId())
	}
}

// Get returns the corresponding value, else math.MaxUint64 if key is not found
func (t *Tree) Get(k uint64) uint64 {
	if k == math.MaxUint64 {
		panic("Does not support getting MaxUint64")
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
	if idx == maxKeys || n.key(idx) == 0 {
		return 0
	}
	child := t.node(n.uint64(valOffset(idx)))
	assert(child != nil)
	return t.get(child, k)
}

// DeleteBelow sets value 0, for all the keys which have value below ts
func (t *Tree) DeleteBelow(ts uint64) {
	fn := func(n node, idx int) {
		if n.val(idx) < ts {
			n.setAt(valOffset(idx), 0)
		}
	}
	t.Iterate(fn)
}

func (t *Tree) iterate(n node, fn func(node, int), parentId uint64) {
	if n.isLeaf() {
		n.iterate(fn, parentId)
		return
	}
	pid := n.pageId()
	for i := 0; i < maxKeys; i++ {
		if n.key(i) == 0 {
			return
		}
		childId := n.uint64(valOffset(i))
		child := t.node(childId)
		t.iterate(child, fn, pid)
	}
}

// Iterate iterates ove the tree
func (t *Tree) Iterate(fn func(node, int)) {
	root := t.node(1)
	t.iterate(root, fn, 0)
	fmt.Println("Done iterating")
}

func (t *Tree) print(n node, parentId uint64) {
	// fmt.Printf("print called with n: %v, parentId: %d\n", &n[0], parentId)
	n.print(parentId)
	if n.isLeaf() {
		return
	}
	pid := n.pageId()
	for i := 0; i < maxKeys; i++ {
		if n.key(i) == 0 {
			return
		}
		childId := n.uint64(valOffset(i))
		child := t.node(childId)
		t.print(child, pid)
	}
}

func (t *Tree) Print() {
	root := t.node(1)
	t.print(root, 0)
	fmt.Println("Done")
}

// For internal nodes, they contain <key, ptr>.
// where all entries <= key are stored in the corresponding ptr.
func (t *Tree) set(n node, k, v uint64) {
	if n.isLeaf() {
		n.set(k, v)
		return
	}

	// This is an internal node.
	idx := n.search(k)
	if idx >= maxKeys {
		// We can just upgrade the key at maxKeys-1, so all key-values under k
		// would get stored in the same child.
		panic("this shouldn't happen")
		n.setAt(keyOffset(maxKeys-1), k)
		idx = maxKeys - 1
	}
	// fmt.Printf("found key: %d for inserting %d for pageid: %d\n", n.key(idx), k, n.pageId())
	// If no key at idx.
	if n.key(idx) == 0 {
		n.setAt(keyOffset(idx), k)
	}
	child := t.node(n.uint64(valOffset(idx)))
	if child == nil {
		child = t.newNode(bitLeaf)
		n.setAt(valOffset(idx), child.pageId())
		fmt.Printf("child is nil. Created child with id: %d\n", child.pageId())
	}
	fmt.Printf("setting %d at child: %d\n", k, child.pageId())
	t.set(child, k, v)

	if child.isFull() {
		// Split child.
		fmt.Println("Before")
		n.print(0)
		fmt.Printf("Child %d is full. Splitting\n", child.pageId())
		nn := t.split(child)

		// Set children.
		n.set(child.maxKey(), child.pageId())
		n.set(nn.maxKey(), nn.pageId())
		n.print(0)
		child.print(n.pageId())
		nn.print(n.pageId())
	}
}

func (t *Tree) split(n node) node {
	if !n.isFull() {
		panic("This should be called only when n is full")
	}
	rightHalf := n[keyOffset(maxKeys/2):keyOffset(maxKeys)]

	// Create a new node nn, copy over half the keys from n, and set the parent to n's parent.
	nn := t.newNode(n.bits())
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

func keyOffset(i int) int        { return 16 * i }   // Last 16 bytes are kept off limits.
func valOffset(i int) int        { return 16*i + 8 } // Last 16 bytes are kept off limits.
func (n node) pageId() uint64    { return n.uint64(keyOffset(maxKeys)) }
func (n node) key(i int) uint64  { return n.uint64(keyOffset(i)) }
func (n node) val(i int) uint64  { return n.uint64(valOffset(i)) }
func (n node) id() uint64        { return n.key(maxKeys) }
func (n node) data(i int) []byte { return n[keyOffset(i):keyOffset(i+1)] }

func (n node) next(k uint64) node {
	return nil
}
func (n node) setAt(start int, k uint64) {
	// fmt.Printf("setAt: %d %d with pageId: %d\n", start, k, n.pageId())
	binary.BigEndian.PutUint64(n[start:start+8], k)
}
func (n node) moveRight(lo int) {
	hi := n.search(math.MaxUint64)
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

func (n node) setBit(b uint64) {
	vo := valOffset(maxKeys)
	v := n.uint64(vo)
	n.setAt(vo, v|b)
}
func (n node) bits() uint64 {
	vo := valOffset(maxKeys)
	return n.uint64(vo)
}

func (n node) isLeaf() bool {
	return n.uint64(valOffset(maxKeys))&bitLeaf > 0
}

// isFull checks that the node is already full.
func (n node) isFull() bool {
	return n.key(maxKeys-1) > 0
}
func (n node) search(k uint64) int {
	return sort.Search(maxKeys, func(i int) bool {
		// ks := n.keyAt(i + 1)
		ks := n.key(i)
		if ks == 0 {
			return true
		}
		return ks >= k
	})
}
func (n node) maxKey() uint64 {
	idx := n.search(math.MaxUint64)
	// idx points to the first key which is zero.
	if idx > 0 {
		idx--
	}
	return n.key(idx)
}
func (n node) get(k uint64) uint64 {
	idx := n.search(k)
	// key is not found
	if idx == maxKeys {
		return 0
	}
	if ki := n.key(idx); ki == k {
		return n.val(idx)
	}
	return 0
}
func (n node) set(k, v uint64) {
	idx := n.search(k)
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
		n.moveRight(idx)
		n.setAt(keyOffset(idx), k)
		n.setAt(valOffset(idx), v)
		return
	}
	panic("shouldn't reach here")
}

func (n node) iterate(fn func(node, int), parentId uint64) {
	for i := 0; i < maxKeys; i++ {
		if k := n.key(i); k > 0 {
			fn(n, i)
		} else {
			break
		}
	}
}

func (n node) print(parentId uint64) {
	var keys []string
	for i := 0; i < maxKeys; i++ {
		if k := n.key(i); k > 0 {
			keys = append(keys, fmt.Sprintf("%d", k))
		} else {
			break
		}
	}
	if len(keys) > 8 {
		copy(keys[4:], keys[len(keys)-4:])
		keys[3] = "..."
		keys = keys[:8]
	}
	idx := n.search(math.MaxUint64)
	numKeys := idx
	if idx > 0 {
		idx--
	}
	fmt.Printf("%d Child of: %d bits: %04b num keys: %d, keys: %s\n",
		n.pageId(), parentId, n.bits(), numKeys, strings.Join(keys, " "))
}
