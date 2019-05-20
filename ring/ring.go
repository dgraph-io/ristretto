package ring

import (
	"sync/atomic"
	"time"
)

type Consumer interface {
	Push([]uint64)
}

type Buffer struct {
	cons Consumer

	data []uint64
	size uint64
	head uint64
	busy uint64
}

func NewBuffer(size uint64, consumer Consumer) *Buffer {
	return &Buffer{
		cons: consumer,
		data: make([]uint64, size),
		size: size,
	}
}

func (b *Buffer) Push(element uint64) bool {
	if head := atomic.AddUint64(&b.head, 1); head >= b.size {
		if atomic.CompareAndSwapUint64(&b.busy, 0, 1) {
			b.cons.Push(append(b.data[:0:0], b.data...))
			b.data[0] = element
			atomic.StoreUint64(&b.head, 0)
			atomic.StoreUint64(&b.busy, 0)
			return true
		}
		return false
	} else {
		b.data[head] = element
		return true
	}
}

type Buffers struct {
	rows []*Buffer

	mask uint64
	busy uint64
	rand uint64
}

func NewBuffers(rows, size uint64, consumer Consumer) *Buffers {
	buffers := &Buffers{
		rows: make([]*Buffer, rows),
		mask: rows - 1,
		rand: uint64(time.Now().UnixNano()),
	}

	for i := range buffers.rows {
		buffers.rows[i] = NewBuffer(size, consumer)
	}

	return buffers
}

func (b *Buffers) id() uint64 {
	b.rand ^= b.rand << 13
	b.rand ^= b.rand >> 7
	b.rand ^= b.rand << 17
	return b.rand & b.mask
}

func (b *Buffers) Push(element uint64) {
	for {
		if b.rows[b.id()].Push(element) {
			return
		}
	}
}
