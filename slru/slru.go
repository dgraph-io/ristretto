package slru

import (
	"container/list"
)

type segment int

const (
	probation segment = iota
	protected
)

// Cache is a segmented LRU cache. It is not safe for concurrent access.
type Cache struct {
	items        map[string]*list.Element
	probation    *list.List
	protected    *list.List
	maxProbation int
	maxProtected int
}

type entry struct {
	key     []byte
	value   []byte
	segment segment
}

// New creates a new SLRU cache.
//
// Segment capacities must be positive.
func New(maxProbation int, maxProtected int) *Cache {
	if maxProbation < 1 || maxProtected < 1 {
		panic("slru: segment capacities must be positive")
	}

	return &Cache{
		items:        make(map[string]*list.Element),
		probation:    list.New(),
		maxProbation: maxProbation,
		protected:    list.New(),
		maxProtected: maxProtected,
	}
}

// Get looks up a key's value from the cache.
func (c *Cache) Get(key []byte) (value []byte, ok bool) {
	keyStr := string(key)
	e, hit := c.items[keyStr]
	if !hit {
		return
	}
	ent := e.Value.(*entry)
	value, ok = ent.value, true

	if ent.segment == protected {
		c.protected.MoveToFront(e)
		return
	}

	// Just promote the entry there's room in the next segment.
	if c.protected.Len() < c.maxProtected {
		c.probation.Remove(e)
		ent.segment = protected
		c.items[keyStr] = c.protected.PushFront(ent)
		return
	}

	// Swap the entry with the oldest protected one in-place to minimize allocations.

	prot := c.protected.Back()
	victim := prot.Value.(*entry)
	victim.segment = probation
	ent.segment = protected
	prot.Value, e.Value = e.Value, prot.Value

	c.protected.MoveToFront(prot)
	c.probation.MoveToFront(e)
	c.items[keyStr] = prot
	c.items[string(victim.key)] = e
	return
}

// Add adds a value to the cache.
func (c *Cache) Add(key []byte, value []byte) {
	keyStr := string(key)
	if _, ok := c.items[keyStr]; ok {
		panic("slru: replacement of existing keys is not supported")
	}

	newItem := entry{key: key, value: value}

	if c.probation.Len() < c.maxProbation || c.Len() < c.maxProbation+c.maxProtected {
		c.items[keyStr] = c.probation.PushFront(&newItem)
		return
	}

	// Reuse the tail item.
	e := c.probation.Back()
	item := e.Value.(*entry)

	delete(c.items, string(item.key))

	*item = newItem
	c.items[keyStr] = e
	c.probation.MoveToFront(e)
}

// Oldest looks up the next value to be removed from the cache. If the cache is
// not at capacity, no value is returned.
//
// This is not counted as an access and therefore does not update recency.
func (c *Cache) Oldest() (value []byte, ok bool) {
	if c.Len() < c.maxProbation+c.maxProtected {
		return
	}
	return c.probation.Back().Value.(*entry).value, true
}

// Len returns the number of items in the cache.
func (c *Cache) Len() int {
	return c.probation.Len() + c.protected.Len()
}

// Remove removes the provided key from the cache.
func (c *Cache) Remove(key []byte) {
	keyStr := string(key)
	e, ok := c.items[keyStr]
	if !ok {
		return
	}

	if e.Value.(*entry).segment == protected {
		c.protected.Remove(e)
	} else {
		c.probation.Remove(e)
	}

	delete(c.items, keyStr)
}
