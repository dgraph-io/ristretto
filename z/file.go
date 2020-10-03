package z

import (
	"encoding/binary"
	"fmt"
	"io"
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

	// If file is empty, truncate it to maxSz.
	if fileSize == 0 {
		if err := fd.Truncate(int64(maxSz)); err != nil {
			return nil, errors.Wrapf(err, "error while truncation")
		}

	} else if fileSize > int64(maxSz) {
		return nil, errors.Errorf("file size %d does not match zero or max size %d",
			fileSize, maxSz)
	} else {
		maxSz = int(fileSize)
	}

	buf, err := Mmap(fd, true, int64(maxSz)) // Mmap up to file size.
	if err != nil {
		return nil, errors.Wrapf(err, "while mmapping %s with size: %d", fd.Name(), fileSize)
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

type mmapReader struct {
	Data   []byte
	offset int
}

func (mr *mmapReader) Read(buf []byte) (int, error) {
	if mr.offset > len(mr.Data) {
		return 0, io.EOF
	}
	n := copy(buf, mr.Data[mr.offset:])
	mr.offset += n
	if n < len(buf) {
		return n, io.EOF
	}
	return n, nil
}

func (m *MmapFile) NewReader(offset int) io.Reader {
	return &mmapReader{
		Data:   m.Data,
		offset: offset,
	}
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
	m.Data = nil
	if err := m.Fd.Truncate(0); err != nil {
		return fmt.Errorf("while truncate file: %s, error: %v\n", m.Fd.Name(), err)
	}
	return os.Remove(m.Fd.Name())
}

// Truncate would truncate the mmapped file to the given size. On Linux and
// others, we could directly just truncate the underlying file, but in Windows,
// we can't do that. So, unmap first, then truncate, then re-map.
func (m *MmapFile) Truncate(maxSz int64) error {
	if err := Munmap(m.Data); err != nil {
		return fmt.Errorf("while munmap file: %s, error: %v\n", m.Fd.Name(), err)
	}
	if err := m.Fd.Truncate(maxSz); err != nil {
		return fmt.Errorf("while truncate file: %s, error: %v\n", m.Fd.Name(), err)
	}
	var err error
	m.Data, err = Mmap(m.Fd, true, maxSz) // Mmap up to max size.
	return err
}

func (m *MmapFile) Close(maxSz int64) error {
	if err := Munmap(m.Data); err != nil {
		return fmt.Errorf("while munmap file: %s, error: %v\n", m.Fd.Name(), err)
	}
	if maxSz >= 0 {
		if err := m.Fd.Truncate(maxSz); err != nil {
			return fmt.Errorf("while truncate file: %s, error: %v\n", m.Fd.Name(), err)
		}
	}
	return m.Fd.Close()
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
