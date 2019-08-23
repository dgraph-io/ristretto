# Ristretto

Ristretto is a fast, efficient, memory bounded cache library. To read more about
the technical details, check out [design.md](./design.md).

## Usage

```go
package main

import (
	"fmt"
	"time"

	"github.com/dgraph-io/ristretto"
)

func main() {
	// create a cache instance
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1000000 * 10,
		MaxCost:     1000000,
		BufferItems: 64,
	})
	if err != nil {
		panic(err)
	}

	// set a value
	cache.Set("key", "value", 1)

	// wait for value to pass through buffers
	time.Sleep(time.Second / 100)

	// get a value, given a key
	value, found := cache.Get("key")
	if !found {
		panic("missing value")
	}

	fmt.Println(value)

	// delete a value, given a key
	cache.Del("key")
}
```

### Config

## Benchmarks

Our benchmark suite is in the `bench` subdirectory package. We try to use a
variety of workload traces to best get an understanding of Ristretto's
performance compared to other Go cache implementations.

The two sides of the "performance" coin are throughput and efficiency. Often
times they are inversely correlated. Ristretto achieves a balance by only
sacrificing efficiency in times of high contention (by dropping items), in order
to keep throughput high.

### Throughput

### Efficiency (Hit Ratio)

The following benchmarks were both made using ARC trace files.

> Nimrod Megiddo and Dharmendra S. Modha, "ARC: A Self-Tuning, Low Overhead Replacement Cache," USENIX Conference on File and Storage Technologies (FAST 03), San Francisco, CA, pp. 115-130, March 31-April 2, 2003. 

#### Search (S3)

<p align="center">
    <img src="https://i.imgur.com/jljJa8w.png" />
</p>

#### Database (DS1)

<p align="center">
    <img src="https://i.imgur.com/VjKZb0p.png" />
</p>
