package ristretto

import (
	"testing"

	"github.com/dgraph-io/ristretto/z"
)

func TestStoreSetGet(t *testing.T) {
	s := newStore()
	hashes := z.KeyToHash(1)
	s.Set(hashes, 2)
	if val, ok := s.Get(hashes); (val == nil || !ok) || val.(int) != 2 {
		t.Fatal("set/get error")
	}
	s.Set(hashes, 3)
	if val, ok := s.Get(hashes); (val == nil || !ok) || val.(int) != 3 {
		t.Fatal("set/get overwrite error")
	}
	s.Set(z.KeyToHash(2), 2)
	if val, ok := s.Get(z.KeyToHash(2)); !ok || val.(int) != 2 {
		t.Fatal("set/get nil key error")
	}
}

func TestStoreDel(t *testing.T) {
	s := newStore()
	hashes := z.KeyToHash(1)
	s.Set(hashes, 1)
	s.Del(hashes)
	if val, ok := s.Get(hashes); val != nil || ok {
		t.Fatal("del error")
	}
	s.Del([2]uint64{2, 0})
}

func TestStoreClear(t *testing.T) {
	s := newStore()
	for i := uint64(0); i < 1000; i++ {
		s.Set(z.KeyToHash(i), i)
	}
	s.Clear()
	for i := uint64(0); i < 1000; i++ {
		if val, ok := s.Get(z.KeyToHash(i)); val != nil || ok {
			t.Fatal("clear operation failed")
		}
	}
}

func TestStoreUpdate(t *testing.T) {
	s := newStore()
	hashedOne := z.KeyToHash(1)
	s.Set(hashedOne, 1)
	if updated := s.Update(hashedOne, 2); !updated {
		t.Fatal("value should have been updated")
	}
	if val, ok := s.Get(hashedOne); val == nil || !ok {
		t.Fatal("value was deleted")
	}
	if val, ok := s.Get(hashedOne); val.(int) != 2 || !ok {
		t.Fatal("value wasn't updated")
	}
	if !s.Update(hashedOne, 3) {
		t.Fatal("value should have been updated")
	}
	if val, ok := s.Get(hashedOne); val.(int) != 3 || !ok {
		t.Fatal("value wasn't updated")
	}
	hashedTwo := z.KeyToHash(2)
	if updated := s.Update(hashedTwo, 2); updated {
		t.Fatal("value should not have been updated")
	}
	if val, ok := s.Get(hashedTwo); val != nil || ok {
		t.Fatal("value should not have been updated")
	}
}

func TestStoreCollision(t *testing.T) {
	s := newShardedMap()
	s.shards[1].Lock()
	s.shards[1].data[1] = storeItem{
		hashes: [2]uint64{1, 0},
		value:  1,
	}
	s.shards[1].Unlock()
	if val, ok := s.Get([2]uint64{1, 1}); val != nil || ok {
		t.Fatal("collision should return nil")
	}
	s.Set([2]uint64{1, 1}, 2)
	if val, ok := s.Get([2]uint64{1, 0}); !ok || val == nil || val.(int) == 2 {
		t.Fatal("collision should prevent Set update")
	}
	if s.Update([2]uint64{1, 1}, 2) {
		t.Fatal("collision should prevent Update")
	}
	if val, ok := s.Get([2]uint64{1, 0}); !ok || val == nil || val.(int) == 2 {
		t.Fatal("collision should prevent Update")
	}
	s.Del([2]uint64{1, 1})
	if val, ok := s.Get([2]uint64{1, 0}); !ok || val == nil {
		t.Fatal("collision should prevent Del")
	}
}

func BenchmarkStoreGet(b *testing.B) {
	s := newStore()
	hashes := z.KeyToHash(1)
	s.Set(hashes, 1)
	b.SetBytes(1)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Get(hashes)
		}
	})
}

func BenchmarkStoreSet(b *testing.B) {
	s := newStore()
	hashes := z.KeyToHash(1)
	b.SetBytes(1)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Set(hashes, 1)
		}
	})
}

func BenchmarkStoreUpdate(b *testing.B) {
	s := newStore()
	hashes := z.KeyToHash(1)
	s.Set(hashes, 1)
	b.SetBytes(1)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Update(hashes, 2)
		}
	})
}
