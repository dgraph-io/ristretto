package ring

import (
	"math/rand"
	"testing"
)

func TestBuffers(t *testing.T) {
	drained := uint64(0)
	record := make(map[uint64]struct{})
	buffers := NewBuffers(&Config{
		Consumer: &consumer{
			wrap: func(consume func()) {
				drained++
				consume()
			},
			push: func(element uint64) {
				record[element] = struct{}{}
			},
		},

		Size: 16,
		Rows: 4,
	})
	rounds := (uint64(buffers.Config.Size) * buffers.Config.Rows) * 1024

	for i := uint64(1); i <= rounds; i++ {
		buffers.Add(rand.Uint64())
	}

	count := len(record)
	for i := range buffers.rows {
		for j := range buffers.rows[i].data {
			if buffers.rows[i].data[j] == 0 {
				break
			}

			count++
		}
	}

	if rounds != uint64(count) {
		t.Fatal("elements missing")
	}
}

func BenchmarkBuffers(b *testing.B) {
	buffers := NewBuffers(&Config{
		Consumer: &consumer{
			wrap: func(consume func()) {},
			push: func(element uint64) {},
		},

		Size: 128,
		Rows: 16,
	})

	for n := 0; n < b.N; n++ {
		buffers.Add(uint64(n))
	}
}

func BenchmarkBuffersParallel(b *testing.B) {
	buffers := NewBuffers(&Config{
		Consumer: &consumer{
			wrap: func(consume func()) {},
			push: func(element uint64) {},
		},

		Size: 128,
		Rows: 16,
	})

	b.RunParallel(func(pb *testing.PB) {
		for element := uint64(0); pb.Next(); element++ {
			buffers.Add(element)
		}
	})
}
