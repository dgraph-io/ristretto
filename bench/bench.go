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

package main

import (
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

var (
	// PATH is the user-defined location for writing the stats.csv file.
	PATH = flag.String(
		"path",
		"stats.csv",
		"Filepath for benchmark CSV data.",
	)
	// CACHE determines what libraries to include in the benchmarks.
	CACHE = flag.String(
		"cache",
		"ristretto",
		`Libraries to include in the benchmark: either "all" or "ristretto".`,
	)
	// SUITE is the flag determing what collection of benchmarks to run.
	SUITE = flag.String(
		"suite",
		"full",
		`You can chose from the following options:
		"full"  - hit ratio and speed performance
		"hits"  - hit ratio
		"speed" - throughput
		`,
	)
	// PARALLEL is the goroutine multiplier to use for benchmarking performance
	// using a variable number of goroutines.
	PARALLEL = flag.Int(
		"parallel",
		1,
		"The goroutine multiplier (see runtime.GOMAXPROCS()).",
	)
	// CAPACITY is the hard limit of items to keep in a cache.
	CAPACITY = flag.Int(
		"capacity",
		1024,
		`The number of "items" to keep in a cache.`,
	)
)

// Benchmark is used to generate benchmarks.
type Benchmark struct {
	// Name is the cache implementation identifier.
	Name string
	// Label is for denoting variations within implementations.
	Label string
	// Bencher is the function for generating testing.B benchmarks for running
	// the actual iterations and collecting runtime information.
	Bencher func(*Benchmark, chan *Stats) func(*testing.B)
	// Para is the multiple of runtime.GOMAXPROCS(0) to use for this benchmark.
	Para int
	// Create is the lazily evaluated function for creating new instances of the
	// underlying cache.
	Create func() Cache
}

type benchSuite struct {
	label   string
	bencher func(*Benchmark, chan *Stats) func(*testing.B)
}

func NewBenchmarks(kind string, para, capa int, cache *benchCache) []*Benchmark {
	suite := make([]*benchSuite, 0)
	// create the bench suite from the suite param (SUITE flag)
	if kind == "hits" || kind == "full" {
		suite = append(suite, []*benchSuite{
			{"hits-uniform", HitsUniform},
			{"hits-zipf", HitsZipf},
		}...)
	}
	if kind == "speed" || kind == "full" {
		suite = append(suite, []*benchSuite{
			{"get-same", GetSame},
			{"get-zipf", GetZipf},
			{"set-get ", SetGet},
			{"set-same", SetSame},
			{"set-zipf", SetZipf},
		}...)
	}
	// create benchmarks from bench suite
	benchmarks := make([]*Benchmark, len(suite))
	for i := range benchmarks {
		benchmarks[i] = &Benchmark{
			Name:    cache.name,
			Label:   suite[i].label,
			Bencher: suite[i].bencher,
			Para:    para,
			Create:  func() Cache { return cache.create(capa) },
		}
	}
	return benchmarks
}

type benchCache struct {
	name   string
	create func(int) Cache
}

// getBenchCaches() returns a slice of benchCache's depending on the value of
// the include param (which is the *CACHE flag passed from main).
func getBenchCaches(include string) []*benchCache {
	caches := []*benchCache{
		{"ristretto", NewBenchRistretto},
	}
	if include == "ristretto" {
		return caches
	}
	if include == "all" {
		caches = append(caches, []*benchCache{
			{"base-mutex ", NewBenchBaseMutex},
			{"goburrow   ", NewBenchGoburrow},
			{"bigcache   ", NewBenchBigCache},
			{"fastcache  ", NewBenchFastCache},
			{"freecache  ", NewBenchFreeCache},
		}...)
	}
	return caches
}

func init() {
	flag.Parse()
}

// TestMain is the entry point for running this benchmark suite.
func main() {
	var (
		caches     = getBenchCaches(*CACHE)
		logs       = make([]*Log, 0)
		benchmarks = make([]*Benchmark, 0)
	)
	// create benchmark generators for each cache
	for _, cache := range caches {
		benchmarks = append(benchmarks, NewBenchmarks(
			*SUITE, *PARALLEL, *CAPACITY, cache)...)
	}

	for _, benchmark := range benchmarks {
		log.Printf("running: %s (%s) * %d",
			benchmark.Name, benchmark.Label, benchmark.Para)
		n := time.Now()
		stop := make(chan struct{})
		stats := make(chan *Stats)
		var totalHits uint64
		var totalReqs uint64
		go func() {
			for {
				select {
				case s := <-stats:
					totalHits += s.Hits
					totalReqs += s.Reqs
				case <-stop:
					return
				}
			}
		}()
		// get testing.BenchMarkResult
		result := testing.Benchmark(benchmark.Bencher(benchmark, stats))
		stop <- struct{}{}
		// append to logs
		logs = append(logs, &Log{
			benchmark,
			NewResult(result, Stats{
				Reqs: totalReqs,
				Hits: totalHits,
			}),
		})
		log.Printf("\t ... %v\n", time.Since(n))
		runtime.GC()
	}
	// save CSV to disk
	if err := save(logs); err != nil {
		log.Panic(err)
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
	for i := range records {
		for j := range records[i] {
			seg := records[i][j] + ","
			if j != 0 {
				seg = " " + seg
			}
			if j == len(records[i])-1 {
				seg = seg[:len(seg)-1]
			}
			if _, err := file.WriteString(seg); err != nil {
				return err
			}
		}
		if _, err := file.WriteString("\n"); err != nil {
			return err
		}
	}
	return nil
}

// Labels returns the column headers of the CSV data. The order is important and
// should correspond with Log.Record().
func Labels() []string {
	return []string{
		"name       ",
		"label        ",
		"gr",
		"ops  ",
		"ac",
		"byt",
		"hits    ",
		"reqs    ",
		"rate",
	}
}

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
		fmt.Sprintf("%d", l.Benchmark.Para*l.Result.Procs),
		fmt.Sprintf("%5.2f", l.Result.Ops),
		fmt.Sprintf("%02d", l.Result.Allocs),
		fmt.Sprintf("%03d", l.Result.Bytes),
		fmt.Sprintf("%08d", l.Result.Stats.Hits),
		fmt.Sprintf("%08d", l.Result.Stats.Reqs),
		fmt.Sprintf("%6.2f%%",
			100*(float64(l.Result.Stats.Hits)/float64(l.Result.Stats.Reqs))),
	}
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
	// Hit ratio information
	Stats Stats
}

// NewResult extracts the data we're interested in from a BenchmarkResult.
func NewResult(result testing.BenchmarkResult, stats Stats) *Result {
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
		Stats:  stats,
	}
}
