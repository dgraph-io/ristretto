package z

import (
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTree(t *testing.T) {
	f, err := ioutil.TempFile(".", "tree")
	require.NoError(t, err)
	defer os.Remove(f.Name())

	mf, err := OpenMmapFileUsing(f, 1<<30, true)
	if err != NewFile {
		require.NoError(t, err)
	}
	defer mf.Close(0)

	bt := NewTree(mf, 8)
	bt.Print()

	N := uint64(256 * 256)
	for i := uint64(1); i < N; i++ {
		bt.Set(i, i)
	}
	for i := uint64(1); i < N; i++ {
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

		bt := NewTree(mf, 8)

		N := uint64(256 * 256)
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
	defer func() {
		pageSize = os.Getpagesize()
		maxKeys = (pageSize / 16) - 1
	}()
	pageSize = 16 << 4
	maxKeys = (pageSize / 16) - 1
	setAndGet()
}

func TestNode(t *testing.T) {
	n := node(make([]byte, pageSize))
	for i := uint64(1); i < 16; i *= 2 {
		n.set(i, i)
	}
	n.print(0)
	require.True(t, math.MaxUint64 == n.get(5))
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
