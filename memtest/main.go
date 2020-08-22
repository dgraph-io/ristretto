package main

import (
	"fmt"
	"log"
	"math/rand"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/dgraph-io/ristretto/z"
)

type S struct {
	key  uint64
	val  []byte
	next *S
}

var (
	ssz      = int(unsafe.Sizeof(S{}))
	lo, hi   = int64(16 << 30), int64(24 << 30)
	increase = true
)

func newS(sz int) *S {
	b := z.Calloc(ssz)
	s := (*S)(unsafe.Pointer(&b[0]))
	s.val = z.Calloc(sz)
	rand.Read(s.val)
	return s
}

func freeS(s *S) {
	z.Free(s.val)
	buf := (*[z.MaxArrayLen]byte)(unsafe.Pointer(s))[:ssz:ssz]
	z.Free(buf)
}

func (s *S) allocateNext(sz int) {
	ns := newS(sz)
	s.next, ns.next = ns, s.next
}

func (s *S) deallocNext() {
	if s.next == nil {
		log.Fatal("next should not be nil")
	}
	next := s.next
	s.next = next.next
	freeS(next)
}

func memory() {
	curMem := atomic.LoadInt64(&z.NumAllocBytes)
	if increase {
		if curMem > hi {
			increase = false
		}
	} else {
		if curMem < lo {
			increase = true
		}
	}
	fmt.Printf("Current Memory: %.2f G. Increase? %v\n", float64(curMem)/float64(1<<30), increase)
}

func viaLL() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	root := newS(1)

	increase := true
	for range ticker.C {
		if increase {
			root.allocateNext(rand.Intn(1024) << 20)
		} else {
			root.deallocNext()
		}
		memory()
	}
}

func viaMap() {
	m := make(map[uint64][]byte)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	increase := true
	for range ticker.C {
		if increase {
			k := rand.Uint64()
			sz := rand.Intn(1024) << 20
			if prev, has := m[k]; has {
				z.Free(prev)
			}
			buf := z.Calloc(sz)
			rand.Read(buf)
			m[k] = buf
		} else {
			for k, val := range m {
				delete(m, k)
				z.Free(val)
				break
			}
		}
		memory()
	}
}

func main() {
	viaLL()
}
