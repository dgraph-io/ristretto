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
	"fmt"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/dgraph-io/ristretto/bench/sim"
)

func TestCacheLFU(t *testing.T) {
	c := NewCache(&Config{
		CacheSize:  16,
		BufferSize: 16,
		Policy:     NewLFU,
		Log:        true,
	})
	u := sim.Collection(sim.NewZipfian(1.01, 2, 256), 256)
	for i := 0; i < 256; i++ {
		c.Set(fmt.Sprintf("%d", u[i]), i)
	}
	spew.Dump(c.Log())
}

func TestCacheLRU(t *testing.T) {
	c := NewCache(&Config{
		CacheSize:  16,
		BufferSize: 16,
		Policy:     NewLRU,
		Log:        true,
	})
	u := sim.Collection(sim.NewZipfian(1.01, 2, 256), 256)
	for i := 0; i < 256; i++ {
		c.Set(fmt.Sprintf("%d", u[i]), i)
	}
	spew.Dump(c.Log())
}

func TestCacheTinyLFU(t *testing.T) {
	c := NewCache(&Config{
		CacheSize:  16,
		BufferSize: 16,
		Policy:     NewTinyLFU,
		Log:        true,
	})
	u := sim.Collection(sim.NewZipfian(1.01, 2, 256), 256)
	for i := 0; i < 256; i++ {
		c.Set(fmt.Sprintf("%d", u[i]), i)
	}
	spew.Dump(c.Log())
}
