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

package bench

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

type (
	Cache interface {
		Get(string) interface{}
		Set(string, interface{})
		Del(string)
		Bench() *Stats
	}
	Stats struct {
		Reqs uint64
		Hits uint64
	}
)

// Log is the primary unit of the CSV output files.
type Log struct {
	Benchmark *Benchmark
	Result    *Result
}

// Record generates a CSV record.
func (l *Log) Record() []string {
	return []string{
		l.Benchmark.Name,
		l.Benchmark.Label,
		fmt.Sprintf("%d", l.Benchmark.Para),
		fmt.Sprintf("%d", l.Result.Ops),
		fmt.Sprintf("%d", l.Result.Allocs),
		fmt.Sprintf("%d", l.Result.Bytes),
	}
}

func Labels() []string {
	return []string{
		"name",
		"label",
		"para",
		"ops",
		"allocs",
		"bytes",
	}
}

// Benchmark is used to generate benchmarks.
type Benchmark struct {
	// Name is the cache implementation identifier.
	Name string
	// Label is for denoting variations within implementations.
	Label string
	// Para is the multiple of runtime.GOMAXPROCS(0) to use for this benchmark.
	Para int
	// Create is a lazily evaluated function for creating new instances of the
	// underlying cache.
	Create func() Cache
}

// Result is a wrapper for testing.BenchmarkResult that adds fields needed for
// our CSV data.
type Result struct {
	// Ops is the operations processed per second.
	Ops uint64
	// Allocs is the number of allocations per iteration.
	Allocs uint64
	// Bytes is the number of bytes allocated per iteration.
	Bytes uint64
}

func NewResult(result testing.BenchmarkResult) *Result {
	memops := strings.Trim(strings.Split(result.String(), "\t")[2], " MB/s")
	opsraw, err := strconv.ParseFloat(memops, 64)
	if err != nil {
		panic(err)
	}
	return &Result{
		Ops:    uint64(opsraw*100) * 10000,
		Allocs: uint64(result.AllocsPerOp()),
		Bytes:  uint64(result.AllocedBytesPerOp()),
	}
}

func Save(logs []*Log) error {
	records := make([][]string, len(logs))
	for i := range records {
		records[i] = logs[i].Record()
	}
	records = append([][]string{Labels()}, records...)
	return csv.NewWriter(os.Stdout).WriteAll(records)
}

func TestMain(m *testing.M) {
	benchmarks := []*Benchmark{
		{"fastcache", "", 1, func() Cache { return NewBenchFastCache(16) }},
		{"another", "", 1, func() Cache { return NewBenchFastCache(16) }},
	}
	logs := make([]*Log, 0)
	for _, benchmark := range benchmarks {
		logs = append(logs, &Log{
			Benchmark: benchmark,
			Result: NewResult(
				testing.Benchmark(GetSame(benchmark)),
			),
		})
	}
	Save(logs)
}
