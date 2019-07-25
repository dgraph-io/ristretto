// The MIT License (MIT)
// Copyright (c) 2014 Andreas Briese, eduToolbox@Bri-C GmbH, Sarstedt

// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package z

import (
	"bytes"
	"encoding/json"
	"log"
	"math"
	"reflect"
	"unsafe"

	"github.com/coocood/rtutil"
)

// helper
var mask = []uint8{1, 2, 4, 8, 16, 32, 64, 128}

func getSize(ui64 uint64) (size uint64, exponent uint64) {
	if ui64 < uint64(512) {
		ui64 = uint64(512)
	}
	size = uint64(1)
	for size < ui64 {
		size <<= 1
		exponent++
	}
	return size, exponent
}

func calcSizeByWrongPositives(numEntries, wrongs float64) (uint64, uint64) {
	size := -1 * numEntries * math.Log(wrongs) / math.Pow(float64(0.69314718056), 2)
	locs := math.Ceil(float64(0.69314718056) * size / numEntries)
	return uint64(size), uint64(locs)
}

// New
// returns a new bloomfilter
func New(params ...float64) (bloomfilter Bloom) {
	var entries, locs uint64
	if len(params) == 2 {
		if params[1] < 1 {
			entries, locs = calcSizeByWrongPositives(params[0], params[1])
		} else {
			entries, locs = uint64(params[0]), uint64(params[1])
		}
	} else {
		log.Fatal("usage: New(float64(number_of_entries), float64(number_of_hashlocations)) i.e. New(float64(1000), float64(3)) or New(float64(number_of_entries), float64(number_of_hashlocations)) i.e. New(float64(1000), float64(0.03))")
	}
	size, exponent := getSize(uint64(entries))
	bloomfilter = Bloom{
		sizeExp: exponent,
		size:    size - 1,
		setLocs: locs,
		shift:   64 - exponent,
	}
	bloomfilter.Size(size)
	return bloomfilter
}

// NewWithBoolset
// takes a []byte slice and number of locs per entry
// returns the bloomfilter with a bitset populated according to the input []byte
func NewWithBoolset(bs *[]byte, locs uint64) (bloomfilter Bloom) {
	bloomfilter = New(float64(len(*bs)<<3), float64(locs))
	ptr := uintptr(unsafe.Pointer(&bloomfilter.bitset[0]))
	for _, b := range *bs {
		*(*uint8)(unsafe.Pointer(ptr)) = b
		ptr++
	}
	return bloomfilter
}

// bloomJSONImExport
// Im/Export structure used by JSONMarshal / JSONUnmarshal
type bloomJSONImExport struct {
	FilterSet []byte
	SetLocs   uint64
}

// JSONUnmarshal
// takes JSON-Object (type bloomJSONImExport) as []bytes
// returns bloom32 / bloom64 object
func JSONUnmarshal(dbData []byte) Bloom {
	bloomImEx := bloomJSONImExport{}
	json.Unmarshal(dbData, &bloomImEx)
	buf := bytes.NewBuffer(bloomImEx.FilterSet)
	bs := buf.Bytes()
	bf := NewWithBoolset(&bs, bloomImEx.SetLocs)
	return bf
}

//
// Bloom filter
type Bloom struct {
	bitset  []uint64
	ElemNum uint64
	sizeExp uint64
	size    uint64
	setLocs uint64
	shift   uint64
}

// <--- http://www.cse.yorku.ca/~oz/hash.html
// modified Berkeley DB Hash (32bit)
// hash is casted to l, h = 16bit fragments
// func (bl Bloom) absdbm(b *[]byte) (l, h uint64) {
// 	hash := uint64(len(*b))
// 	for _, c := range *b {
// 		hash = uint64(c) + (hash << 6) + (hash << bl.sizeExp) - hash
// 	}
// 	h = hash >> bl.shift
// 	l = hash << bl.shift >> bl.shift
// 	return l, h
// }

