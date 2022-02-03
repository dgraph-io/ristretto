/*
 * Copyright 2019 Dgraph Labs, Inc. and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Ristretto is a fast, fixed size, in-memory cache with a dual focus on
// throughput and hit ratio performance. You can easily add Ristretto to an
// existing system and keep the most valuable data where you need it.
package ristretto

import (
	"errors"
	"sync"
	"time"
	"unsafe"

	"github.com/etecs-ru/ristretto/z"
)

// TODO: find the optimal value for this or make it configurable
var setBufSize = 32 * 1024 //nolint:gochecknoglobals // adopt fork, do not touch it

type itemCallback func(*Item)

const itemSize = int64(unsafe.Sizeof(storeItem{}))

// Cache is a thread-safe implementation of a hashmap with a TinyLFU admission
// policy and a Sampled LFU eviction policy. You can use the same Cache instance
// from as many goroutines as you want.
type Cache struct {
	store              *shardedMap
	policy             *lfuPolicy
	getBuf             *ringBuffer
	setBuf             chan *Item
	onEvict            itemCallback
	onReject           itemCallback
	onExit             func(interface{})
	keyToHash          func(interface{}) (uint64, uint64)
	stop               chan struct{}
	cleanupTicker      *time.Ticker
	cost               func(value interface{}) int64
	Metrics            *Metrics
	ignoreInternalCost bool
	isClosed           bool
}

// Config is passed to NewCache for creating new Cache instances.
type Config struct {
	OnExit             func(val interface{})
	KeyToHash          func(key interface{}) (uint64, uint64)
	ShouldUpdate       func(prev, cur interface{}) bool
	Cost               func(value interface{}) int64
	OnEvict            func(item *Item)
	OnReject           func(item *Item)
	NumCounters        int64
	MaxCost            int64
	BufferItems        int64
	Metrics            bool
	IgnoreInternalCost bool
}

type itemFlag byte

const (
	itemNew itemFlag = iota
	itemDelete
	itemUpdate
)

// Item is passed to setBuf so items can eventually be added to the cache.
type Item struct {
	Expiration time.Time
	Value      interface{}
	wg         *sync.WaitGroup
	Key        uint64
	Conflict   uint64
	Cost       int64
	flag       itemFlag
}

// NewCache returns a new Cache instance and any configuration errors, if any.
func NewCache(config *Config) (*Cache, error) {
	switch {
	case config.NumCounters == 0:
		return nil, errors.New("NumCounters can't be zero")
	case config.MaxCost == 0:
		return nil, errors.New("MaxCost can't be zero")
	case config.BufferItems == 0:
		return nil, errors.New("BufferItems can't be zero")
	}
	policy := newPolicy(config.NumCounters, config.MaxCost)
	cache := &Cache{
		store:              newShardedMap(config.ShouldUpdate),
		policy:             policy,
		getBuf:             newRingBuffer(policy, config.BufferItems),
		setBuf:             make(chan *Item, setBufSize),
		keyToHash:          config.KeyToHash,
		stop:               make(chan struct{}),
		cost:               config.Cost,
		ignoreInternalCost: config.IgnoreInternalCost,
		cleanupTicker:      time.NewTicker(time.Duration(bucketDurationSecs) * time.Second / 2),
	}
	cache.onExit = func(val interface{}) {
		if config.OnExit != nil && val != nil {
			config.OnExit(val)
		}
	}
	cache.onEvict = func(item *Item) {
		if config.OnEvict != nil {
			config.OnEvict(item)
		}
		cache.onExit(item.Value)
	}
	cache.onReject = func(item *Item) {
		if config.OnReject != nil {
			config.OnReject(item)
		}
		cache.onExit(item.Value)
	}
	cache.store.shouldUpdate = func(prev, cur interface{}) bool {
		if config.ShouldUpdate != nil {
			return config.ShouldUpdate(prev, cur)
		}
		return true
	}
	if cache.keyToHash == nil {
		cache.keyToHash = z.KeyToHash
	}
	if config.Metrics {
		cache.collectMetrics()
	}
	// NOTE: benchmarks seem to show that performance decreases the more
	//       goroutines we have running cache.processItems(), so 1 should
	//       usually be sufficient
	go cache.processItems()
	return cache, nil
}

func (c *Cache) Wait() {
	if c == nil || c.isClosed {
		return
	}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	c.setBuf <- &Item{wg: wg}
	wg.Wait()
}

// Get returns the value (if any) and a boolean representing whether the
// value was found or not. The value can be nil and the boolean can be true at
// the same time.
func (c *Cache) Get(key interface{}) (interface{}, bool) {
	if c == nil || c.isClosed || key == nil {
		return nil, false
	}
	keyHash, conflictHash := c.keyToHash(key)
	c.getBuf.Push(keyHash)
	value, ok := c.store.Get(keyHash, conflictHash)
	if ok {
		c.Metrics.add(hit, keyHash, 1)
	} else {
		c.Metrics.add(miss, keyHash, 1)
	}
	return value, ok
}

// Set attempts to add the key-value item to the cache. If it returns false,
// then the Set was dropped and the key-value item isn't added to the cache. If
// it returns true, there's still a chance it could be dropped by the policy if
// its determined that the key-value item isn't worth keeping, but otherwise the
// item will be added and other items will be evicted in order to make room.
//
// To dynamically evaluate the items cost using the Config.Coster function, set
// the cost parameter to 0 and Coster will be ran when needed in order to find
// the items true cost.
func (c *Cache) Set(key, value interface{}, cost int64) bool {
	return c.SetWithTTL(key, value, cost, 0*time.Second)
}

// SetWithTTL works like Set but adds a key-value pair to the cache that will expire
// after the specified TTL (time to live) has passed. A zero value means the value never
// expires, which is identical to calling Set. A negative value is a no-op and the value
// is discarded.
func (c *Cache) SetWithTTL(key, value interface{}, cost int64, ttl time.Duration) bool {
	return c.setInternal(key, value, cost, ttl, false)
}

// SetIfPresent is like Set, but only updates the value of an existing key. It
// does NOT add the key to cache if it's absent.
func (c *Cache) SetIfPresent(key, value interface{}, cost int64) bool {
	return c.setInternal(key, value, cost, 0*time.Second, true)
}

func (c *Cache) setInternal(key, value interface{},
	cost int64, ttl time.Duration, onlyUpdate bool) bool {
	if c == nil || c.isClosed || key == nil {
		return false
	}

	var expiration time.Time
	switch {
	case ttl == 0:
		// No expiration.
		break
	case ttl < 0:
		// Treat this a a no-op.
		return false
	default:
		expiration = time.Now().Add(ttl)
	}

	keyHash, conflictHash := c.keyToHash(key)
	i := &Item{
		flag:       itemNew,
		Key:        keyHash,
		Conflict:   conflictHash,
		Value:      value,
		Cost:       cost,
		Expiration: expiration,
	}
	if onlyUpdate {
		i.flag = itemUpdate
	}
	// cost is eventually updated. The expiration must also be immediately updated
	// to prevent items from being prematurely removed from the map.
	if prev, ok := c.store.Update(i); ok {
		c.onExit(prev)
		i.flag = itemUpdate
	} else if onlyUpdate {
		// The instruction was to update the key, but store.Update failed. So,
		// this is a NOOP.
		return false
	}
	// Attempt to send item to policy.
	select {
	case c.setBuf <- i:
		return true
	default:
		if i.flag == itemUpdate {
			// Return true if this was an update operation since we've already
			// updated the store. For all the other operations (set/delete), we
			// return false which means the item was not inserted.
			return true
		}
		c.Metrics.add(dropSets, keyHash, 1)
		return false
	}
}

// Del deletes the key-value item from the cache if it exists.
func (c *Cache) Del(key interface{}) {
	if c == nil || c.isClosed || key == nil {
		return
	}
	keyHash, conflictHash := c.keyToHash(key)
	// Delete immediately.
	_, prev := c.store.Del(keyHash, conflictHash)
	c.onExit(prev)
	// If we've set an item, it would be applied slightly later.
	// So we must push the same item to `setBuf` with the deletion flag.
	// This ensures that if a set is followed by a delete, it will be
	// applied in the correct order.
	c.setBuf <- &Item{
		flag:     itemDelete,
		Key:      keyHash,
		Conflict: conflictHash,
	}
}

// GetTTL returns the TTL for the specified key and a bool that is true if the
// item was found and is not expired.
func (c *Cache) GetTTL(key interface{}) (time.Duration, bool) {
	if c == nil || key == nil {
		return 0, false
	}

	keyHash, conflictHash := c.keyToHash(key)
	if _, ok := c.store.Get(keyHash, conflictHash); !ok {
		// not found
		return 0, false
	}

	expiration := c.store.Expiration(keyHash)
	if expiration.IsZero() {
		// found but no expiration
		return 0, true
	}

	if time.Now().After(expiration) {
		// found but expired
		return 0, false
	}

	return time.Until(expiration), true
}

// Close stops all goroutines and closes all channels.
func (c *Cache) Close() {
	if c == nil || c.isClosed {
		return
	}
	c.Clear()

	// Block until processItems goroutine is returned.
	c.stop <- struct{}{}
	close(c.stop)
	close(c.setBuf)
	c.policy.Close()
	c.isClosed = true
}

// Clear empties the hashmap and zeroes all policy counters. Note that this is
// not an atomic operation (but that shouldn't be a problem as it's assumed that
// Set/Get calls won't be occurring until after this).
func (c *Cache) Clear() {
	if c == nil || c.isClosed {
		return
	}
	// Block until processItems goroutine is returned.
	c.stop <- struct{}{}

	// Clear out the setBuf channel.
loop:
	for {
		select {
		case i := <-c.setBuf:
			if i.wg != nil {
				i.wg.Done()
				continue
			}
			if i.flag != itemUpdate {
				// In itemUpdate, the value is already set in the store.  So, no need to call
				// onEvict here.
				c.onEvict(i)
			}
		default:
			break loop
		}
	}

	// Clear value hashmap and policy data.
	c.policy.Clear()
	c.store.Clear(c.onEvict)
	// Only reset metrics if they're enabled.
	if c.Metrics != nil {
		c.Metrics.Clear()
	}
	// Restart processItems goroutine.
	go c.processItems()
}

// MaxCost returns the max cost of the cache.
func (c *Cache) MaxCost() int64 {
	if c == nil {
		return 0
	}
	return c.policy.MaxCost()
}

// UpdateMaxCost updates the maxCost of an existing cache.
func (c *Cache) UpdateMaxCost(maxCost int64) {
	if c == nil {
		return
	}
	c.policy.UpdateMaxCost(maxCost)
}

// processItems is ran by goroutines processing the Set buffer.
func (c *Cache) processItems() {
	startTs := make(map[uint64]time.Time)
	numToKeep := 100000 // TODO: Make this configurable via options.

	trackAdmission := func(key uint64) {
		if c.Metrics == nil {
			return
		}
		startTs[key] = time.Now()
		if len(startTs) > numToKeep {
			for k := range startTs {
				if len(startTs) <= numToKeep {
					break
				}
				delete(startTs, k)
			}
		}
	}
	onEvict := func(i *Item) {
		if ts, has := startTs[i.Key]; has {
			c.Metrics.trackEviction(int64(time.Since(ts) / time.Second))
			delete(startTs, i.Key)
		}
		if c.onEvict != nil {
			c.onEvict(i)
		}
	}

	for {
		select {
		case i := <-c.setBuf:
			if i.wg != nil {
				i.wg.Done()
				continue
			}
			// Calculate item cost value if new or update.
			if i.Cost == 0 && c.cost != nil && i.flag != itemDelete {
				i.Cost = c.cost(i.Value)
			}
			if !c.ignoreInternalCost {
				// Add the cost of internally storing the object.
				i.Cost += itemSize
			}

			switch i.flag {
			case itemNew:
				victims, added := c.policy.Add(i.Key, i.Cost)
				if added {
					c.store.Set(i)
					c.Metrics.add(keyAdd, i.Key, 1)
					trackAdmission(i.Key)
				} else {
					c.onReject(i)
				}
				for _, victim := range victims {
					victim.Conflict, victim.Value = c.store.Del(victim.Key, 0)
					onEvict(victim)
				}

			case itemUpdate:
				c.policy.Update(i.Key, i.Cost)

			case itemDelete:
				c.policy.Del(i.Key) // Deals with metrics updates.
				_, val := c.store.Del(i.Key, i.Conflict)
				c.onExit(val)
			}
		case <-c.cleanupTicker.C:
			c.store.Cleanup(c.policy, onEvict)
		case <-c.stop:
			return
		}
	}
}
