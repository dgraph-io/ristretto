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

// Ristretto is a fast, fixed size, in-memory cache with a dual focus on
// throughput and hit ratio performance. You can easily add Ristretto to an
// existing system and keep the most valuable data where you need it.
package ristretto

import "sync"

// ClearKey contains an array referencing the keys saved in the Ristretto
// cache and implements Mutex to ensure the array didn't get updated during
// inspection or update. It ensure deadlocks or any crash won't happen.
type ClearKey struct {
	keys map[string]string
	mu sync.RWMutex
}

// NewClearKey generate a new ClearKey object and returns it
func NewClearKey() *ClearKey {
	ck := &ClearKey{}
	ck.keys = make(map[string]string)
	return ck
}

// ListKeys returns the saved keys list to the client.
func (c *ClearKey) ListKeys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	keys := make([]string, 0, len(c.keys))
	for k := range c.keys {
		keys = append(keys, k)
	}
	return keys
}

// AddKey is called to update keys array with non-existing key or replacing
// the existing one.
func (c *ClearKey) AddKey(key string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	c.keys[key] = ""
}

// DelKey is called to delete a key if exists, it does nothing otherwise
func (c *ClearKey) DelKey(key string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	delete(c.keys, key)
}
