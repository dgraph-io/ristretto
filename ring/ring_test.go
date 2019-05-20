package ring

import (
	"testing"
)

const (
	STRIPE_SIZE  = 128
	STRIPE_COUNT = 16
)

type BaseConsumer struct{}

func (c *BaseConsumer) Push(elements []uint64) {}

type TestConsumer struct {
	push func([]uint64)
}

func (c *TestConsumer) Push(elements []uint64) { c.push(elements) }

func TestBuffers(t *testing.T) {
	buffer := newBuffer(STRIPE_SIZE, &TestConsumer{
		push: func(elements []uint64) {
			// spew.Dump(elements)
		},
	})

	for i := uint64(0); i < STRIPE_SIZE; i++ {
		buffer.push(i)
	}
}

func BenchmarkBuffer(b *testing.B) {
	buffer := newBuffer(STRIPE_SIZE, new(BaseConsumer))

	b.SetBytes(1)
	for n := 0; n < b.N; n++ {
		buffer.push(1)
	}
}

func BenchmarkBufferParallel(b *testing.B) {
	buffer := newBuffer(STRIPE_SIZE, new(BaseConsumer))

	b.SetBytes(1)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buffer.push(1)
		}
	})
}

func BenchmarkBuffers(b *testing.B) {
	buffer := NewBuffers(STRIPE_COUNT, STRIPE_SIZE, new(BaseConsumer))

	b.SetBytes(1)
	for n := 0; n < b.N; n++ {
		buffer.Push(1)
	}
}

func BenchmarkBuffersParallel(b *testing.B) {
	buffer := NewBuffers(STRIPE_COUNT, STRIPE_SIZE, new(BaseConsumer))

	b.SetBytes(1)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buffer.Push(1)
		}
	})
}
