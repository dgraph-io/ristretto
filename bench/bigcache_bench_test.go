package bench

import (
	"math/rand"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/allegro/bigcache"
)

func initBigCache(maxEntries, numElems int) *bigcache.BigCache {
	numShards := 256

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
	for i := 0; i < 2*numElems; i++ {
		cache.Set(strconv.Itoa(i), []byte("data"))
	}
	cache.Reset()

	return cache
}

func runBigCacheBenchmark(b *testing.B,
	cache *bigcache.BigCache, data []string, pctWrites uint64) {

	size := len(data)
	mask := size - 1

	var writeRoutineCounter uint64
	if pctWrites == 0 {
		writeRoutineCounter = 1<<64 - 1
	} else {
		// TODO: only works when no remainder
		writeRoutineCounter = 100 / pctWrites
	}
	routineCoutner := uint64(0)

	// initialize cache
	for i := 0; i < size; i++ {
		cache.Set(data[i], []byte("data"))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		index := rand.Int() & mask
		myCounter := atomic.AddUint64(&routineCoutner, 1)

		if myCounter%writeRoutineCounter == 0 {
			for pb.Next() {
				cache.Set(data[index&mask], []byte("data"))
				index = index + 1
			}
		} else {
			for pb.Next() {
				cache.Get(data[index&mask])
				index = index + 1
			}
		}
	})
}

func BenchmarkBigCacheRead(b *testing.B) {
	cache := initBigCache(b.N, workloadDataSize)
	data := initAccessPatternString(workloadDataSize)
	runBigCacheBenchmark(b, cache, data, 0)
}

func BenchmarkBigCacheWrite(b *testing.B) {
	cache := initBigCache(b.N, workloadDataSize)
	data := initAccessPatternString(workloadDataSize)
	runBigCacheBenchmark(b, cache, data, 100)
}

// 25% write and 75% read benchmark
func BenchmarkBigCacheReadWrite(b *testing.B) {
	cache := initBigCache(b.N, workloadDataSize)
	data := initAccessPatternString(workloadDataSize)
	runBigCacheBenchmark(b, cache, data, 25)
}

func BenchmarkBigCacheHotKeyRead(b *testing.B) {
	cache := initBigCache(b.N, workloadDataSize)
	data := initHotKeyAccessPatternString(workloadDataSize)
	runBigCacheBenchmark(b, cache, data, 0)
}

func BenchmarkBigCacheHotKeyWrite(b *testing.B) {
	cache := initBigCache(b.N, workloadDataSize)
	data := initHotKeyAccessPatternString(workloadDataSize)
	runBigCacheBenchmark(b, cache, data, 100)
}

// 25% write and 75% read benchmark
func BenchmarkBigCacheHotKeyReadWrite(b *testing.B) {
	cache := initBigCache(b.N, workloadDataSize)
	data := initHotKeyAccessPatternString(workloadDataSize)
	runBigCacheBenchmark(b, cache, data, 25)
}
