package z

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"sort"
	"testing"

	"github.com/dustin/go-humanize"
	"github.com/stretchr/testify/require"
)

var tmp int

func setPageSize(sz int) {
	pageSize = sz
	maxKeys = (pageSize / 16) - 1
}

func createMmapFile(t require.TestingT, sz int) (*MmapFile, *os.File) {
	f, err := ioutil.TempFile(".", "tree")
	require.NoError(t, err)
	mf, err := OpenMmapFileUsing(f, 1<<30, true)
	if err != NewFile {
		require.NoError(t, err)
	}
	return mf, f
}

func cleanup(mf *MmapFile, f *os.File) {
	mf.Close(0)
	os.Remove(f.Name())
}
func TestTree(t *testing.T) {
	mf, f := createMmapFile(t, 1<<30)
	defer cleanup(mf, f)

	bt := NewTree(mf)
	bt.Print()

	N := uint64(256 * 256)
	for i := uint64(1); i < N; i++ {
		bt.Set(i, i)
	}
	for i := uint64(1); i < N; i++ {
		require.Equal(t, i, bt.Get(i))
	}

	bt.DeleteBelow(100)
	for i := uint64(1); i < 100; i++ {
		require.Equal(t, uint64(0), bt.Get(i))
	}
	for i := uint64(100); i < N; i++ {
		require.Equal(t, i, bt.Get(i))
	}

	bt.Print()
}

func TestTreeBasic(t *testing.T) {
	setAndGet := func() {
		mf, f := createMmapFile(t, 1<<30)
		defer cleanup(mf, f)

		bt := NewTree(mf)

		N := uint64(1 << 20)
		mp := make(map[uint64]uint64)
		for i := uint64(1); i < N; i++ {
			key := uint64(rand.Int63n(1<<60) + 1)
			bt.Set(key, key)
			mp[key] = key
		}
		for k, v := range mp {
			require.Equal(t, v, bt.Get(k))
		}
	}
	setAndGet()
	defer setPageSize(os.Getpagesize())
	setPageSize(16 << 5)
	setAndGet()
}

func TestNode(t *testing.T) {
	n := node(make([]byte, pageSize))
	for i := uint64(1); i < 16; i *= 2 {
		n.set(i, i)
	}
	n.print(0)
	require.True(t, 0 == n.get(5))
	n.set(5, 5)
	n.print(0)
}

func TestNodeBasic(t *testing.T) {
	n := node(make([]byte, pageSize))
	N := uint64(256)
	mp := make(map[uint64]uint64)
	for i := uint64(1); i < N; i++ {
		key := uint64(rand.Int63n(1<<60) + 1)
		n.set(key, key)
		mp[key] = key
	}
	for k, v := range mp {
		require.Equal(t, v, n.get(k))
	}
}

func TestNode_MoveRight(t *testing.T) {
	n := node(make([]byte, pageSize))
	N := uint64(10)
	for i := uint64(1); i < N; i++ {
		n.set(i, i)
	}
	n.moveRight(5)
	n.iterate(func(n node, i int) {
		if i < 5 {
			require.Equal(t, uint64(i+1), n.key(i))
			require.Equal(t, uint64(i+1), n.val(i))
		} else if i > 5 {
			require.Equal(t, uint64(i), n.key(i))
			require.Equal(t, uint64(i), n.val(i))
		}
	})
}

func TestNodeCompact(t *testing.T) {
	n := node(make([]byte, pageSize))
	n.setBit(bitLeaf)
	N := uint64(256)
	mp := make(map[uint64]uint64)
	for i := uint64(1); i < N; i++ {
		key := uint64(rand.Int63n(1<<60) + 1)
		val := uint64(0)
		if i%2 == 0 {
			val = key / 2
			mp[key] = val
		}
		n.set(key, val)
	}
	require.Equal(t, 255/2, n.compact())
	for k, v := range mp {
		require.Equal(t, v, n.get(k))
	}
}

func BenchmarkWrite(b *testing.B) {
	b.Run("map", func(b *testing.B) {
		mp := make(map[uint64]uint64)
		for n := 0; n < b.N; n++ {
			k := rand.Uint64()
			mp[k] = k
		}
	})
	b.Run("btree", func(b *testing.B) {
		mf, f := createMmapFile(b, 1<<30)
		defer cleanup(mf, f)

		bt := NewTree(mf)
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			k := rand.Uint64()
			bt.Set(k, k)
		}
	})
}

func BenchmarkRead(b *testing.B) {
	N := 10 << 20
	mp := make(map[uint64]uint64)
	for i := 0; i < N; i++ {
		k := uint64(rand.Intn(2*N)) + 1
		mp[k] = k
	}
	b.Run("map", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			k := uint64(rand.Intn(2 * N))
			v, ok := mp[k]
			_, _ = v, ok
		}
	})

	mf, f := createMmapFile(b, 1<<30)
	defer cleanup(mf, f)

	bt := NewTree(mf)
	for i := 0; i < N; i++ {
		k := uint64(rand.Intn(2*N)) + 1
		bt.Set(k, k)
	}
	np := bt.NumPages()
	fmt.Printf("Num pages: %d Size: %s\n", np, humanize.IBytes(uint64(np*pageSize)))
	fmt.Println("Writes done.")

	b.Run("btree", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			k := uint64(rand.Intn(2*N)) + 1
			v := bt.Get(k)
			_ = v
		}
	})
}

func BenchmarkSearch(b *testing.B) {
	linear := func(n node, k uint64, N int) int {
		for i := 0; i < N; i++ {
			if ki := n.key(i); ki >= k {
				return i
			}
		}
		return N
	}
	binary := func(n node, k uint64, N int) int {
		return sort.Search(N, func(i int) bool {
			return n.key(i) >= k
		})
	}

	for sz := 1; sz < 256; sz *= 2 {
		f, err := ioutil.TempFile(".", "tree")
		require.NoError(b, err)

		mf, err := OpenMmapFileUsing(f, pageSize, true)
		if err != NewFile {
			require.NoError(b, err)
		}

		n := node(mf.Data)
		for i := 1; i <= sz; i++ {
			n.set(uint64(i), uint64(i))
		}

		b.Run(fmt.Sprintf("linear-%d", sz), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				tmp = linear(n, uint64(sz), sz)
			}
		})
		b.Run(fmt.Sprintf("binary-%d", sz), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				tmp = binary(n, uint64(sz), sz)
			}
		})
		mf.Close(0)
		os.Remove(f.Name())
	}
}
