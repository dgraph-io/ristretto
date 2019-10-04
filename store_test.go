package ristretto

import "testing"

func TestStoreSetGet(t *testing.T) {
	s := newStore()
	s.Set(1, 2)
	if val, ok := s.Get(1); (val == nil || !ok) || val.(int) != 2 {
		t.Fatal("set/get error")
	}
	s.Set(1, 3)
	if val, ok := s.Get(1); (val == nil || !ok) || val.(int) != 3 {
		t.Fatal("set/get overwrite error")
	}
}

func TestStoreDel(t *testing.T) {
	s := newStore()
	s.Set(1, 1)
	s.Del(1)
	if val, ok := s.Get(1); val != nil || ok {
		t.Fatal("del error")
	}
}

func TestStoreClear(t *testing.T) {
	s := newStore()
	for i := uint64(0); i < 1000; i++ {
		s.Set(i, i)
	}
	s.Clear()
	for i := uint64(0); i < 1000; i++ {
		if val, ok := s.Get(i); val != nil || ok {
			t.Fatal("clear operation failed")
		}
	}
}

func TestStoreUpdate(t *testing.T) {
	s := newStore()
	s.Set(1, 1)
	if updated := s.Update(1, 2); !updated {
		t.Fatal("value should have been updated")
	}
	if val, ok := s.Get(1); val == nil || !ok {
		t.Fatal("value was deleted")
	}
	if val, ok := s.Get(1); val.(int) != 2 || !ok {
		t.Fatal("value wasn't updated")
	}
	if updated := s.Update(2, 2); updated {
		t.Fatal("value should not have been updated")
	}
	if val, ok := s.Get(2); val != nil || ok {
		t.Fatal("value should not have been updated")
	}
}

func BenchmarkStoreGet(b *testing.B) {
	s := newStore()
	s.Set(1, 1)
	b.SetBytes(1)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Get(1)
		}
	})
}

func BenchmarkStoreSet(b *testing.B) {
	s := newStore()
	b.SetBytes(1)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Set(1, 1)
		}
	})
}

func BenchmarkStoreUpdate(b *testing.B) {
	s := newStore()
	s.Set(1, 1)
	b.SetBytes(1)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Update(1, 2)
		}
	})
}
