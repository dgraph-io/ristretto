package ristretto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type CacheItem struct {
	Key  uint64
	Hits uint64
}

func (p CacheItem) Less(other *CacheItem) bool {
	return p.Hits < other.Hits
}

func TestMinHeap(t *testing.T) {
	heap := NewMinHeap[CacheItem]()

	// Test insertion
	heap.Insert(&CacheItem{100, 30})
	heap.Insert(&CacheItem{200, 25})

	peek, _ := heap.Peek()
	require.Equal(t, uint64(25), peek.Hits, "Peek returned incorrect item")

	heap.Insert(&CacheItem{300, 35})
	heap.Insert(&CacheItem{400, 20})

	require.Equalf(t, 4, heap.Size(), "Expected heap size 4, got %d", heap.Size())

	// Test extraction
	expectedHits := []uint64{20, 25, 30, 35}
	for i, expectedHit := range expectedHits {
		item, ok := heap.Extract()
		require.Truef(t, ok, "Failed to extract item %d", i)
		require.Equalf(t, expectedHit, item.Hits, "Expected hit %d, got %d", expectedHit, item.Hits)
	}

	// Test empty heap
	_, ok := heap.Extract()
	require.False(t, ok, "Expected false when extracting from empty heap")
}
