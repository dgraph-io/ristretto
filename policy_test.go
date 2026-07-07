/*
 * SPDX-FileCopyrightText: © 2017-2025 Istari Digital, Inc.
 * SPDX-License-Identifier: Apache-2.0
 */

package ristretto

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPolicy(t *testing.T) {
	defer func() {
		require.Nil(t, recover())
	}()
	newPolicy[int](100, 10)
}

func TestPolicyMetrics(t *testing.T) {
	p := newDefaultPolicy[int](100, 10)
	p.CollectMetrics(newMetrics())
	require.NotNil(t, p.metrics)
	require.NotNil(t, p.evict.metrics)
}

func TestPolicyProcessItems(t *testing.T) {
	p := newDefaultPolicy[int](100, 10)
	p.itemsCh <- []uint64{1, 2, 2}
	time.Sleep(wait)
	p.admitMu.Lock()
	require.Equal(t, int64(2), p.admit.Estimate(2))
	require.Equal(t, int64(1), p.admit.Estimate(1))
	p.admitMu.Unlock()

	p.stop <- struct{}{}
	<-p.done
	p.itemsCh <- []uint64{3, 3, 3}
	time.Sleep(wait)
	p.admitMu.Lock()
	require.Equal(t, int64(0), p.admit.Estimate(3))
	p.admitMu.Unlock()
}

func TestPolicyProcessItemsDoesNotWaitForEvictionLock(t *testing.T) {
	p := newDefaultPolicy[int](100, 10)
	defer p.Close()

	p.evictMu.Lock()
	defer p.evictMu.Unlock()

	p.itemsCh <- []uint64{9}
	require.Eventually(t, func() bool {
		p.admitMu.Lock()
		defer p.admitMu.Unlock()
		return p.admit.Estimate(9) == 1
	}, time.Second, time.Millisecond)
}

func waitForEvictionLockAvailable[V any](p *defaultPolicy[V], timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if p.evictMu.TryLock() {
			p.evictMu.Unlock()
			return true
		}
		time.Sleep(time.Millisecond)
	}
	return false
}

type policyAddResult[V any] struct {
	victims []*Item[V]
	added   bool
}

func waitForPolicyAdd[V any](t *testing.T, done <-chan policyAddResult[V]) policyAddResult[V] {
	t.Helper()
	select {
	case result := <-done:
		return result
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Add")
		return policyAddResult[V]{}
	}
}

func TestPolicyAddDoesNotHoldEvictionLockWhileEstimating(t *testing.T) {
	p := newDefaultPolicy[int](100, 10)
	defer p.Close()
	p.Add(1, 10)

	p.admitMu.Lock()
	addDone := make(chan policyAddResult[int], 1)
	go func() {
		victims, added := p.Add(2, 1)
		addDone <- policyAddResult[int]{victims: victims, added: added}
	}()

	releasedEvict := waitForEvictionLockAvailable(p, time.Second)
	p.admitMu.Unlock()
	result := waitForPolicyAdd(t, addDone)

	require.True(t, releasedEvict, "Add held evictMu while waiting on admission estimates")
	require.True(t, result.added)
	require.Len(t, result.victims, 1)
	require.True(t, p.Has(2))
}

func TestPolicyAddRetriesWhenSampleVictimDisappears(t *testing.T) {
	p := newDefaultPolicy[int](100, 10)
	defer p.Close()
	p.Add(1, 10)

	p.admitMu.Lock()
	addDone := make(chan policyAddResult[int], 1)
	go func() {
		victims, added := p.Add(2, 1)
		addDone <- policyAddResult[int]{victims: victims, added: added}
	}()

	releasedEvict := waitForEvictionLockAvailable(p, time.Second)
	if releasedEvict {
		p.Del(1)
	}
	p.admitMu.Unlock()
	result := waitForPolicyAdd(t, addDone)

	require.True(t, releasedEvict, "Add held evictMu while waiting on admission estimates")
	require.True(t, result.added)
	require.Empty(t, result.victims)
	require.False(t, p.Has(1))
	require.True(t, p.Has(2))
	require.Equal(t, int64(9), p.Cap())
}

