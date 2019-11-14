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
	key      uint64
	conflict uint64
	value    interface{}
}

// store is the interface fulfilled by all hash map implementations in this
// file. Some hash map implementations are better suited for certain data
// distributions than others, so this allows us to abstract that out for use
// in Ristretto.
//
// Every store is safe for concurrent usage.
type store interface {
	// Get returns the value associated with the key parameter.
	Get(uint64, uint64) (interface{}, bool)
	// Set adds the key-value pair to the Map or updates the value if it's
	// already present.
	Set(uint64, uint64, interface{})
	// Del deletes the key-value pair from the Map.
	Del(uint64, uint64) (uint64, interface{})
	// Update attempts to update the key with a new value and returns true if
	// successful.
	Update(uint64, uint64, interface{}) bool
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

func (sm *shardedMap) Get(key, conflict uint64) (interface{}, bool) {
	return sm.shards[key%numShards].Get(key, conflict)
}

func (sm *shardedMap) Set(key, conflict uint64, value interface{}) {
	sm.shards[key%numShards].Set(key, conflict, value)
}

func (sm *shardedMap) Del(key, conflict uint64) (uint64, interface{}) {
	return sm.shards[key%numShards].Del(key, conflict)
}

func (sm *shardedMap) Update(key, conflict uint64, value interface{}) bool {
	return sm.shards[key%numShards].Update(key, conflict, value)
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

func (m *lockedMap) Get(key, conflict uint64) (interface{}, bool) {
	m.RLock()
	item, ok := m.data[key]
	m.RUnlock()
	if !ok {
		return nil, false
	}
	if conflict != 0 && (conflict != item.conflict) {
		return nil, false
	}
	return item.value, true
}

func (m *lockedMap) Set(key, conflict uint64, value interface{}) {
	m.Lock()
	item, ok := m.data[key]
	if !ok {
		m.data[key] = storeItem{
			key:      key,
			conflict: conflict,
			value:    value,
		}
		m.Unlock()
		return
	}
	if conflict != 0 && (conflict != item.conflict) {
		m.Unlock()
		return
	}
	m.data[key] = storeItem{
		key:      key,
		conflict: conflict,
		value:    value,
	}
	m.Unlock()
}

func (m *lockedMap) Del(key, conflict uint64) (uint64, interface{}) {
	m.Lock()
	item, ok := m.data[key]
	if !ok {
		m.Unlock()
		return 0, nil
	}
	if conflict != 0 && (conflict != item.conflict) {
		m.Unlock()
		return 0, nil
	}
	delete(m.data, key)
	m.Unlock()
	return item.conflict, item.value
}

func (m *lockedMap) Update(key, conflict uint64, value interface{}) bool {
	m.Lock()
	item, ok := m.data[key]
	if !ok {
		m.Unlock()
		return false
	}
	if conflict != 0 && (conflict != item.conflict) {
		m.Unlock()
		return false
	}
	m.data[key] = storeItem{
		key:      key,
		conflict: conflict,
		value:    value,
	}
	m.Unlock()
	return true
}

func (m *lockedMap) Clear() {
	m.Lock()
	m.data = make(map[uint64]storeItem)
	m.Unlock()
}
