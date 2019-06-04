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
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Rather than just writing to stdout, we can use a user-defined file location
// for saving benchmark data so we can use stdout for logs.
var PATH = flag.String("path", "stats.csv", "Filepath for benchmark CSV data.")

func init() {
	flag.Parse()
}

// TestMain is the entry point for running this benchmark suite.
func TestMain(m *testing.M) {
	logs := make([]*Log, 0)
	benchmarks := []*Benchmark{{
		"fastCache      ", "get-same", GetSame, 1,
		func() Cache { return NewBenchFastCache(16) },
	}, {
		"bigCache       ", "get-same", GetSame, 1,
		func() Cache { return NewBenchBigCache(16) },
	}, {
		"freeCache      ", "get-same", GetSame, 1,
		func() Cache { return NewBenchFreeCache(16) },
	}, {
		"baseMutex      ", "get-same", GetSame, 1,
		func() Cache { return NewBenchBaseMutex(16) },
	}, {
		"goburrow       ", "get-same", GetSame, 1,
		func() Cache { return NewBenchGoburrow(16) },
	}, {
		"ristretto      ", "get-same", GetSame, 1,
		func() Cache { return NewBenchRistretto(16) },
	}}

	for _, benchmark := range benchmarks {
		log.Printf("running: %s (%s) * %d",
			benchmark.Name, benchmark.Label, benchmark.Para)
		n := time.Now()
		// get testing.BenchMarkResult
		result := testing.Benchmark(benchmark.Bencher(benchmark))
		// append to logs
		logs = append(logs, &Log{benchmark, NewResult(result)})
		log.Printf("\t ... %v\n", time.Since(n))
	}

	// save CSV to disk
	if err := save(logs); err != nil {
		log.Panic(err)
	}
}

func GetSame(benchmark *Benchmark) func(b *testing.B) {
	return func(b *testing.B) {
		cache := benchmark.Create()
		cache.Set("*", []byte("*"))
		b.SetParallelism(benchmark.Para)
		b.SetBytes(1)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				cache.Get("*")
			}
		})
	}
}

// save writes all logs to the PATH file in CSV format.
func save(logs []*Log) error {
	// will hold all log records with the first row being column labels
	records := make([][]string, 0)
	for _, log := range logs {
		records = append(records, log.Record())
	}
	// write csv data
	records = append([][]string{Labels()}, records...)
	// create file for writing
	file, err := os.OpenFile(*PATH, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	// clear contents
	if err = file.Truncate(0); err != nil {
		return err
	}
	defer file.Close()
	return csv.NewWriter(file).WriteAll(records)
}

// Labels returns the column headers of the CSV data. The order is important and
// should correspond with Log.Record().
func Labels() []string {
	return []string{
		"name", "label", "goroutines", "ops", "allocs", "bytes"}
}

// Log is the primary unit of the CSV output files.
type Log struct {
	Benchmark *Benchmark
	Result    *Result
}

// Record generates a CSV record.
func (l *Log) Record() []string {
	return []string{
		strings.Trim(l.Benchmark.Name, " "),
		l.Benchmark.Label,
		fmt.Sprintf("%d", l.Benchmark.Para*l.Result.Procs),
		fmt.Sprintf("%.2f", l.Result.Ops),
		fmt.Sprintf("%d", l.Result.Allocs),
		fmt.Sprintf("%d", l.Result.Bytes),
	}
}

// Benchmark is used to generate benchmarks.
type Benchmark struct {
	// Name is the cache implementation identifier.
	Name string
	// Label is for denoting variations within implementations.
	Label string
	// Bencher is the function for generating testing.B benchmarks for running
	// the actual iterations and collecting runtime information.
	Bencher func(*Benchmark) func(b *testing.B)
	// Para is the multiple of runtime.GOMAXPROCS(0) to use for this benchmark.
	Para int
	// Create is the lazily evaluated function for creating new instances of the
	// underlying cache.
	Create func() Cache
}

// Result is a wrapper for testing.BenchmarkResult that adds fields needed for
// our CSV data.
type Result struct {
	// Ops represents millions of operations per second.
	Ops float64
	// Allocs is the number of allocations per iteration.
	Allocs uint64
	// Bytes is the number of bytes allocated per iteration.
	Bytes uint64
	// Procs is the value of runtime.GOMAXPROCS(0) at the time result was
	// recorded.
	Procs int
}

// NewResult extracts the data we're interested in from a BenchmarkResult.
func NewResult(result testing.BenchmarkResult) *Result {
	memops := strings.Trim(strings.Split(result.String(), "\t")[2], " MB/s")
	opsraw, err := strconv.ParseFloat(memops, 64)
	if err != nil {
		log.Panic(err)
	}
	return &Result{
		Ops:    opsraw,
		Allocs: uint64(result.AllocsPerOp()),
		Bytes:  uint64(result.AllocedBytesPerOp()),
		Procs:  runtime.GOMAXPROCS(0),
	}
}
