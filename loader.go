package ristretto

import (
	"context"
	"sync"
)

// loader is the interface fulfilled by all loader implementations in
// this file.
type loader interface {
	// Do runs and returns the results of the given function, making
	// sure that only one execution is running for a given key at a
	// time. If a duplicate comes in, the duplicate caller waits for the
	// original to complete and receives the same results.
	Do(ctx context.Context, key interface{}, keyHash uint64, fn LoadFunc) (interface{}, error)
}

// newLoader returns the default loader implementation.
func newLoader() loader {
	return newShardedCaller()
}

type shardedCaller struct {
	shards []*lockedCaller
}

func newShardedCaller() *shardedCaller {
	sm := &shardedCaller{
		shards: make([]*lockedCaller, int(numShards)),
	}
	for i := range sm.shards {
		sm.shards[i] = newLockedCaller()
	}
	return sm
}

func (m *shardedCaller) Do(ctx context.Context, key interface{}, keyHash uint64, fn LoadFunc) (interface{}, error) {
	return m.shards[keyHash%numShards].do(ctx, key, keyHash, fn)
}

// lockedCaller calls a load function with a key, ensuring that only one
// call is in-flight for a given key at a time.
type lockedCaller struct {
	mu sync.Mutex
	m  map[uint64]*call
}

func newLockedCaller() *lockedCaller {
	return &lockedCaller{
		m: make(map[uint64]*call),
	}
}

func (m *lockedCaller) do(ctx context.Context, key interface{}, keyHash uint64, fn LoadFunc) (interface{}, error) {
	m.mu.Lock()
	if c, ok := m.m[keyHash]; ok {
		m.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}

	c := &call{}
	c.wg.Add(1)
	m.m[keyHash] = c
	m.mu.Unlock()

	c.val, c.err = fn(ctx, key)
	c.wg.Done()

	m.mu.Lock()
	delete(m.m, keyHash)
	m.mu.Unlock()

	return c.val, c.err
}

// call is a running or completed Do call
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}
