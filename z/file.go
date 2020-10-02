package z

import (
	"encoding/binary"
	"fmt"
	"os"
	"path"

	"github.com/pkg/errors"
)

// MmapFile represents an mmapd file and includes both the buffer to the data
// and the file descriptor.
type MmapFile struct {
	Data []byte
	Fd   *os.File
}

var NewFile = errors.New("Create a new file")

// OpenMmapFile opens an existing file or creates a new file. If the file is
// created, it would truncate the file to maxSz. In both cases, it would mmap
// the file to maxSz and returned it. In case the file is created, z.NewFile is
// returned.
func OpenMmapFile(filename string, flag int, maxSz int) (*MmapFile, error) {
	fd, err := os.OpenFile(filename, flag, 0666)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to open: %s", filename)
	}
	fi, err := fd.Stat()
	if err != nil {
		return nil, errors.Wrapf(err, "cannot stat file: %s", filename)
	}
	fileSize := fi.Size()
	if fileSize > int64(maxSz) {
		return nil, errors.Errorf("file size %d does not match zero or max size %d",
			fileSize, maxSz)
	}
	if err := fd.Truncate(int64(maxSz)); err != nil {
		return nil, errors.Wrapf(err, "error while truncation")
	}
	buf, err := Mmap(fd, true, int64(maxSz)) // Mmap up to max size.
	if err != nil {
		return nil, errors.Wrapf(err, "while mmapping %s with size: %d", fd.Name(), maxSz)
	}

	err = nil
	if fileSize == 0 {
		err = NewFile
		dir, _ := path.Split(filename)
		go SyncDir(dir)
	}
	return &MmapFile{
		Data: buf,
		Fd:   fd,
	}, err
}

// Slice returns the slice at the given offset.
func (m *MmapFile) Slice(offset int) []byte {
	sz := binary.BigEndian.Uint32(m.Data[offset:])
	start := offset + 4
	next := start + int(sz)
	if next > len(m.Data) {
		return []byte{}
	}
	res := m.Data[start:next]
	return res
}

// AllocateSlice allocates a slice of the given size at the given offset.
func (m *MmapFile) AllocateSlice(sz, offset int) ([]byte, int) {
	binary.BigEndian.PutUint32(m.Data[offset:], uint32(sz))
	return m.Data[offset+4 : offset+4+sz], offset + 4 + sz
}

func (m *MmapFile) Sync() error {
	return Msync(m.Data)
}

func (m *MmapFile) Delete() error {
	if err := Munmap(m.Data); err != nil {
		return fmt.Errorf("while munmap file: %s, error: %v\n", m.Fd.Name(), err)
	}
	if err := m.Fd.Truncate(0); err != nil {
		return fmt.Errorf("while truncate file: %s, error: %v\n", m.Fd.Name(), err)
	}
	return os.Remove(m.Fd.Name())
}

func SyncDir(dir string) error {
	df, err := os.Open(dir)
	if err != nil {
		return errors.Wrapf(err, "while opening %s", dir)
	}
	if err := df.Sync(); err != nil {
		return errors.Wrapf(err, "while syncing %s", dir)
	}
	if err := df.Close(); err != nil {
		return errors.Wrapf(err, "while closing %s", dir)
	}
	return nil
}
