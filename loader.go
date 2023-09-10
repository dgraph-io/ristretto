package ristretto

import (
	"context"
	"sync"
)

// loader is the interface fulfilled by all loader implementations in
// this file.
type loader[K any, V any] interface {
	// Do runs and returns the results of the given function, making
	// sure that only one execution is running for a given key at a
	// time. If a duplicate comes in, the duplicate caller waits for the
	// original to complete and receives the same results.
	Do(ctx context.Context, key K, keyHash uint64, fn LoadFunc[K, V]) (V, error)
}

// newLoader returns the default loader implementation.
func newLoader[K any, V any]() loader[K, V] {
	return newShardedCaller[K, V]()
}

type shardedCaller[K any, V any] struct {
	shards []*lockedCaller[K, V]
}

func newShardedCaller[K any, V any]() *shardedCaller[K, V] {
	sm := &shardedCaller[K, V]{
		shards: make([]*lockedCaller[K, V], int(numShards)),
	}
	for i := range sm.shards {
		sm.shards[i] = newLockedCaller[K, V]()
	}
	return sm
}

func (c *shardedCaller[K, V]) Do(ctx context.Context, key K, keyHash uint64, fn LoadFunc[K, V]) (V, error) {
	return c.shards[keyHash%numShards].do(ctx, key, keyHash, fn)
}

// lockedCaller calls a load function with a key, ensuring that only one
// call is in-flight for a given key at a time.
type lockedCaller[K any, V any] struct {
	mu sync.Mutex
	m  map[uint64]*call[V]
}

func newLockedCaller[K any, V any]() *lockedCaller[K, V] {
	return &lockedCaller[K, V]{
		m: make(map[uint64]*call[V]),
	}
}

func (lc *lockedCaller[K, V]) do(ctx context.Context, key K, keyHash uint64, fn LoadFunc[K, V]) (V, error) {
	lc.mu.Lock()
	if c, ok := lc.m[keyHash]; ok {
		lc.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}

	c := &call[V]{}
	c.wg.Add(1)
	lc.m[keyHash] = c
	lc.mu.Unlock()

	c.val, c.err = fn(ctx, key)
	c.wg.Done()

	lc.mu.Lock()
	delete(lc.m, keyHash)
	lc.mu.Unlock()

	return c.val, c.err
}

// call is a running or completed Do call
type call[V any] struct {
	wg  sync.WaitGroup
	val V
	err error
}
