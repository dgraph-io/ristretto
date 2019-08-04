package ristretto

import (
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPage(t *testing.T) {
	p := newPage(1, 32)
	for i := 0; i < 32*4+1; i++ {
		t.Logf("Slot: %v\n", p.allocateSlot())
	}
	t.Logf("Final: %s\n", p)
	for i := int32(0); i < 32*4; i++ {
		p.releaseSlot(i)
		t.Logf("Final: %s\n", p)
	}
}

func TestArena(t *testing.T) {
	a := NewArena()
	require.Equal(t, int32(8), a.classFor(7).objectSize)
	require.Equal(t, int32(16), a.classFor(9).objectSize)
	require.Equal(t, int32(16), a.classFor(16).objectSize)
	require.Equal(t, int32(32), a.classFor(17).objectSize)
	require.Equal(t, int32(32), a.classFor(31).objectSize)
	require.Equal(t, int32(32), a.classFor(32).objectSize)
	require.Equal(t, int32(1024), a.classFor(513).objectSize)
	require.Equal(t, int32(32*1024), a.classFor(16*1024+1).objectSize)
}

func TestAllocation(t *testing.T) {
	a := NewArena()
	buf, slot := a.Allocate(13)
	require.NotEqual(t, 0, slot)
	rand.Read(buf)
	require.Equal(t, buf, a.Get(slot))

	slots := make(map[uint64]struct{})
	for i := 0; i < 100; i++ {
		buf, slot = a.Allocate(452)
		if slot == 0 {
			pg := a.classFor(452).availablePage()
			t.Logf("avail: %d", pg.availableSlots())
		}
		require.NotEqual(t, 0, slot)
		require.Equal(t, 512, len(buf))
		if _, has := slots[slot]; has {
			require.False(t, has)
		}
		slots[slot] = struct{}{}
		t.Logf("%s", Readable(slot))
	}
}

func BenchmarkAlloc(b *testing.B) {
	a := NewArena()
	var success, fail uint64
	b.RunParallel(func(pb *testing.PB) {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		var ok, not uint64
		for pb.Next() {
			sz := uint32(r.Intn(512))
			_, slot := a.Allocate(sz)
			if slot == 0 {
				not++
			} else {
				ok++
			}
		}
		atomic.AddUint64(&fail, not)
		atomic.AddUint64(&success, ok)
	})
	b.Logf("OK: %d. Fail: %d", success, fail)
}
