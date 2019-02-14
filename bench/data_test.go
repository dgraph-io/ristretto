package bench

import (
	"math/rand"
	"strconv"
	"time"
)

const (
	// zipf distribution parameters
	s = 2.0
	v = 10.0

	maxEntrySize     = 10
	workloadDataSize = 2 << 5
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func initPatternZipf(length int) [][]byte {
	source := rand.NewSource(time.Now().UnixNano())
	rv := rand.New(source)
	items := uint64(length / 3)
	z := rand.NewZipf(rv, s, v, items)

	data := make([][]byte, length)
	for i := 0; i < length; i++ {
		data[i] = []byte(strconv.Itoa(int(z.Uint64())))
	}

	return data
}

func initPatternHotKey(length int) [][]byte {
	v := rand.Int() % length
	data := make([][]byte, length)
	for i := 0; i < length; i++ {
		data[i] = []byte(strconv.Itoa(v))
	}

	return data
}
