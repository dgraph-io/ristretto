package bench

import (
	"math/rand"
	"sync/atomic"
	"testing"

	"github.com/dgraph-io/caffeine"
)

func runCacheBenchmark(b *testing.B, cache caffeine.Cache,
	data [][]byte, pctWrites uint64) {

	size := len(data)
	mask := size - 1
	rc := uint64(0)

	// initialize cache
	for i := 0; i < size; i++ {
		cache.Set(data[i], []byte("data"))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		index := rand.Int() & mask
		mc := atomic.AddUint64(&rc, 1)

		if pctWrites*mc/100 == pctWrites*(mc-1)/100 {
			for pb.Next() {
				_ = cache.Set(data[index&mask], []byte("data"))
				index = index + 1
			}
		} else {
			for pb.Next() {
				_, _ = cache.Get(data[index&mask])
				index = index + 1
			}
		}
	})
}
