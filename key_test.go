package ristretto

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewClearKey(t *testing.T) {
	c := NewClearKey()
	require.True(t, c != nil)
	require.True(t, len(c.keys) == 0)
}

func TestClearKey_AddKey(t *testing.T) {
	c := NewClearKey()
	c.AddKey("newKey")
	c.AddKey("anotherKey")
	require.True(t, len(c.keys) == 2)
}

func TestClearKey_ListKeys(t *testing.T) {
	firstKey := "firstKey"
	secondKey := "secondKey"

	c := NewClearKey()
	c.AddKey(firstKey)
	c.AddKey(secondKey)
	require.True(t, len(c.ListKeys()) == 2)
	require.True(t, c.ListKeys()[0] == firstKey)
	require.True(t, c.ListKeys()[1] == secondKey)
}

func TestClearKey_DelKey(t *testing.T) {
	firstKey := "firstKey"
	secondKey := "secondKey"
	thirdKey := "thirdKey"

	c := NewClearKey()
	c.AddKey(firstKey)
	c.AddKey(secondKey)
	c.AddKey(thirdKey)

	c.DelKey(secondKey)
	listKey := c.ListKeys()
	require.True(t, len(listKey) == 2)
	require.True(t, listKey[0] == firstKey)
	require.True(t, listKey[1] == thirdKey)

	c.DelKey(firstKey)
	require.True(t, len(c.ListKeys()) == 1)
	require.True(t, c.ListKeys()[0] == thirdKey)

	c.DelKey("nonexistent")
	require.True(t, len(c.ListKeys()) == 1)
	require.True(t, c.ListKeys()[0] == thirdKey)
}
