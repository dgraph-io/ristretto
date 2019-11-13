package ristretto

import (
	"testing"

	"github.com/dgraph-io/ristretto/z"
)

func TestStoreSetGet(t *testing.T) {
	s := newStore()
	key, conflict := z.KeyToHash(1)
	s.Set(key, conflict, 2)
	if val, ok := s.Get(key, conflict); (val == nil || !ok) || val.(int) != 2 {
		t.Fatal("set/get error")
	}
	s.Set(key, conflict, 3)
	if val, ok := s.Get(key, conflict); (val == nil || !ok) || val.(int) != 3 {
		t.Fatal("set/get overwrite error")
	}
	key, conflict = z.KeyToHash(2)
	s.Set(key, conflict, 2)
	if val, ok := s.Get(key, conflict); !ok || val.(int) != 2 {
		t.Fatal("set/get nil key error")
	}
}

func TestStoreDel(t *testing.T) {
	s := newStore()
	key, conflict := z.KeyToHash(1)
	s.Set(key, conflict, 1)
	s.Del(key, conflict)
	if val, ok := s.Get(key, conflict); val != nil || ok {
		t.Fatal("del error")
	}
	s.Del(2, 0)
}

func TestStoreClear(t *testing.T) {
	s := newStore()
	for i := uint64(0); i < 1000; i++ {
		key, conflict := z.KeyToHash(i)
		s.Set(key, conflict, i)
	}
	s.Clear()
	for i := uint64(0); i < 1000; i++ {
		key, conflict := z.KeyToHash(i)
		if val, ok := s.Get(key, conflict); val != nil || ok {
			t.Fatal("clear operation failed")
		}
	}
}

func TestStoreUpdate(t *testing.T) {
	s := newStore()
	key, conflict := z.KeyToHash(1)
	s.Set(key, conflict, 1)
	if updated := s.Update(key, conflict, 2); !updated {
		t.Fatal("value should have been updated")
	}
	if val, ok := s.Get(key, conflict); val == nil || !ok {
		t.Fatal("value was deleted")
	}
	if val, ok := s.Get(key, conflict); val.(int) != 2 || !ok {
		t.Fatal("value wasn't updated")
	}
	if !s.Update(key, conflict, 3) {
		t.Fatal("value should have been updated")
	}
	if val, ok := s.Get(key, conflict); val.(int) != 3 || !ok {
		t.Fatal("value wasn't updated")
	}
	key, conflict = z.KeyToHash(2)
	if updated := s.Update(key, conflict, 2); updated {
		t.Fatal("value should not have been updated")
	}
	if val, ok := s.Get(key, conflict); val != nil || ok {
		t.Fatal("value should not have been updated")
	}
}

func TestStoreCollision(t *testing.T) {
	s := newShardedMap()
	s.shards[1].Lock()
	s.shards[1].data[1] = storeItem{
		key:      1,
		conflict: 0,
		value:    1,
	}
	s.shards[1].Unlock()
	if val, ok := s.Get(1, 1); val != nil || ok {
		t.Fatal("collision should return nil")
	}
	s.Set(1, 1, 2)
	if val, ok := s.Get(1, 0); !ok || val == nil || val.(int) == 2 {
		t.Fatal("collision should prevent Set update")
	}
	if s.Update(1, 1, 2) {
		t.Fatal("collision should prevent Update")
	}
	if val, ok := s.Get(1, 0); !ok || val == nil || val.(int) == 2 {
		t.Fatal("collision should prevent Update")
	}
	s.Del(1, 1)
	if val, ok := s.Get(1, 0); !ok || val == nil {
		t.Fatal("collision should prevent Del")
	}
}

func BenchmarkStoreGet(b *testing.B) {
	s := newStore()
	key, conflict := z.KeyToHash(1)
	s.Set(key, conflict, 1)
	b.SetBytes(1)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Get(key, conflict)
		}
	})
}

func BenchmarkStoreSet(b *testing.B) {
	s := newStore()
	key, conflict := z.KeyToHash(1)
	b.SetBytes(1)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Set(key, conflict, 1)
		}
	})
}

func BenchmarkStoreUpdate(b *testing.B) {
	s := newStore()
	key, conflict := z.KeyToHash(1)
	s.Set(key, conflict, 1)
	b.SetBytes(1)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Update(key, conflict, 2)
		}
	})
}
