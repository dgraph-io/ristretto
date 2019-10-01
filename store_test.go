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
	GenerateTest(newStore)(t)
}

func TestStoreLockedMap(t *testing.T) {
	GenerateTest(func() store { return newLockedMap() })(t)
}

func GenerateTest(create func() store) func(*testing.T) {
	return func(t *testing.T) {
		t.Run("set/get", func(t *testing.T) {
			m := create()
			m.Set(1, 1)
			if val, _ := m.Get(1); val != nil && val.(int) != 1 {
				t.Fatal("set-get error")
			}
		})
		t.Run("set", func(t *testing.T) {
			m := create()
			m.Set(1, 1)
			// overwrite
			m.Set(1, 2)
			if val, _ := m.Get(1); val != nil && val.(int) != 2 {
				t.Fatal("set update error")
			}
		})
		t.Run("del", func(t *testing.T) {
			m := create()
			m.Set(1, 1)
			// delete item
			m.Del(1)
			if val, found := m.Get(1); val != nil || found {
				t.Fatal("del error")
			}
		})
		t.Run("clear", func(t *testing.T) {
			m := create()
			// set a lot of values
			for i := uint64(0); i < 1000; i++ {
				m.Set(i, i)
			}
			// clear
			m.Clear()
			// check if any of the values exist
			for i := uint64(0); i < 1000; i++ {
				if _, ok := m.Get(i); ok {
					t.Fatal("clear operation failed")
				}
      }
    })
		t.Run("update", func(t *testing.T) {
			m := create()
			m.Set(1, 1)
			if updated := m.Update(1, 2); !updated {
				t.Fatal("value should have been updated")
			}
			if val, _ := m.Get(1); val.(int) != 2 {
				t.Fatal("value wasn't updated")
			}
			if updated := m.Update(2, 2); updated {
				t.Fatal("value should not have been updated")
			}
			if val, found := m.Get(2); val != nil || found {
				t.Fatal("value should not have been updated")
			}
		})
	}
}
