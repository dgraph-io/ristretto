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
)

// store is the interface fulfilled by all hash map implementations in this
// file. Some hash map implementations are better suited for certain data
// distributions than others, so this allows us to abstract that out for use
// in Ristretto.
//
// Every store is safe for concurrent usage.
type store interface {
	// Get returns the value associated with the key parameter.
	Get(uint64) (interface{}, bool)
	// Set adds the key-value pair to the Map or updates the value if it's
	// already present.
	Set(uint64, interface{})
	// Del deletes the key-value pair from the Map.
	Del(uint64)
	// Run applies the function parameter to random key-value pairs. No key
	// will be visited more than once. If the function returns false, the
	// iteration stops. If the function returns true, the iteration will
	// continue until every key has been visited once.
	Run(func(interface{}, interface{}) bool)
}

// newStore returns the default store implementation.
func newStore() store {
	return newSyncMap()
}

type syncMap struct {
	*sync.Map
}

func newSyncMap() store {
	return &syncMap{&sync.Map{}}
}

func (m *syncMap) Get(key uint64) (interface{}, bool) {
	return m.Load(key)
}

func (m *syncMap) Set(key uint64, value interface{}) {
	m.Store(key, value)
}

func (m *syncMap) Del(key uint64) {
	m.Delete(key)
}

func (m *syncMap) Run(f func(key, value interface{}) bool) {
	m.Range(f)
}

type lockedMap struct {
	sync.RWMutex
	data map[uint64]interface{}
}

func newLockedMap() *lockedMap {
	return &lockedMap{data: make(map[uint64]interface{})}
}

func (m *lockedMap) Get(key uint64) interface{} {
	m.RLock()
	defer m.RUnlock()
	return m.data[key]
}

func (m *lockedMap) Set(key uint64, value interface{}) {
	m.Lock()
	defer m.Unlock()
	m.data[key] = value
}

func (m *lockedMap) Del(key uint64) {
	m.Lock()
	defer m.Unlock()
	delete(m.data, key)
}

func (m *lockedMap) Run(f func(interface{}, interface{}) bool) {
	m.RLock()
	defer m.RUnlock()
	for k, v := range m.data {
		if !f(k, v) {
			return
		}
	}
}

// lazy map

type MyTryLock struct {
	reader int32
	state  int32 // 0 - unlocked, 1 - locked
}

func (t *MyTryLock) RLock() {
	// check for any active lock
	for {
		// starving can be improved by the runtime semaphore thingy
		// need to study more about it.
		if atomic.LoadInt32(&t.state) == int32(0) {
			atomic.AddInt32(&t.reader, 1)
			return
		}
	}
}

func (t *MyTryLock) RUnLock() {
	atomic.AddInt32(&t.reader, -1)
}

func (t *MyTryLock) TryLock() bool {
	if atomic.LoadInt32(&t.reader) == 0 {
		ok := atomic.CompareAndSwapInt32(&t.state, 0, 1)
		return ok
	}
	return false
}

func (t *MyTryLock) UnLock() {
	if !atomic.CompareAndSwapInt32(&t.state, 1, 0) {
		panic("race")
	}
}

type kv struct {
	key uint64
	val interface{}
}

var kvPool = sync.Pool{
	New: func() interface{} {
		return new(kv)
	},
}

type LazyMap struct {
	inner map[uint64]interface{}
	buf   chan *kv
	del   chan uint64
	sync.RWMutex
}

func NewLazyMap() *LazyMap {
	return &LazyMap{
		inner: make(map[uint64]interface{}, 100),
		buf:   make(chan *kv, 10000000),
		del:   make(chan uint64, 10000000),
	}
}

func (l *LazyMap) Set(k uint64, val interface{}) {
	if l.TryLock() {
		defer l.Unlock()
		l.inner[k] = val
		return
	}
	kv := kvPool.Get().(*kv)
	kv.key = k
	kv.val = val
	l.buf <- kv
}

func (l *LazyMap) Get(k uint64) (interface{}, bool) {
	if (len(l.buf) > 0 || len(l.del) > 0) && l.TryLock() {
	del:
		for {
			select {
			case k := <-l.del:
				delete(l.inner, k)
			default:
				break del
			}
		}
		for {
			select {
			case kv := <-l.buf:
				l.inner[kv.key] = kv.val
				kvPool.Put(kv)
			default:
				v, ok := l.inner[k]
				l.Unlock()
				return v, ok
			}
		}
	}
	l.RLock()
	v, ok := l.inner[k]
	l.RUnlock()
	return v, ok
}

func (l *LazyMap) Del(k uint64) {
	select {
	case l.del <- k:
	default:
	}

}

func (l *LazyMap) Run(f func(interface{}, interface{}) bool) {
	l.RLock()
	defer l.RUnlock()
	for k, v := range l.inner {
		if !f(k, v) {
			return
		}
	}
}