func TestPolicyPush(t *testing.T) {
	p := newDefaultPolicy[int](100, 10)
	require.True(t, p.Push([]uint64{}))

	keepCount := 0
	for i := 0; i < 10; i++ {
		if p.Push([]uint64{1, 2, 3, 4, 5}) {
			keepCount++
		}
	}
	require.NotEqual(t, 0, keepCount)
}

func TestPolicyAdd(t *testing.T) {
	p := newDefaultPolicy[int](1000, 100)
	if victims, added := p.Add(1, 101); victims != nil || added {
		t.Fatal("can't add an item bigger than entire cache")
	}
	p.evictMu.Lock()
	p.evict.add(1, 1)
	p.evictMu.Unlock()
	p.admitMu.Lock()
	p.admit.Increment(1)
	p.admit.Increment(2)
	p.admit.Increment(3)
	p.admitMu.Unlock()

	victims, added := p.Add(1, 1)
	require.Nil(t, victims)
	require.False(t, added)

	victims, added = p.Add(2, 20)
	require.Nil(t, victims)
	require.True(t, added)

	victims, added = p.Add(3, 90)
	require.NotNil(t, victims)
	require.True(t, added)

	victims, added = p.Add(4, 20)
	require.NotNil(t, victims)
	require.False(t, added)
}

func TestPolicyHas(t *testing.T) {
	p := newDefaultPolicy[int](100, 10)
	p.Add(1, 1)
	require.True(t, p.Has(1))
	require.False(t, p.Has(2))
}

func TestPolicyDel(t *testing.T) {
	p := newDefaultPolicy[int](100, 10)
	p.Add(1, 1)
	p.Del(1)
	p.Del(2)
	require.False(t, p.Has(1))
	require.False(t, p.Has(2))
}

func TestPolicyCap(t *testing.T) {
	p := newDefaultPolicy[int](100, 10)
	p.Add(1, 1)
	require.Equal(t, int64(9), p.Cap())
}

func TestPolicyUpdate(t *testing.T) {
	p := newDefaultPolicy[int](100, 10)
	p.Add(1, 1)
	p.Update(1, 2)
	p.evictMu.RLock()
	require.Equal(t, int64(2), p.evict.keyCosts[1])
	p.evictMu.RUnlock()
}

func TestPolicyCost(t *testing.T) {
	p := newDefaultPolicy[int](100, 10)
	p.Add(1, 2)
	require.Equal(t, int64(2), p.Cost(1))
	require.Equal(t, int64(-1), p.Cost(2))
}

func TestPolicyClear(t *testing.T) {
	p := newDefaultPolicy[int](100, 10)
	p.Add(1, 1)
	p.Add(2, 2)
	p.Add(3, 3)
	p.Clear()
	require.Equal(t, int64(10), p.Cap())
	require.False(t, p.Has(1))
	require.False(t, p.Has(2))
	require.False(t, p.Has(3))
}

func TestPolicyConcurrentOperations(t *testing.T) {
	p := newDefaultPolicy[int](1<<12, 256)
	defer p.Close()
	for i := 0; i < 256; i++ {
		p.Add(uint64(i), 1)
	}

	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(seed uint64) {
			defer wg.Done()
			for i := uint64(0); i < 1000; i++ {
				key := seed*1000 + i
				switch i & 7 {
				case 0:
					p.Push([]uint64{key, key + 1, key + 2})
				case 1:
					p.Add(key, 1)
				case 2:
					_ = p.Has(key & 255)
				case 3:
					_ = p.Cost(key & 255)
				case 4:
					_ = p.Cap()
				case 5:
					p.Update(key&255, 1)
				case 6:
					p.Del(key & 255)
				default:
					p.Clear()
				}
			}
		}(uint64(g))
	}
	wg.Wait()
}

func TestPolicyClose(t *testing.T) {
	defer func() {
		require.NotNil(t, recover())
	}()

	p := newDefaultPolicy[int](100, 10)
	p.Add(1, 1)
	p.Close()
	p.itemsCh <- []uint64{1}
}

func TestPushAfterClose(t *testing.T) {
	p := newDefaultPolicy[int](100, 10)
	p.Close()
	require.False(t, p.Push([]uint64{1, 2}))
}

