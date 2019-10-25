package ristretto

import (
	"testing"

	"github.com/dgraph-io/ristretto/z"
)

func TestStoreSetGet(t *testing.T) {
	s := newStore(2)
	hashed := z.KeyToHash(1, 0)
	s.Set(hashed, 1, 2)
	if val, ok := s.Get(hashed, 1); (val == nil || !ok) || val.(int) != 2 {
		t.Fatal("set/get error")
	}
	s.Set(hashed, 1, 3)
	if val, ok := s.Get(hashed, 1); (val == nil || !ok) || val.(int) != 3 {
		t.Fatal("set/get overwrite error")
	}
	s.Set(z.KeyToHash(2, 0), nil, 2)
	if val, ok := s.Get(z.KeyToHash(2, 0), nil); !ok || val.(int) != 2 {
		t.Fatal("set/get nil key error")
	}
}

func TestStoreDel(t *testing.T) {
	s := newStore(2)
	hashed := z.KeyToHash(1, 0)
	s.Set(hashed, 1, 1)
	s.Del(hashed, 1)
	if val, ok := s.Get(hashed, 1); val != nil || ok {
		t.Fatal("del error")
	}
	s.Del(2, 2)
}

func TestStoreClear(t *testing.T) {
	s := newStore(2)
	for i := uint64(0); i < 1000; i++ {
		s.Set(z.KeyToHash(i, 0), i, i)
	}
	s.Clear()
	for i := uint64(0); i < 1000; i++ {
		if val, ok := s.Get(z.KeyToHash(i, 0), i); val != nil || ok {
			t.Fatal("clear operation failed")
		}
	}
}

func TestStoreUpdate(t *testing.T) {
	s := newStore(2)
	hashedOne := z.KeyToHash(1, 0)
	s.Set(hashedOne, 1, 1)
	if updated := s.Update(hashedOne, 1, 2); !updated {
		t.Fatal("value should have been updated")
	}
	if val, ok := s.Get(hashedOne, 1); val == nil || !ok {
		t.Fatal("value was deleted")
	}
	if val, ok := s.Get(hashedOne, 1); val.(int) != 2 || !ok {
		t.Fatal("value wasn't updated")
	}
	if !s.Update(hashedOne, nil, 3) {
		t.Fatal("value should have been updated")
	}
	if val, ok := s.Get(hashedOne, 1); val.(int) != 3 || !ok {
		t.Fatal("value wasn't updated")
	}
	hashedTwo := z.KeyToHash(2, 0)
	if updated := s.Update(hashedTwo, 2, 2); updated {
		t.Fatal("value should not have been updated")
	}
	if val, ok := s.Get(hashedTwo, 2); val != nil || ok {
		t.Fatal("value should not have been updated")
	}
}

func TestStoreCollision(t *testing.T) {
	s := newShardedMap(2)
	s.shards[1].Lock()
	s.shards[1].data[1] = storeItem{
		keyHash: 1,
		hashes:  []uint64{2},
		value:   1,
	}
	s.shards[1].Unlock()
	if val, ok := s.Get(1, 1); val != nil || ok {
		t.Fatal("collision should return nil")
	}
	s.Set(1, 1, 2)
	if val, ok := s.Get(1, 2); !ok || val == nil || val.(int) == 2 {
		t.Fatal("collision should prevent Set update")
	}
	if s.Update(1, 1, 2) {
		t.Fatal("collision should prevent Update")
	}
	if val, ok := s.Get(1, 2); !ok || val == nil || val.(int) == 2 {
		t.Fatal("collision should prevent Update")
	}
	s.Del(1, 1)
	if val, ok := s.Get(1, 2); !ok || val == nil {
		t.Fatal("collision should prevent Del")
	}
}

func BenchmarkStoreGet(b *testing.B) {
	s := newStore(2)
	hashed := z.KeyToHash(1, 0)
	s.Set(hashed, 1, 1)
	b.SetBytes(1)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Get(hashed, 1)
		}
	})
}

func BenchmarkStoreSet(b *testing.B) {
	s := newStore(2)
	hashed := z.KeyToHash(1, 0)
	b.SetBytes(1)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Set(hashed, 1, 1)
		}
	})
}

func BenchmarkStoreUpdate(b *testing.B) {
	s := newStore(2)
	hashed := z.KeyToHash(1, 0)
	s.Set(hashed, 1, 1)
	b.SetBytes(1)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Update(hashed, 1, 2)
		}
	})
}
