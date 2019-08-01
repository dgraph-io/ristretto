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

// sim is a package encapsulating the generation/simulation of keys for
// benchmarking cache implementations.
package sim

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrDone    = errors.New("no more values in the simulator")
	ErrBadLine = errors.New("bad line for trace format")
)

type Simulator func() (uint64, error)

func NewZipfian(s, v float64, n uint64) Simulator {
	u := &sync.Mutex{}
	z := rand.NewZipf(rand.New(rand.NewSource(time.Now().UnixNano())), s, v, n)
	return func() (uint64, error) {
		u.Lock()
		defer u.Unlock()
		return z.Uint64(), nil
	}
}

func NewUniform(n uint64) Simulator {
	u := &sync.Mutex{}
	m := int64(n)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return func() (uint64, error) {
		u.Lock()
		defer u.Unlock()
		return uint64(r.Int63n(m)), nil
	}
}

type Parser func(string, error) ([]uint64, error)

func NewReader(parser Parser, file io.Reader) Simulator {
	u := &sync.Mutex{}
	b := bufio.NewReader(file)
	s := make([]uint64, 0)
	i := -1
	var err error
	return func() (uint64, error) {
		u.Lock()
		defer u.Unlock()
		// only parse a new line when we've run out of items
		if i++; i == len(s) {
			// parse sequence from line
			if s, err = parser(b.ReadString('\n')); err != nil {
				return 0, err
			}
			i = 0
		}
		return s[i], nil
	}
}

func ParseLirs(line string, err error) ([]uint64, error) {
	if line != "" {
		// example: "1\r\n"
		key, err := strconv.ParseUint(strings.TrimSpace(line), 10, 64)
		return []uint64{key}, err
	}
	return nil, ErrDone
}

func ParseArc(line string, err error) ([]uint64, error) {
	if line != "" {
		// example: "0 5 0 0\r\n"
		//
		// -  first block: starting number in sequence
		// - second block: number of items in sequence
		// -  third block: ignore
		// - fourth block: global line number (not used)
		cols := strings.Fields(line)
		if len(cols) != 4 {
			return nil, ErrBadLine
		}
		start, err := strconv.ParseUint(cols[0], 10, 64)
		if err != nil {
			return nil, err
		}
		count, err := strconv.ParseUint(cols[1], 10, 64)
		if err != nil {
			return nil, err
		}
		// populate sequence from start to start + count
		seq := make([]uint64, count)
		for i := range seq {
			seq[i] = start + uint64(i)
		}
		return seq, nil
	}
	return nil, ErrDone
}

func Collection(simulator Simulator, size uint64) []uint64 {
	collection := make([]uint64, size)
	for i := range collection {
		collection[i], _ = simulator()
	}
	return collection
}

func StringCollection(simulator Simulator, size uint64) []string {
	collection := make([]string, size)
	for i := range collection {
		n, _ := simulator()
		collection[i] = fmt.Sprintf("%d", n)
	}
	return collection
}
