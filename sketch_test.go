package ristretto

import (
	"testing"
)

func TestSketch(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("no panic with bad param numCounters")
		}
	}()
	s := newCmSketch(5)
	if s.mask != 7 {
		t.Fatal("not rounding up to next power of 2")
	}
	newCmSketch(0)
}

func TestSketchIncrement(t *testing.T) {
	s := newCmSketch(16)
	s.Increment(1)
	s.Increment(5)
	s.Increment(9)
	for i := 0; i < cmDepth; i++ {
		if s.rows[i].string() != s.rows[0].string() {
			break
		}
		if i == cmDepth-1 {
			t.Fatal("identical rows, bad seeding")
		}
	}
}

func TestSketchEstimate(t *testing.T) {
	s := newCmSketch(16)
	s.Increment(1)
	s.Increment(1)
	if s.Estimate(1) != 2 {
		t.Fatal("estimate should be 2")
	}
	if s.Estimate(0) != 0 {
		t.Fatal("estimate should be 0")
	}
}

func TestSketchReset(t *testing.T) {
	s := newCmSketch(16)
	s.Increment(1)
	s.Increment(1)
	s.Increment(1)
	s.Increment(1)
	s.Reset()
	if s.Estimate(1) != 2 {
		t.Fatal("reset failed, estimate should be 2")
	}
}

func TestSketchClear(t *testing.T) {
	s := newCmSketch(16)
	for i := 0; i < 16; i++ {
		s.Increment(uint64(i))
	}
	s.Clear()
	for i := 0; i < 16; i++ {
		if s.Estimate(uint64(i)) != 0 {
			t.Fatal("clear failed")
		}
	}
}

func BenchmarkSketchIncrement(b *testing.B) {
	s := newCmSketch(16)
	b.SetBytes(1)
	for n := 0; n < b.N; n++ {
		s.Increment(1)
	}
}

func BenchmarkSketchEstimate(b *testing.B) {
	s := newCmSketch(16)
	s.Increment(1)
	b.SetBytes(1)
	for n := 0; n < b.N; n++ {
		s.Estimate(1)
	}
}
