package bench

import (
	"math/rand"
	"sync"
	"testing"
)

func BenchmarkSyncMapRead(b *testing.B) {
	var cache sync.Map

	ints := initAccessPatternString(size)
	for i := 0; i < size; i++ {
		cache.Store(ints[i], []byte("data"))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		counter := rand.Int() & mask
		for pb.Next() {
			cache.Load(ints[counter&mask])
			counter = counter + 1
		}
	})
}

func BenchmarkSyncMapWrite(b *testing.B) {
	var cache sync.Map

	ints := initAccessPatternString(size)
	for i := 0; i < size; i++ {
		cache.Store(ints[i], []byte("data"))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		counter := rand.Int() & mask
		for pb.Next() {
			cache.Store(ints[counter&mask], []byte("data"))
			counter = counter + 1
		}
	})
}

// 25% write and 75% read benchmark
func BenchmarkSyncMapReadWrite(b *testing.B) {
	var cache sync.Map

	ints := initAccessPatternString(size)
	for i := 0; i < size; i++ {
		cache.Store(ints[i], []byte("data"))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		counter := rand.Int() & mask

		if rand.Uint64()%4 == 0 {
			for pb.Next() {
				cache.Store(ints[counter&mask], []byte("data"))
				counter = counter + 1
			}
		} else {
			for pb.Next() {
				cache.Load(ints[counter&mask])
				counter = counter + 1
			}
		}
	})
}
