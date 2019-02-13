package bench

import (
	"math/rand"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/coocood/freecache"
)

func initFreeCache(maxEntries, numElements int) *freecache.Cache {
	cache := freecache.NewCache(maxEntries * maxEntrySize)

	// Enforce full initialization of internal structures
	for i := 0; i < 2*numElements; i++ {
		cache.Set([]byte(strconv.Itoa(i)), []byte("data"), 0)
	}
	cache.Clear()

	return cache
}

func runFreeCacheBenchmark(b *testing.B,
	cache *freecache.Cache, data [][]byte, pctWrites uint64) {

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
		cache.Set(data[i], []byte("data"), 0)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		index := rand.Int() & mask
		myCounter := atomic.AddUint64(&routineCoutner, 1)

		if myCounter%writeRoutineCounter == 0 {
			for pb.Next() {
				cache.Set(data[index&mask], []byte("data"), 0)
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

func BenchmarkFreeCacheRead(b *testing.B) {
	cache := initFreeCache(b.N, workloadDataSize)
	data := initAccessPatternBytes(workloadDataSize)
	runFreeCacheBenchmark(b, cache, data, 0)
}

func BenchmarkFreeCacheWrite(b *testing.B) {
	cache := initFreeCache(b.N, workloadDataSize)
	data := initAccessPatternBytes(workloadDataSize)
	runFreeCacheBenchmark(b, cache, data, 100)
}

// 25% write and 75% read benchmark
func BenchmarkFreeCacheReadWrite(b *testing.B) {
	cache := initFreeCache(b.N, workloadDataSize)
	data := initAccessPatternBytes(workloadDataSize)
	runFreeCacheBenchmark(b, cache, data, 25)
}

func BenchmarkFreeCacheHotKeyRead(b *testing.B) {
	cache := initFreeCache(b.N, workloadDataSize)
	data := initHotKeyAccessPatternBytes(workloadDataSize)
	runFreeCacheBenchmark(b, cache, data, 0)
}

func BenchmarkFreeCacheHotKeyWrite(b *testing.B) {
	cache := initFreeCache(b.N, workloadDataSize)
	data := initHotKeyAccessPatternBytes(workloadDataSize)
	runFreeCacheBenchmark(b, cache, data, 100)
}

// 25% write and 75% read benchmark
func BenchmarkFreeCacheHotKeyReadWrite(b *testing.B) {
	cache := initFreeCache(b.N, workloadDataSize)
	data := initHotKeyAccessPatternBytes(workloadDataSize)
	runFreeCacheBenchmark(b, cache, data, 25)
}
