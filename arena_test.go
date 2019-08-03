package ristretto

import "testing"

func TestPage(t *testing.T) {
	p := newPage(1, 32)
	for i := 0; i < 32*4+1; i++ {
		t.Logf("Slot: %v\n", p.allocateSlot())
	}
	t.Logf("Final: %s\n", p)
	for i := 0; i < 32*4; i++ {
		p.releaseSlot(i)
		t.Logf("Final: %s\n", p)
	}
}
