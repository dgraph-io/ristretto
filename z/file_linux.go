package z

import (
	"fmt"
)

// Truncate would truncate the mmapped file to the given size. On Linux and
// others, we could directly just truncate the underlying file, but in Windows,
// we can't do that. So, unmap first, then truncate, then re-map.
func (m *MmapFile) Truncate(maxSz int64) error {
	if err := m.Sync(); err != nil {
		return fmt.Errorf("while sync file: %s, error: %v\n", m.Fd.Name(), err)
	}
	if err := m.Fd.Truncate(maxSz); err != nil {
		return fmt.Errorf("while truncate file: %s, error: %v\n", m.Fd.Name(), err)
	}

	var err error
	m.Data, err = mremap(m.Data, int(maxSz)) // Mmap up to max size.
	return err
}
