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

package bench

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/allegro/bigcache"
	"github.com/coocood/freecache"
	"github.com/dgraph-io/ristretto"
	goburrow "github.com/goburrow/cache"
	"github.com/golang/groupcache/lru"
)

// Cache needs to be fulfilled by the cache implementations in order for the
// benchmarks to run properly.
type Cache interface {
	Get(string) interface{}
	Set(string, interface{})
	Del(string)
	Bench() *Stats
}

// Stats holds hit/miss information after a round of benchmark iterations has
// been ran.
type Stats struct {
	Reqs uint64
	Hits uint64
}

////////////////////////////////////////////////////////////////////////////////

type BenchRistretto struct {
	cache *ristretto.Cache
	stats *Stats
}

func NewBenchRistretto(capacity int) Cache {
	return &BenchRistretto{
		cache: ristretto.NewCache(uint64(capacity)),
		stats: &Stats{},
	}
}

func (c *BenchRistretto) Get(key string) interface{} {
	atomic.AddUint64(&c.stats.Reqs, 1)
	value := c.cache.Get(key)
	if value != nil {
		atomic.AddUint64(&c.stats.Hits, 1)
	} else {
		c.Set(key, []byte("*"))
	}
	return value
}

func (c *BenchRistretto) Set(key string, value interface{}) {
	c.cache.Set(key, value)
}

func (c *BenchRistretto) Del(key string) {
	c.cache.Del(key)
}

func (c *BenchRistretto) Bench() *Stats {
	return c.stats
}

////////////////////////////////////////////////////////////////////////////////

type BenchBaseMutex struct {
	sync.Mutex
	cache *lru.Cache
	stats *Stats
}

func NewBenchBaseMutex(capacity int) Cache {
	return &BenchBaseMutex{
		cache: lru.New(capacity),
		stats: &Stats{},
	}
}

func (c *BenchBaseMutex) Get(key string) interface{} {
	c.Lock()
	defer c.Unlock()
	atomic.AddUint64(&c.stats.Reqs, 1)
	value, _ := c.cache.Get(key)
	if value != nil {
		atomic.AddUint64(&c.stats.Hits, 1)
	} else {
		// same as c.Set but without nested mutex calls
		c.cache.Add(key, []byte("*"))
	}
	return value
}

func (c *BenchBaseMutex) Set(key string, value interface{}) {
	c.Lock()
	defer c.Unlock()
	c.cache.Add(key, value)
}

func (c *BenchBaseMutex) Del(key string) {
	c.cache.Remove(key)
}

func (c *BenchBaseMutex) Bench() *Stats {
	return c.stats
}

////////////////////////////////////////////////////////////////////////////////

type BenchBigCache struct {
	cache *bigcache.BigCache
	stats *Stats
}

func NewBenchBigCache(capacity int) Cache {
	// create a bigcache instance with default config values except for the
	// logger - we don't want them messing with our stdout
	//
	// https://github.com/allegro/bigcache/blob/master/config.go#L47
	cache, err := bigcache.NewBigCache(bigcache.Config{
		Shards:             1024,
		LifeWindow:         time.Second * 30,
		CleanWindow:        0,
		MaxEntriesInWindow: 1000 * 10 * 60,
		MaxEntrySize:       500,
		Verbose:            true,
		Hasher:             newBigCacheHasher(),
		HardMaxCacheSize:   0,
		Logger:             nil,
	})
	if err != nil {
		log.Panic(err)
	}
	return &BenchBigCache{
		cache: cache,
		stats: &Stats{},
	}
}

func (c *BenchBigCache) Get(key string) interface{} {
	atomic.AddUint64(&c.stats.Reqs, 1)
	value, _ := c.cache.Get(key)
	if value != nil {
		atomic.AddUint64(&c.stats.Hits, 1)
	} else {
		c.Set(key, []byte("*"))
	}
	return value
}

