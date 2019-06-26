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

package ristretto

import (
	"github.com/dgraph-io/ristretto/ring"
	"github.com/dgraph-io/ristretto/store"
)

// Cache ties everything together. The three main components are:
//
//     1) The hash map: this is the store.Map interface.
//     2) The admission and eviction policy: this is the Policy interface.
//     3) The bp-wrapper buffer: this is the ring.Buffer struct.
//
// All three of these components work together to try and keep the most valuable
// key-value pairs in the hash map. Value is determined by the Policy, and
// BP-Wrapper keeps the Policy fast (by batching metadata updates).
type Cache struct {
	data   store.Map
	policy Policy
	buffer *ring.Buffer
}

type Config struct {
	CacheSize  uint64
	BufferSize uint64
	Policy     func(uint64, store.Map) Policy
	Log        bool
}

func NewCache(config *Config) *Cache {
	// data is the hash map for the entire cache, it's initialized outside of
	// the cache struct declaration because it may need to be passed to the
	// policy in some cases
	data := store.NewMap()
	// initialize the policy (with a recorder wrapping if logging is enabled)
	var policy Policy = config.Policy(config.CacheSize, data)
	if config.Log {
		policy = NewRecorder(config.Policy(config.CacheSize, data), data)
	}
	return &Cache{
		data:   data,
		policy: policy,
		buffer: ring.NewBuffer(ring.LOSSY, &ring.Config{
			Consumer: policy,
			Capacity: config.BufferSize,
		}),
	}
}

/*
// TODO: finalize this and make it atomic
func (c *Cache) GetOrSet(key string, get func() interface{}) interface{} {
	value := c.Get(key)
	if value == nil {
		value = get()
		c.Set(key, value)
	}
	return value
}
*/

func (c *Cache) Get(key string) interface{} {
	c.buffer.Push(ring.Element(key))
	return c.data.Get(key)
}

func (c *Cache) Set(key string, value interface{}) {
	// attempt to add the key to the admission/eviction policy
	if victim, added := c.policy.Add(key); added {
		// delete the victim if there was an eviction
		if victim != "" {
			c.data.Del(key)
		}
		// since the key was added to the policy, add it to the data store too
		c.data.Set(key, value)
	}
}

// TODO: also delete from policy?
func (c *Cache) Del(key string) {
	c.data.Del(key)
}

func (c *Cache) Log() *PolicyLog {
	return c.policy.Log()
}
