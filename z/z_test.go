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

func verifyHashProduct(t *testing.T, wantKey, wantConflict, key, conflict uint64) {
	if wantKey != key || wantConflict != conflict {
		t.Errorf("expected (%d, %d) but got (%d, %d)\n",
			wantKey, wantConflict, key, conflict)
	}
}

func TestKeyToHash(t *testing.T) {
	var key uint64
	var conflict uint64

	key, conflict = KeyToHash(uint64(1))
	verifyHashProduct(t, 1, 0, key, conflict)

	key, conflict = KeyToHash(1)
	verifyHashProduct(t, 1, 0, key, conflict)

	key, conflict = KeyToHash(int32(2))
	verifyHashProduct(t, 2, 0, key, conflict)

	key, conflict = KeyToHash(int32(-2))
	verifyHashProduct(t, math.MaxUint64-1, 0, key, conflict)

	key, conflict = KeyToHash(int64(-2))
	verifyHashProduct(t, math.MaxUint64-1, 0, key, conflict)

	key, conflict = KeyToHash(uint32(3))
	verifyHashProduct(t, 3, 0, key, conflict)

	key, conflict = KeyToHash(int64(3))
	verifyHashProduct(t, 3, 0, key, conflict)
}
