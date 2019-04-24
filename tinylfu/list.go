package tinylfu

// list implements a doubly linked list. It is based on Go's built-in list.List,
// but simplified for TinyLFU to reduce size per element and to remove the need
// for allocations when moving elements between lists. Unlike the built-in list,
// this struct must be initialized prior to use.
type list struct {
	// To simplify the implementation, internally a list l is implemented as a
	// ring, such that root is both the next element of the l.Back() and the
	// previous element of l.Front()
	root element

	// Current list length excluding the root.
	len int
}

// newList returns an initialized list.
func newList() *list { return new(list).Init() }

// Init initializes or clears the list.
func (l *list) Init() *list {
	l.root.next = &l.root
	l.root.prev = &l.root
	l.len = 0
	return l
}

// Len returns the number of elements in the list.
func (l *list) Len() int { return l.len }

// Front returns the first element of the list or nil if the list is empty.
func (l *list) Front() *element {
	if l.len == 0 {
		return nil
	}
	return l.root.next
}

// Back returns the last element of the list or nil if the list is empty.
func (l *list) Back() *element {
	if l.len == 0 {
		return nil
	}
	return l.root.prev
}

// PushFront inserts an element into a list.
func (l *list) PushFront(e *element) {
	if e.list != nil {
		e.Remove()
	}
	e.next = l.root.next
	e.prev = &l.root
	l.root.next = e
	e.next.prev = e
	e.list = l
	l.len++
}

func (l *list) PushBack(e *element) {
	if e.list != nil {
		e.Remove()
	}
	e.prev = l.root.prev
	e.next = &l.root
	l.root.prev = e
	e.prev.next = e
	e.list = l
	l.len++
}

// element is a node within a linked list.
type element struct {
	next, prev *element
	list       *list

	Value uint64
}

// Next returns the next list element or nil.
func (e *element) Next() *element {
	if p := e.next; e.list != nil && p != &e.list.root {
		return p
	}
	return nil
}

// Prev returns the previous list element or nil.
func (e *element) Prev() *element {
	if p := e.prev; e.list != nil && p != &e.list.root {
		return p
	}
	return nil
}

// List returns the list containing the element or nil.
func (e *element) List() *list {
	return e.list
}

// Remove removes an element from its list.
func (e *element) Remove() {
	if e.list == nil {
		return
	}

	e.list.len--
	e.prev.next = e.next
	e.next.prev = e.prev
	e.next = nil
	e.prev = nil
	e.list = nil
}

// MoveToFront moves an element to the front of its list. The element must be
// initialized and inserted into a list.
func (e *element) MoveToFront() {
	root := &e.list.root
	if root.next == e {
		return
	}

	e.prev.next = e.next
	e.next.prev = e.prev
	e.prev = root
	e.next = root.next
	root.next.prev = e
	root.next = e
}
