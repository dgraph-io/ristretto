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
	bucketSize = 5
)

func timeToBucket(t time.Time) int {
	return t.Second() / bucketSize
}

type bucketMap map[uint64]uint64

type expirationMap struct {
	sync.RWMutex
	m map[int]bucketMap
}

func newExpirationMap() *expirationMap {
	return &expirationMap{
		m: make(map[int]bucketMap),
	}
}

func (m *expirationMap) Add(key, conflict uint64, expiration time.Time) {
	if expiration.IsZero() {
		return
	}

	m.Lock()
	defer m.Unlock()

	bucketNum := timeToBucket(expiration)
	_, ok := m.m[bucketNum]
	if !ok {
		m.m[bucketNum] = make(bucketMap)
	}
	m.m[bucketNum][key] = conflict
}
