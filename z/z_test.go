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

package z

import (
	"math"
	"testing"
)

func verifyHashProduct(t *testing.T, wants, got uint64) {
	if wants != got {
		t.Errorf("expected hash product to equal %d. Got %d", wants, got)
	}
}

func TestKeyToHash(t *testing.T) {
	verifyHashProduct(t, 1, KeyToHash(uint64(1), 0))
	verifyHashProduct(t, 1, KeyToHash(1, 0))
	verifyHashProduct(t, 2, KeyToHash(int32(2), 0))
	verifyHashProduct(t, math.MaxUint64-1, KeyToHash(int32(-2), 0))
	verifyHashProduct(t, math.MaxUint64-1, KeyToHash(int64(-2), 0))
	verifyHashProduct(t, 3, KeyToHash(uint32(3), 0))
	verifyHashProduct(t, 3, KeyToHash(int64(3), 0))
  last := KeyToHash("data", 0)
	if KeyToHash("data", 1) == last {
		t.Fatal("seed not being used")
	}
}
