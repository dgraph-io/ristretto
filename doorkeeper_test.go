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

	"github.com/dgraph-io/ristretto/z"
)

func TestDoorkeeper(t *testing.T) {
	d := z.NewBloomFilter(float64(1374), 0.01)
	hash := z.MemHashString("*")
	if d.Has(hash) {
		t.Fatal("item exists but was never added")
	}
	if d.AddIfNotHas(hash) != true {
		t.Fatal("item didn't exist so Set() should return true")
	}
	if d.AddIfNotHas(hash) != false {
		t.Fatal("item did exist so Set() should return false")
	}
	if !d.Has(hash) {
		t.Fatal("item was added but Has() is false")
	}
	d.Clear()
	if d.Has(hash) {
		t.Fatal("doorkeeper was reset but Has() returns true")
	}
}
