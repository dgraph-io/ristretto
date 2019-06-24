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

// LFU_SAMPLE is the number of items to sample when looking at eviction
// candidates.
const LFU_SAMPLE = 5

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
	logging bool
	log     *PolicyLog
	data    store.Map
	policy  Policy
	buffer  *ring.Buffer
}

type Config struct {
	CacheSize  uint64
	BufferSize uint64
	Log        bool
}

func NewCache(config *Config) *Cache {
	data := store.NewMap()
	policy := NewTinyLFU(config, data)
	return &Cache{
		logging: config.Log,
		log:     &PolicyLog{},
		data:    data,
		policy:  policy,
		buffer: ring.NewBuffer(ring.LOSSY, &ring.Config{
			Consumer: policy,
			Capacity: config.BufferSize,
		}),
	}
}

func (c *Cache) Get(key string) interface{} {
	c.buffer.Push(ring.Element(key))
	return c.data.Get(key)
}

// TODO: clean up the policy logging, right now it's here to abstract it out
//       from the individual policies (for example, it's impossible to track
//       from within the probablistic tinylfu policy)
func (c *Cache) Set(key string, value interface{}) {
	// if already exists, just update the value
	if rawValue := c.data.Get(key); rawValue != nil {
		if c.logging {
			c.log.Hit()
		}
		c.data.Set(key, value)
	}
	// attempt to add and delete victim if needed
	if victim, added := c.policy.Add(key); added {
		if c.logging {
			c.log.Miss()
		}
		// check if there was an eviction victim
		if victim != "" {
			if c.logging {
				c.log.Evict()
			}
			c.data.Del(key)
		}
		// since the key was added to the policy, add it to the data store too
		c.data.Set(key, value)
	}
}

func (c *Cache) Del(key string) {
	c.data.Del(key)
}

func (c *Cache) Log() *PolicyLog {
	return c.log
}
