/*
 * SPDX-FileCopyrightText: © Hypermode Inc. <hello@hypermode.com>
 * SPDX-License-Identifier: Apache-2.0
 */

package ristretto

import (
	"testing"
	"time"

	"github.com/dgraph-io/ristretto/v2/z"
	"github.com/stretchr/testify/require"
)

func TestStoreSetGet(t *testing.T) {
	s := newStore[int]()
	key, conflict := z.KeyToHash(1)
	i := Item[int]{
		Key:      key,
		Conflict: conflict,
		Value:    2,
	}
	s.Set(&i)
	val, ok := s.Get(key, conflict)
	require.True(t, ok)
	require.Equal(t, 2, val)

	i.Value = 3
	s.Set(&i)
	val, ok = s.Get(key, conflict)
	require.True(t, ok)
	require.Equal(t, 3, val)

	key, conflict = z.KeyToHash(2)
	i = Item[int]{
		Key:      key,
		Conflict: conflict,
		Value:    2,
	}
	s.Set(&i)
	val, ok = s.Get(key, conflict)
	require.True(t, ok)
	require.Equal(t, 2, val)
}

func TestStoreDel(t *testing.T) {
	s := newStore[int]()
	key, conflict := z.KeyToHash(1)
	i := Item[int]{
		Key:      key,
		Conflict: conflict,
		Value:    1,
	}
	s.Set(&i)
	s.Del(key, conflict)
	val, ok := s.Get(key, conflict)
	require.False(t, ok)
	require.Empty(t, val)

	s.Del(2, 0)
}

func TestStoreClear(t *testing.T) {
	s := newStore[uint64]()
	for i := uint64(0); i < 1000; i++ {
		key, conflict := z.KeyToHash(i)
		it := Item[uint64]{
			Key:      key,
			Conflict: conflict,
			Value:    i,
		}
		s.Set(&it)
	}
	s.Clear(nil)
	for i := uint64(0); i < 1000; i++ {
		key, conflict := z.KeyToHash(i)
		val, ok := s.Get(key, conflict)
		require.False(t, ok)
		require.Empty(t, val)
	}
}

func TestShouldUpdate(t *testing.T) {
	// Create a should update function where the value only increases.
	s := newStore[int]()
	s.SetShouldUpdateFn(func(cur, prev int) bool {
		return cur > prev
	})

	key, conflict := z.KeyToHash(1)
	i := Item[int]{
		Key:      key,
		Conflict: conflict,
		Value:    2,
	}
	s.Set(&i)
	i.Value = 1
	_, ok := s.Update(&i)
	require.False(t, ok)

	i.Value = 3
	_, ok = s.Update(&i)
	require.True(t, ok)
}

func TestStoreUpdate(t *testing.T) {
	s := newStore[int]()
	key, conflict := z.KeyToHash(1)
	i := Item[int]{
		Key:      key,
		Conflict: conflict,
		Value:    1,
	}
	s.Set(&i)
	i.Value = 2
	_, ok := s.Update(&i)
	require.True(t, ok)

	val, ok := s.Get(key, conflict)
	require.True(t, ok)
	require.NotNil(t, val)

	val, ok = s.Get(key, conflict)
	require.True(t, ok)
	require.Equal(t, 2, val)

	i.Value = 3
	_, ok = s.Update(&i)
	require.True(t, ok)

	val, ok = s.Get(key, conflict)
	require.True(t, ok)
	require.Equal(t, 3, val)

	key, conflict = z.KeyToHash(2)
	i = Item[int]{
		Key:      key,
		Conflict: conflict,
		Value:    2,
	}
	_, ok = s.Update(&i)
	require.False(t, ok)
	val, ok = s.Get(key, conflict)
	require.False(t, ok)
	require.Empty(t, val)
}

func TestStoreCollision(t *testing.T) {
	s := newShardedMap[int]()
	s.shards[1].Lock()
	s.shards[1].data[1] = storeItem[int]{
		key:      1,
		conflict: 0,
		value:    1,
	}
	s.shards[1].Unlock()
	val, ok := s.Get(1, 1)
	require.False(t, ok)
	require.Empty(t, val)

	i := Item[int]{
		Key:      1,
		Conflict: 1,
		Value:    2,
	}
	s.Set(&i)
	val, ok = s.Get(1, 0)
	require.True(t, ok)
	require.NotEqual(t, 2, val)

	_, ok = s.Update(&i)
	require.False(t, ok)
	val, ok = s.Get(1, 0)
	require.True(t, ok)
	require.NotEqual(t, 2, val)

	s.Del(1, 1)
	val, ok = s.Get(1, 0)
	require.True(t, ok)
	require.NotEmpty(t, val)
}