func (c *BenchBigCache) Set(key string, value interface{}) {
	if err := c.cache.Set(key, value.([]byte)); err != nil {
		log.Panic(err)
	}
}

func (c *BenchBigCache) Del(key string) {
	if err := c.cache.Delete(key); err != nil {
		log.Panic(err)
	}
}

func (c *BenchBigCache) Bench() *Stats {
	return c.stats
}

////////////////////////////////////////////////////////////////////////////////

// bigCacheHasher is just trying to mimic bigcache's internal implementation of
// a 64bit fnv-1a hasher
//
// https://github.com/allegro/bigcache/blob/master/fnv.go
type bigCacheHasher struct{}

func newBigCacheHasher() *bigCacheHasher { return &bigCacheHasher{} }

func (h bigCacheHasher) Sum64(key string) uint64 {
	hash := uint64(14695981039346656037)
	for i := 0; i < len(key); i++ {
		hash ^= uint64(key[i])
		hash *= 1099511628211
	}
	return hash
}

////////////////////////////////////////////////////////////////////////////////

type BenchFastCache struct {
	cache *fastcache.Cache
	stats *Stats
}

func NewBenchFastCache(capacity int) Cache {
	return &BenchFastCache{
		cache: fastcache.New(capacity),
		stats: &Stats{},
	}
}

func (c *BenchFastCache) Get(key string) interface{} {
	atomic.AddUint64(&c.stats.Reqs, 1)
	var value []byte
	c.cache.Get(value, []byte(key))
	if value != nil {
		atomic.AddUint64(&c.stats.Hits, 1)
	} else {
		c.Set(key, []byte("*"))
	}
	return value
}

func (c *BenchFastCache) Set(key string, value interface{}) {
	c.cache.Set([]byte(key), []byte("*"))
}

func (c *BenchFastCache) Del(key string) {
	c.cache.Del([]byte(key))
}

func (c *BenchFastCache) Bench() *Stats {
	return c.stats
}

////////////////////////////////////////////////////////////////////////////////

type BenchFreeCache struct {
	cache *freecache.Cache
	stats *Stats
}

func NewBenchFreeCache(capacity int) Cache {
	return &BenchFreeCache{
		cache: freecache.NewCache(capacity),
		stats: &Stats{},
	}
}

func (c *BenchFreeCache) Get(key string) interface{} {
	atomic.AddUint64(&c.stats.Reqs, 1)
	value, _ := c.cache.Get([]byte(key))
	if value != nil {
		atomic.AddUint64(&c.stats.Hits, 1)
	} else {
		c.Set(key, []byte("*"))
	}
	return value
}

func (c *BenchFreeCache) Set(key string, value interface{}) {
	if err := c.cache.Set([]byte(key), value.([]byte), 0); err != nil {
		log.Panic(err)
	}
}

func (c *BenchFreeCache) Del(key string) {
	c.cache.Del([]byte(key))
}

func (c *BenchFreeCache) Bench() *Stats {
	return c.stats
}

////////////////////////////////////////////////////////////////////////////////

type BenchGoburrow struct {
	cache goburrow.Cache
	stats *Stats
}

func NewBenchGoburrow(capacity int) Cache {
	return &BenchGoburrow{
		cache: goburrow.New(
			goburrow.WithMaximumSize(capacity),
		),
		stats: &Stats{},
	}
}

func (c *BenchGoburrow) Get(key string) interface{} {
	atomic.AddUint64(&c.stats.Reqs, 1)
	value, _ := c.cache.GetIfPresent(key)
	if value != nil {
		atomic.AddUint64(&c.stats.Hits, 1)
	} else {
		c.Set(key, []byte("*"))
	}
	return value
}

func (c *BenchGoburrow) Set(key string, value interface{}) {
	c.cache.Put(key, value)
}

func (c *BenchGoburrow) Del(key string) {
	c.cache.Invalidate(key)
}

func (c *BenchGoburrow) Bench() *Stats {
	return c.stats
}
