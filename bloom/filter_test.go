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

package bloom

import (
	"testing"
)

func TestFilter(t *testing.T) {
	f := NewFilter(1374, 0.01)
	if f.keys != 7 || len(f.data)*8 < 13170 {
		t.Fatal("bad initialization based on size and false positive rate")
	}
	if f.Has("*") {
		t.Fatal("item exists but was never added")
	}
	if f.Set("*") != true {
		t.Fatal("item didn't exist so Set() should return true")
	}
	if f.Set("*") != false {
		t.Fatal("item did exist so Set() should return false")
	}
	if !f.Has("*") {
		t.Fatal("item was added but Has() is false")
	}
	f.Reset()
	if f.Has("*") {
		t.Fatal("filter was reset but Has() returns true")
	}
}
