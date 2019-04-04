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
	"errors"
	"sync"

	"github.com/dgraph-io/ristretto/lru"
)

//Cache interface
type Cache interface {
	Get(key []byte) ([]byte, error)
	Set(key []byte, value []byte) error
}

//BasicCache implementation
type BasicCache struct {
	c   *lru.Cache
	mux sync.Mutex
}

//Get a value from cache
func (r *BasicCache) Get(key []byte) ([]byte, error) {
	r.mux.Lock()
	defer r.mux.Unlock()
	v, ok := r.c.Get(string(key)) //
	if ok {
		return v.([]byte), nil
	}
	return nil, errors.New("key not found")
}

//Set a value in cache for a key
func (r *BasicCache) Set(key, value []byte) error {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.c.Add(string(key), value)
	return nil
}

//New a BasicCache
func New(size int) *BasicCache {
	return &BasicCache{c: lru.New(size)}
}
