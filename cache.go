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

// Cache ties everything together. The three main components are:
//
//     1) The hash map: this is the Map interface.
//     2) The admission and eviction policy: this is the Policy interface.
//     3) The bp-wrapper buffer: this is the Buffer struct.
//
// All three of these components work together to try and keep the most valuable
// key-value pairs in the hash map. Value is determined by the Policy, and
// BP-Wrapper keeps the Policy fast (by batching metadata updates).
type Cache struct {
	data   Map
	policy Policy
	buffer *Buffer
	notify func(string)
}

type Config struct {
	CacheSize  uint64
	BufferSize uint64
	CostAware  bool
	Policy     func(uint64, bool) Policy
	OnEvict    func(string)
	Log        bool
}

func NewCache(config *Config) *Cache {
	// data is the hash map for the entire cache, it's initialized outside of
	// the cache struct declaration because it may need to be passed to the
	// policy in some cases
	data := NewMap()
	// initialize the policy (with a recorder wrapping if logging is enabled)
	var policy Policy = config.Policy(config.CacheSize, config.CostAware)
	if config.Log {
		policy = NewRecorder(policy, data)
	}
	return &Cache{
		data:   data,
		policy: policy,
		buffer: NewBuffer(LOSSY, &RingConfig{
			Consumer: policy,
			Capacity: config.BufferSize,
		}),
		notify: config.OnEvict,
	}
}

func (c *Cache) Get(key string) interface{} {
	c.buffer.Push(Element(key))
	return c.data.Get(key)
}

func (c *Cache) Set(key string, val interface{}, cost uint64) ([]string, bool) {
	victims, added := c.policy.Add(key, cost)
	if !added {
		return nil, false
	}
	for _, victim := range victims {
		c.data.Del(victim)
		if c.notify != nil {
			c.notify(victim)
		}
	}
	c.data.Set(key, val)
	return victims, true
}

func (c *Cache) Del(key string) {
	c.policy.Del(key)
	c.data.Del(key)
}

func (c *Cache) Log() *PolicyLog {
	return c.policy.Log()
}
