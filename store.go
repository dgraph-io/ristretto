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

type storeItem struct {
	hashes [2]uint64
	value  interface{}
}

// store is the interface fulfilled by all hash map implementations in this
// file. Some hash map implementations are better suited for certain data
// distributions than others, so this allows us to abstract that out for use
// in Ristretto.
//
// Every store is safe for concurrent usage.
type store interface {
	// Get returns the value associated with the key parameter.
	Get([2]uint64) (interface{}, bool)
	// Set adds the key-value pair to the Map or updates the value if it's
	// already present.
	Set([2]uint64, interface{})
	// Del deletes the key-value pair from the Map.
	Del([2]uint64)
	// Update attempts to update the key with a new value and returns true if
	// successful.
	Update([2]uint64, interface{}) bool
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
	sm := &shardedMap{
		shards: make([]*lockedMap, int(numShards)),
	}
	for i := range sm.shards {
		sm.shards[i] = newLockedMap()
	}
	return sm
}

func (sm *shardedMap) Get(hashes [2]uint64) (interface{}, bool) {
	return sm.shards[hashes[0]%numShards].Get(hashes)
}

func (sm *shardedMap) Set(hashes [2]uint64, value interface{}) {
	sm.shards[hashes[0]%numShards].Set(hashes, value)
}

func (sm *shardedMap) Del(hashes [2]uint64) {
	sm.shards[hashes[0]%numShards].Del(hashes)
}

func (sm *shardedMap) Update(hashes [2]uint64, value interface{}) bool {
	return sm.shards[hashes[0]%numShards].Update(hashes, value)
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

func (m *lockedMap) Get(hashes [2]uint64) (interface{}, bool) {
	m.RLock()
	item, ok := m.data[hashes[0]]
	m.RUnlock()
	if !ok {
		return nil, false
	}
	if (item.hashes[1] != hashes[1]) && hashes[1] != 0 {
		return nil, false
	}
	return item.value, true
}

func (m *lockedMap) Set(hashes [2]uint64, value interface{}) {
	m.Lock()
	item, ok := m.data[hashes[0]]
	if !ok {
		m.data[hashes[0]] = storeItem{
			hashes: hashes,
			value:  value,
		}
		m.Unlock()
		return
	}
	if (item.hashes[1] != hashes[1]) && hashes[1] != 0 {
		m.Unlock()
		return
	}
	m.data[hashes[0]] = storeItem{
		hashes: item.hashes,
		value:  value,
	}
	m.Unlock()
}

func (m *lockedMap) Del(hashes [2]uint64) {
	m.Lock()
	item, ok := m.data[hashes[0]]
	if !ok {
		m.Unlock()
		return
	}
	if (item.hashes[1] != hashes[1]) && hashes[1] != 0 {
		m.Unlock()
		return
	}
	delete(m.data, hashes[0])
	m.Unlock()
}

func (m *lockedMap) Update(hashes [2]uint64, value interface{}) bool {
	m.Lock()
	item, ok := m.data[hashes[0]]
	if !ok {
		m.Unlock()
		return false
	}
	if (item.hashes[1] != hashes[1]) && hashes[1] != 0 {
		m.Unlock()
		return false
	}
	m.data[hashes[0]] = storeItem{
		hashes: item.hashes,
		value:  value,
	}
	m.Unlock()
	return true
}

func (m *lockedMap) Clear() {
	m.Lock()
	m.data = make(map[uint64]storeItem)
	m.Unlock()
}
