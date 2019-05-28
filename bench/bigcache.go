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
	"time"

	"github.com/allegro/bigcache"
)

type BenchBigCache struct {
	cache *bigcache.BigCache
	stats Stats
}

func NewBenchBigCache(capacity int) *BenchBigCache {
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
	return &BenchBigCache{cache: cache}
}

func (c *BenchBigCache) Get(key string) interface{} {
	c.stats.Reqs++
	data, err := c.cache.Get(key)
	if err != nil {
		log.Panic(err)
	}
	// entry found
	if data != nil {
		c.stats.Hits++
	}
	return data
}

func (c *BenchBigCache) Set(key string, value interface{}) {
	if err := c.cache.Set(key, value.([]byte)); err != nil {
		log.Panic(err)
	}
}

func (c *BenchBigCache) Del(key string) {
	c.cache.Delete(key)
}

func (c *BenchBigCache) Bench() *Stats {
	return &c.stats
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
