/*
 * Copyright 2019 Dgraph Labs, Inc. and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package caffeine

import (
	"errors"
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/allegro/bigcache"
	"github.com/cespare/xxhash"
	"github.com/coocood/freecache"
	"github.com/golang/groupcache/lru"
	"github.com/pingcap/go-ycsb/pkg/generator"
)

const (
	// based on 21million dataset, we observed a maximum key length of 77,
	// with minimum length being 6 and average length being 25. We also
	// observed that 99% of keys had length <64 bytes.
	maxKeyLength = 128
	// workloadSize is the size of array storing sequence of keys that we
	// have in our workload. In the benchmark, we iterate over this array b.N
	// number of times in circular fashion starting at a random position.
	workloadSize = 2 << 20
)

var (
	errKeyNotFound  = errors.New("key not found")
	errInvalidValue = errors.New("invalid value")
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

//========================================================================
//                           Possible workloads
//========================================================================

func zipfKeyList() [][]byte {
	// To ensure repetition of keys in the array,
	// we are generating keys in the range from 0 to workloadSize/3.
	maxKey := int64(workloadSize) / 3

	// scrambled zipfian to ensure same keys are not together
	z := generator.NewScrambledZipfian(0, maxKey, generator.ZipfianConstant)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	keys := make([][]byte, workloadSize)
	for i := 0; i < workloadSize; i++ {
		keys[i] = []byte(strconv.Itoa(int(z.Next(r))))
	}

	return keys
}

func oneKeyList() [][]byte {
	v := rand.Int() % (workloadSize / 3)
	s := []byte(strconv.Itoa(v))

	keys := make([][]byte, workloadSize)
	for i := 0; i < workloadSize; i++ {
		keys[i] = s
	}

	return keys
}

//========================================================================
//                               BigCache
//========================================================================

type BigCache struct {
	c *bigcache.BigCache
}

func (b *BigCache) Get(key []byte) ([]byte, error) {
	return b.c.Get(string(key))
}

func (b *BigCache) Set(key, value []byte) error {
	return b.c.Set(string(key), value)
}

func newBigCache(keysInWindow int) *BigCache {
	cache, err := bigcache.NewBigCache(bigcache.Config{
		Shards:             256,
		LifeWindow:         0,
		MaxEntriesInWindow: keysInWindow,
		MaxEntrySize:       maxKeyLength,
		Verbose:            false,
	})
	if err != nil {
		panic(err)
	}

	// Enforce full initialization of internal structures. This is taken
	// from GetPutBenchmark.java from java caffeine. It is required in
	// caffeine given that it keeps buffering the keys for applying the
	// necessary changes later. This is probably unnecessary here.
	for i := 0; i < 2*workloadSize; i++ {
		_ = cache.Set(strconv.Itoa(i), []byte("data"))
	}
	_ = cache.Reset()

	return &BigCache{cache}
}

//========================================================================
//                            FreeCache
//========================================================================

type FreeCache struct {
	c *freecache.Cache
}

func (f *FreeCache) Get(key []byte) ([]byte, error) {
	return f.c.Get(key)
}

func (f *FreeCache) Set(key, value []byte) error {
	return f.c.Set(key, value, 0)
}

func newFreeCache(keysInWindow int) *FreeCache {
	cache := freecache.NewCache(keysInWindow * maxKeyLength)

	// Enforce full initialization of internal structures
	// (probably not required, see above in bigcache)
	for i := 0; i < 2*workloadSize; i++ {
		_ = cache.Set([]byte(strconv.Itoa(i)), []byte("data"), 0)
	}
	cache.Clear()

	return &FreeCache{cache}
}

//========================================================================
//                            GroupCache
//========================================================================

const (
	segmentAndOpVal = 255
)

type GroupCache struct {
	shards [256]*lru.Cache
	locks  [256]sync.Mutex
}

func (g *GroupCache) Get(key []byte) ([]byte, error) {
	hashVal := xxhash.Sum64(key)
	shardNum := hashVal & segmentAndOpVal

	g.locks[shardNum].Lock()
	v, ok := g.shards[shardNum].Get(string(key))
	g.locks[shardNum].Unlock()

	if ok {
		return v.([]byte), nil
	}
	return nil, errKeyNotFound
}

func (g *GroupCache) Set(key, value []byte) error {
	hashVal := xxhash.Sum64(key)
	shardNum := hashVal & segmentAndOpVal

	g.locks[shardNum].Lock()
	g.shards[shardNum].Add(string(key), value)
	g.locks[shardNum].Unlock()

	return nil
}

func newGroupCache(keysInWindow int) *GroupCache {
	gc := &GroupCache{}
	for i := 0; i < 256; i++ {
		gc.shards[i] = lru.New(keysInWindow / 256)
	}

	// Enforce full initialization of internal structures
	for j := 0; j < 2*workloadSize; j++ {
		gc.Set([]byte(strconv.Itoa(j)), []byte("data"))
	}
	for i := 0; i < 256; i++ {
		gc.shards[i].Clear()
	}

	return gc
}

//========================================================================
//                              sync.Map
//========================================================================

type SyncMap struct {
	c *sync.Map
}

func (m *SyncMap) Get(key []byte) ([]byte, error) {
	v, ok := m.c.Load(string(key))
	if !ok {
		return nil, errKeyNotFound
	}

	tv, ok := v.([]byte)
	if !ok {
		return nil, errInvalidValue
	}

	return tv, nil
}

func (m *SyncMap) Set(key, value []byte) error {
	// We are not performing any initialization here unlike other caches
	// given that there is no function available to reset the map.
	m.c.Store(string(key), value)
	return nil
}

func newSyncMap() *SyncMap {
	return &SyncMap{new(sync.Map)}
}

//========================================================================
//                         Benchmark Code
//========================================================================

func runCacheBenchmark(b *testing.B, cache Cache, keys [][]byte, pctWrites uint64) {
	b.ReportAllocs()

	size := len(keys)
	mask := size - 1
	rc := uint64(0)

	// initialize cache
	for i := 0; i < size; i++ {
		_ = cache.Set(keys[i], []byte("data"))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		index := rand.Int() & mask
		mc := atomic.AddUint64(&rc, 1)

		if pctWrites*mc/100 != pctWrites*(mc-1)/100 {
			for pb.Next() {
				_ = cache.Set(keys[index&mask], []byte("data"))
				index = index + 1
			}
		} else {
			for pb.Next() {
				_, _ = cache.Get(keys[index&mask])
				index = index + 1
			}
		}
	})
}

func BenchmarkCaches(b *testing.B) {
	zipfList := zipfKeyList()
	oneList := oneKeyList()

	// two datasets (zipf, onekey)
	// 3 caches (bigcache, freecache, sync.Map)
	// 3 types of benchmark (read, write, mixed)
	benchmarks := []struct {
		name      string
		cache     Cache
		keys      [][]byte
		pctWrites uint64
	}{
		{"BigCacheZipfRead", newBigCache(b.N), zipfList, 0},
		{"FreeCacheZipfRead", newFreeCache(b.N), zipfList, 0},
		{"GroupCacheZipfRead", newGroupCache(b.N), zipfList, 0},
		{"SyncMapZipfRead", newSyncMap(), zipfList, 0},

		{"BigCacheOneKeyRead", newBigCache(b.N), oneList, 0},
		{"FreeCacheOneKeyRead", newFreeCache(b.N), oneList, 0},
		{"GroupCacheOneKeyRead", newGroupCache(b.N), oneList, 0},
		{"SyncMapOneKeyRead", newSyncMap(), oneList, 0},

		{"BigCacheZipfWrite", newBigCache(b.N), zipfList, 100},
		{"FreeCacheZipfWrite", newFreeCache(b.N), zipfList, 100},
		{"GroupCacheZipfWrite", newGroupCache(b.N), zipfList, 100},
		{"SyncMapZipfWrite", newSyncMap(), zipfList, 100},

		{"BigCacheOneKeyWrite", newBigCache(b.N), oneList, 100},
		{"FreeCacheOneKeyWrite", newFreeCache(b.N), oneList, 100},
		{"GroupCacheOneKeyWrite", newGroupCache(b.N), oneList, 100},
		{"SyncMapOneKeyWrite", newSyncMap(), oneList, 100},

		{"BigCacheZipfMixed", newBigCache(b.N), zipfList, 25},
		{"FreeCacheZipfMixed", newFreeCache(b.N), zipfList, 25},
		{"GroupCacheZipfMixed", newGroupCache(b.N), zipfList, 25},
		{"SyncMapZipfMixed", newSyncMap(), zipfList, 25},

		{"BigCacheOneKeyMixed", newBigCache(b.N), oneList, 25},
		{"FreeCacheOneKeyMixed", newFreeCache(b.N), oneList, 25},
		{"GroupCacheOneKeyMixed", newGroupCache(b.N), oneList, 25},
		{"SyncMapOneKeyMixed", newSyncMap(), oneList, 25},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			runCacheBenchmark(b, bm.cache, bm.keys, bm.pctWrites)
		})
	}
}
