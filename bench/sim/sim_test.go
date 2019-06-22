package sim

import (
	"bytes"
	"testing"
)

func TestZipfian(t *testing.T) {
	s := NewZipfian(1.25, 2, 100)
	for i := 0; i < 100; i++ {
		if _, err := s(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestUniform(t *testing.T) {
	s := NewUniform(100)
	for i := 0; i < 100; i++ {
		if _, err := s(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestParseLirs(t *testing.T) {
	s := NewReader(ParseLirs, bytes.NewReader([]byte{
		'0', '\r', '\n',
		'1', '\r', '\n',
		'2', '\r', '\n',
	}))
	for i := uint64(0); i < 3; i++ {
		v, err := s()
		if err != nil {
			t.Fatal(err)
		}
		if v != i {
			t.Fatal("value mismatch")
		}
	}
}

func TestCollect(t *testing.T) {
	s := NewUniform(100)
	c := Collection(s, 100)
	if len(c) != 100 {
		t.Fatal("collection not full")
	}
}
