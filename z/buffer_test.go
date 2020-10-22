/*
 * Copyright 2020 Dgraph Labs, Inc. and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package z

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuffer(t *testing.T) {
	rand.Seed(time.Now().Unix())

	for btype := UseCalloc; btype < UseInvalid; btype++ {
		name := fmt.Sprintf("Using mode %s", btype)
		t.Run(name, func(t *testing.T) {
			var bytesBuffer bytes.Buffer // This is just for verifying result.
			bytesBuffer.Grow(512)

			cBuffer, err := NewBufferWith(512, 4<<30, btype)
			require.Nil(t, err)
			defer cBuffer.Release()

			// Writer small []byte
			var smallBytes [256]byte
			rand.Read(smallBytes[:])
			var bigBytes [1024]byte
			rand.Read(bigBytes[:])

			_, err = cBuffer.Write(smallBytes[:])
			require.NoError(t, err, "unable to write data to page buffer")
			_, err = cBuffer.Write(bigBytes[:])
			require.NoError(t, err, "unable to write data to page buffer")

			// Write data to bytesBuffer also, just to match result.
			bytesBuffer.Write(smallBytes[:])
			bytesBuffer.Write(bigBytes[:])

			require.True(t, bytes.Equal(cBuffer.Bytes(), bytesBuffer.Bytes()))
		})
	}
}

func TestBufferWrite(t *testing.T) {
	rand.Seed(time.Now().Unix())

	for btype := UseCalloc; btype < UseInvalid; btype++ {
		name := fmt.Sprintf("Using mode %s", btype)
		t.Run(name, func(t *testing.T) {
			var wb [128]byte
			rand.Read(wb[:])

			cb, err := NewBufferWith(32, 4<<30, btype)
			require.Nil(t, err)
			defer cb.Release()

			bb := new(bytes.Buffer)

			end := 32
			for i := 0; i < 3; i++ {
				n, err := cb.Write(wb[:end])
				require.NoError(t, err, "unable to write bytes to buffer")
				require.Equal(t, n, end, "length of buffer and length written should be equal")

				// append to bb also for testing.
				bb.Write(wb[:end])

				require.True(t, bytes.Equal(cb.Bytes(), bb.Bytes()), "Both bytes should match")
				end = end * 2
			}
		})
	}
}

func TestBufferAutoMmap(t *testing.T) {
	buf := NewBuffer(1 << 20)
	buf.AutoMmapAfter(64 << 20)

	N := 128 << 10
	var wb [1024]byte
	for i := 0; i < N; i++ {
		rand.Read(wb[:])
		b := buf.SliceAllocate(len(wb))
		copy(b, wb[:])
	}
	t.Logf("Buffer size: %d\n", buf.LenWithPadding())

	buf.SortSlice(func(l, r []byte) bool {
		return bytes.Compare(l, r) < 0
	})
	t.Logf("sort done\n")

	var count int
	var last []byte
	buf.SliceIterate(func(slice []byte) error {
		require.True(t, bytes.Compare(slice, last) >= 0)
		last = append(last[:0], slice...)
		count++
		return nil
	})
	require.Equal(t, N, count)
}

func TestBufferSimpleSort(t *testing.T) {
	buf := NewBuffer(1 << 20)
	defer buf.Release()
	for i := 0; i < 25600; i++ {
		b := buf.SliceAllocate(4)
		binary.BigEndian.PutUint32(b, uint32(rand.Int31n(256000)))
	}
	buf.SortSlice(func(ls, rs []byte) bool {
		left := binary.BigEndian.Uint32(ls)
		right := binary.BigEndian.Uint32(rs)
		return left < right
	})
	var last uint32
	var i int
	buf.SliceIterate(func(slice []byte) error {
		num := binary.BigEndian.Uint32(slice)
		if num < last {
			fmt.Printf("num: %d idx: %d last: %d\n", num, i, last)
		}
		i++
		require.GreaterOrEqual(t, num, last)
		last = num
		// fmt.Printf("Got number: %d\n", num)
		return nil
	})
}

func TestBufferSlice(t *testing.T) {
	for btype := UseCalloc; btype < UseInvalid; btype++ {
		name := fmt.Sprintf("Using mode %s", btype)
		t.Run(name, func(t *testing.T) {
			buf, err := NewBufferWith(0, 0, btype)
			require.Nil(t, err)
			defer buf.Release()

			count := 10000
			exp := make([][]byte, 0, count)

			// Create "count" number of slices.
			for i := 0; i < count; i++ {
				sz := 1 + rand.Intn(8)
				testBuf := make([]byte, sz)
				rand.Read(testBuf)

				newSlice := buf.SliceAllocate(sz)
				require.Equal(t, sz, copy(newSlice, testBuf))

				// Save testBuf for verification.
				exp = append(exp, testBuf)
			}

			compare := func() {
				i := 0
				buf.SliceIterate(func(slice []byte) error {
					// All the slices returned by the buffer should be equal to what we
					// inserted earlier.
					if !bytes.Equal(exp[i], slice) {
						fmt.Printf("exp: %s got: %s\n", hex.Dump(exp[i]), hex.Dump(slice))
						t.Fail()
					}
					require.Equal(t, exp[i], slice)
					i++
					return nil
				})
				require.Equal(t, len(exp), i)
			}
			compare() // same order as inserted.

			t.Logf("Sorting using sort.Slice\n")
			sort.Slice(exp, func(i, j int) bool {
				return bytes.Compare(exp[i], exp[j]) < 0
			})
			t.Logf("Sorting using buf.SortSlice\n")
			buf.SortSlice(func(a, b []byte) bool {
				return bytes.Compare(a, b) < 0
			})
			t.Logf("Done sorting\n")
			compare() // same order after sort.
		})
	}
}

func TestBufferSort(t *testing.T) {
	for btype := UseCalloc; btype < UseInvalid; btype++ {
		name := fmt.Sprintf("Using mode %s", btype)
		t.Run(name, func(t *testing.T) {
			buf, err := NewBufferWith(0, 0, btype)
			require.Nil(t, err)
			defer buf.Release()

			N := 10000

			for i := 0; i < N; i++ {
				newSlice := buf.SliceAllocate(8)
				uid := uint64(rand.Int63())
				binary.BigEndian.PutUint64(newSlice, uid)
			}

			test := func(start, end int) {
				start = padding + 12*start
				end = padding + 12*end
				buf.SortSliceBetween(start, end, func(ls, rs []byte) bool {
					lhs := binary.BigEndian.Uint64(ls)
					rhs := binary.BigEndian.Uint64(rs)
					return lhs < rhs
				})

				slice, next := []byte{}, start
				var last uint64
				var count int
				for next != 0 && next < end {
					slice, next = buf.Slice(next)
					uid := binary.BigEndian.Uint64(slice)
					require.GreaterOrEqual(t, uid, last)
					last = uid
					count++
				}
				require.Equal(t, (end-start)/12, count)
			}
			for i := 10; i <= N; i += 10 {
				test(i-10, i)
			}
			test(0, N)
		})
	}
}
