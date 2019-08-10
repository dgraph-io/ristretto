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
	"sync"
	"sync/atomic"
	"testing"
)

var (
	// PATH is the user-defined location for writing the stats.csv file.
	flagPath = flag.String(
		"path",
		"stats.csv",
		"Filepath for benchmark CSV data.",
	)
	// CACHE determines what libraries to include in the benchmarks.
	flagCache = flag.String(
		"cache",
		"ristretto",
		`Libraries to include in the benchmark: either "all" or "ristretto".`,
	)
	// SUITE is the flag determing what collection of benchmarks to run.
	flagSuite = flag.String(
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
	flagParallel = flag.Int(
		"parallel",
		1,
		"The goroutine multiplier (see runtime.GOMAXPROCS()).",
	)
)

// Benchmark is used to generate benchmarks.
type Benchmark struct {
	// Name is the cache implementation identifier.
	Name string
	// Label is for denoting variations within implementations.
	Label string
	// TODO - document and clean
	HitsBencher  func(*Benchmark, *LogCollection) func()
	SpeedBencher func(*Benchmark, *LogCollection) func(*testing.B)
	// Para is the multiple of runtime.GOMAXPROCS(0) to use for this benchmark.
	Para int
	// Create is the lazily evaluated function for creating new instances of the
	// underlying cache.
	Create func(hits bool) Cache
}

func (b *Benchmark) Log() {
	log.Printf("running: %s (%s) * %d", b.Name, b.Label, b.Para)
}

type benchSuite struct {
	label      string
	benchHits  func(*Benchmark, *LogCollection) func()
	benchSpeed func(*Benchmark, *LogCollection) func(*testing.B)
}

func NewBenchmarks(kind string, para, capa int, cache *benchCache) []*Benchmark {
	suite := make([]*benchSuite, 0)
	// create the bench suite from the suite param (SUITE flag)
	if kind == "hits" || kind == "full" {
		suite = append(suite, []*benchSuite{
			{"hits-zipf     ", HitsZipf, nil},
			{"hits-lirs-gli ", HitsLIRS("gli"), nil},
			{"hits-lirs-loop", HitsLIRS("loop"), nil},
			{"hits-arc-ds1  ", HitsARC("ds1"), nil},
			{"hits-arc-p3   ", HitsARC("p3"), nil},
			{"hits-arc-p8   ", HitsARC("p8"), nil},
			{"hits-arc-s3   ", HitsARC("s3"), nil},
		}...)
	}
	if kind == "speed" || kind == "full" {
		suite = append(suite, []*benchSuite{
			{"get-same      ", nil, GetSame},
			{"get-zipf      ", nil, GetZipf},
			{"set-get       ", nil, SetGet},
			{"set-same      ", nil, SetSame},
			{"set-zipf      ", nil, SetZipf},
			{"set-get-zipf  ", nil, SetGetZipf},
		}...)
	}
	// create benchmarks from bench suite
	benchmarks := make([]*Benchmark, len(suite))
	for i := range benchmarks {
		benchmarks[i] = &Benchmark{
			Name:   cache.name,
			Label:  suite[i].label,
			Para:   para,
			Create: func(hits bool) Cache { return cache.create(capa, hits) },
		}
		if suite[i].benchHits != nil {
			benchmarks[i].HitsBencher = suite[i].benchHits
		} else if suite[i].benchSpeed != nil {
			benchmarks[i].SpeedBencher = suite[i].benchSpeed
		}
	}
	return benchmarks
}

type benchCache struct {
	name   string
	create func(int, bool) Cache
}

// getBenchCaches() returns a slice of benchCache's depending on the value of
// the include params (which is the cache/suite flags passed from main).
func getBenchCaches(include, suite string) []*benchCache {
	caches := []*benchCache{
		{"ristretto  ", NewBenchRistretto},
	}
	if include == "ristretto" {
		return caches
	}
	if include == "all" {
		if suite == "hits" {
			// BenchOptimal is not safe for concurrent access, so it's only
			// included if the hit ratio suite is being ran.
			caches = append(caches, []*benchCache{
				{"optimal    ", NewBenchOptimal},
			}...)
		}
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

func main() {
	var (
		caches     = getBenchCaches(*flagCache, *flagSuite)
		logs       = make([]*Log, 0)
		benchmarks = make([]*Benchmark, 0)
	)
	// create benchmark generators for each cache
	for _, cache := range caches {
		benchmarks = append(benchmarks,
			NewBenchmarks(*flagSuite, *flagParallel, capacity, cache)...,
		)
	}
	for _, benchmark := range benchmarks {
		// log the current benchmark to keep user updated
		benchmark.Log()
		// collection of policy logs for hit ratio analysis
		coll := NewLogCollection()

		var result testing.BenchmarkResult
		if benchmark.HitsBencher != nil {
			benchmark.HitsBencher(benchmark, coll)()
		} else if benchmark.SpeedBencher != nil {
			result = testing.Benchmark(benchmark.SpeedBencher(benchmark, coll))
		}
		// append benchmark result to logs
		logs = append(logs, &Log{benchmark, NewResult(result, coll)})
		// clear GC after each benchmark to reduce random effects on the data
		runtime.GC()
	}
	// save logs CSV to disk
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
	file, err := os.OpenFile(*flagPath, os.O_WRONLY|os.O_CREATE, 0666)
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
		"label         ",
		"go",
		" mop/s",
		" ns/op",
		"ac",
		"byt",
		"hits    ",
		"misses  ",
		"  ratio ",
	}
}

// Log is the primary unit of the CSV output files.
type Log struct {
	Benchmark *Benchmark
	Result    *Result
}

// Record generates a CSV record.
func (l *Log) Record() []string {
	var (
		goroutines  string = fmt.Sprintf("%2d", l.Benchmark.Para*l.Result.Procs)
		mOpsPerSec  string = fmt.Sprintf("%6.2f", l.Result.Ops)
		allocsPerOp string = fmt.Sprintf("%02d", l.Result.Allocs)
		bytesPerOp  string = fmt.Sprintf("%03d", l.Result.Bytes)
		nsPerOp     string = fmt.Sprintf("%6d", l.Result.NsOp)
		totalHits   string = fmt.Sprintf("%08d", l.Result.Hits)
		totalMisses string = fmt.Sprintf("%08d", l.Result.Misses)
		hitRatio    string = fmt.Sprintf("%6.2f%%",
			100*(float64(l.Result.Hits)/float64(l.Result.Hits+l.Result.Misses)),
		)
	)
	if l.Benchmark.Label[:4] == "hits" {
		mOpsPerSec = "------"
		allocsPerOp = "--"
		bytesPerOp = "---"
		nsPerOp = "------"
	} else {
		totalHits = "--------"
		totalMisses = "--------"
		hitRatio = "-------"
	}
	return []string{
		l.Benchmark.Name,
		l.Benchmark.Label,
		// throughput stats
		goroutines,
		mOpsPerSec,
		nsPerOp,
		allocsPerOp,
		bytesPerOp,
		// hit ratio stats
		totalHits,
		totalMisses,
		hitRatio,
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
	Procs  int
	Hits   int64
	Misses int64
	NsOp   int64
}

// NewResult extracts the data we're interested in from a BenchmarkResult.
func NewResult(res testing.BenchmarkResult, coll *LogCollection) *Result {
	result := &Result{}
	if res.N == 0 {
		result.Hits = coll.Hits()
		result.Misses = coll.Misses()
		return result
	}
	memops := strings.Trim(strings.Split(res.String(), "\t")[2], " MB/s")
	opsraw, err := strconv.ParseFloat(memops, 64)
	if err != nil {
		log.Panic(err)
	}
	if coll == nil {
		coll = &LogCollection{}
	}
	return &Result{
		Ops:    opsraw,
		Allocs: uint64(res.AllocsPerOp()),
		Bytes:  uint64(res.AllocedBytesPerOp()),
		Procs:  runtime.GOMAXPROCS(0),
		Hits:   coll.Hits(),
		Misses: coll.Misses(),
		NsOp:   res.NsPerOp(),
	}
}

type LogCollection struct {
	sync.Mutex
	Logs []*policyLog
}

func NewLogCollection() *LogCollection {
	return &LogCollection{
		Logs: make([]*policyLog, 0),
	}
}

func (c *LogCollection) Append(plog *policyLog) {
	c.Lock()
	defer c.Unlock()
	c.Logs = append(c.Logs, plog)
}

func (c *LogCollection) Hits() int64 {
	c.Lock()
	defer c.Unlock()
	var sum int64
	for i := range c.Logs {
		sum += c.Logs[i].GetHits()
	}
	return sum
}

func (c *LogCollection) Misses() int64 {
	c.Lock()
	defer c.Unlock()
	var sum int64
	for i := range c.Logs {
		sum += c.Logs[i].GetMisses()
	}
	return sum
}

type policyLog struct {
	hits      int64
	misses    int64
	evictions int64
}

func (p *policyLog) Hit() {
	atomic.AddInt64(&p.hits, 1)
}

func (p *policyLog) Miss() {
	atomic.AddInt64(&p.misses, 1)
}

func (p *policyLog) Evict() {
	atomic.AddInt64(&p.evictions, 1)
}

func (p *policyLog) GetMisses() int64 {
	return atomic.LoadInt64(&p.misses)
}

func (p *policyLog) GetHits() int64 {
	return atomic.LoadInt64(&p.hits)
}

func (p *policyLog) GetEvictions() int64 {
	return atomic.LoadInt64(&p.evictions)
}

func (p *policyLog) Ratio() float64 {
	hits := atomic.LoadInt64(&p.hits)
	misses := atomic.LoadInt64(&p.misses)
	return float64(hits) / float64(hits+misses)
}
