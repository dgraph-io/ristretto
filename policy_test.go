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

func TestPolicy(t *testing.T) {
	t.Run("uniform-push", func(t *testing.T) {
		policy := newPolicy(1024, 1024)
		values := make([]uint64, 1024)
		for i := range values {
			values[i] = uint64(i)
		}
		policy.Add(0, 1)
		policy.Push(values)
		if !policy.Has(0) || policy.Has(999999) {
			t.Fatal("add/push error")
		}
	})
	t.Run("uniform-add", func(t *testing.T) {
		policy := newPolicy(1024, 1024)
		for i := int64(0); i < 1024; i++ {
			policy.Add(uint64(i), 1)
		}
		if victims, added := policy.Add(999999, 1); victims == nil || !added {
			t.Fatal("add/eviction error")
		}
	})
	t.Run("variable-push", func(t *testing.T) {
		policy := newPolicy(1024, 1024*4)
		values := make([]uint64, 1024)
		for i := range values {
			values[i] = uint64(i)
		}
		policy.Add(0, 1)
		policy.Push(values)
		if !policy.Has(0) || policy.Has(999999) {
			t.Fatal("add/push error")
		}
	})
	t.Run("variable-add", func(t *testing.T) {
		policy := newPolicy(1024, 1024*4)
		for i := int64(0); i < 1024; i++ {
			policy.Add(uint64(i), 4)
		}
		if victims, added := policy.Add(999999, 1); victims == nil || !added {
			t.Fatal("add/eviction error")
		}
	})
}
