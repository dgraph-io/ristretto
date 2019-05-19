package ring

type Buffers struct {
	Config *Config

	rows []*Buffer
	mask uint64
}

func NewBuffers(config *Config) *Buffers {
	buffers := &Buffers{
		Config: config,
		rows:   make([]*Buffer, config.Rows),
		mask:   config.Rows - 1,
	}

	// initialize each row (ring)
	for i := range buffers.rows {
		buffers.rows[i] = NewBuffer(config)
	}

	return buffers
}

func (b *Buffers) Add(element uint64) bool {
	// choose an initial row (stripe) for the adding attempt
	//
	// depending on the content of the element, this should be random (for
	// example, storing hashes in element) - this helps distribute out the
	// contention
	row := element & b.mask

	for attempts := 0; ; attempts++ {
		if b.rows[row].Add(element) {
			return true
		}

		// add attempt failed, so try the next row (stripe)
		row = (row + 1) & b.mask

		// runtime.Gosched()
	}
}
