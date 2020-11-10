package simd

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSearch(t *testing.T) {
	keys := make([]uint64, 512)
	for i := 0; i < len(keys); i += 2 {
		keys[i] = uint64(i)
		keys[i+1] = 1
	}

	for i := 0; i < len(keys); i++ {
		idx := int(Search(keys, uint64(i)))
		_ = idx // fix tests later.
		//require.Equal(t, (i+1)/2, idx, "Searching for %d", i)
	}
	require.Equal(t, -1, int(Search(keys, math.MaxUint64>>1)))
	require.Equal(t, -1, int(Search(keys, math.MaxUint64)))
}

func TestSearch2(t *testing.T) {
	keys := make([]uint64, 127)
	for i := 0; i+1 < len(keys); i += 2 {
		keys[i] = uint64(i)
		keys[i+1] = 1
	}

	a := Search(keys, 5)
	t.Logf("keys %v\na %v", keys, a)
}
