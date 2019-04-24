// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tinylfu

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPushFront(t *testing.T) {
	t.Run("PushFront", func(t *testing.T) {
		l := newList()
		assert.Nil(t, l.Front())

		var elements []*element
		for i := 0; i < 3; i++ {
			e := &element{Value: uint64(i)}
			elements = append([]*element{e}, elements...)
			l.PushFront(e)

			assert.Equal(t, e, l.Front())
			checkList(t, l, elements)
		}
	})

	t.Run("ChangeLists", func(t *testing.T) {
		l1, l2 := newList(), newList()
		e := &element{Value: 42}
		l1.PushFront(e)
		l2.PushFront(e)
		checkList(t, l1, nil)
		checkList(t, l2, []*element{e})
	})
}

func TestPushBack(t *testing.T) {
	t.Run("PushFront", func(t *testing.T) {
		l := newList()
		assert.Nil(t, l.Back())

		var elements []*element
		for i := 0; i < 3; i++ {
			e := &element{Value: uint64(i)}
			elements = append(elements, e)
			l.PushBack(e)

			assert.Equal(t, e, l.Back())
			checkList(t, l, elements)
		}
	})

	t.Run("ChangeLists", func(t *testing.T) {
		l1, l2 := newList(), newList()
		e := &element{Value: 42}
		l1.PushBack(e)
		l2.PushBack(e)
		checkList(t, l1, nil)
		checkList(t, l2, []*element{e})
	})
}

func TestRemove(t *testing.T) {
	t.Run("Uninitialized", func(t *testing.T) {
		e := &element{}
		assert.NotPanics(t, func() { e.Remove() })
	})

	t.Run("SingleElement", func(t *testing.T) {
		e := &element{}
		l := newList()
		l.PushFront(e)
		checkList(t, l, []*element{e})
		e.Remove()
		checkList(t, l, nil)
	})

	// Test removal of the head, middle, and tail.
	for i := 0; i < 3; i++ {
		l := newList()
		var elements []*element
		var remove *element

		for ei := 0; ei < 3; ei++ {
			e := &element{Value: uint64(ei)}
			l.PushBack(e)
			if ei == i {
				remove = e
			} else {
				elements = append(elements, e)
			}
		}

		t.Run(fmt.Sprintf("Remove%dOf3", i), func(t *testing.T) {
			remove.Remove()
			assert.Nil(t, remove.prev)
			assert.Nil(t, remove.next)
			assert.Nil(t, remove.list)
			checkList(t, l, elements)
		})
	}
}

func TestMoveToFront(t *testing.T) {
	t.Run("Uninitialized", func(t *testing.T) {
		e := &element{}
		assert.Panics(t, func() { e.MoveToFront() })
	})

	t.Run("SingleElement", func(t *testing.T) {
		e := &element{}
		l := newList()
		l.PushFront(e)
		e.MoveToFront()
		checkList(t, l, []*element{e})
	})

	// Test removal of the head, middle, and tail.
	for i := 0; i < 3; i++ {
		l := newList()
		var elements []*element
		var move *element

		for ei := 0; ei < 3; ei++ {
			e := &element{Value: uint64(ei)}
			l.PushBack(e)
			if ei == i {
				move = e
				elements = append([]*element{e}, elements...)
			} else {
				elements = append(elements, e)
			}
		}

		t.Run(fmt.Sprintf("Move%dOf3", i), func(t *testing.T) {
			move.MoveToFront()
			checkList(t, l, elements)
		})
	}
}

func checkList(t *testing.T, l *list, elements []*element) {
	t.Helper()

	root := &l.root
	if !assert.Equal(t, len(elements), l.Len(), "list length") {
		return
	}

	// Special case: empty list
	if len(elements) == 0 {
		if root.next != nil && root.next != root || root.prev != nil && root.prev != root {
			t.Errorf("l.root.next = %p, l.root.prev = %p; both should both be nil or %p", l.root.next, l.root.prev, root)
		}
		return
	}

	for i, e := range elements {
		assert.Equal(t, l, e.List())

		if i > 0 {
			assert.Equal(t, elements[i-1], e.prev, "internal prev pointer")
			assert.Equal(t, elements[i-1], e.Prev(), "external prev pointer")
		} else {
			assert.Equal(t, root, e.prev, "internal prev pointer")
			assert.Nil(t, e.Prev(), "external prev pointer")
		}

		if i < len(elements)-1 {
			assert.Equal(t, elements[i+1], e.next, "internal next pointer")
			assert.Equal(t, elements[i+1], e.Next(), "external next pointer")
		} else {
			assert.Equal(t, root, e.next, "internal next pointer")
			assert.Nil(t, e.Next(), "external next pointer")
		}
	}
}
