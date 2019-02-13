package bench

import (
	"math/rand"
	"strconv"
	"testing"

	"github.com/coocood/freecache"
)

func initFreeCache(maxEntries int) *freecache.Cache {
	cache := freecache.NewCache(maxEntries * maxEntrySize)

	// Enforce full initialization of internal structures
	for i := 0; i < 2*size; i++ {
		cache.Set([]byte(strconv.Itoa(i)), []byte("data"), 0)
	}

	return cache
}

func BenchmarkFreeCacheRead(b *testing.B) {
	cache := initFreeCache(b.N)
	ints := initAccessPatternBytes(size)
	for i := 0; i < size; i++ {
		cache.Set(ints[i], []byte("data"), 0)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		counter := rand.Int() & mask
		for pb.Next() {
			cache.Get(ints[counter&mask])
			counter = counter + 1
		}
	})
}

func BenchmarkFreeCacheWrite(b *testing.B) {
	cache := initFreeCache(b.N)
	ints := initAccessPatternBytes(size)
	for i := 0; i < size; i++ {
		cache.Set(ints[i], []byte("data"), 0)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		counter := rand.Int() & mask
		for pb.Next() {
			cache.Set(ints[counter&mask], []byte("data"), 0)
			counter = counter + 1
		}
	})
}

// 25% write and 75% read benchmark
func BenchmarkFreeCacheReadWrite(b *testing.B) {
	cache := initFreeCache(b.N)
	ints := initAccessPatternBytes(size)
	for i := 0; i < size; i++ {
		cache.Set(ints[i], []byte("data"), 0)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		counter := rand.Int() & mask

		if rand.Uint64()%4 == 0 {
			for pb.Next() {
				cache.Set(ints[counter&mask], []byte("data"), 0)
				counter = counter + 1
			}
		} else {
			for pb.Next() {
				cache.Get(ints[counter&mask])
				counter = counter + 1
			}
		}
	})
}
