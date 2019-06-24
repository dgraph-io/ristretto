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

	"github.com/dgraph-io/ristretto/ring"
	"github.com/dgraph-io/ristretto/store"
)

func TestTinyLFU(t *testing.T) {
	t.Run("push", func(t *testing.T) {
		m := store.NewMap()
		p := NewTinyLFU(16, m)
		m.Set("1", nil)
		p.Push([]ring.Element{"1", "1", "1"})
		if p.sketch.Estimate("1") != 3 {
			t.Fatal("push error")
		}
	})
	t.Run("add", func(t *testing.T) {
		c := uint64(16)
		// tinylfu counters need a map for eviction
		m := store.NewMap()
		p := NewTinyLFU(c, m)
		// fill it up
		for i := uint64(0); i < c; i++ {
			k := fmt.Sprintf("%d", i)
			// need to add it to the map as well because that's how eviction is
			// done
			m.Set(k, nil)
			p.Add(k)
		}
		if victim, _ := p.Add("16"); victim == "" {
			t.Fatal("eviction error")
		}
	})
}

////////////////////////////////////////////////////////////////////////////////

func TestLRU(t *testing.T) {
	t.Run("push", func(t *testing.T) {
		p := NewLRU(4)
		p.Add("1")
		p.Add("3")
		p.Add("2")
		p.Push([]ring.Element{"1", "3", "1"})
		if p.String() != "[1, 3, 2]" {
			t.Fatal("push order error")
		}
	})
	t.Run("add", func(t *testing.T) {
		p := NewLRU(4)
		p.Add("1")
		p.Add("2")
		p.Add("3")
		p.Add("4")
		p.Push([]ring.Element{"1", "3"})
		victim, added := p.Add("5")
		if added && victim != "2" {
			t.Fatal("eviction error")
		}
	})
}

////////////////////////////////////////////////////////////////////////////////

func TestLFU(t *testing.T) {
	t.Run("push", func(t *testing.T) {
		p := NewLFU(&Config{4, 4, true})
		p.Add("1")
		p.Push([]ring.Element{"1", "1", "1"})
		if p.data["1"] != 4 {
			t.Fatal("push error")
		}
	})
	t.Run("add", func(t *testing.T) {
		p := NewLFU(&Config{4, 4, true})
		p.Add("1")
		p.Add("2")
		p.Add("3")
		p.Add("4")
		p.Push([]ring.Element{
			"1", "1", "1", "1",
			"2", "2", "2",
			"3",
			"4", "4",
		})
		victim, added := p.Add("5")
		if added && victim != "3" {
			t.Fatal("eviction error")
		}
	})
}

func BenchmarkLFU(b *testing.B) {
	k := "1"
	data := []ring.Element{"1", "1"}
	b.Run("single", func(b *testing.B) {
		p := NewLFU(&Config{1000000, 1000000, true})
		p.Add(k)
		b.SetBytes(1)
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			p.hit(k)
		}
	})
	b.Run("parallel", func(b *testing.B) {
		p := NewLFU(&Config{1000000, 1000000, true})
		p.Add(k)
		b.SetBytes(1)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				p.Push(data)
			}
		})
	})
}

////////////////////////////////////////////////////////////////////////////////

func TestClairvoyant(t *testing.T) {
	p := NewClairvoyant(5)
	p.Push([]ring.Element{
		"1", "2", "3", "4", "5", "6", "3", "9",
		"4", "3", "1", "7", "8", "9", "5", "3",
		"5", "7",
	})
	l := p.Log()
	if l.GetHits() != 9 || l.GetHits()+l.GetMisses() != 18 || l.GetEvictions() != 4 {
		t.Fatal("log error")
	}
}
