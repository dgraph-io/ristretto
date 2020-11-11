package simd

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSearchClever(t *testing.T) {
	Search := Clever
	keys := make([]uint64, 512)
	for i := 0; i < len(keys); i += 2 {
		keys[i] = uint64(i)
		keys[i+1] = 1
	}

	for i := 0; i < len(keys); i++ {
		idx := int(Search(keys, uint64(i)))
		require.Equal(t, (i+1)/2, idx, "%v\n%v", i, keys)
	}
	require.Equal(t, 256, int(Search(keys, math.MaxUint64>>1)))
	require.Equal(t, 256, int(Search(keys, math.MaxUint64)))
}

func TestSearchNaive(t *testing.T) {
	Search := Naive
	keys := make([]uint64, 512)
	for i := 0; i < len(keys); i += 2 {
		keys[i] = uint64(i)
		keys[i+1] = 1
	}

	for i := 0; i < len(keys); i++ {
		idx := int(Search(keys, uint64(i)))
		require.Equal(t, (i+1)/2, idx, "%v\n%v", i, keys)
	}
	require.Equal(t, 256, int(Search(keys, math.MaxUint64>>1)))
	require.Equal(t, 256, int(Search(keys, math.MaxUint64)))
}

func Test_cmp2(t *testing.T) {
	a := cmp2_native([2]uint64{2, 1}, [2]uint64{2, 2})
	b := cmp2([2]uint64{2, 2}, [2]uint64{2, 2})
	t.Logf("a %v b %v", a, b)
	//t.Logf("a %v b %v c %v", a, b, c)

	a = cmp2_native([2]uint64{1, 2}, [2]uint64{2, 2})
	b = cmp2([2]uint64{1, 2}, [2]uint64{2, 2})
	t.Logf("a %v b %v", a, b)

	a = cmp2_native([2]uint64{2, 2}, [2]uint64{2, 2})
	b = cmp2([2]uint64{2, 2}, [2]uint64{2, 2})
	t.Logf("a %v b %v", a, b)

	a = cmp2_native([2]uint64{1, 1}, [2]uint64{2, 2})
	b = cmp2([2]uint64{1, 1}, [2]uint64{2, 2})
	t.Logf("a %v b %v", a, b)
}

func Test_cmp4(t *testing.T) {
	a := cmp4([4]uint64{0, 1, 2, 3}, [4]uint64{2, 2, 2, 2})
	b := cmp4([4]uint64{0, 1, 2, 3}, [4]uint64{2, 2, 2, 2})
	t.Logf("a %v b %v", a, b)
}

func Benchmark_cmp2_native(b *testing.B) {
	b.StopTimer()
	twos := [2]uint64{2, 1}
	pks := [2]uint64{2, 2}
	b.ResetTimer()
	b.StartTimer()
	var idx int16
	for i := 0; i < b.N; i++ {
		idx = cmp2_native(twos, pks)
	}
	_ = idx
}

func Benchmark_cmp2_sse(b *testing.B) {
	b.StopTimer()
	twos := [2]uint64{2, 1}
	pks := [2]uint64{2, 2}
	b.ResetTimer()
	b.StartTimer()
	var idx int16
	for i := 0; i < b.N; i++ {
		idx = cmp2(twos, pks)
	}
	_ = idx
}

func Benchmark_cmp4_native(b *testing.B) {
	b.StopTimer()
	fours := [4]uint64{1, 2, 3, 4}
	pk := [4]uint64{2, 2, 2, 2}
	b.ResetTimer()
	b.StartTimer()

	var idx int16
	for i := 0; i < b.N; i++ {
		idx = cmp4_native(fours, pk)
	}
	_ = idx
}

func Benchmark_cmp4_avx2(b *testing.B) {
	b.StopTimer()
	fours := [4]uint64{1, 2, 3, 4}
	pk := [4]uint64{2, 2, 2, 2}
	b.ResetTimer()
	b.StartTimer()

	var idx int16
	for i := 0; i < b.N; i++ {
		idx = cmp4(fours, pk)
	}
	_ = idx
}

func BenchmarkNaive(b *testing.B) {
	b.StopTimer()
	keys := make([]uint64, 512)
	for i := 0; i < len(keys); i += 2 {
		keys[i] = uint64(i)
		keys[i+1] = 1
	}
	b.ResetTimer()
	b.StartTimer()
	var idx int16
	for i := 0; i < b.N; i++ {
		for j := 0; j < len(keys); j++ {
			idx = Naive(keys, uint64(j))
		}
	}
	_ = idx
}

func BenchmarkClever(b *testing.B) {
	b.StopTimer()
	keys := make([]uint64, 512)
	for i := 0; i < len(keys); i += 2 {
		keys[i] = uint64(i)
		keys[i+1] = 1
	}
	b.ResetTimer()
	b.StartTimer()
	var idx int16
	for i := 0; i < b.N; i++ {
		for j := 0; j < len(keys); j++ {
			idx = Clever(keys, uint64(j))
		}
	}
	_ = idx
}
