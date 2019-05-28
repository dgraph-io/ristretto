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
	"sync"

	"github.com/dgraph-io/ristretto/ring"
	"github.com/golang/groupcache/lru"
)

type BenchBaseMutex struct {
	sync.Mutex
	cache *lru.Cache
	stats Stats
}

func NewBenchBaseMutex(capacity int) *BenchBaseMutex {
	return &BenchBaseMutex{
		cache: lru.New(capacity),
	}
}

func (c *BenchBaseMutex) Get(key string) interface{} {
	c.Lock()
	defer c.Unlock()
	c.stats.Reqs++
	value, _ := c.cache.Get(key)
	// value found
	if value != nil {
		c.stats.Hits++
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
	return &c.stats
}

////////////////////////////////////////////////////////////////////////////////

type BenchBaseMutexWrap struct {
	sync.RWMutex
	cache  *lru.Cache
	buffer *ring.Buffer
	stats  Stats
}

func NewBenchBaseMutexWrap(capacity int) *BenchBaseMutexWrap {
	cache := &BenchBaseMutexWrap{
		cache: lru.New(capacity),
	}
	cache.buffer = ring.NewBuffer(ring.LOSSY, &ring.Config{
		Consumer: cache,
		Capacity: capacity * 16,
	})
	return cache
}

// TODO: can't use groupcache lru for this because we need access to the linked
//       list for bp-wrapper
func (c *BenchBaseMutexWrap) Push(keys []ring.Element) {
	c.Lock()
	defer c.Unlock()
	/*
		for _, key := range keys {
			// move elements to the front of the list
		}
	*/
}

func (c *BenchBaseMutexWrap) Get(key string) interface{} {
	return nil
}

func (c *BenchBaseMutexWrap) Set(key string, value interface{}) {

}

func (c *BenchBaseMutexWrap) Del(key string) {

}

func (c *BenchBaseMutexWrap) Bench() *Stats {
	return &c.stats
}
