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

	"github.com/stretchr/testify/require"
)

func TestKeyToHash(t *testing.T) {
	require.Equal(t, uint64(1), KeyToHash(uint64(1)))
	require.Equal(t, uint64(1), KeyToHash(1))
	require.Equal(t, uint64(2), KeyToHash(int32(2)))
	require.Equal(t, uint64(math.MaxUint64)-1, KeyToHash(int32(-2)))
	require.Equal(t, uint64(math.MaxUint64)-1, KeyToHash(int64(-2)))
	require.Equal(t, uint64(3), KeyToHash(uint32(3)))
	require.Equal(t, uint64(3), KeyToHash(int64(3)))
}
