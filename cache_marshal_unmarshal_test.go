package ristretto

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func createMarshalCache[K Key, V any](_ *testing.T) (*Cache[K, V], error) {
	cfg := &Config[K, V]{
		NumCounters: 100,  // small for test
		MaxCost:     1000, // big enough
		BufferItems: 64,
		Metrics:     false,
	}
	return NewCache[K, V](cfg)
}

func TestMarshalUnmarshal_Basic(t *testing.T) {
	// 1) Create a small cache instance
	cache, err := createMarshalCache[string, string](t)
	require.NoError(t, err, "NewCache should not error")

	// 2) Insert a couple of items
	ok := cache.Set("foo", "bar", int64(len("bar")))
	require.True(t, ok, "Set(foo) should succeed")
	ok = cache.Set("baz", "qux", 1)
	require.True(t, ok, "Set(baz) should succeed")

	// 3) Wait for asynchronous Sets to land
	cache.Wait()

	// 4) Verify that the items are in the cache
	v, ok := cache.Get("foo")
	require.True(t, ok, "Get(foo) should find an entry")
	require.Equal(t, "bar", v, "Get(foo) value")
	v, ok = cache.Get("baz")
	require.True(t, ok, "Get(baz) should find an entry")
	require.Equal(t, "qux", v, "Get(baz) value")

	// 5) MarshalBinary the entire cache
	data, err := cache.MarshalBinary()
	require.NoError(t, err, "MarshalBinary should not error")
	require.NotEmpty(t, data, "MarshalBinary should return a non-empty blob")

	// 6) Clear the cache and verify it's empty
	cache.Clear()
	_, ok = cache.Get("foo")
	require.False(t, ok, "foo should be evicted after Clear()")
	_, ok = cache.Get("baz")
	require.False(t, ok, "baz should be evicted after Clear()")

	// 7) UnmarshalBinary from the dump
	require.NoError(t, cache.UnmarshalBinary(data), "UnmarshalBinary should not error")

	// 8) Verify that the items are back
	v, ok = cache.Get("foo")
	require.True(t, ok, "Get(foo) after unmarshal should find an entry")
	require.Equal(t, "bar", v, "Get(foo) value after unmarshal")
	v, ok = cache.Get("baz")
	require.True(t, ok, "Get(baz) after unmarshal should find an entry")
	require.Equal(t, "qux", v, "Get(baz) value after unmarshal")
}

func TestMarshalUnmarshal_WithTTL(t *testing.T) {
	cache, err := createMarshalCache[string, string](t)
	require.NoError(t, err, "NewCache should not error")

	// 2) Set with a short TTL
	ttl := 50 * time.Millisecond
	ok := cache.SetWithTTL("t1", "expires", 1, ttl)
	require.True(t, ok, "SetWithTTL should succeed")
	cache.Wait()

	// 3) MarshalBinary immediately
	data, err := cache.MarshalBinary()
	require.NoError(t, err, "MarshalBinary should not error")

	// 4) Clear & unmarshal
	cache.Clear()
	require.NoError(t, cache.UnmarshalBinary(data), "UnmarshalBinary should not error")

	// 5) It should still be there initially
	_, ok = cache.Get("t1")
	require.True(t, ok, "Entry should exist immediately after unmarshal")

	// 6) Wait until after original TTL would have expired
	time.Sleep(ttl)
	_, ok = cache.Get("t1")
	require.False(t, ok, "Entry should expire after TTL")
}

func TestMarshalUnmarshal_Empty(t *testing.T) {
	cache, err := createMarshalCache[string, string](t)
	require.NoError(t, err, "NewCache should not error")

	data, err := cache.MarshalBinary()
	require.NoError(t, err, "MarshalBinary should not error on empty cache")
	require.NotEmpty(t, data, "MarshalBinary of empty cache should still return a blob")

	cache.Clear()
	require.NoError(t, cache.UnmarshalBinary(data), "UnmarshalBinary should not error on empty blob")

	// still empty
	_, ok := cache.Get("whatever")
	require.False(t, ok, "Cache should remain empty after restoring an empty dump")
}

func TestUnmarshal_Corrupt(t *testing.T) {
	cache, err := createMarshalCache[string, string](t)
	require.NoError(t, err, "NewCache should not error")

	err = cache.UnmarshalBinary([]byte("not a gob"))
	require.Error(t, err, "UnmarshalBinary should error on corrupt data")
}

func TestMarshal_ExcludeExpired(t *testing.T) {
	cache, err := createMarshalCache[string, string](t)
	require.NoError(t, err, "NewCache should not error")

	cache.Set("xx", "yy", 1)
	ok := cache.SetWithTTL("x", "y", 1, 1*time.Millisecond)
	require.True(t, ok, "SetWithTTL should succeed")
	cache.Wait()

	time.Sleep(2 * time.Millisecond) // let it expire before MarshalBinary
	data, err := cache.MarshalBinary()
	require.NoError(t, err, "MarshalBinary should not error")

	cache.Clear()
	require.NoError(t, cache.UnmarshalBinary(data), "UnmarshalBinary should not error")

	_, ok = cache.Get("x")
	require.False(t, ok, "Expired items should not survive dump/unmarshal")

	v, ok := cache.Get("xx")
	require.True(t, ok, "Non-expired item should survive dump/unmarshal")
	require.Equal(t, "yy", v, "Value of surviving item")
}

func TestUnmarshal_ReuseMarshalMultipleTimes(t *testing.T) {
	cache, err := createMarshalCache[string, string](t)
	require.NoError(t, err, "NewCache should not error")

	ok := cache.Set("a", "b", 1)
	require.True(t, ok, "Set(a) should succeed")
	cache.Wait()

	data, err := cache.MarshalBinary()
	require.NoError(t, err, "MarshalBinary should not error")

	for i := 0; i < 3; i++ {
		cache.Clear()
		require.NoError(t, cache.UnmarshalBinary(data), "UnmarshalBinary iteration %d should not error", i)
		v, ok := cache.Get("a")
		require.True(t, ok, "Iteration %d: entry should be present", i)
		require.Equal(t, "b", v, "Iteration %d: value should match", i)
	}
}
