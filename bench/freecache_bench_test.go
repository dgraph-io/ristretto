package bench

import (
	"strconv"
	"testing"

	"github.com/coocood/freecache"
)

type FreeCache struct {
	c *freecache.Cache
}

func (f *FreeCache) Get(key []byte) ([]byte, error) {
	return f.c.Get(key)
}

func (f *FreeCache) Set(key, value []byte) error {
	return f.c.Set(key, value, 0)
}

func initFreeCache(maxEntries, numElements int) *FreeCache {
	cache := freecache.NewCache(maxEntries * maxEntrySize)

	// Enforce full initialization of internal structures
	for i := 0; i < 2*numElements; i++ {
		cache.Set([]byte(strconv.Itoa(i)), []byte("data"), 0)
	}
	cache.Clear()

	return &FreeCache{cache}
}

func BenchmarkFreeCacheRead(b *testing.B) {
	cache := initFreeCache(b.N, workloadDataSize)
	data := initPatternZipf(workloadDataSize)
	runCacheBenchmark(b, cache, data, 0)
}

func BenchmarkFreeCacheWrite(b *testing.B) {
	cache := initFreeCache(b.N, workloadDataSize)
	data := initPatternZipf(workloadDataSize)
	runCacheBenchmark(b, cache, data, 100)
}

// 25% write and 75% read benchmark
func BenchmarkFreeCacheReadWrite(b *testing.B) {
	cache := initFreeCache(b.N, workloadDataSize)
	data := initPatternZipf(workloadDataSize)
	runCacheBenchmark(b, cache, data, 25)
}

func BenchmarkFreeCacheHotKeyRead(b *testing.B) {
	cache := initFreeCache(b.N, workloadDataSize)
	data := initPatternHotKey(workloadDataSize)
	runCacheBenchmark(b, cache, data, 0)
}

func BenchmarkFreeCacheHotKeyWrite(b *testing.B) {
	cache := initFreeCache(b.N, workloadDataSize)
	data := initPatternHotKey(workloadDataSize)
	runCacheBenchmark(b, cache, data, 100)
}

// 25% write and 75% read benchmark
func BenchmarkFreeCacheHotKeyReadWrite(b *testing.B) {
	cache := initFreeCache(b.N, workloadDataSize)
	data := initPatternHotKey(workloadDataSize)
	runCacheBenchmark(b, cache, data, 25)
}
