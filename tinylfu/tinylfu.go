// Package tinylfu is an implementation of the W-TinyLFU caching algorithm.
// See details at http://arxiv.org/abs/1512.00727
package tinylfu

// Policy implements a windowed TinyLFU eviction policy. It is not safe for concurrent access.
type Policy struct {
	data     map[uint64]*element
	admittor AdmissionPolicy
	stats    StatsRecorder

	window    *list
	probation *list
	protected *list

	capacity     int
	maxWindow    int
	maxProtected int
}

// New creates a new TinyLFU cache.
func New(capacity int, opts ...Option) *Policy {
	// Consistent behavior relies on capacity for one element in each segment.
	if capacity < 3 {
		panic("tinylfu: capacity must be positive")
	}

	p := &Policy{
		data:      make(map[uint64]*element),
		window:    newList(),
		probation: newList(),
		protected: newList(),
		capacity:  capacity,
	}

	WithSegmentation(0.99, 0.8)(p)
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Len returns the number of items in the cache.
func (p *Policy) Len() int {
	return p.window.Len() + p.probation.Len() + p.protected.Len()
}

// Record updates the policy when an entry is accessed.
func (p *Policy) Record(key uint64) {
	if p.admittor != nil {
		p.admittor.Record(key)
	}

	node, ok := p.data[key]
	if !ok {
		if p.stats != nil {
			p.stats.RecordMiss()
		}
		p.onMiss(key)
		return
	}

	switch node.List() {
	case p.window, p.protected:
		if p.stats != nil {
			p.stats.RecordHit()
		}
		node.MoveToFront()

	case p.probation:
		if p.stats != nil {
			p.stats.RecordHit()
		}

		// Promote the accessed item to the protected segment.
		p.protected.PushFront(node)

		// Demote the oldest protected item if needed.
		if p.protected.Len() > p.maxProtected {
			p.probation.PushFront(p.protected.Back())
		}
	}
}

// onMiss adds the entry to the admission window, evicting if necessary.
func (p *Policy) onMiss(key uint64) {
	// This assumes maxWindow >= 1 or the following promotion panics.
	if p.window.Len() < p.maxWindow {
		p.insertNew(key)
		return
	}

	candidate := p.window.Back()
	p.probation.PushFront(candidate)

	// This assumes capacity >= 2 or the following eviction panics.
	if len(p.data) < p.capacity {
		p.insertNew(key)
		return
	}

	victim, evict := p.probation.Back(), candidate
	if p.admittor == nil || p.admittor.Admit(candidate.Value, victim.Value) {
		evict = victim
	}

	delete(p.data, evict.Value)
	evict.Value = key
	p.data[key] = evict
	p.window.PushFront(evict)

	if p.stats != nil {
		p.stats.RecordEviction()
	}
}

// insertNew allocates a new element and adds it to the admission window segment.
// This is the only time a node is allocated.
func (p *Policy) insertNew(key uint64) {
	node := &element{Value: key}
	p.window.PushFront(node)
	p.data[key] = node
}
