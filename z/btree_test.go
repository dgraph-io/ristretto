package z

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func setPageSize(sz int) {
	pageSize = sz
	maxKeys = (pageSize / 16) - 1
}
func TestTree(t *testing.T) {
	f, err := ioutil.TempFile(".", "tree")
	require.NoError(t, err)
	defer os.Remove(f.Name())

	mf, err := OpenMmapFileUsing(f, 1<<30, true)
	if err != NewFile {
		require.NoError(t, err)
	}
	defer mf.Close(0)

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
		f, err := ioutil.TempFile(".", "tree")
		require.NoError(t, err)
		defer os.Remove(f.Name())

		mf, err := OpenMmapFileUsing(f, 1<<30, true)
		if err != NewFile {
			require.NoError(t, err)
		}
		defer mf.Close(0)

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
		f, err := ioutil.TempFile(".", "tree")
		require.NoError(b, err)
		defer os.Remove(f.Name())

		mf, err := OpenMmapFileUsing(f, 1<<30, true)
		if err != NewFile {
			require.NoError(b, err)
		}
		defer mf.Close(0)

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

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			k := uint64(rand.Intn(2 * N))
			v, ok := mp[k]
			_, _ = v, ok
		}
	})

	f, err := ioutil.TempFile(".", "tree")
	require.NoError(b, err)
	defer os.Remove(f.Name())

	mf, err := OpenMmapFileUsing(f, 1<<30, true)
	if err != NewFile {
		require.NoError(b, err)
	}
	defer mf.Close(0)

	bt := NewTree(mf)
	for i := 0; i < N; i++ {
		k := uint64(rand.Intn(2*N)) + 1
		bt.Set(k, k)
	}
	np := bt.NumPages()
	fmt.Printf("Num pages: %d Size: %d\n", np, np*pageSize)
	fmt.Println("Writes done")

	b.Run("btree", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			k := uint64(rand.Intn(2*N)) + 1
			v := bt.Get(k)
			_ = v
		}
	})
}
