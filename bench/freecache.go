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

	"github.com/coocood/freecache"
)

type BenchFreeCache struct {
	cache *freecache.Cache
	stats Stats
}

func NewBenchFreeCache(capacity int) *BenchFreeCache {
	return &BenchFreeCache{
		cache: freecache.NewCache(capacity),
	}
}

func (c *BenchFreeCache) Get(key string) interface{} {
	c.stats.Reqs++
	value, err := c.cache.Get([]byte(key))
	if err != nil {
		log.Panic(err)
	}
	// value found
	if value != nil {
		c.stats.Hits++
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
	return &c.stats
}
