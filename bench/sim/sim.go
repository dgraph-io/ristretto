/*
 * Copyright 2019 Dgraph Labs, Inc. and Contributors
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

package sim

import (
	"bufio"
	"io"
	"math/rand"
	"strconv"
	"time"
)

type Simulator func() (uint64, error)

func NewZipfian(s, v float64, n uint64) Simulator {
	z := rand.NewZipf(rand.New(rand.NewSource(time.Now().UnixNano())), s, v, n)
	return func() (uint64, error) {
		return z.Uint64(), nil
	}
}

func NewUniform(n uint64) Simulator {
	m := int64(n)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return func() (uint64, error) {
		return uint64(r.Int63n(m)), nil
	}
}

type Parser func(string, error) (uint64, error)

func NewReader(parser Parser, file io.Reader) Simulator {
	b := bufio.NewReader(file)
	return func() (uint64, error) {
		return parser(b.ReadString('\n'))
	}
}

func ParseLirs(line string, err error) (uint64, error) {
	if line != "" {
		// example: "1\r\n"
		return strconv.ParseUint(line[:len(line)-2], 10, 64)
	}
	return 0, ErrDone
}

func Collection(simulator Simulator, size uint64) []uint64 {
	collection := make([]uint64, size)
	for i := range collection {
		collection[i], _ = simulator()
	}
	return collection
}
