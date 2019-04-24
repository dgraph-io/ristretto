package tinylfu

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithSegmentation(t *testing.T) {
	// Minimal
	p := New(3)
	assert.Equal(t, 3, p.capacity)
	assert.Equal(t, 1, p.maxWindow)
	assert.Equal(t, 1, p.maxProtected)

	// Defaults
	p = New(1000)
	assert.Equal(t, 10, p.maxWindow)
	assert.Equal(t, 792, p.maxProtected)

	// Non-default
	p = New(50, WithSegmentation(.8, .5))
	assert.Equal(t, 10, p.maxWindow)
	assert.Equal(t, 20, p.maxProtected)

	// Maximum window
	p = New(1000, WithSegmentation(0, 0))
	assert.Equal(t, 998, p.maxWindow)
	assert.Equal(t, 1, p.maxProtected)

	// Maximum probation
	p = New(1000, WithSegmentation(1, 0))
	assert.Equal(t, 1, p.maxWindow)
	assert.Equal(t, 1, p.maxProtected)

	// Maximum protected
	p = New(1000, WithSegmentation(1, 1))
	assert.Equal(t, 1, p.maxWindow)
	assert.Equal(t, 998, p.maxProtected)
}
