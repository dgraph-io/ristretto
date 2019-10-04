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
	"sync"
)

type storeHash struct {
	primary   uint64
	secondary uint64
}

type storeItem struct {
	key   storeHash
	value interface{}
}

var emptyStoreItem = storeItem{key: storeHash{0, 0}, value: nil}

// store is the interface fulfilled by all hash map implementations in this
// file. Some hash map implementations are better suited for certain data
// distributions than others, so this allows us to abstract that out for use
// in Ristretto.
//
// Every store is safe for concurrent usage.
type store interface {
	// Get returns the value associated with the key parameter.
	Get(storeHash) (interface{}, bool)
	// Set adds the key-value pair to the Map or updates the value if it's
	// already present.
	Set(storeHash, interface{})
	// Del deletes the key-value pair from the Map.
	Del(storeHash)
	// Update attempts to update the key with a new value and returns true if
	// successful.
	Update(storeHash, interface{}) bool
	// Clear clears all contents of the store.
	Clear()
}

// newStore returns the default store implementation.
func newStore() store {
	return newShardedMap()
}

const numShards uint64 = 256

type shardedMap struct {
	shards []*lockedMap
}

func newShardedMap() *shardedMap {
	sm := &shardedMap{shards: make([]*lockedMap, int(numShards))}
	for i := range sm.shards {
		sm.shards[i] = newLockedMap()
	}
	return sm
}

func (sm *shardedMap) Get(key storeHash) (interface{}, bool) {
	idx := key.primary % numShards
	return sm.shards[idx].Get(key)
}

func (sm *shardedMap) Set(key storeHash, value interface{}) {
	idx := key.primary % numShards
	sm.shards[idx].Set(key, value)
}

func (sm *shardedMap) Del(key storeHash) {
	idx := key.primary % numShards
	sm.shards[idx].Del(key)
}

func (sm *shardedMap) Update(key storeHash, value interface{}) bool {
	idx := key.primary % numShards
	return sm.shards[idx].Update(key, value)
}

func (sm *shardedMap) Clear() {
	for i := uint64(0); i < numShards; i++ {
		sm.shards[i].Clear()
	}
}

type lockedMap struct {
	sync.RWMutex
	data map[uint64]storeItem
}

func newLockedMap() *lockedMap {
	return &lockedMap{
		data: make(map[uint64]storeItem),
	}
}

func (m *lockedMap) Get(key storeHash) (interface{}, bool) {
	m.RLock()
	item, found := m.data[key.primary]
	m.RUnlock()
	if item == emptyStoreItem || !found {
		return nil, false
	}
	if item.key.secondary == key.secondary {
		return item.value, true
	}
	// TODO: log collision
	return nil, false
}

func (m *lockedMap) Set(key storeHash, value interface{}) {
	m.Lock()
	if item, ok := m.data[key.primary]; ok {
		if item.key.secondary != key.secondary {
			// TODO: log collision
			return
		}
	}
	m.data[key.primary] = storeItem{key, value}
	m.Unlock()
}

func (m *lockedMap) Del(key storeHash) {
	m.Lock()
	if item, ok := m.data[key.primary]; ok {
		if item.key.secondary != key.secondary {
			// TODO: log collision
			return
		}
	}
	delete(m.data, key.primary)
	m.Unlock()
}

func (m *lockedMap) Update(key storeHash, value interface{}) bool {
	m.Lock()
	defer m.Unlock()
	if item, ok := m.data[key.primary]; ok {
		if item.key.secondary != key.secondary {
			// TODO: log collision
			return false
		}
		m.data[key.primary] = storeItem{key, value}
		return true
	}
	return false
}

func (m *lockedMap) Clear() {
	m.Lock()
	m.data = make(map[uint64]storeItem)
	m.Unlock()
}
