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
	"hash/maphash"
)

// We need two seeds, one for each hash value returned by
// KeyToHash.
var (
	seed1 = maphash.MakeSeed()
	seed2 = maphash.MakeSeed()
)

// KeyToHash returns two hashes of the given key. It's the default KeyToHash
// function used by ristretto. It accepts string, bytes and all integer types except uintptr.
func KeyToHash(key interface{}) (uint64, uint64) {
	if key == nil {
		return 0, 0
	}
	switch k := key.(type) {
	case uint64:
		return k, 0
	case byte:
		return uint64(k), 0
	case int:
		return uint64(k), 0
	case int32:
		return uint64(k), 0
	case uint32:
		return uint64(k), 0
	case int64:
		return uint64(k), 0
	case string:
		return memHashString(seed1, k), memHashString(seed2, k)
	case []byte:
		return memHash(seed1, k), memHash(seed2, k)
	default:
		panic("Key type not supported")
	}
}

func memHash(seed maphash.Seed, data []byte) uint64 {
	var h maphash.Hash
	h.SetSeed(seed)
	h.Write(data)
	return h.Sum64()
}

func memHashString(seed maphash.Seed, str string) uint64 {
	var h maphash.Hash
	h.SetSeed(seed)
	h.WriteString(str)
	return h.Sum64()
}
