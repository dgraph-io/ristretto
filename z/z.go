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
	"github.com/cespare/xxhash"
)

func KeyToHash(key interface{}) [2]uint64 {
	if key == nil {
		return [2]uint64{0, 0}
	}
	switch k := key.(type) {
	case uint64:
		return [2]uint64{k, 0}
	case string:
		raw := []byte(k)
		return [2]uint64{MemHash(raw), xxhash.Sum64(raw)}
	case []byte:
		return [2]uint64{MemHash(k), xxhash.Sum64(k)}
	case byte:
		return [2]uint64{uint64(k), 0}
	case int:
		return [2]uint64{uint64(k), 0}
	case int32:
		return [2]uint64{uint64(k), 0}
	case uint32:
		return [2]uint64{uint64(k), 0}
	case int64:
		return [2]uint64{uint64(k), 0}
	default:
		panic("Key type not supported")
	}
}
