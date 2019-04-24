package tinylfu

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUniformSaturation(t *testing.T) {
	p := newTestPolicy(t)
	for i := 0; i < 20; i++ {
		p.Record(uint64(i))
	}

	checkData(t, p, []uint64{12, 13, 14, 15, 16, 17, 18, 19})
	checkSegment(t, p.window, []uint64{19, 18})
	checkSegment(t, p.probation, []uint64{17, 16, 15, 14, 13, 12})
	checkSegment(t, p.protected, nil)
}

func TestTinyLFU(t *testing.T) {
	p := newTestPolicy(t)

	// Saturate the window and probation segments.
	for i := 0; i < 8; i++ {
		p.Record(uint64(i))
	}

	// Access some probation, but don't evict or demote anything yet.
	for i := 0; i < 4; i++ {
		p.Record(uint64(i))
	}

	checkData(t, p, []uint64{0, 1, 2, 3, 4, 5, 6, 7})
	checkSegment(t, p.window, []uint64{7, 6})
	checkSegment(t, p.probation, []uint64{5, 4})
	checkSegment(t, p.protected, []uint64{3, 2, 1, 0})

	// Refresh something in the protected region and promote something from probation.
	p.Record(2)
	p.Record(5) // Demote 0

	checkData(t, p, []uint64{0, 1, 2, 3, 4, 5, 6, 7})
	checkSegment(t, p.window, []uint64{7, 6})
	checkSegment(t, p.probation, []uint64{0, 4})
	checkSegment(t, p.protected, []uint64{5, 2, 3, 1})

	// Evict a few values.
	for i := 10; i < 13; i++ {
		p.Record(uint64(i))
	}

	checkData(t, p, []uint64{1, 2, 3, 5, 7, 10, 11, 12})
	checkSegment(t, p.window, []uint64{12, 11})
	checkSegment(t, p.probation, []uint64{10, 7})
	checkSegment(t, p.protected, []uint64{5, 2, 3, 1})

	// Finally, promote a window value.
	p.Record(11)

	checkData(t, p, []uint64{1, 2, 3, 5, 7, 10, 11, 12})
	checkSegment(t, p.window, []uint64{11, 12})
	checkSegment(t, p.probation, []uint64{10, 7})
	checkSegment(t, p.protected, []uint64{5, 2, 3, 1})
}

func newTestPolicy(t *testing.T) *Policy {
	// Create a policy with 2 window, 2 probation, 4 protected slots. This is
	// enough to fully exercise most cases without being onerous to validate
	// comprehensively.
	return New(8, WithSegmentation(.75, .67))
}

// Verify a policy's data map contains the given keys in any order.
func checkData(t *testing.T, p *Policy, values []uint64) {
	t.Helper()
	if !assert.Equal(t, len(values), len(p.data), "data size") {
		return
	}

	for _, v := range values {
		e, ok := p.data[v]
		if assert.True(t, ok, "key %d exists", v) {
			assert.Equal(t, v, e.Value, "entry node matches key")
		}
	}
}

// Verify a segment contains the given values in order.
func checkSegment(t *testing.T, l *list, values []uint64) {
	t.Helper()
	if !assert.Equal(t, len(values), l.Len(), "segment size") {
		return
	}

	node := l.Front()
	for _, v := range values {
		assert.Equal(t, v, node.Value)
		node = node.Next()
	}
}