// Add
// set the bit(s) for entry; Adds an entry to the Bloom filter
func (bl *Bloom) Add(entry []byte) {
	hash := AESHash(entry)
	bl.AddHash(hash)
}

// AddAESHash accepts an AES hash of the entry calculated by the caller.
func (bl *Bloom) AddAESHash(hash uint64) {
	h := hash >> bl.shift
	l := hash << bl.shift >> bl.shift
	for i := uint64(0); i < (*bl).setLocs; i++ {
		(*bl).Set((h + i*l) & (*bl).size)
		(*bl).ElemNum++
	}
}

// Has
// check if bit(s) for entry is/are set
// returns true if the entry was added to the Bloom Filter
func (bl Bloom) Has(entry []byte) bool {
	hash := rtutil.AESHash(entry)
	h := hash >> bl.shift
	l := hash << bl.shift >> bl.shift
	for i := uint64(0); i < bl.setLocs; i++ {
		switch bl.IsSet((h + i*l) & bl.size) {
		case false:
			return false
		}
	}
	return true
}

// AddIfNotHas
// Only Add entry if it's not present in the bloomfilter
// returns true if entry was added
// returns false if entry was allready registered in the bloomfilter
func (bl Bloom) AddIfNotHas(entry []byte) (added bool) {
	if bl.Has(entry[:]) {
		return added
	}
	bl.Add(entry[:])
	return true
}

// Size
// make Bloom filter with as bitset of size sz
func (bl *Bloom) Size(sz uint64) {
	(*bl).bitset = make([]uint64, sz>>6)
}

// Clear
// resets the Bloom filter
func (bl *Bloom) Clear() {
	for i, _ := range (*bl).bitset {
		(*bl).bitset[i] = 0
	}
}

// Set
// set the bit[idx] of bitsit
func (bl *Bloom) Set(idx uint64) {
	ptr := unsafe.Pointer(uintptr(unsafe.Pointer(&bl.bitset[idx>>6])) + uintptr((idx%64)>>3))
	*(*uint8)(ptr) |= mask[idx%8]
}

// IsSet
// check if bit[idx] of bitset is set
// returns true/false
func (bl *Bloom) IsSet(idx uint64) bool {
	ptr := unsafe.Pointer(uintptr(unsafe.Pointer(&bl.bitset[idx>>6])) + uintptr((idx%64)>>3))
	r := ((*(*uint8)(ptr)) >> (idx % 8)) & 1
	return r == 1
}

func (bl Bloom) BinaryMarshal() []byte {
	data := make([]byte, 8*5+len(bl.bitset)*8)
	unsafeMarshal(data, uintptr(unsafe.Pointer(&bl.ElemNum)), 40)
	copy(data[40:], sliceU64ToU8(bl.bitset))
	return data
}

func (bl *Bloom) BinaryUnmarshal(data []byte) {
	unsafeUnmarshal(uintptr(unsafe.Pointer(&bl.ElemNum)), 40, data)
	bl.bitset = sliceU8ToU64(data[40:])
	return
}

func sliceU64ToU8(u64Slice []uint64) []byte {
	var u8s []byte
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&u8s))
	hdr.Len = len(u64Slice) * 8
	hdr.Cap = len(u64Slice) * 8
	hdr.Data = uintptr(unsafe.Pointer(&u64Slice[0]))
	return u8s
}

func sliceU8ToU64(u8s []byte) []uint64 {
	var u64s []uint64
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&u64s))
	hdr.Len = len(u8s) / 8
	hdr.Cap = len(u8s) / 8
	hdr.Data = uintptr(unsafe.Pointer(&u8s[0]))
	return u64s
}

func unsafeMarshal(dest []byte, ptr uintptr, n int) {
	var u8s []byte
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&u8s))
	hdr.Len = n
	hdr.Cap = n
	hdr.Data = ptr
	copy(dest, u8s)
}

func unsafeUnmarshal(ptr uintptr, n int, src []byte) {
	var u8s []byte
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&u8s))
	hdr.Len = n
	hdr.Cap = n
	hdr.Data = ptr
	copy(u8s, src)
}
