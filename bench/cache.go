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

package main

import (
	"github.com/dgraph-io/ristretto"
)

type Cache interface {
	Get(string) interface{}
	Set(string, interface{})
	Del(string)
	Log() *ristretto.PolicyLog
}

type BenchRistretto struct {
	cache *ristretto.Cache
}

func NewBenchRistretto(capacity int) Cache {
	return &BenchRistretto{
		cache: ristretto.NewCache(&ristretto.Config{
			CacheSize:  uint64(capacity),
			BufferSize: uint64(capacity),
			Log:        true,
		}),
	}
}

func (c *BenchRistretto) Get(key string) interface{} {
	return c.cache.Get(key)
}

func (c *BenchRistretto) Set(key string, value interface{}) {
	c.cache.Set(key, value)
}

func (c *BenchRistretto) Del(key string) {
	c.cache.Del(key)
}

func (c *BenchRistretto) Log() *ristretto.PolicyLog {
	return c.cache.Log()
}

/*
type BenchBaseMutex struct {
	sync.Mutex
	cache *lru.Cache
}

func NewBenchBaseMutex(capacity int) Cache {
	return &BenchBaseMutex{
		cache: lru.New(capacity),
	}
}

func (c *BenchBaseMutex) Get(key string) interface{} {
	c.Lock()
	defer c.Unlock()
	value, _ := c.cache.Get(key)
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

type BenchBigCache struct {
	cache *bigcache.BigCache
	stats *Stats
}

func NewBenchBigCache(capacity int) Cache {
	// NOTE: bigcache automatically allocates more memory, there's no way to set
	//       a hard limit on the item/memory capacity
	//
	// create a bigcache instance with default config values except for the
	// logger - we don't want them messing with our stdout
	//
	// https://github.com/allegro/bigcache/blob/master/config.go#L47
	cache, err := bigcache.NewBigCache(bigcache.Config{
		Shards:             2,
		LifeWindow:         time.Second * 30,
		CleanWindow:        0,
		MaxEntriesInWindow: capacity,
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

func (c *BenchBigCache) GetFast(key string) interface{} {
	value, _ := c.cache.Get(key)
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

type BenchFastCache struct {
	cache *fastcache.Cache
	stats *Stats
}

func NewBenchFastCache(capacity int) Cache {
	// NOTE: if capacity is less than 32MB, then fastcache sets it to 32MB
	return &BenchFastCache{
		cache: fastcache.New(capacity),
		stats: &Stats{},
	}
}

func (c *BenchFastCache) Get(key string) interface{} {
	atomic.AddUint64(&c.stats.Reqs, 1)
	value := c.cache.Get(nil, []byte(key))
	if value != nil {
		atomic.AddUint64(&c.stats.Hits, 1)
	} else {
		c.Set(key, []byte("*"))
	}
	return value
}

func (c *BenchFastCache) GetFast(key string) interface{} {
	return c.cache.Get(nil, []byte(key))
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

type BenchFreeCache struct {
	cache *freecache.Cache
	stats *Stats
}

func NewBenchFreeCache(capacity int) Cache {
	// NOTE: if capacity is less than 512KB, then freecache sets it to 512KB
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

func (c *BenchFreeCache) GetFast(key string) interface{} {
	value, _ := c.cache.Get([]byte(key))
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

func (c *BenchGoburrow) GetFast(key string) interface{} {
	value, _ := c.cache.GetIfPresent(key)
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
*/