func TestAddAfterClose(t *testing.T) {
	p := newDefaultPolicy[int](100, 10)
	p.Close()
	p.Add(1, 1)
}

func TestSampledLFUAdd(t *testing.T) {
	e := newSampledLFU(4, 100)
	e.add(1, 1)
	e.add(2, 2)
	e.add(3, 1)
	require.Equal(t, int64(4), e.used)
	require.Equal(t, int64(2), e.keyCosts[2])
}

func TestSampledLFUDel(t *testing.T) {
	e := newSampledLFU(4, 100)
	e.add(1, 1)
	e.add(2, 2)
	e.del(2)
	require.Equal(t, int64(1), e.used)
	_, ok := e.keyCosts[2]
	require.False(t, ok)
	e.del(4)
}

func TestSampledLFUUpdate(t *testing.T) {
	e := newSampledLFU(4, 100)
	e.add(1, 1)
	require.True(t, e.updateIfHas(1, 2))
	require.Equal(t, int64(2), e.used)
	require.False(t, e.updateIfHas(2, 2))
}

func TestSampledLFUClear(t *testing.T) {
	e := newSampledLFU(4, 100)
	e.add(1, 1)
	e.add(2, 2)
	e.add(3, 1)
	e.clear()
	require.Equal(t, 0, len(e.keyCosts))
	require.Equal(t, int64(0), e.used)
}

func TestSampledLFURoom(t *testing.T) {
	e := newSampledLFU(16, 1000)
	e.add(1, 1)
	e.add(2, 2)
	e.add(3, 3)
	require.Equal(t, int64(6), e.roomLeft(4))
}

func TestSampledLFUSample(t *testing.T) {
	e := newSampledLFU(16, 1000)
	e.add(4, 4)
	e.add(5, 5)
	sample := e.fillSample([]*policyPair{
		{1, 1},
		{2, 2},
		{3, 3},
	})
	k := sample[len(sample)-1].key
	require.Equal(t, 5, len(sample))
	require.NotEqual(t, 1, k)
	require.NotEqual(t, 2, k)
	require.NotEqual(t, 3, k)
	require.Equal(t, len(sample), len(e.fillSample(sample)))
	e.del(5)
	sample = e.fillSample(sample[:len(sample)-2])
	require.Equal(t, 4, len(sample))
}

func TestTinyLFUIncrement(t *testing.T) {
	a := newTinyLFU(4)
	a.Increment(1)
	a.Increment(1)
	a.Increment(1)
	require.True(t, a.door.Has(1))
	require.Equal(t, int64(2), a.freq.Estimate(1))

	a.Increment(1)
	require.False(t, a.door.Has(1))
	require.Equal(t, int64(1), a.freq.Estimate(1))
}

func TestTinyLFUEstimate(t *testing.T) {
	a := newTinyLFU(8)
	a.Increment(1)
	a.Increment(1)
	a.Increment(1)
	require.Equal(t, int64(3), a.Estimate(1))
	require.Equal(t, int64(0), a.Estimate(2))
}

func TestTinyLFUPush(t *testing.T) {
	a := newTinyLFU(16)
	a.Push([]uint64{1, 2, 2, 3, 3, 3})
	require.Equal(t, int64(1), a.Estimate(1))
	require.Equal(t, int64(2), a.Estimate(2))
	require.Equal(t, int64(3), a.Estimate(3))
	require.Equal(t, int64(6), a.incrs)
}

func TestTinyLFUClear(t *testing.T) {
	a := newTinyLFU(16)
	a.Push([]uint64{1, 3, 3, 3})
	a.clear()
	require.Equal(t, int64(0), a.incrs)
	require.Equal(t, int64(0), a.Estimate(3))
}

func BenchmarkSampledLFUPopulate(b *testing.B) {
	sizes := []struct {
		name string
		n    int
	}{
		{"1K", 1000},
		{"10K", 10000},
		{"100K", 100000},
		{"1M", 1000000},
	}
	for _, s := range sizes {
		b.Run(s.name, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				e := newSampledLFU(int64(s.n*100), int64(s.n*10))
				for i := 0; i < s.n; i++ {
					e.add(uint64(i), 1)
				}
			}
		})
	}
}

