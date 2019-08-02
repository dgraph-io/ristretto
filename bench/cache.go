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
	"log"
	"sync"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/allegro/bigcache"
	"github.com/coocood/freecache"
	"github.com/dgraph-io/ristretto"
	goburrow "github.com/goburrow/cache"
	"github.com/golang/groupcache/lru"
)

type Cache interface {
	Get(string) (interface{}, bool)
	Set(string, interface{})
	Del(string)
	Log() *policyLog
}

type BenchRistretto struct {
	cache *ristretto.Cache
	track bool
	log   *policyLog
}

func NewBenchRistretto(capacity int, track bool) Cache {
	c, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: int64(capacity * 10),
		MaxCost:     int64(capacity),
		BufferItems: 64,
	})
	if err != nil {
		panic(err)
	}
	return &BenchRistretto{
		cache: c,
		track: track,
		log:   &policyLog{},
	}
}

func (c *BenchRistretto) Get(key string) (interface{}, bool) {
	return c.cache.Get(key)
}

func (c *BenchRistretto) Set(key string, value interface{}) {
	if c.track {
		if _, ok := c.cache.Get(key); ok {
			c.log.Hit()
		} else {
			c.log.Miss()
		}
	}
	c.cache.Set(key, value, 1)
}

func (c *BenchRistretto) Del(key string) {
	c.cache.Del(key)
}

func (c *BenchRistretto) Log() *policyLog {
	return c.log
}

type BenchBaseMutex struct {
	sync.Mutex
	cache *lru.Cache
	log   *policyLog
	track bool
}

func NewBenchBaseMutex(capacity int, track bool) Cache {
	return &BenchBaseMutex{
		cache: lru.New(capacity),
		log:   &policyLog{},
		track: track,
	}
}

func (c *BenchBaseMutex) Get(key string) (interface{}, bool) {
	c.Lock()
	defer c.Unlock()
	return c.cache.Get(key)
}

func (c *BenchBaseMutex) Set(key string, value interface{}) {
	c.Lock()
	defer c.Unlock()
	if c.track {
		if value, _ := c.cache.Get(key); value != nil {
			c.log.Hit()
		} else {
			c.log.Miss()
		}
	}
	c.cache.Add(key, value)
}

func (c *BenchBaseMutex) Del(key string) {
	c.cache.Remove(key)
}

func (c *BenchBaseMutex) Log() *policyLog {
	return c.log
}

type BenchBigCache struct {
	cache *bigcache.BigCache
	log   *policyLog
	track bool
}

func NewBenchBigCache(capacity int, track bool) Cache {
	// NOTE: bigcache automatically allocates more memory, there's no way to set
	//       a hard limit on the item/memory capacity
	//
	// create a bigcache instance with default config values except for the
	// logger - we don't want them messing with our stdout
	//
	// https://github.com/allegro/bigcache/blob/master/config.go#L47
	cache, err := bigcache.NewBigCache(bigcache.Config{
		Shards:             256,
		LifeWindow:         0,
		MaxEntriesInWindow: capacity,
		MaxEntrySize:       128,
		Verbose:            false,
	})
	if err != nil {
		log.Panic(err)
	}
	return &BenchBigCache{
		cache: cache,
		log:   &policyLog{},
		track: track,
	}
}

func (c *BenchBigCache) Get(key string) (interface{}, bool) {
	value, err := c.cache.Get(key)
	if err != nil {
		panic(err)
		return nil, false
	}
	return value, true
}

func (c *BenchBigCache) Set(key string, value interface{}) {
	if c.track {
		if value, _ := c.cache.Get(key); value != nil {
			c.log.Hit()
		} else {
			c.log.Miss()
		}
	}
	if err := c.cache.Set(key, value.([]byte)); err != nil {
		log.Panic(err)
	}
}

func (c *BenchBigCache) Del(key string) {
	if err := c.cache.Delete(key); err != nil {
		log.Panic(err)
	}
}

func (c *BenchBigCache) Log() *policyLog {
	return c.log
}

type BenchFastCache struct {
	cache *fastcache.Cache
	log   *policyLog
	track bool
}

func NewBenchFastCache(capacity int, track bool) Cache {
	// NOTE: if capacity is less than 32MB, then fastcache sets it to 32MB
	return &BenchFastCache{
		cache: fastcache.New(capacity),
		log:   &policyLog{},
		track: track,
	}
}

func (c *BenchFastCache) Get(key string) (interface{}, bool) {
	value := c.cache.Get(nil, []byte(key))
	if len(value) > 0 {
		return value, true
	}
	return value, false
}

func (c *BenchFastCache) GetFast(key string) interface{} {
	return c.cache.Get(nil, []byte(key))
}

func (c *BenchFastCache) Set(key string, value interface{}) {
	if c.track {
		if c.cache.Get(nil, []byte(key)) != nil {
			c.log.Hit()
		} else {
			c.log.Miss()
		}
	}
	c.cache.Set([]byte(key), []byte("*"))
}

func (c *BenchFastCache) Del(key string) {
	c.cache.Del([]byte(key))
}

func (c *BenchFastCache) Log() *policyLog {
	return c.log
}

type BenchFreeCache struct {
	cache *freecache.Cache
	log   *policyLog
	track bool
}

func NewBenchFreeCache(capacity int, track bool) Cache {
	// NOTE: if capacity is less than 512KB, then freecache sets it to 512KB
	return &BenchFreeCache{
		cache: freecache.NewCache(capacity),
		log:   &policyLog{},
		track: track,
	}
}

func (c *BenchFreeCache) Get(key string) (interface{}, bool) {
	value, err := c.cache.Get([]byte(key))
	if err != nil {
		return value, false
	}
	return value, true
}

func (c *BenchFreeCache) Set(key string, value interface{}) {
	if c.track {
		if value, _ := c.cache.Get([]byte(key)); value != nil {
			c.log.Hit()
		} else {
			c.log.Miss()
		}
	}
	if err := c.cache.Set([]byte(key), value.([]byte), 0); err != nil {
		log.Panic(err)
	}
}

func (c *BenchFreeCache) Del(key string) {
	c.cache.Del([]byte(key))
}

func (c *BenchFreeCache) Log() *policyLog {
	return c.log
}

type BenchGoburrow struct {
	cache goburrow.Cache
	log   *policyLog
	track bool
}

func NewBenchGoburrow(capacity int, track bool) Cache {
	return &BenchGoburrow{
		cache: goburrow.New(
			goburrow.WithMaximumSize(capacity),
		),
		log:   &policyLog{},
		track: track,
	}
}

func (c *BenchGoburrow) Get(key string) (interface{}, bool) {
	return c.cache.GetIfPresent(key)
}

func (c *BenchGoburrow) Set(key string, value interface{}) {
	if c.track {
		if value, _ := c.cache.GetIfPresent(key); value != nil {
			c.log.Hit()
		} else {
			c.log.Miss()
		}
	}
	c.cache.Put(key, value)
}

func (c *BenchGoburrow) Del(key string) {
	c.cache.Invalidate(key)
}

func (c *BenchGoburrow) Log() *policyLog {
	return c.log
}
