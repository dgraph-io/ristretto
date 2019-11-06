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

	"github.com/dgraph-io/ristretto/z"
)

type storeItem struct {
	keyHash uint64
	hashes  []uint64
	value   interface{}
}

// store is the interface fulfilled by all hash map implementations in this
// file. Some hash map implementations are better suited for certain data
// distributions than others, so this allows us to abstract that out for use
// in Ristretto.
//
// Every store is safe for concurrent usage.
type store interface {
	// Get returns the value associated with the key parameter.
	Get(uint64, interface{}) (interface{}, bool)
	// Set adds the key-value pair to the Map or updates the value if it's
	// already present.
	Set(uint64, interface{}, interface{})
	// Del deletes the key-value pair from the Map.
	Del(uint64, interface{})
	// Update attempts to update the key with a new value and returns true if
	// successful.
	Update(uint64, interface{}, interface{}) bool
	// Clear clears all contents of the store.
	Clear()
}

// newStore returns the default store implementation.
func newStore(rounds uint8) store {
	return newShardedMap(rounds)
}

const numShards uint64 = 256

type shardedMap struct {
	shards []*lockedMap
}

func newShardedMap(rounds uint8) *shardedMap {
	sm := &shardedMap{
		shards: make([]*lockedMap, int(numShards)),
	}
	for i := range sm.shards {
		sm.shards[i] = newLockedMap(rounds)
	}
	return sm
}

func (sm *shardedMap) Get(hashed uint64, key interface{}) (interface{}, bool) {
	return sm.shards[hashed%numShards].Get(hashed, key)
}

func (sm *shardedMap) Set(hashed uint64, key, value interface{}) {
	sm.shards[hashed%numShards].Set(hashed, key, value)
}

func (sm *shardedMap) Del(hashed uint64, key interface{}) {
	sm.shards[hashed%numShards].Del(hashed, key)
}

func (sm *shardedMap) Update(hashed uint64, key, value interface{}) bool {
	return sm.shards[hashed%numShards].Update(hashed, key, value)
}

func (sm *shardedMap) Clear() {
	for i := uint64(0); i < numShards; i++ {
		sm.shards[i].Clear()
	}
}

type lockedMap struct {
	sync.RWMutex
	data   map[uint64]storeItem
	rounds uint8
}

func newLockedMap(rounds uint8) *lockedMap {
	return &lockedMap{
		data:   make(map[uint64]storeItem),
		rounds: rounds,
	}
}

func (m *lockedMap) Get(keyHash uint64, key interface{}) (interface{}, bool) {
	m.RLock()
	item, ok := m.data[keyHash]
	m.RUnlock()
	if !ok {
		return nil, false
	}
	if key != nil {
		for i := uint8(1); i < m.rounds; i++ {
			if z.KeyToHash(key, i) != item.hashes[i-1] {
				return nil, false
			}
		}
	}
	return item.value, true
}

func (m *lockedMap) Set(keyHash uint64, key, value interface{}) {
	m.Lock()
	item, ok := m.data[keyHash]
	if !ok {
		hashes := make([]uint64, m.rounds)
		for i := uint8(1); i < m.rounds; i++ {
			hashes[i-1] = z.KeyToHash(key, i)
		}
		m.data[keyHash] = storeItem{
			keyHash: keyHash,
			hashes:  hashes,
			value:   value,
		}
		m.Unlock()
		return
	}
	if key != nil {
		for i := uint8(1); i < m.rounds; i++ {
			if z.KeyToHash(key, i) != item.hashes[i-1] {
				m.Unlock()
				return
			}
		}
	}
	m.data[keyHash] = storeItem{
		keyHash: keyHash,
		hashes:  item.hashes,
		value:   value,
	}
	m.Unlock()
}

func (m *lockedMap) Del(keyHash uint64, key interface{}) {
	m.Lock()
	item, ok := m.data[keyHash]
	if !ok {
		m.Unlock()
		return
	}
	if key != nil {
		for i := uint8(1); i < m.rounds; i++ {
			if z.KeyToHash(key, i) != item.hashes[i-1] {
				m.Unlock()
				return
			}
		}
	}
	delete(m.data, keyHash)
	m.Unlock()
}

func (m *lockedMap) Update(keyHash uint64, key, value interface{}) bool {
	m.Lock()
	item, ok := m.data[keyHash]
	if !ok {
		m.Unlock()
		return false
	}
	if key != nil {
		for i := uint8(1); i < m.rounds; i++ {
			if z.KeyToHash(key, i) != item.hashes[i-1] {
				m.Unlock()
				return false
			}
		}
	}
	m.data[keyHash] = storeItem{
		keyHash: keyHash,
		hashes:  item.hashes,
		value:   value,
	}
	m.Unlock()
	return true
}

func (m *lockedMap) Clear() {
	m.Lock()
	m.data = make(map[uint64]storeItem)
	m.Unlock()
}
