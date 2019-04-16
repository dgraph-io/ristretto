package slru

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	key1, val1 := []byte{1}, []byte("one")
	key2, val2 := []byte{2}, []byte("two")

	c := New(1, 1)
	c.Set(key1, val1)
	c.Set(key2, val2)

	v, ok := c.Get([]byte("missing"))
	assert.Nil(t, v, "Get nonexistent")
	assert.False(t, ok)

	v, ok = c.Get(key1)
	assert.Equal(t, val1, v, "Get first during probation")
	assert.True(t, ok)

	v, ok = c.Get(key1)
	assert.Equal(t, val1, v, "Get first after promotion")
	assert.True(t, ok)

	v, ok = c.Get(key2)
	assert.Equal(t, val2, v, "Get second during probation")
	assert.True(t, ok)

	v, ok = c.Get(key1)
	assert.Equal(t, val1, v, "Get first after demotion")
	assert.True(t, ok)
}

func TestRemove(t *testing.T) {
	key1, val1 := []byte{1}, []byte("one")
	key2, val2 := []byte{2}, []byte("two")

	c := New(1, 1)
	c.Set(key1, val1)
	c.Set(key2, val2)
	c.Get(key1) // Promote the first key.

	c.Remove([]byte("missing")) // Test negatives.

	c.Remove(key1)
	_, ok := c.Get(key1)
	assert.False(t, ok, "Remove from protected segment")

	c.Remove(key2)
	_, ok = c.Get(key2)
	assert.False(t, ok, "Remove from probation segment")
}

func TestEviction(t *testing.T) {

	tests := []struct {
		Name          string
		Gets          []int
		ExpectedEvict int
	}{
		{"all_probational", nil, 0},
		{"half_promoted", []int{0, 1}, 2},
		{"each_accessed_once", []int{0, 1, 2, 3}, 0},
		{"each_accessed_reverse", []int{3, 2, 1, 0}, 3},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			c := New(2, 2)

			var keys [][]byte
			for i := byte(0); i < 4; i++ {
				keys = append(keys, []byte{'k', 'e', 'y', i})
				c.Set(keys[i], []byte{i})
			}

			for _, keyIndex := range tt.Gets {
				c.Get(keys[keyIndex])
			}

			c.Set([]byte("new"), []byte("value"))

			for i, key := range keys {
				val, ok := c.Get(key)
				if i == tt.ExpectedEvict {
					assert.False(t, ok)
				} else {
					assert.True(t, ok)
					assert.Equal(t, val, []byte{byte(i)})
				}
			}
		})
	}
}