func BenchmarkSampledLFUFillSample(b *testing.B) {
	sizes := []struct {
		name string
		n    int
	}{
		{"100", 100},
		{"1K", 1000},
		{"10K", 10000},
		{"100K", 100000},
	}
	for _, s := range sizes {
		b.Run(s.name, func(b *testing.B) {
			e := newSampledLFU(int64(s.n*100), int64(s.n*10))
			for i := 0; i < s.n; i++ {
				e.add(uint64(i), 1)
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				e.fillSample(make([]*policyPair, 0, lfuSample))
			}
		})
	}
}

func BenchmarkPolicyConcurrentHas(b *testing.B) {
	b.ReportAllocs()
	p := newDefaultPolicy[int](1<<20, 1<<20)
	defer p.Close()
	for i := 0; i < 4096; i++ {
		p.Add(uint64(i), 1)
	}

	var workerID atomic.Uint64
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		key := workerID.Add(1) * 2654435761
		for pb.Next() {
			_ = p.Has(key & 4095)
			key++
		}
	})
}

func BenchmarkPolicyConcurrentCost(b *testing.B) {
	b.ReportAllocs()
	p := newDefaultPolicy[int](1<<20, 1<<20)
	defer p.Close()
	for i := 0; i < 4096; i++ {
		p.Add(uint64(i), 1)
	}

	var workerID atomic.Uint64
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		key := workerID.Add(1) * 2654435761
		for pb.Next() {
			_ = p.Cost(key & 4095)
			key++
		}
	})
}

func BenchmarkPolicyConcurrentCap(b *testing.B) {
	b.ReportAllocs()
	p := newDefaultPolicy[int](1<<20, 1<<20)
	defer p.Close()
	for i := 0; i < 4096; i++ {
		p.Add(uint64(i), 1)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = p.Cap()
		}
	})
}

func BenchmarkPolicyAddWithEviction(b *testing.B) {
	b.ReportAllocs()
	p := newDefaultPolicy[int](1<<20, 1024)
	defer p.Close()
	for i := 0; i < 1024; i++ {
		p.Add(uint64(i), 1)
	}

	key := uint64(1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Add(key, 1)
		key++
	}
}

func processPolicyFrequencyBatches(b *testing.B, p *defaultPolicy[int], batch []uint64) {
	for i := 0; i < b.N; i++ {
		p.itemsCh <- batch
	}
}

func reportPolicyFrequencyThroughput(b *testing.B, batchSize int) {
	elapsed := b.Elapsed().Seconds()
	if elapsed > 0 {
		b.ReportMetric(float64(b.N*batchSize)/elapsed, "frequency-keys/s")
	}
}

func BenchmarkPolicyProcessItems(b *testing.B) {
	p := newDefaultPolicy[int](1<<20, 1024)
	defer p.Close()
	batch := []uint64{1, 2, 3, 4}

	b.SetBytes(int64(len(batch) * 8))
	b.ResetTimer()
	processPolicyFrequencyBatches(b, p, batch)
	b.StopTimer()
	reportPolicyFrequencyThroughput(b, len(batch))
}

func BenchmarkPolicyProcessItemsDuringEviction(b *testing.B) {
	p := newDefaultPolicy[int](1<<20, 1024)
	defer p.Close()
	for i := 0; i < 1024; i++ {
		p.Add(uint64(i), 1)
	}

	var stop atomic.Bool
	var adds atomic.Uint64
	start := make(chan struct{})
	addDone := make(chan struct{})
	go func() {
		defer close(addDone)
		<-start
		key := uint64(1024)
		for !stop.Load() {
			p.Add(key, 1)
			adds.Add(1)
			key++
		}
	}()

	batch := []uint64{1, 2, 3, 4}
	b.SetBytes(int64(len(batch) * 8))
	b.ResetTimer()
	close(start)
	processPolicyFrequencyBatches(b, p, batch)
	b.StopTimer()

	stop.Store(true)
	<-addDone

	reportPolicyFrequencyThroughput(b, len(batch))
	elapsed := b.Elapsed().Seconds()
	if elapsed > 0 {
		b.ReportMetric(float64(adds.Load())/elapsed, "add-calls/s")
	}
}
