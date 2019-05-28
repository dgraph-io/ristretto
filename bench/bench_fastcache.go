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

import "github.com/VictoriaMetrics/fastcache"

type BenchFastCache struct {
	cache *fastcache.Cache
	stats Stats
}

func NewBenchFastCache(capacity int) *BenchFastCache {
	return &BenchFastCache{
		cache: fastcache.New(capacity),
	}
}

func (c *BenchFastCache) Get(key string) interface{} {
	c.stats.Reqs++

	var data []byte
	c.cache.Get(data, []byte(key))

	if data != nil {
		c.stats.Hits++
	}

	return data
}

func (c *BenchFastCache) Set(key string, value interface{}) {
	c.cache.Set([]byte(key), []byte("*"))
}

func (c *BenchFastCache) Del(key string) {
	c.cache.Del([]byte(key))
}

func (c *BenchFastCache) Bench() *Stats {
	return &c.stats
}
