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

func verifyHashProduct(t *testing.T, wants, got [2]uint64) {
	if wants[0] != got[0] || wants[1] != got[1] {
		t.Errorf("expected [2]uint64{%d, %d}, but got [2]uint64{%d, %d}\n",
			wants[0], wants[1], got[0], got[1])
	}
}

func TestKeyToHash(t *testing.T) {
	verifyHashProduct(t, [2]uint64{1, 0}, KeyToHash(uint64(1)))
	verifyHashProduct(t, [2]uint64{1, 0}, KeyToHash(1))
	verifyHashProduct(t, [2]uint64{2, 0}, KeyToHash(int32(2)))
	verifyHashProduct(t, [2]uint64{math.MaxUint64 - 1, 0}, KeyToHash(int32(-2)))
	verifyHashProduct(t, [2]uint64{math.MaxUint64 - 1, 0}, KeyToHash(int64(-2)))
	verifyHashProduct(t, [2]uint64{3, 0}, KeyToHash(uint32(3)))
	verifyHashProduct(t, [2]uint64{3, 0}, KeyToHash(int64(3)))
}
