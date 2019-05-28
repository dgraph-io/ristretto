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
	"testing"
)

// Rather than just writing to stdout, we can use a user-defined file location
// for saving benchmark data so we can use stdout for logs.
var PATH = flag.String("path", "stats.csv", "Filepath for benchmark CSV data.")

func init() {
	flag.Parse()

	h := newBigCacheHasher()
	fmt.Printf("%2x\n", h.Sum64("testing"))
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
	file, err := os.OpenFile(*PATH, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Panic(err)
	}
	defer file.Close()
	return csv.NewWriter(file).WriteAll(records)
}

// TestMain is the entry point for running this benchmark suite.
func TestMain(m *testing.M) {
	logs := make([]*Log, 0)
	benchmarks := []*Benchmark{
		{"fastcache", "", 1, func() Cache { return NewBenchFastCache(16) }},
		{"bigcache", "", 1, func() Cache { return NewBenchBigCache(16) }},
	}

	for _, benchmark := range benchmarks {
		fmt.Println("bench: ", benchmark)
		logs = append(logs, &Log{
			Benchmark: benchmark,
			Result: NewResult(
				testing.Benchmark(GetSame(benchmark)),
			),
		})
	}

	save(logs)
}
