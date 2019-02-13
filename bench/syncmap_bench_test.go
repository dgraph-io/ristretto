package bench

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
)

func runSyncMapBenchmark(b *testing.B,
	cache sync.Map, data []string, pctWrites uint64) {

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
		cache.Store(data[i], []byte("data"))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		index := rand.Int() & mask
		myCounter := atomic.AddUint64(&routineCoutner, 1)

		if myCounter%writeRoutineCounter == 0 {
			for pb.Next() {
				cache.Store(data[index&mask], []byte("data"))
				index = index + 1
			}
		} else {
			for pb.Next() {
				cache.Load(data[index&mask])
				index = index + 1
			}
		}
	})
}

func BenchmarkSyncMapRead(b *testing.B) {
	var cache sync.Map
	data := initAccessPatternString(workloadDataSize)
	runSyncMapBenchmark(b, cache, data, 0)
}

func BenchmarkSyncMapWrite(b *testing.B) {
	var cache sync.Map
	data := initAccessPatternString(workloadDataSize)
	runSyncMapBenchmark(b, cache, data, 100)
}

// 25% write and 75% read benchmark
func BenchmarkSyncMapReadWrite(b *testing.B) {
	var cache sync.Map
	data := initAccessPatternString(workloadDataSize)
	runSyncMapBenchmark(b, cache, data, 25)
}

func BenchmarkSyncMapHotKeyRead(b *testing.B) {
	var cache sync.Map
	data := initHotKeyAccessPatternString(workloadDataSize)
	runSyncMapBenchmark(b, cache, data, 0)
}

func BenchmarkSyncMapHotKeyWrite(b *testing.B) {
	var cache sync.Map
	data := initHotKeyAccessPatternString(workloadDataSize)
	runSyncMapBenchmark(b, cache, data, 100)
}

// 25% write and 75% read benchmark
func BenchmarkSyncMapHotKeyReadWrite(b *testing.B) {
	var cache sync.Map
	data := initHotKeyAccessPatternString(workloadDataSize)
	runSyncMapBenchmark(b, cache, data, 25)
}
