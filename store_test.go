package ristretto

import (
	"github.com/dgraph-io/ristretto/z"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestStoreSetGet(t *testing.T) {
	s := newStore()
	key, conflict := z.KeyToHash(1)
	i := item{
		key:      key,
		conflict: conflict,
		value:    2,
	}
	s.Set(&i)
	val, ok := s.Get(key, conflict)
	require.True(t, ok)
	require.Equal(t, 2, val.(int))

	i.value = 3
	s.Set(&i)
	val, ok = s.Get(key, conflict)
	require.True(t, ok)
	require.Equal(t, 3, val.(int))

	key, conflict = z.KeyToHash(2)
	i = item{
		key:      key,
		conflict: conflict,
		value:    2,
	}
	s.Set(&i)
	val, ok = s.Get(key, conflict)
	require.True(t, ok)
	require.Equal(t, 2, val.(int))
}

func TestStoreExpiration(t *testing.T) {
	s := newStore()
	key, conflict := z.KeyToHash(1)

	expiration := time.Now().Add(1 * time.Second)
	i := item{
		key:        key,
		conflict:   conflict,
		value:      1,
		expiration: expiration,
	}

	s.Set(&i)
	val, ok := s.Get(key, conflict)
	require.True(t, ok)
	require.Equal(t, 1, val.(int))

	ttl := s.Expiration(key)
	require.Equal(t, expiration, ttl)

	s.Del(key, conflict)
	val, ok = s.Get(key, conflict)
	require.False(t, ok)

	ttl = s.Expiration(key)
	require.True(t, ttl.IsZero())
}

func TestStoreExpirationForMissingItem(t *testing.T) {
	s := newStore()
	key, _ := z.KeyToHash(1)

	ttl := s.Expiration(key)
	require.True(t, ttl.IsZero())
}

func TestStoreDel(t *testing.T) {
	s := newStore()
	key, conflict := z.KeyToHash(1)
	i := item{
		key:      key,
		conflict: conflict,
		value:    1,
	}
	s.Set(&i)
	s.Del(key, conflict)
	val, ok := s.Get(key, conflict)
	require.False(t, ok)
	require.Nil(t, val)

	s.Del(2, 0)
}

func TestStoreClear(t *testing.T) {
	s := newStore()
	for i := uint64(0); i < 1000; i++ {
		key, conflict := z.KeyToHash(i)
		it := item{
			key:      key,
			conflict: conflict,
			value:    i,
		}
		s.Set(&it)
	}
	s.Clear()
	for i := uint64(0); i < 1000; i++ {
		key, conflict := z.KeyToHash(i)
		val, ok := s.Get(key, conflict)
		require.False(t, ok)
		require.Nil(t, val)
	}
}

func TestStoreUpdate(t *testing.T) {
	s := newStore()
	key, conflict := z.KeyToHash(1)
	i := item{
		key:      key,
		conflict: conflict,
		value:    1,
	}
	s.Set(&i)
	i.value = 2
	require.True(t, s.Update(&i))

	val, ok := s.Get(key, conflict)
	require.True(t, ok)
	require.NotNil(t, val)

	val, ok = s.Get(key, conflict)
	require.True(t, ok)
	require.Equal(t, 2, val.(int))

	i.value = 3
	require.True(t, s.Update(&i))

	val, ok = s.Get(key, conflict)
	require.True(t, ok)
	require.Equal(t, 3, val.(int))

	key, conflict = z.KeyToHash(2)
	i = item{
		key:      key,
		conflict: conflict,
		value:    2,
	}
	require.False(t, s.Update(&i))
	val, ok = s.Get(key, conflict)
	require.False(t, ok)
	require.Nil(t, val)
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
	val, ok := s.Get(1, 1)
	require.False(t, ok)
	require.Nil(t, val)

	i := item{
		key:      1,
		conflict: 1,
		value:    2,
	}
	s.Set(&i)
	val, ok = s.Get(1, 0)
	require.True(t, ok)
	require.NotEqual(t, 2, val.(int))

	require.False(t, s.Update(&i))
	val, ok = s.Get(1, 0)
	require.True(t, ok)
	require.NotEqual(t, 2, val.(int))

	s.Del(1, 1)
	val, ok = s.Get(1, 0)
	require.True(t, ok)
	require.NotNil(t, val)
}

func BenchmarkStoreGet(b *testing.B) {
	s := newStore()
	key, conflict := z.KeyToHash(1)
	i := item{
		key:      key,
		conflict: conflict,
		value:    1,
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
	s := newStore()
	key, conflict := z.KeyToHash(1)
	b.SetBytes(1)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := item{
				key:      key,
				conflict: conflict,
				value:    1,
			}
			s.Set(&i)
		}
	})
}

func BenchmarkStoreUpdate(b *testing.B) {
	s := newStore()
	key, conflict := z.KeyToHash(1)
	i := item{
		key:      key,
		conflict: conflict,
		value:    1,
	}
	s.Set(&i)
	b.SetBytes(1)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Update(&item{
				key:      key,
				conflict: conflict,
				value:    2,
			})
		}
	})
}
