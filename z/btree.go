package z

import (
	"encoding/binary"
	"os"
)

var pageSize = os.Getpagesize()
var bf = pageSize / 16

// Each node in the node is of size pageSize. Two kinds of nodes. Leaf nodes and internal nodes.
// Leaf nodes only contain the data. Internal nodes would contain the key and the offset to the child node.
// Internal node would have first entry as
// <0 offset to child>, <1000 offset>, <5000 offset>, and so on...
// Leaf nodes would just have: <key, value>, <key, value>, and so on...
type node []byte

func (n node) keyAt(i int) uint64 {
	return binary.BigEndian.Uint64(n[16*i : 16*i+8])
}

func (n node) setAt(i int, k, v uint64) {
	start := 16 * i
	binary.BigEndian.PutUint64(n[start:start+8], k)
	binary.BigEndian.PutUint64(n[start+8:start+16], v)
}

func (n node) moveRight(i int) {
	start := 16 * i
	copy(n[start+16:], n[start:])
	// TODO: maybe we can optimize this to avoid moving the entries which are already zero.
}

// isFull checks that the node is already full.
func (n node) isFull() bool {
	return n.keyAt(bf-1) > 0
}

func (n node) Set(k, v uint64) {
	for i := 0; i < bf; i++ {
		ks := n.keyAt(i)
		if ks == 0 {
			n.setAt(i, k, v)
			return
		}
		if k > ks {
			continue
		} else if k == ks {
			n.setAt(i, k, v)
			return
		}
		if n.isFull() {
			// TODO: Split this node into two.
		}
		// Found the first entry which is greater than k. So, we need to fit k just before it. For
		// that, we should move the rest of the data in the node to the right to make space for k.
		n.moveRight(i)
		n.setAt(i, k, v)
		return
	}
}
