# bench

This is a comprehensive benchmarking suite for Go cache libraries. We measure
cache performance by tracking raw throughput and hit ratios, for a variety of
access distributions / trace files.

## how to run

#### 1. download trace files using git lfs

```
[./ristretto/bench]$ git lfs --fetch all
```

#### 2. compile bench

```
[./ristretto/bench]$ go build
```

#### 3. run bench with parameters

```
[./ristretto/bench]$ ./bench -suite    [ all | speed | hits ] 
                             -cache    [ all | ristretto ]
                             -parallel [ 1... ]
                             -path     [ output_file.csv ]
```

Note: The `parallel` flag is the goroutine multiplier to use when running the
benchmarks. This is useful for simulating contention.

#### 4. use the output.csv file

The output CSV file is useful for creating charts and comparing implementations.
Here's an example of the output when running the speed and hits suite:

```
name       , label         , go,  mop/s,  ns/op, ac, byt, hits    , misses  ,   ratio 
ristretto  , hits-zipf     ,  0, ------, ------, --, ---, 00059405, 00040595,  59.40%
ristretto  , hits-lirs-gli ,  0, ------, ------, --, ---, 00003480, 00002535,  57.86%
ristretto  , hits-lirs-loop,  0, ------, ------, --, ---, 00098988, 00001012,  98.99%
ristretto  , hits-arc-ds1  ,  0, ------, ------, --, ---, 00007647, 00092353,   7.65%
ristretto  , hits-arc-p3   ,  0, ------, ------, --, ---, 00001471, 00098529,   1.47%
ristretto  , hits-arc-p8   ,  0, ------, ------, --, ---, 00002102, 00097898,   2.10%
ristretto  , hits-arc-s3   ,  0, ------, ------, --, ---, 00000183, 00099817,   0.18%
ristretto  , get-same      ,  4,  19.75,     50, 00, 000, --------, --------, -------
ristretto  , get-zipf      ,  4,  18.56,     53, 00, 000, --------, --------, -------
ristretto  , set-get       ,  4,   3.51,    284, 01, 034, --------, --------, -------
ristretto  , set-same      ,  4,   9.04,    110, 02, 064, --------, --------, -------
ristretto  , set-zipf      ,  4,   8.94,    111, 02, 064, --------, --------, -------
ristretto  , set-get-zipf  ,  4,  17.98,     55, 00, 000, --------, --------, -------
```

The dashed-out blocks are to be ignored. Because of the nature of those
benchmarks, those blocks could be misleading if they were left visible.
