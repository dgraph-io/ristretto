package bench

import (
	"math/rand"
	"strconv"
	"time"
)

const (
	s = 2.0
	v = 10.0
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func initAccessPatternString(length int) []string {
	source := rand.NewSource(time.Now().UnixNano())
	rv := rand.New(source)
	z := rand.NewZipf(rv, s, v, items)

	ints := make([]string, length)
	for i := 0; i < length; i++ {
		ints[i] = strconv.Itoa(int(z.Uint64()))
	}

	return ints
}

func initAccessPatternBytes(length int) [][]byte {
	source := rand.NewSource(time.Now().UnixNano())
	rv := rand.New(source)
	z := rand.NewZipf(rv, s, v, items)

	ints := make([][]byte, length)
	for i := 0; i < length; i++ {
		ints[i] = []byte(strconv.Itoa(int(z.Uint64())))
	}

	return ints
}
