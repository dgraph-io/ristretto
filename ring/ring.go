package ring

import "sync/atomic"

type Consumer interface {
	Wrap(func())
	Push(uint64)
}

type Buffer struct {
	cons Consumer
	data []uint64
	busy int32
	size int32
	head int32
}

func NewBuffer(size int, consumer Consumer) *Buffer {
	return &Buffer{
		cons: consumer,
		data: make([]uint64, size),
		size: int32(size),
		head: -1,
	}
}

func (b *Buffer) Add(element uint64) bool {
	if atomic.CompareAndSwapInt32(&b.busy, 0, 1) {
		b.head++
		if b.head == b.size {
			b.cons.Wrap(func() {
				for i := range b.data {
					b.cons.Push(b.data[i])
				}
			})
			b.head = 0
		}

		b.data[b.head] = element
		atomic.StoreInt32(&b.busy, 0)
		return true
	}
	return false
}

type Buffers struct {
	rows []*Buffer
	mask uint64
}

func NewBuffers(rows, size int, consumer Consumer) *Buffers {
	buffers := &Buffers{
		rows: make([]*Buffer, rows),
		mask: uint64(rows) - 1,
	}

	for i := range buffers.rows {
		buffers.rows[i] = NewBuffer(size, consumer)
	}

	return buffers
}

func (b *Buffers) Add(element uint64) bool {
	for row := element & b.mask; ; row = (row + 1) & b.mask {
		if b.rows[row].Add(element) {
			return true
		}
	}
}