func TestStoreExpiration(t *testing.T) {
	s := newStore[int]()
	key, conflict := z.KeyToHash(1)
	expiration := time.Now().Add(time.Second)
	i := Item[int]{
		Key:        key,
		Conflict:   conflict,
		Value:      1,
		Expiration: expiration,
	}
	s.Set(&i)
	val, ok := s.Get(key, conflict)
	require.True(t, ok)
	require.Equal(t, 1, val)

	ttl := s.Expiration(key)
	require.Equal(t, expiration, ttl)

	s.Del(key, conflict)

	_, ok = s.Get(key, conflict)
	require.False(t, ok)
	require.True(t, s.Expiration(key).IsZero())

	// missing item
	key, _ = z.KeyToHash(4340958203495)
	ttl = s.Expiration(key)
	require.True(t, ttl.IsZero())
}

// TestSnapshotEntries verifies that snapshotEntries correctly captures all storeItems
// including key, conflict, value, and expiration.
func TestSnapshotEntries(t *testing.T) {
	// Create a new shardedMap for string values
	sm := newShardedMap[string]()

	// Prepare two entries with distinct shards and expirations
	exp1 := time.Now().Add(5 * time.Minute)
	si1 := storeItem[string]{
		key:        100,
		conflict:   1,
		value:      "one",
		expiration: exp1,
	}
	sm.shards[0].Lock()
	sm.shards[0].data[100] = si1
	sm.shards[0].Unlock()

	exp2 := time.Time{} // no expiration
	si2 := storeItem[string]{
		key:        200,
		conflict:   2,
		value:      "two",
		expiration: exp2,
	}
	sm.shards[1].Lock()
	sm.shards[1].data[200] = si2
	sm.shards[1].Unlock()

	// Snapshot and verify
	entries := sm.snapshotEntries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Map entries by key for easier assertions
	found := make(map[uint64]storeItem[string])
	for _, e := range entries {
		found[e.key] = e
	}

	// Verify first entry
	e1, ok := found[100]
	if !ok {
		t.Errorf("missing entry for key 100")
	} else {
		if e1.conflict != 1 {
			t.Errorf("expected conflict 1 for key 100, got %d", e1.conflict)
		}
		if e1.value != "one" {
			t.Errorf("expected value 'one' for key 100, got '%s'", e1.value)
		}
		if !e1.expiration.Equal(exp1) {
			t.Errorf("expected expiration %v for key 100, got %v", exp1, e1.expiration)
		}
	}

	// Verify second entry
	e2, ok := found[200]
	if !ok {
		t.Errorf("missing entry for key 200")
	} else {
		if e2.conflict != 2 {
			t.Errorf("expected conflict 2 for key 200, got %d", e2.conflict)
		}
		if e2.value != "two" {
			t.Errorf("expected value 'two' for key 200, got '%s'", e2.value)
		}
		if !e2.expiration.IsZero() {
			t.Errorf("expected no expiration for key 200, got %v", e2.expiration)
		}
	}
}

func BenchmarkStoreGet(b *testing.B) {
	s := newStore[int]()
	key, conflict := z.KeyToHash(1)
	i := Item[int]{
		Key:      key,
		Conflict: conflict,
		Value:    1,
	}
	s.Set(&i)
	b.SetBytes(1)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Get(key, conflict)
		}
	})
}

func BenchmarkStoreSet(b *testing.B) {
	s := newStore[int]()
	key, conflict := z.KeyToHash(1)
	b.SetBytes(1)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := Item[int]{
				Key:      key,
				Conflict: conflict,
				Value:    1,
			}
			s.Set(&i)
		}
	})
}

func BenchmarkStoreUpdate(b *testing.B) {
	s := newStore[int]()
	key, conflict := z.KeyToHash(1)
	i := Item[int]{
		Key:      key,
		Conflict: conflict,
		Value:    1,
	}
	s.Set(&i)
	b.SetBytes(1)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Update(&Item[int]{
				Key:      key,
				Conflict: conflict,
				Value:    2,
			})
		}
	})
}
