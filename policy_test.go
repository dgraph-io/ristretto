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
	"fmt"
	"testing"

	"github.com/dgraph-io/ristretto/ring"
	"github.com/dgraph-io/ristretto/store"
)

type PolicyCreator func(uint64, store.Map) Policy

func GeneratePolicyTest(create PolicyCreator) func(*testing.T) {
	iterations := uint64(4)
	return func(t *testing.T) {
		t.Run("push", func(t *testing.T) {
			data := store.NewMap()
			policy := create(iterations, data)
			values := make([]ring.Element, iterations)
			for i := range values {
				values[i] = ring.Element(fmt.Sprintf("%d", i))
			}
			data.Set("0", 1)
			policy.Add("0")
			policy.Push(values)
			if !policy.Has("0") || policy.Has("*") {
				t.Fatal("add/push error")
			}
		})
		t.Run("add", func(t *testing.T) {
			data := store.NewMap()
			policy := create(iterations, data)
			for i := uint64(0); i < iterations; i++ {
				data.Set(fmt.Sprintf("%d", i), i)
				policy.Add(fmt.Sprintf("%d", i))
			}
			if victim, added := policy.Add("*"); victim == "" || !added {
				fmt.Println(victim, added)
				t.Fatal("add/eviction error")
			}
		})
	}
}

func TestPolicy(t *testing.T) {
	policies := []PolicyCreator{NewLFU, NewLRU, NewTinyLFU}
	for _, policy := range policies {
		GeneratePolicyTest(policy)(t)
	}
}
