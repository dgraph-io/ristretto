package ring

import (
	"sync"
	"sync/atomic"
	"testing"
)

type consumer struct {
	wrap func(func())
	push func(uint64)
}

func (c *consumer) Wrap(consume func()) { c.wrap(consume) }
func (c *consumer) Push(element uint64) { c.push(element) }

func TestBuffer(t *testing.T) {
	record := make(map[uint64]struct{})
	buffer := NewBuffer(&Config{
		Consumer: &consumer{
			wrap: func(consume func()) { consume() },
			push: func(element uint64) { record[element] = struct{}{} },
		},

		Size: 16,
	})
	for i := uint64(1); i <= uint64(buffer.Config.Size+1); i++ {
		buffer.Add(i)
	}

	if buffer.data[0] != 17 {
		t.Fatal("wrapping around")
	}

	if buffer.data[1] != 0 {
		t.Fatal("clearing")
	}

	for i := uint64(1); i <= uint64(buffer.Config.Size); i++ {
		if _, exists := record[i]; !exists {
			t.Fatalf("missing element: %d\n", i)
		}
	}
}

func TestBufferParallel(t *testing.T) {
	var wg sync.WaitGroup

	mutex := &sync.Mutex{}
	record := make(map[uint64]struct{})
	buffer := NewBuffer(&Config{
		Consumer: &consumer{
			wrap: func(consume func()) {
				mutex.Lock()
				defer mutex.Unlock()
				consume()
			},
			push: func(element uint64) {
				record[element] = struct{}{}
			},
		},

		Size: 16,
	})
	element := uint64(1)
	added := uint64(0)

	for a := 0; a < 8; a++ {
		wg.Add(1)
		go func() {
			for i := 0; i < int(buffer.Config.Size); i++ {
				if buffer.Add(atomic.AddUint64(&element, 1)) {
					atomic.AddUint64(&added, 1)
				}
			}
			wg.Done()
		}()
	}

	wg.Wait()

	remain := uint64(0)
	for i := range buffer.data {
		if buffer.data[i] != 0 {
			remain++
		}
	}

	if uint64(len(record))+remain != added {
		t.Fatal("elements missing")
	}
}

func BenchmarkBuffer(b *testing.B) {
	buffer := NewBuffer(&Config{
		Consumer: &consumer{
			wrap: func(consume func()) {},
			push: func(element uint64) {},
		},

		Size: 64,
	})

	for n := 0; n < b.N; n++ {
		buffer.Add(uint64(n))
	}
}

func BenchmarkBufferParallel(b *testing.B) {
	buffer := NewBuffer(&Config{
		Consumer: &consumer{
			wrap: func(consume func()) {},
			push: func(element uint64) {},
		},

		Size: 128,
	})

	b.RunParallel(func(pb *testing.PB) {
		for element := uint64(0); pb.Next(); element++ {
			buffer.Add(element)
		}
	})
}
