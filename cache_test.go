package ristretto

import (
	"sync"
	"testing"
	"time"
)

var wait = time.Millisecond * 10

func TestCache(t *testing.T) {
	if _, err := NewCache(&Config{
		NumCounters: 0,
	}); err == nil {
		t.Fatal("numCounters can't be 0")
	}
	if _, err := NewCache(&Config{
		NumCounters: 100,
		MaxCost:     0,
	}); err == nil {
		t.Fatal("maxCost can't be 0")
	}
	if _, err := NewCache(&Config{
		NumCounters: 100,
		MaxCost:     10,
		BufferItems: 0,
	}); err == nil {
		t.Fatal("bufferItems can't be 0")
	}
	if c, err := NewCache(&Config{
		NumCounters: 100,
		MaxCost:     10,
		BufferItems: 64,
		Metrics:     true,
	}); c == nil || err != nil {
		t.Fatal("config should be good")
	}
}

func TestCacheProcessItems(t *testing.T) {
	m := &sync.Mutex{}
	evicted := make(map[uint64]struct{})
	c, err := NewCache(&Config{
		NumCounters: 100,
		MaxCost:     10,
		BufferItems: 64,
		Cost: func(value interface{}) int64 {
			return int64(value.(int))
		},
		OnEvict: func(key uint64, value interface{}, cost int64) {
			m.Lock()
			defer m.Unlock()
			evicted[key] = struct{}{}
		},
	})
	if err != nil {
		panic(err)
	}
	c.setBuf <- &item{flag: itemNew, key: 1, value: 1, cost: 0}
	time.Sleep(wait)
	if !c.policy.Has(1) || c.policy.Cost(1) != 1 {
		t.Fatal("cache processItems didn't add new item")
	}
	c.setBuf <- &item{flag: itemUpdate, key: 1, value: 2, cost: 0}
	time.Sleep(wait)
	if c.policy.Cost(1) != 2 {
		t.Fatal("cache processItems didn't update item cost")
	}
	c.setBuf <- &item{flag: itemDelete, key: 1}
	time.Sleep(wait)
	if val, ok := c.store.Get(1); val != nil || ok {
		t.Fatal("cache processItems didn't delete item")
	}
	if c.policy.Has(1) {
		t.Fatal("cache processItems didn't delete item")
	}
	c.setBuf <- &item{flag: itemNew, key: 2, value: 2, cost: 3}
	c.setBuf <- &item{flag: itemNew, key: 3, value: 3, cost: 3}
	c.setBuf <- &item{flag: itemNew, key: 4, value: 3, cost: 3}
	c.setBuf <- &item{flag: itemNew, key: 5, value: 3, cost: 5}
	time.Sleep(wait)
	m.Lock()
	if len(evicted) == 0 {
		m.Unlock()
		t.Fatal("cache processItems not evicting or calling OnEvict")
	}
	m.Unlock()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("cache processItems didn't stop")
		}
	}()
	c.Close()
	c.setBuf <- &item{flag: itemNew}
}

func TestCacheGet(t *testing.T) {
	c, err := NewCache(&Config{
		NumCounters: 100,
		MaxCost:     10,
		BufferItems: 64,
		Metrics:     true,
	})
	if err != nil {
		panic(err)
	}
	c.store.Set(1, 1)
	if val, ok := c.Get(1); val == nil || !ok {
		t.Fatal("get should be successful")
	}
	if val, ok := c.Get(2); val != nil || ok {
		t.Fatal("get should not be successful")
	}
	if c.stats.Ratio() != 0.5 {
		t.Fatal("get should record metrics")
	}
	c = nil
	if val, ok := c.Get(0); val != nil || ok {
		t.Fatal("get should not be successful with nil cache")
	}
}

func TestCacheSet(t *testing.T) {
	c, err := NewCache(&Config{
		NumCounters: 100,
		MaxCost:     10,
		BufferItems: 64,
		Metrics:     true,
	})
	if err != nil {
		panic(err)
	}
	if c.Set(1, 1, 1) {
		time.Sleep(wait)
		if val, ok := c.Get(1); val == nil || val.(int) != 1 || !ok {
			t.Fatal("set/get returned wrong value")
		}
	} else {
		if val, ok := c.Get(1); val != nil || ok {
			t.Fatal("set was dropped but value still added")
		}
	}
	c.Set(1, 2, 2)
	if val, ok := c.store.Get(1); val == nil || val.(int) != 2 || !ok {
		t.Fatal("set/update was unsuccessful")
	}
	c.stop <- struct{}{}
	for i := 0; i < setBufSize; i++ {
		c.setBuf <- &item{itemUpdate, 1, 1, 1}
	}
	if c.Set(2, 2, 1) {
		t.Fatal("set should be dropped with full setBuf")
	}
	if c.stats.Get(dropSets) != 1 {
		t.Fatal("set should track dropSets")
	}
	close(c.setBuf)
	close(c.stop)
	c = nil
	if c.Set(1, 1, 1) {
		t.Fatal("set shouldn't be successful with nil cache")
	}
}

