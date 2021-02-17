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
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/zhangyunhao116/skipmap"
)

// TODO: Do we need this to be a separate struct from Item?
type storeItem struct {
	key        uint64
	conflict   uint64
	value      interface{}
	expiration int64
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
	// Expiration returns the expiration time for this key.
	Expiration(uint64) int64
	// Set adds the key-value pair to the Map or updates the value if it's
	// already present. The key-value pair is passed as a pointer to an
	// item object.
	Set(*Item)
	// Del deletes the key-value pair from the Map.
	Del(uint64, uint64) (uint64, interface{})
	// Update attempts to update the key with a new value and returns true if
	// successful.
	Update(*Item) (interface{}, bool)
	// Cleanup removes items that have an expired TTL.
	Cleanup(policy policy, onEvict itemCallback)
	// Clear clears all contents of the store.
	Clear(onEvict itemCallback)
}

// newStore returns the default store implementation.
func newStore() store {
	return newShardedMap()
}

const numShards uint64 = 256

type shardedMap struct {
	shards    []*lockedMap
	expiryMap *expirationMap
}

func newShardedMap() *shardedMap {
	sm := &shardedMap{
		shards:    make([]*lockedMap, int(numShards)),
		expiryMap: newExpirationMap(),
	}
	for i := range sm.shards {
		sm.shards[i] = newLockedMap(sm.expiryMap)
	}
	return sm
}

func (sm *shardedMap) Get(key, conflict uint64) (interface{}, bool) {
	return sm.shards[key%numShards].get(key, conflict)
}

func (sm *shardedMap) Expiration(key uint64) int64 {
	return sm.shards[key%numShards].Expiration(key)
}

func (sm *shardedMap) Set(i *Item) {
	if i == nil {
		// If item is nil make this Set a no-op.
		return
	}

	sm.shards[i.Key%numShards].Set(i)
}

func (sm *shardedMap) Del(key, conflict uint64) (uint64, interface{}) {
	return sm.shards[key%numShards].Del(key, conflict)
}

func (sm *shardedMap) Update(newItem *Item) (interface{}, bool) {
	return sm.shards[newItem.Key%numShards].Update(newItem)
}

func (sm *shardedMap) Cleanup(policy policy, onEvict itemCallback) {
	sm.expiryMap.cleanup(sm, policy, onEvict)
}

func (sm *shardedMap) Clear(onEvict itemCallback) {
	for i := uint64(0); i < numShards; i++ {
		sm.shards[i].Clear(onEvict)
	}
}

type lockedMap struct {
	_data unsafe.Pointer // *skipmap.Uint64Map
	mu    sync.Mutex

	em *expirationMap
}

func newLockedMap(em *expirationMap) *lockedMap {
	return &lockedMap{
		_data: unsafe.Pointer(skipmap.NewUint64()),
		em:    em,
	}
}

func (m *lockedMap) get(key, conflict uint64) (interface{}, bool) {
	val, ok := m.data().Load(key)
	if !ok {
		return nil, false
	}
	item := val.(storeItem)
	if conflict != 0 && (conflict != item.conflict) {
		return nil, false
	}

	// Handle expired items.
	if item.expiration != 0 && time.Now().Unix() > item.expiration {
		return nil, false
	}
	return item.value, true
}

func (m *lockedMap) Expiration(key uint64) int64 {
	val, _ := m.data().Load(key)
	return val.(storeItem).expiration
}

func (m *lockedMap) Set(i *Item) {
	if i == nil {
		// If the item is nil make this Set a no-op.
		return
	}

	val, ok := m.data().Load(i.Key)

	if ok {
		item := val.(storeItem)
		// The item existed already. We need to check the conflict key and reject the
		// update if they do not match. Only after that the expiration map is updated.
		if i.Conflict != 0 && (i.Conflict != item.conflict) {
			return
		}
		m.em.update(i.Key, i.Conflict, item.expiration, i.Expiration)
	} else {
		// The value is not in the map already. There's no need to return anything.
		// Simply add the expiration map.
		m.em.add(i.Key, i.Conflict, i.Expiration)
	}

	m.data().Store(i.Key, storeItem{
		key:        i.Key,
		conflict:   i.Conflict,
		value:      i.Value,
		expiration: i.Expiration,
	})
}

func (m *lockedMap) Del(key, conflict uint64) (uint64, interface{}) {
	val, ok := m.data().Load(key)
	if !ok {
		return 0, nil
	}
	item := val.(storeItem)
	if conflict != 0 && (conflict != item.conflict) {
		return 0, nil
	}

	if item.expiration != 0 {
		m.em.del(key, item.expiration)
	}

	m.data().Delete(key)
	return item.conflict, item.value
}

func (m *lockedMap) Update(newItem *Item) (interface{}, bool) {
	val, ok := m.data().Load(newItem.Key)
	if !ok {
		return nil, false
	}
	item := val.(storeItem)
	if newItem.Conflict != 0 && (newItem.Conflict != item.conflict) {
		return nil, false
	}

	m.em.update(newItem.Key, newItem.Conflict, item.expiration, newItem.Expiration)
	m.data().Store(newItem.Key, storeItem{
		key:        newItem.Key,
		conflict:   newItem.Conflict,
		value:      newItem.Value,
		expiration: newItem.Expiration,
	})
	return item.value, true
}

func (m *lockedMap) Clear(onEvict itemCallback) {
	i := &Item{}
	if onEvict != nil {
		m.data().Range(func(_ uint64, value interface{}) bool {
			si := value.(storeItem)
			i.Key = si.key
			i.Conflict = si.conflict
			i.Value = si.value
			onEvict(i)
			return true
		})
	}
	m.setdata(skipmap.NewUint64())
}

func (m *lockedMap) data() *skipmap.Uint64Map {
	return (*skipmap.Uint64Map)(atomic.LoadPointer(&m._data))
}

func (m *lockedMap) setdata(mp *skipmap.Uint64Map) {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&m._data)), unsafe.Pointer(mp))
}
