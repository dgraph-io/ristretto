package ring

import (
	"math/rand"
	"testing"
	"time"
)

type TestConsumer struct {
	wrap func(func())
	push func(uint64)
}

func (c *TestConsumer) Wrap(consume func()) { c.wrap(consume) }
func (c *TestConsumer) Push(element uint64) { c.push(element) }

func TestBuffer(t *testing.T) {
	runs := uint64(5)

	// TODO:
	buffer := NewBuffer(int(runs), &TestConsumer{
		wrap: func(consume func()) { consume() },
		push: func(element uint64) {},
	})

	for i := uint64(1); i <= runs+1; i++ {
		buffer.Add(i)
	}
}

type BaseConsumer struct{}

func (c *BaseConsumer) Wrap(consume func()) {}
func (c *BaseConsumer) Push(element uint64) {}

func Zipfian(size int) []uint64 {
	zipf := rand.NewZipf(
		rand.New(rand.NewSource(time.Now().UnixNano())),
		1.1, 2, 100000)

	values := make([]uint64, size)
	for i := range values {
		values[i] = zipf.Uint64()
	}

	return values
}

const (
	BYTES = 1
)

func BenchmarkBuffers(b *testing.B) {
	b.Run("sp-uniform", func(b *testing.B) {
		buffer := NewBuffers(16, 128, new(BaseConsumer))

		b.SetBytes(BYTES)
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			buffer.Add(1)
		}
	})

	b.Run("sp-zipfian", func(b *testing.B) {
		buffer := NewBuffers(16, 128, new(BaseConsumer))
		values := Zipfian(b.N)

		b.SetBytes(BYTES)
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			buffer.Add(values[n])
		}
	})

	b.Run("mp-uniform", func(b *testing.B) {
		buffer := NewBuffers(16, 128, new(BaseConsumer))

		b.SetBytes(BYTES)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buffer.Add(1)
			}
		})
	})

	b.Run("mp-zipfian", func(b *testing.B) {
		buffer := NewBuffers(16, 128, new(BaseConsumer))

		b.SetBytes(BYTES)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			values := Zipfian(b.N)
			for i := 0; pb.Next(); i++ {
				buffer.Add(values[i])
			}
		})
	})
}
