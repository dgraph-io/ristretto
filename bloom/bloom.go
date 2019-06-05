package bloom

import (
	"sync"
	"sync/atomic"

	"github.com/dgraph-io/ristretto/ring"
)

type Counter struct {
	sync.Mutex
	data   map[string]*uint64
	sample uint64
}

func NewCounter(sample uint64) *Counter {
	return &Counter{
		data:   make(map[string]*uint64),
		sample: sample,
	}
}

func (c *Counter) update(key string) {
	if counter, exists := c.data[key]; exists {
		*counter++
		return
	}
	// make a new counter for this key
	counter := uint64(1)
	c.data[key] = &counter
}

// Push fulfills the ring.Consumer interface for BP-Wrapper batched updates.
func (c *Counter) Push(keys []ring.Element) {
	c.Lock()
	defer c.Unlock()
	for _, key := range keys {
		c.update(string(key))
	}
}

func (c *Counter) Evict() string {
	c.Lock()
	defer c.Unlock()
	victim := struct {
		key   string
		count uint64
	}{}
	i := uint64(0)
	for key, counter := range c.data {
		count := atomic.LoadUint64(counter)
		if i == 0 || count < victim.count {
			victim.key, victim.count = key, count
		}
		if i == c.sample {
			break
		}
		i++
	}
	// delete victim and return key
	delete(c.data, victim.key)
	return victim.key
}

////////////////////////////////////////////////////////////////////////////////

type Clock struct{}
