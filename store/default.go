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

package store

import "sync"

type Default struct {
	*sync.Map
}

func NewDefault() Map {
	return &Default{&sync.Map{}}
}

func (m *Default) Get(key string) interface{} {
	value, _ := m.Load(key)
	return value
}

func (m *Default) Set(key string, value interface{}) {
	m.Store(key, value)
}

func (m *Default) Del(key string) {
	m.Delete(key)
}

func (m *Default) Run(f func(key, value interface{}) bool) {
	m.Range(f)
}
