package ring

import (
	"sync/atomic"
	"time"
)

// Consumer is the user-defined object responsible for receiving and processing
// elements in batches when buffers are drained.
type Consumer interface {
	Push([]uint64)
}

// Buffer is a singular ("stripe") lossy ring buffer.
type Buffer struct {
	cons Consumer
	data []uint64
	size uint64
	head uint64
	busy uint64
}

func newBuffer(size uint64, consumer Consumer) *Buffer {
	return &Buffer{
		cons: consumer,
		data: make([]uint64, size),
		size: size,
	}
}

func (b *Buffer) push(element uint64) bool {
	// increment head to get the next available position in the buffer, and
	// check if we need to drain
	if head := atomic.AddUint64(&b.head, 1); head >= b.size {
		// the buffer is full, so attempt to get exclusive access to the data
		// so we can push the data to the Consumer
		if atomic.CompareAndSwapUint64(&b.busy, 0, 1) {
			// push buffer contents to the consumer
			b.cons.Push(append(b.data[:0:0], b.data...))
			// since the old elements are cleared out, place the new element
			// at the front of the buffer
			b.data[0] = element
			// reset buffer head and unlock the busy mutex
			atomic.StoreUint64(&b.head, 0)
			atomic.StoreUint64(&b.busy, 0)
			return true
		}
		return false
	} else {
		// buffer has space, so just add the element and move along
		b.data[head] = element
		return true
	}
}

// Buffers stores a slice of buffers (stripes) and evenly distributes element
// pushing between them.
//
// This implements the "batching" process described in the BP-Wrapper paper
// (section III part A).
type Buffers struct {
	rows []*Buffer
	mask uint64
	busy uint64
	rand uint64
}

// NewBuffers returns a striped, lossy ring buffer. Rows determines the
// number of stripes. Size determines the size of each stripe. Consumer is
// called by each stripe when they are full and trying to drain.
func NewBuffers(rows, size uint64, consumer Consumer) *Buffers {
	buffers := &Buffers{
		rows: make([]*Buffer, rows),
		mask: rows - 1,
		rand: uint64(time.Now().UnixNano()),
	}

	for i := range buffers.rows {
		buffers.rows[i] = newBuffer(size, consumer)
	}

	return buffers
}

func (b *Buffers) id() uint64 {
	b.rand ^= b.rand << 13
	b.rand ^= b.rand >> 7
	b.rand ^= b.rand << 17
	return b.rand & b.mask
}

// Push adds the element to the buffer and will probably, eventually be sent
// to the Consumer.
func (b *Buffers) Push(element uint64) {
	for {
		if b.rows[b.id()].push(element) {
			return
		}
	}
}
