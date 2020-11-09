package z

import (
	"fmt"
	"reflect"
	"unsafe"

	"golang.org/x/sys/unix"
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

func mremap(data []byte, size int) ([]byte, error) {
	// from <https://github.com/torvalds/linux/blob/f8394f232b1eab649ce2df5c5f15b0e528c92091/include/uapi/linux/mman.h#L8>
	const MREMAP_MAYMOVE = 0x1

	header := (*reflect.SliceHeader)(unsafe.Pointer(&data))
	mmapAddr, mmapSize, errno := unix.Syscall6(
		unix.SYS_MREMAP,
		header.Data,
		uintptr(header.Len),
		uintptr(size),
		uintptr(MREMAP_MAYMOVE),
		0,
		0,
	)
	if errno != 0 {
		return nil, fmt.Errorf("mremap failed with errno: %s", errno)
	}
	if mmapSize != uintptr(size) {
		return nil, fmt.Errorf("mremap size mismatch: requested: %d got: %d", size, mmapSize)
	}

	header.Data = mmapAddr
	header.Cap = size
	header.Len = size
	return data, nil
}
