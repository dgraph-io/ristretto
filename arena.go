package ristretto

import (
	"bytes"
	"fmt"
	"math/bits"
)

type Arena struct {
	classes    []*class
	bigObjects map[uint64]*page
}

type class struct {
	pages map[uint32]*page
}

type page struct {
	size int
	data []byte
	bits []uint32
}

func (p *page) getSlot() int {
	for idx, bit := range p.bits {
		tmp := ^bit
		tmp &= -tmp
		tmp -= 1
		slot := bits.OnesCount32(tmp)
		if slot == 32 {
			continue
		}

		p.bits[idx] |= 1 << uint(slot) // Mark it as used.
		return 32*idx + slot
	}
	return -1
}

func (p *page) releaseSlot(slot int) {
	if slot < 0 {
		return
	}
	mask := uint32(1 << uint(slot%32))
	p.bits[slot/32] &= ^mask
}

func (p *page) buffer(slot int) []byte {
	start := slot * size
	return p.data[start : start+size]
}

func (p *page) String() string {
	var buf bytes.Buffer
	for idx, bit := range p.bits {
		fmt.Fprintf(&buf, "idx=%d slots=%0b ", idx, bit)
	}
	return buf.String()
}

const maxSize int = 32 << 10

func (a *Arena) Allocate(data []byte) uint64 {
	if len(data) > maxSize {
		// allocate on big objects.
	}
	// pow := math.Ceil((math.Log2(float64(len(data)))))
	// c := a.classes[int(pow)]

	return 0
}
