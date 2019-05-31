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

package bench

import "github.com/dgraph-io/ristretto"

type BenchRistretto struct {
	cache ristretto.Cache
}

func NewBenchRistretto(capacity int) *BenchRistretto {
	return &BenchRistretto{
		cache: ristretto.New(capacity),
	}
}

func (c *BenchRistretto) Get(key string) interface{} {
	value, _ := c.cache.Get([]byte(key))
	return value
}

func (c *BenchRistretto) Set(key string, value interface{}) {
	c.cache.Set([]byte(key), value.([]byte))
}

func (c *BenchRistretto) Del(key string) {
	//c.cache.Del(key)
}

func (c *BenchRistretto) Bench() *Stats {
	return nil
}
