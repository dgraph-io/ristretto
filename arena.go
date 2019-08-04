package ristretto

import (
	"bytes"
	"fmt"
	"math/bits"
	"sync"
	"sync/atomic"
	"unsafe"
)

type Arena struct {
	classes []*class
}

func NewArena() *Arena {
	// TODO: Later support values bigger than 32KB.
	a := new(Arena)
	// TODO: Make the sizes more granular. TCMalloc uses 170 size classes.
	var idx uint64
	for size := int32(8); size <= 32*1024; size *= 2 {
		a.classes = append(a.classes, newClass(size, idx))
		idx++
	}
	return a
}

func (a *Arena) classFor(size uint32) *class {
	if size <= 8 {
		return a.classes[0]
	}
	sub := 3
	if bits.OnesCount32(size) == 1 {
		sub += 1
	}
	return a.classes[bits.Len32(size)-sub]
}

func (a *Arena) Allocate(sz uint32) ([]byte, uint64) {
	c := a.classFor(sz)
	if c == nil {
		return nil, 0
	}
	page := c.allocatePage()
	slot := page.allocateSlot()
	if slot < 0 {
		return nil, 0
	}
	return page.buffer(slot), uint64(slot)<<32 | uint64(page.pid)
}

func (a *Arena) Get(slot uint64) []byte {
	pid := slot & 0xffffffff
	cid := int(pid & 0xff)
	if cid >= len(a.classes) {
		return nil
	}
	c := a.classes[cid]
	p, ok := c.pages.Load(pid)
	if !ok {
		return nil
	}
	page := p.(*page)
	return page.buffer(int32(slot >> 32))
}

func Readable(slot uint64) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Class: %d ", slot&0xff)
	fmt.Fprintf(&buf, "Page: %d ", (slot>>8)&0xffffff)
	fmt.Fprintf(&buf, "Slot: %d ", slot>>32)
	return buf.String()
}

type class struct {
	mask       uint64 // Uses 8-bits in LSB.
	nextId     *uint32
	objectSize int32
	pages      *sync.Map
	latest     *unsafe.Pointer
}

func newClass(size int32, mask uint64) *class {
	return &class{
		mask:       mask,
		nextId:     new(uint32),
		objectSize: size,
		pages:      new(sync.Map),
		latest:     new(unsafe.Pointer),
	}
}

func (c *class) availablePage() *page {
	if cached := (*page)(atomic.LoadPointer(c.latest)); cached != nil {
		if cached.availableSlots() > 0 {
			return cached
		}
	}
	var out *page
	c.pages.Range(func(key, value interface{}) bool {
		p := value.(*page)
		if p.availableSlots() > 0 {
			out = p
			return false
		}
		return true
	})
	if out != nil {
		atomic.StorePointer(c.latest, unsafe.Pointer(out))
	}
	return out
}

const maxNextId = uint32(1<<24 - 1)

func (c *class) allocatePage() *page {
	if p := c.availablePage(); p != nil {
		return p
	}
	// Need to figure out locking situation here.
	nextId := atomic.AddUint32(c.nextId, 1)
	if nextId >= maxNextId {
		return nil // We can only use 24-bits.
	}
	pid := c.mask | uint64(nextId<<8)
	out := newPage(pid, c.objectSize)
	c.pages.Store(out.pid, out)
	return out
}

type page struct {
	pid       uint64
	size      int32
	data      []byte
	bits      []*uint32
	allocated *int32 // These two can help figure out when to release this page.
	released  *int32 // Not being used right now.
	leftSlots *int32
}

const defaultPageSize int32 = 4096

func newPage(pid uint64, size int32) *page {
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
		leftSlots: &slots,
	}
}

func (p *page) availableSlots() int32 {
	return atomic.LoadInt32(p.leftSlots)
}

// Does not need any more locks beyond atomics.
func (p *page) allocateSlot() int32 {
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
		atomic.AddInt32(p.leftSlots, -1)
		return int32(32*idx + slot)
	}
	return -1
}

// Does not need any more locks beyond atomics.
func (p *page) releaseSlot(slot int32) {
	if slot < 0 {
		return
	}
	mask := uint32(1 << uint(slot%32))
	idx := slot / 32
	for {
		old := atomic.LoadUint32(p.bits[idx])
		if atomic.CompareAndSwapUint32(p.bits[idx], old, old & ^mask) {
			atomic.AddInt32(p.leftSlots, 1)
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
