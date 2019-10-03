package ristretto

import (
	"testing"
	"time"
)

func TestPolicyMetrics(t *testing.T) {
	p := newDefaultPolicy(100, 10)
	p.CollectMetrics(newMetrics())
	if p.stats == nil || p.evict.stats == nil {
		t.Fatal("policy metrics initialization error")
	}
}

func TestPolicyProcessItems(t *testing.T) {
	p := newDefaultPolicy(100, 10)
	p.itemsCh <- []uint64{1, 2, 2}
	time.Sleep(time.Millisecond)
	if p.admit.Estimate(2) != 2 || p.admit.Estimate(1) != 1 {
		t.Fatal("policy processItems not pushing to tinylfu counters")
	}
	p.stop <- struct{}{}
	p.itemsCh <- []uint64{3, 3, 3}
	time.Sleep(time.Millisecond)
	if p.admit.Estimate(3) != 0 {
		t.Fatal("policy processItems not stopping")
	}
}

func TestPolicyPush(t *testing.T) {
	p := newDefaultPolicy(100, 10)
	if !p.Push([]uint64{}) {
		t.Fatal("push empty slice should be good")
	}
}

func TestSampledLFUAdd(t *testing.T) {
	e := newSampledLFU(4)
	e.add(1, 1)
	e.add(2, 2)
	e.add(3, 1)
	if e.used != 4 {
		t.Fatal("used not being incremented")
	}
	if e.keyCosts[2] != 2 {
		t.Fatal("keyCosts not being updated")
	}
}

func TestSampledLFUDel(t *testing.T) {
	e := newSampledLFU(4)
	e.add(1, 1)
	e.add(2, 2)
	e.del(2)
	if e.used != 1 {
		t.Fatal("del not updating used field")
	}
	if _, ok := e.keyCosts[2]; ok {
		t.Fatal("del not deleting value from keyCosts")
	}
	e.del(4)
}

func TestSampledLFUUpdate(t *testing.T) {
	e := newSampledLFU(4)
	e.add(1, 1)
	if !e.updateIfHas(1, 2) {
		t.Fatal("update should be possible")
	}
	if e.used != 2 {
		t.Fatal("update not changing used field")
	}
	if e.updateIfHas(2, 2) {
		t.Fatal("update shouldn't be possible")
	}
}

func TestSampledLFUClear(t *testing.T) {
	e := newSampledLFU(4)
	e.add(1, 1)
	e.add(2, 2)
	e.add(3, 1)
	e.clear()
	if len(e.keyCosts) != 0 || e.used != 0 {
		t.Fatal("clear not deleting keyCosts or zeroing used field")
	}
}

func TestSampledLFURoom(t *testing.T) {
	e := newSampledLFU(16)
	e.add(1, 1)
	e.add(2, 2)
	e.add(3, 3)
	if e.roomLeft(4) != 6 {
		t.Fatal("roomLeft returning wrong value")
	}
}

func TestSampledLFUSample(t *testing.T) {
	e := newSampledLFU(16)
	e.add(4, 4)
	e.add(5, 5)
	sample := e.fillSample([]*policyPair{
		{1, 1},
		{2, 2},
		{3, 3},
	})
	k := sample[len(sample)-1].key
	if len(sample) != 5 || k == 1 || k == 2 || k == 3 {
		t.Fatal("fillSample not filling properly")
	}
	if len(sample) != len(e.fillSample(sample)) {
		t.Fatal("fillSample mutating full sample")
	}
	e.del(5)
	if sample = e.fillSample(sample[:len(sample)-2]); len(sample) != 4 {
		t.Fatal("fillSample not returning sample properly")
	}
}

func TestTinyLFUIncrement(t *testing.T) {
	a := newTinyLFU(4)
	a.Increment(1)
	a.Increment(1)
	a.Increment(1)
	if !a.door.Has(1) {
		t.Fatal("doorkeeper bit not set")
	}
	if a.freq.Estimate(1) != 2 {
		t.Fatal("incorrect counter value")
	}
	a.Increment(1)
	if a.door.Has(1) {
		t.Fatal("doorkeeper bit set after reset")
	}
	if a.freq.Estimate(1) != 1 {
		t.Fatal("counter value not halved after reset")
	}
}

func TestTinyLFUEstimate(t *testing.T) {
	a := newTinyLFU(8)
	a.Increment(1)
	a.Increment(1)
	a.Increment(1)
	if a.Estimate(1) != 3 {
		t.Fatal("estimate value incorrect")
	}
	if a.Estimate(2) != 0 {
		t.Fatal("estimate value should be 0")
	}
}

func TestTinyLFUPush(t *testing.T) {
	a := newTinyLFU(16)
	a.Push([]uint64{1, 2, 2, 3, 3, 3})
	if a.Estimate(1) != 1 || a.Estimate(2) != 2 || a.Estimate(3) != 3 {
		t.Fatal("push didn't increment counters properly")
	}
	if a.incrs != 6 {
		t.Fatal("incrs not being incremented")
	}
}

func TestTinyLFUClear(t *testing.T) {
	a := newTinyLFU(16)
	a.Push([]uint64{1, 3, 3, 3})
	a.clear()
	if a.incrs != 0 || a.Estimate(3) != 0 {
		t.Fatal("clear not clearing")
	}
}
