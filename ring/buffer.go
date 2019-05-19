package ring

import "sync/atomic"

type Buffer struct {
	Config *Config

	data []uint64
	head int32
	busy int32

	/* TODO: padding for cpu cache?
	_pd0 [5]uint64
	_pd1 [8]uint64
	_pd2 [8]uint64
	_pd3 [8]uint64
	*/
}

func NewBuffer(config *Config) *Buffer {
	return &Buffer{
		Config: config,
		data:   make([]uint64, config.Size),
		head:   -1,
	}
}

func (b *Buffer) Add(element uint64) bool {
	// atomically increment the head and check if the buffer is full
	if head := atomic.AddInt32(&b.head, 1); head >= b.Config.Size {
		// the buffer is full so we need to drain and notify others that we're
		// currently draining - this is done with the busy flag
		if atomic.CompareAndSwapInt32(&b.busy, 0, 1) {
			// the consumer's wrap function must encapsulate all calls to push()
			// (so the consumer can guarantee one goroutine is receiving values)
			b.Config.Consumer.Wrap(func() {
				for i := range b.data {
					// push element to consumer and clear out the data slot
					//
					// NOTE: this is racy because other goroutines may be still
					//       placing elements in b.data, but it's negligible
					//       (making this a *lossy* ring buffer)
					b.Config.Consumer.Push(b.data[i])
					// TODO: do we even need to clear?
					b.data[i] = 0
				}
			})

			// at this point b.data is cleared, so we can place the element at
			// the front of the buffer
			b.data[0] = element

			// remember: we have one element at the front, so head is 0 and not
			// -1 as with an empty buffer
			atomic.StoreInt32(&b.head, 0)
			// notify others that we're done draining
			atomic.StoreInt32(&b.busy, 0)
			return true
		}

		// the buffer is full but someone else is draining, return as a failure
		// so the caller can try something else (like the next buffer)
		return false
	} else {
		b.data[head] = element
		return true
	}
}
