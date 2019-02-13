package bench

import (
	"math/rand"
	"strconv"
	"testing"

	"github.com/allegro/bigcache"
)

const (
	// benchmark parameters
	size  = 2 << 3
	mask  = size - 1
	items = size / 3

	// bigcache parameters
	numShards    = 256
	maxEntrySize = 10 // in bytes for memory initialization
)

func initBigCache(maxEntries int) *bigcache.BigCache {
	cache, err := bigcache.NewBigCache(bigcache.Config{
		Shards:             numShards,
		LifeWindow:         0,
		MaxEntriesInWindow: maxEntries,
		MaxEntrySize:       maxEntrySize,
		Verbose:            false,
	})
	if err != nil {
		panic(err)
	}

	// Enforce full initialization of internal structures
	for i := 0; i < 2*size; i++ {
		cache.Set(strconv.Itoa(i), []byte("data"))
	}

	return cache
}

func BenchmarkBigCacheRead(b *testing.B) {
	cache := initBigCache(b.N)
	ints := initAccessPatternString(size)
	for i := 0; i < size; i++ {
		cache.Set(ints[i], []byte("data"))
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

func BenchmarkBigCacheWrite(b *testing.B) {
	cache := initBigCache(b.N)
	ints := initAccessPatternString(size)
	for i := 0; i < size; i++ {
		cache.Set(ints[i], []byte("data"))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		counter := rand.Int() & mask
		for pb.Next() {
			cache.Set(ints[counter&mask], []byte("data"))
			counter = counter + 1
		}
	})
}

// 25% write and 75% read benchmark
func BenchmarkBigCacheReadWrite(b *testing.B) {
	cache := initBigCache(b.N)
	ints := initAccessPatternString(size)
	for i := 0; i < size; i++ {
		cache.Set(ints[i], []byte("data"))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		counter := rand.Int() & mask

		if rand.Uint64()%4 == 0 {
			for pb.Next() {
				cache.Set(ints[counter&mask], []byte("data"))
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
