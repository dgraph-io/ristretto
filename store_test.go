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
	"testing"
)

func BenchmarkStoreSyncMap(b *testing.B) {
	GenerateBench(func() store { return newSyncMap() })(b)
}

func BenchmarkStoreLockedMap(b *testing.B) {
	GenerateBench(func() store { return newLockedMap() })(b)
}

func GenerateBench(create func() store) func(*testing.B) {
	return func(b *testing.B) {
		b.Run("get  ", func(b *testing.B) {
			m := create()
			m.Set(1, 1)
			b.SetBytes(1)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					m.Get(1)
				}
			})
		})
	}
}

func TestStore(t *testing.T) {
	GenerateTest(func() store { return newStore() })(t)
}

func TestStoreSyncMap(t *testing.T) {
	GenerateTest(func() store { return newSyncMap() })(t)
}

func TestStoreLockedMap(t *testing.T) {
	GenerateTest(func() store { return newLockedMap() })(t)
}

func GenerateTest(create func() store) func(*testing.T) {
	return func(t *testing.T) {
		t.Run("set/get", func(t *testing.T) {
			m := create()
			m.Set(1, 1)
			if m.Get(1).(int) != 1 {
				t.Fatal("set-get error")
			}
		})
		t.Run("set", func(t *testing.T) {
			m := create()
			m.Set(1, 1)
			// overwrite
			m.Set(1, 2)
			if m.Get(1).(int) != 2 {
				t.Fatal("set update error")
			}
		})
		t.Run("del", func(t *testing.T) {
			m := create()
			m.Set(1, 1)
			// delete item
			m.Del(1)
			if m.Get(1) != nil {
				t.Fatal("del error")
			}
		})
		t.Run("run", func(t *testing.T) {
			m := create()
			n := 10
			// populate with incrementing ints
			for i := 0; i < n; i++ {
				m.Set(uint64(i), i)
			}
			// will hold items collected during Run
			check := make(map[uint64]struct{})
			// iterate over items
			m.Run(func(key, value interface{}) bool {
				check[key.(uint64)] = struct{}{}
				// go until no more items
				return true
			})
			if len(check) != n {
				t.Fatal("run not iterating over all elements")
			}
			// check stopping run iteration
			i := 0
			// iterate 3 times
			m.Run(func(key, value interface{}) bool {
				i++
				return !(i == 3)
			})
			if i != 3 {
				println(i)
				t.Fatal("run not checking return bool")
			}
		})
	}
}
