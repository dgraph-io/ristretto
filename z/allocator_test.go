/*
 * Copyright 2020 Dgraph Labs, Inc. and Contributors
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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAllocate(t *testing.T) {
	a := NewAllocator(1024)
	defer a.Release()
	require.Equal(t, 0, len(a.Allocate(0)))
	require.Equal(t, 1, len(a.Allocate(1)))
	require.Equal(t, 1<<20, len(a.Allocate(1<<20)))
	require.Equal(t, 256<<20, len(a.Allocate(256<<20)))
	require.Panics(t, func() { a.Allocate(1 << 30) })
}
