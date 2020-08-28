package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/dgraph-io/ristretto/z"
)

type S struct {
	key  uint64
	val  []byte
	next *S
	inGo bool
}

var (
	ssz      = int(unsafe.Sizeof(S{}))
	lo, hi   = int64(1 << 30), int64(16 << 30)
	increase = true
	stop     int32
	fill     []byte
	maxMB    = 32
)

func newS(sz int) *S {
	var s *S
	if b := z.CallocNoRef(ssz); len(b) > 0 {
		s = (*S)(unsafe.Pointer(&b[0]))
	} else {
		s = &S{inGo: true}
	}
	s.val = z.Calloc(sz)
	copy(s.val, fill)
	if s.next != nil {
		log.Fatalf("news.next must be nil: %p", s.next)
	}
	return s
}

func freeS(s *S) {
	z.Free(s.val)
	if !s.inGo {
		buf := (*[z.MaxArrayLen]byte)(unsafe.Pointer(s))[:ssz:ssz]
		z.Free(buf)
	}
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
	// In normal mode, z.NumAllocBytes would always be zero. So, this program would misbehave.
	curMem := z.NumAllocBytes()
	if increase {
		if curMem > hi {
			increase = false
		}
	} else {
		if curMem < lo {
			increase = true
			runtime.GC()
			time.Sleep(3 * time.Second)
		}
	}
	fmt.Printf("Current Memory: %05.2f G. Increase? %v\n", float64(curMem)/float64(1<<30), increase)
}

func viaLL() {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	root := newS(1)
	for range ticker.C {
		if atomic.LoadInt32(&stop) == 1 {
			break
		}
		if increase {
			root.allocateNext(rand.Intn(maxMB) << 20)
		} else {
			root.deallocNext()
		}
		memory()
	}
	for root.next != nil {
		root.deallocNext()
		memory()
	}
	freeS(root)
}

func viaMap() {
	m := make(map[int][]byte)
	N := 1000000
	for i := 0; i < N; i++ {
		m[i] = nil
	}

	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		if atomic.LoadInt32(&stop) == 1 {
			break
		}
		if increase {
			k := rand.Intn(1000000)
			sz := rand.Intn(maxMB) << 20

			prev := m[k]
			z.Free(prev)

			buf := z.Calloc(sz)
			copy(buf, fill)
			m[k] = buf
		} else {
			for k, val := range m {
				if val != nil {
					z.Free(val)
					m[k] = nil
					break
				}
			}
		}
		memory()
	}
	for k, val := range m {
		delete(m, k)
		z.Free(val)
		memory()
	}
}

func viaList() {
	var slices [][]byte

	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		if atomic.LoadInt32(&stop) == 1 {
			break
		}
		if increase {
			sz := rand.Intn(maxMB) << 20
			buf := z.Calloc(sz)
			copy(buf, fill)
			slices = append(slices, buf)
		} else {
			idx := len(slices) - 1
			z.Free(slices[idx])
			slices = slices[:idx]
		}
		memory()
	}
	for _, val := range slices {
		z.Free(val)
		memory()
	}
	slices = nil
}

func main() {
	if !z.UsingManualMemory() {
		log.Fatalf("Not using manual memory management. Compile with jemalloc.")
	}
	z.StatsPrint()

	fill = make([]byte, maxMB<<20)
	rand.Read(fill)

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("Stopping")
		atomic.StoreInt32(&stop, 1)
	}()
	go func() {
		if err := http.ListenAndServe("0.0.0.0:8080", nil); err != nil {
			log.Fatalf("Error: %v", err)
		}
	}()

	viaLL()
	// viaMap()
	// viaList()
	if left := z.NumAllocBytes(); left != 0 {
		log.Fatalf("Unable to deallocate all memory: %v\n", left)
	}
	runtime.GC()
	fmt.Println("Done. Reduced to zero memory usage.")
	time.Sleep(5 * time.Second)
}
