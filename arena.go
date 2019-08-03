package ristretto

import (
	"bytes"
	"fmt"
	"math/bits"
	"sync"
	"sync/atomic"
)

type Arena struct {
	classes    []*class
	bigObjects map[uint64]*page
}

type class struct {
	sync.RWMutex
	nextId     uint32
	objectSize int32
	pages      map[uint32]*page
}

func (c *class) availablePage() *page {
	c.RLock()
	defer c.RUnlock()
	for _, p := range c.pages {
		if p.availableSlots() > 0 {
			return p
		}
	}
	return nil
}

func (c *class) allocatePage() *page {
	if p := c.availablePage(); c != nil {
		return p
	}
	// Need to figure out locking situation here.
	c.Lock()
	p := newPage(c.nextId, c.objectSize)
	c.pages[p.pid] = p
	c.nextId++
	c.Unlock()
	return p
}

type page struct {
	pid       uint32
	size      int32
	data      []byte
	bits      []*uint32
	allocated *int32
	released  *int32
}

const defaultPageSize int32 = 4096

func newPage(pid uint32, size int32) *page {
	slots := defaultPageSize / size
	if slots < 32 {
		slots = 32
	}
	if bits.OnesCount32(uint32(slots)) != 1 {
		panic("Size is not a power of 2")
	}
	b := make([]*uint32, slots/32)
	for i := range b {
		b[i] = new(uint32)
	}
	return &page{
		pid:       pid,
		size:      size,
		data:      make([]byte, slots*size),
		bits:      b,
		allocated: new(int32),
		released:  new(int32),
	}
}

func (p *page) availableSlots() int32 {
	return int32(len(p.bits)*32) - (atomic.LoadInt32(p.allocated) - atomic.LoadInt32(p.released))
}

// Does not need any more locks beyond atomics.
func (p *page) allocateSlot() int {
	for idx := 0; idx < len(p.bits); {
		bit := atomic.LoadUint32(p.bits[idx])
		tmp := ^bit
		tmp &= -tmp
		tmp -= 1
		slot := bits.OnesCount32(tmp)
		if slot == 32 {
			idx++
			continue
		}

		newBit := bit | 1<<uint(slot)
		if !atomic.CompareAndSwapUint32(p.bits[idx], bit, newBit) {
			continue // Try again.
		}
		atomic.AddInt32(p.allocated, 1)
		return 32*idx + slot
	}
	return -1
}

// Does not need any more locks beyond atomics.
func (p *page) releaseSlot(slot int) {
	if slot < 0 {
		return
	}
	mask := uint32(1 << uint(slot%32))
	idx := slot / 32
	for {
		old := atomic.LoadUint32(p.bits[idx])
		if atomic.CompareAndSwapUint32(p.bits[idx], old, old & ^mask) {
			atomic.AddInt32(p.released, 1)
			return
		}
	}
}

// Do not need a mutex lock here.
func (p *page) buffer(slot int32) []byte {
	start := slot * p.size
	return p.data[start : start+p.size]
}

func (p *page) String() string {
	var buf bytes.Buffer
	for idx, bit := range p.bits {
		fmt.Fprintf(&buf, "idx=%d slots=%0b ", idx, atomic.LoadUint32(bit))
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
