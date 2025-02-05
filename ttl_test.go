/*
 * SPDX-FileCopyrightText: © Hypermode Inc. <hello@hypermode.com>
 * SPDX-License-Identifier: Apache-2.0
 */

package ristretto

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestExpirationMapCleanup tests the cleanup functionality of the expiration map.
// It verifies that expired items are correctly evicted from the store and that
// non-expired items remain in the store.
func TestExpirationMapCleanup(t *testing.T) {
	// Create a new expiration map
	em := newExpirationMap[int]()
	// Create a new store
	s := newShardedMap[int]()
	// Create a new policy
	p := newDefaultPolicy[int](100, 10)

	// Add items to the store and expiration map
	now := time.Now()
	i1 := &Item[int]{Key: 1, Conflict: 1, Value: 100, Expiration: now.Add(1 * time.Second)}
	s.Set(i1)
	em.add(i1.Key, i1.Conflict, i1.Expiration)

	i2 := &Item[int]{Key: 2, Conflict: 2, Value: 200, Expiration: now.Add(3 * time.Second)}
	s.Set(i2)
	em.add(i2.Key, i2.Conflict, i2.Expiration)

	// Create a map to store evicted items
	evictedItems := make(map[uint64]int)
	evictedItemsOnEvictFunc := func(item *Item[int]) {
		evictedItems[item.Key] = item.Value
	}

	// Wait for the first item to expire
	time.Sleep(2 * time.Second)

	// Cleanup the expiration map
	cleanedBucketsCount := em.cleanup(s, p, evictedItemsOnEvictFunc)
	require.Equal(t, 1, cleanedBucketsCount, "cleanedBucketsCount should be 1 after first cleanup")

	// Check that the first item was evicted
	require.Equal(t, 1, len(evictedItems), "evictedItems should have 1 item")
	require.Equal(t, 100, evictedItems[1], "evictedItems should have the first item")
	_, ok := s.Get(i1.Key, i1.Conflict)
	require.False(t, ok, "i1 should have been evicted")

	// Check that the second item is still in the store
	_, ok = s.Get(i2.Key, i2.Conflict)
	require.True(t, ok, "i2 should still be in the store")

	// Wait for the second item to expire
	time.Sleep(2 * time.Second)

	// Cleanup the expiration map
	cleanedBucketsCount = em.cleanup(s, p, evictedItemsOnEvictFunc)
	require.Equal(t, 1, cleanedBucketsCount, "cleanedBucketsCount should be 1 after second cleanup")

	// Check that the second item was evicted
	require.Equal(t, 2, len(evictedItems), "evictedItems should have 2 items")
	require.Equal(t, 200, evictedItems[2], "evictedItems should have the second item")
	_, ok = s.Get(i2.Key, i2.Conflict)
	require.False(t, ok, "i2 should have been evicted")

	t.Run("Miscalculation of buckets does not cause memory leaks", func(t *testing.T) {
		// Break lastCleanedBucketNum, this can happen if the system time is changed.
		em.lastCleanedBucketNum = storageBucket(now.AddDate(-1, 0, 0))

		cleanedBucketsCount = em.cleanup(s, p, evictedItemsOnEvictFunc)
		require.Equal(t,
			0, cleanedBucketsCount,
			"cleanedBucketsCount should be 0 after cleanup with lastCleanedBucketNum change",
		)
	})
}
