package z

import (
	"io/ioutil"
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
	// TODO: Add a bt.Get and check that it works.
	bt.Print()
}

func TestNode(t *testing.T) {
	n := node(make([]byte, pageSize))
	for i := uint64(1); i < 16; i *= 2 {
		n.set(i, i)
	}
	n.print(0)

	n.set(5, 5)
	n.print(0)
}
