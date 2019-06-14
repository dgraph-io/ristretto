package ristretto

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/dgraph-io/ristretto/ring"
)

func TestMeta(t *testing.T) {
	keys := []string{"1", "2", "3", "4", "5"}
	data := []uint64{4, 3, 2, 1, 0}
	elem := make([]ring.Element, len(keys))
	for i := range elem {
		elem[i] = ring.Element{&keys[i], &data[i]}
	}

	m := NewMeta(4)
	m.Push(elem)

	spew.Dump(m.Victim())

	spew.Dump(m.tracking)
}
