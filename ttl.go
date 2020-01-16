/*
 * Copyright 2020 Dgraph Labs, Inc. and Contributors
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
	"time"
)

const (
	// TODO: find the optimal value or make it configurable.
	bucketSizeSecs = 5
)

func timeToBucket(t time.Time) int {
	return t.Second() / bucketSizeSecs
}

// Map of key to conflict.
type bucket map[uint64]uint64

type expirationMap struct {
	sync.RWMutex
	buckets map[int]bucket
}

func newExpirationMap() *expirationMap {
	return &expirationMap{
		buckets: make(map[int]bucket),
	}
}

func (m *expirationMap) Add(key, conflict uint64, expiration time.Time) {
	if expiration.IsZero() {
		return
	}

	bucketNum := timeToBucket(expiration)
	m.Lock()
	defer m.Unlock()
	_, ok := m.buckets[bucketNum]
	if !ok {
		m.buckets[bucketNum] = make(bucket)
	}
	m.buckets[bucketNum][key] = conflict
}

func (m *expirationMap) Delete(key uint64, expiration time.Time) {
	bucketNum := timeToBucket(expiration)
	m.Lock()
	defer m.Unlock()
	_, ok := m.buckets[bucketNum]
	if !ok {
		return
	}
	delete(m.buckets[bucketNum], key)
}

func (m *expirationMap) CleanUp(store store, policy policy, onEvict onEvictFunc) {
	// Get the bucket number for the current time and substract one. There might be
	// items in the current bucket that have not expired yet but all the items in
	// the previous bucket should have expired.
	bucketNum := timeToBucket(time.Now()) - 1

	m.Lock()
	keys := m.buckets[bucketNum]
	delete(m.buckets, bucketNum)
	m.Unlock()

	for key, conflict := range keys {
		_, value := store.Del(key, 0)
		cost := policy.Cost(key)
		if onEvict != nil {
			onEvict(key, conflict, value, cost)
		}
	}
}
