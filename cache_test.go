package ristretto

import (
	"testing"
)

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
