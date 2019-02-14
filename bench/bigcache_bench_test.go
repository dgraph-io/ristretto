package bench

import (
	"strconv"
	"testing"

	"github.com/allegro/bigcache"
)

type BigCache struct {
	c *bigcache.BigCache
}

func (b *BigCache) Get(key []byte) ([]byte, error) {
	return b.c.Get(string(key))
}

func (b *BigCache) Set(key, value []byte) error {
	return b.c.Set(string(key), value)
}

func initBigCache(maxEntries, numElems int) *BigCache {
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

	return &BigCache{cache}
}

func BenchmarkBigCacheRead(b *testing.B) {
	cache := initBigCache(b.N, workloadDataSize)
	data := initPatternZipf(workloadDataSize)
	runCacheBenchmark(b, cache, data, 0)
}

func BenchmarkBigCacheWrite(b *testing.B) {
	cache := initBigCache(b.N, workloadDataSize)
	data := initPatternZipf(workloadDataSize)
	runCacheBenchmark(b, cache, data, 100)
}

// 25% write and 75% read benchmark
func BenchmarkBigCacheReadWrite(b *testing.B) {
	cache := initBigCache(b.N, workloadDataSize)
	data := initPatternZipf(workloadDataSize)
	runCacheBenchmark(b, cache, data, 25)
}

func BenchmarkBigCacheHotKeyRead(b *testing.B) {
	cache := initBigCache(b.N, workloadDataSize)
	data := initPatternHotKey(workloadDataSize)
	runCacheBenchmark(b, cache, data, 0)
}

func BenchmarkBigCacheHotKeyWrite(b *testing.B) {
	cache := initBigCache(b.N, workloadDataSize)
	data := initPatternHotKey(workloadDataSize)
	runCacheBenchmark(b, cache, data, 100)
}

// 25% write and 75% read benchmark
func BenchmarkBigCacheHotKeyReadWrite(b *testing.B) {
	cache := initBigCache(b.N, workloadDataSize)
	data := initPatternHotKey(workloadDataSize)
	runCacheBenchmark(b, cache, data, 25)
}
