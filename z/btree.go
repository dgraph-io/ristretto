package z

import (
	"encoding/binary"
	"os"
	"sort"
)

var pageSize = os.Getpagesize()
var maxKeys = (pageSize / 16) - 1

type Tree struct {
	mf       *MmapFile
	nextPage uint64
}

func (t *Tree) newPage() uint64 {
	offset := int(t.nextPage) * pageSize
	t.nextPage++
	n := node(t.mf.Data[offset : offset+pageSize])
	n.setBit(bitUsed)
	return t.nextPage - 1
}
func (t *Tree) node(pid uint64) node {
	start := pageSize * int(pid)
	return node(t.mf.Data[start : start+pageSize])
}

// For internal nodes, they contain <key, ptr>.
// where all entries <= key are stored in the corresponding ptr.
func (t *Tree) set(n node, k, v uint64) {
	if n.isLeaf() {
		if n.isFull() {
			// TODO: Split.
		}
		n.set(k, v)
		return
	}
	// This is an internal node.
	idx := n.search(k)
	if idx >= maxKeys {
		// We can just upgrade the key at maxKeys-1, so all key-values under k
		// would get stored in the same child.
		n.setAt(keyOffset(maxKeys-1), k)
		idx = maxKeys - 1
	}
	if n.key(idx) == 0 {
		n.setAt(keyOffset(idx), k)
	}
	child := t.node(n.uint64(valOffset(idx)))
	t.set(child, k, v)
}

func (t *Tree) Set(k, v uint64) {
	n := t.node(0)
	for !n.isLeaf() {

	}
}

func (t *Tree) split(n node) {
	if !n.isFull() {
		return
	}
	rightHalf := n[keyOffset(maxKeys/2):keyOffset(maxKeys)]

	// Create a new node nn, copy over half the keys from n, and set the parent to n's parent.
	ni := t.newPage()
	nn := t.node(ni)
	copy(nn, rightHalf)
	nn.setAt(keyOffset(maxKeys), ni)

	// Remove entries from node n.
	ZeroOut(rightHalf, 0, len(rightHalf))

	// Add nn in the common parent.
	// p := t.node(n.parent())
	// p.set(nn.key(0), uint64(ni))

	// t.split(p) //
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

func keyOffset(i int) int       { return 16 * i }   // Last 16 bytes are kept off limits.
func valOffset(i int) int       { return 16*i + 8 } // Last 16 bytes are kept off limits.
func (n node) key(i int) uint64 { return n.uint64(keyOffset(i)) }
func (n node) id() uint64       { return n.key(maxKeys) }

func (n node) next(k uint64) node {
	return nil
}
func (n node) setAt(start int, k uint64) {
	binary.BigEndian.PutUint64(n[start:start+8], k)
}
func (n node) moveRight(i int) {
	start := keyOffset(i)
	copy(n[start+16:], n[start:])
	// TODO: maybe we can optimize this to avoid moving the entries which are already zero.
}

const (
	bitUsed = iota
	bitInternal
	bitLeaf
)

func (n node) setBit(b int) {
	vo := valOffset(maxKeys)
	v := n.uint64(vo)
	n.setAt(vo, v|(1<<bitInternal))
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
func (n node) set(k, v uint64) {
	idx := n.search(k)
	if idx >= maxKeys {
		return
	}
	ki := n.key(idx)
	if ki == 0 || ki == k {
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
