/*
 * SPDX-FileCopyrightText: © 2017-2026 Istari Digital, Inc.
 * SPDX-License-Identifier: Apache-2.0
 */

package ristretto

import (
	"testing"
	"time"
)

type pausingCleanupStore struct {
	store[int]
	entered chan struct{}
	resume  chan struct{}
}

// Expiration pauses the pre-fix cleanup path before it reads the refreshed item.
func (s *pausingCleanupStore) Expiration(key uint64) time.Time {
	close(s.entered)
	<-s.resume
	return s.store.Expiration(key)
}

// DelExpired pauses conditional deletion until the test replaces the expired item.
func (s *pausingCleanupStore) DelExpired(key, conflict uint64, now time.Time) (uint64, int, time.Time, bool) {
	close(s.entered)
	<-s.resume
	return s.store.(interface {
		DelExpired(uint64, uint64, time.Time) (uint64, int, time.Time, bool)
	}).DelExpired(key, conflict, now)
}

// TestExpirationMapCleanupPreservesFreshReplacement covers an update during TTL cleanup.
func TestExpirationMapCleanupPreservesFreshReplacement(t *testing.T) {
	s := newShardedMap[int]()
	p := newDefaultPolicy[int](100, 100)
	defer p.Close()
	expired := time.Now().Add(-20 * time.Second)
	s.Set(&Item[int]{Key: 1, Conflict: 1, Value: 11, Cost: 1, Expiration: expired})
	p.Add(1, 1)
	em := s.expiryMap
	em.lastCleanedBucketNum = storageBucket(expired) - 1

	// Pause cleanup after it snapshots the expired bucket, then refresh the key.
	paused := &pausingCleanupStore{
		store: s, entered: make(chan struct{}), resume: make(chan struct{}),
	}
	done := make(chan struct{})
	go func() {
		em.cleanup(paused, p, nil)
		close(done)
	}()
	select {
	case <-paused.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("cleanup did not reach expiration check or delete")
	}
	s.Set(&Item[int]{Key: 1, Conflict: 1, Value: 22, Cost: 1})
	close(paused.resume)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("cleanup did not complete")
	}
	got, ok := s.Get(1, 1)
	if !ok || got != 22 {
		t.Fatalf("fresh replacement deleted: got %d, found %v", got, ok)
	}
	if cost := p.Cost(1); cost != 1 {
		t.Fatalf("fresh replacement cost removed: got %d", cost)
	}
}
