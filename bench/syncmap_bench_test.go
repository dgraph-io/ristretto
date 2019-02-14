package bench

import (
	"errors"
	"sync"
	"testing"
)

var (
	ErrKeyNotFound  = errors.New("key not found")
	ErrInvalidValue = errors.New("invalid value")
)

type SyncMap struct {
	c *sync.Map
}

func (m *SyncMap) Get(key []byte) ([]byte, error) {
	v, ok := m.c.Load(string(key))
	if !ok {
		return nil, ErrKeyNotFound
	}

	tv, ok := v.([]byte)
	if !ok {
		return nil, ErrInvalidValue
	}

	return tv, nil
}

func (m *SyncMap) Set(key, value []byte) error {
	m.c.Store(string(key), value)
	return nil
}

func initSyncMap() *SyncMap {
	return &SyncMap{new(sync.Map)}
}

func BenchmarkSyncMapRead(b *testing.B) {
	cache := initSyncMap()
	data := initPatternZipf(workloadDataSize)
	runCacheBenchmark(b, cache, data, 0)
}

func BenchmarkSyncMapWrite(b *testing.B) {
	cache := initSyncMap()
	data := initPatternZipf(workloadDataSize)
	runCacheBenchmark(b, cache, data, 100)
}

// 25% write and 75% read benchmark
func BenchmarkSyncMapReadWrite(b *testing.B) {
	cache := initSyncMap()
	data := initPatternZipf(workloadDataSize)
	runCacheBenchmark(b, cache, data, 25)
}

func BenchmarkSyncMapHotKeyRead(b *testing.B) {
	cache := initSyncMap()
	data := initPatternHotKey(workloadDataSize)
	runCacheBenchmark(b, cache, data, 0)
}

func BenchmarkSyncMapHotKeyWrite(b *testing.B) {
	cache := initSyncMap()
	data := initPatternHotKey(workloadDataSize)
	runCacheBenchmark(b, cache, data, 100)
}

// 25% write and 75% read benchmark
func BenchmarkSyncMapHotKeyReadWrite(b *testing.B) {
	cache := initSyncMap()
	data := initPatternHotKey(workloadDataSize)
	runCacheBenchmark(b, cache, data, 25)
}