func TestCacheDel(t *testing.T) {
	c, err := NewCache(&Config{
		NumCounters: 100,
		MaxCost:     10,
		BufferItems: 64,
	})
	if err != nil {
		panic(err)
	}
	c.Set(1, 1, 1)
	c.Del(1)
	time.Sleep(wait)
	if val, ok := c.Get(1); val != nil || ok {
		t.Fatal("del didn't delete")
	}
	c = nil
	defer func() {
		if r := recover(); r != nil {
			t.Fatal("del panic with nil cache")
		}
	}()
	c.Del(1)
}

func TestCacheClear(t *testing.T) {
	c, err := NewCache(&Config{
		NumCounters: 100,
		MaxCost:     10,
		BufferItems: 64,
		Metrics:     true,
	})
	if err != nil {
		panic(err)
	}
	for i := 0; i < 10; i++ {
		c.Set(i, i, 1)
	}
	time.Sleep(wait)
	if c.stats.Get(keyAdd) != 10 {
		t.Fatal("range of sets not being processed")
	}
	c.Clear()
	if c.stats.Get(keyAdd) != 0 {
		t.Fatal("clear didn't reset metrics")
	}
	for i := 0; i < 10; i++ {
		if val, ok := c.Get(i); val != nil || ok {
			t.Fatal("clear didn't delete values")
		}
	}
}

func TestCacheMetrics(t *testing.T) {
	c, err := NewCache(&Config{
		NumCounters: 100,
		MaxCost:     10,
		BufferItems: 64,
		Metrics:     true,
	})
	if err != nil {
		panic(err)
	}
	for i := 0; i < 10; i++ {
		c.Set(i, i, 1)
	}
	time.Sleep(wait)
	m := c.Metrics()
	if m.KeysAdded != 10 {
		t.Fatal("metrics exporting incorrect fields")
	}
	c = nil
	if c.Metrics() != nil {
		t.Fatal("metrics exporting non-nil with nil cache")
	}
}

func TestMetrics(t *testing.T) {
	newMetrics()
}

func TestMetricsAddGet(t *testing.T) {
	m := newMetrics()
	m.Add(hit, 1, 1)
	m.Add(hit, 2, 2)
	m.Add(hit, 3, 3)
	if m.Get(hit) != 6 {
		t.Fatal("add/get error")
	}
	m = nil
	m.Add(hit, 1, 1)
	if m.Get(hit) != 0 {
		t.Fatal("get with nil struct should return 0")
	}
}

func TestMetricsRatio(t *testing.T) {
	m := newMetrics()
	if m.Ratio() != 0 {
		t.Fatal("ratio with no hits or misses should be 0")
	}
	m.Add(hit, 1, 1)
	m.Add(hit, 2, 2)
	m.Add(miss, 1, 1)
	m.Add(miss, 2, 2)
	if m.Ratio() != 0.5 {
		t.Fatal("ratio incorrect")
	}
	m = nil
	if m.Ratio() != 0.0 {
		t.Fatal("ratio with a nil struct should return 0")
	}
}

func TestMetricsExport(t *testing.T) {
	m := newMetrics()
	m.Add(hit, 1, 1)
	m.Add(miss, 1, 1)
	m.Add(keyAdd, 1, 1)
	m.Add(keyUpdate, 1, 1)
	m.Add(keyEvict, 1, 1)
	m.Add(costAdd, 1, 1)
	m.Add(costEvict, 1, 1)
	m.Add(dropSets, 1, 1)
	m.Add(rejectSets, 1, 1)
	m.Add(dropGets, 1, 1)
	m.Add(keepGets, 1, 1)
	M := exportMetrics(m)
	if M.Hits != 1 || M.Misses != 1 || M.Ratio != 0.5 || M.KeysAdded != 1 ||
		M.KeysUpdated != 1 || M.KeysEvicted != 1 || M.CostAdded != 1 ||
		M.CostEvicted != 1 || M.SetsDropped != 1 || M.SetsRejected != 1 ||
		M.GetsDropped != 1 || M.GetsKept != 1 {
		t.Fatal("exportMetrics wrong value(s)")
	}
}
