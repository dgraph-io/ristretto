# bench

A cache benchmarking framework. The goal is to automatically generate
benchmarking functions for a variety of cache implementations, run them, and
output CSV data for easy analysis and visualization.

## usage 

```
$ go build
$ ./bench -path stats.csv

  2019/05/30 20:00:36 running: fastCache       (get-same) * 1
  2019/05/30 20:00:38      ... 1.552760541s
  2019/05/30 20:00:38 running: bigCache        (get-same) * 1
  2019/05/30 20:00:39      ... 1.49502163s
  2019/05/30 20:00:39 running: freeCache       (get-same) * 1
  2019/05/30 20:00:42      ... 2.536555797s
  2019/05/30 20:00:42 running: baseMutex       (get-same) * 1
  2019/05/30 20:00:44      ... 2.01624646s
  2019/05/30 20:00:44 running: ristretto       (get-same) * 1
  2019/05/30 20:00:45      ... 1.700182571s
  ok      github.com/dgraph-io/ristretto/bench    9.333s

$ cat stats.csv

  name,label,goroutines,ops,allocs,bytes
  fastCache,get-same,12,18.54,1,8
  bigCache,get-same,12,23.94,3,34
  freeCache,get-same,12,4.41,2,33
  baseMutex,get-same,12,10.31,0,0
  ristretto,get-same,12,6.56,2,40
```

## notes


