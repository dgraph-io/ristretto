# Ristretto
Higher performance, concurrent cache library

Branch with Channel:
goos: linux
goarch: amd64
BenchmarkCaches/RistrettoZipfRead-4          	20000000	        95.4 ns/op	       0 B/op	       0 allocs/op
BenchmarkCaches/RistrettoZipfRead-4          	20000000	        96.4 ns/op	       0 B/op	       0 allocs/op
BenchmarkCaches/RistrettoZipfRead-8          	30000000	        49.1 ns/op	       0 B/op	       0 allocs/op
BenchmarkCaches/RistrettoZipfRead-8          	30000000	        48.7 ns/op	       0 B/op	       0 allocs/op
BenchmarkCaches/RistrettoZipfRead-16         	50000000	        25.3 ns/op	       0 B/op	       0 allocs/op
BenchmarkCaches/RistrettoZipfRead-16         	50000000	        25.5 ns/op	       0 B/op	       0 allocs/op
BenchmarkCaches/RistrettoZipfRead-32         	100000000	        17.9 ns/op	       1 B/op	       0 allocs/op
BenchmarkCaches/RistrettoZipfRead-32         	100000000	        17.9 ns/op	       1 B/op	       0 allocs/op
BenchmarkCaches/RistrettoZipfRead-64         	100000000	        18.3 ns/op	       0 B/op	       0 allocs/op
BenchmarkCaches/RistrettoZipfRead-64         	100000000	        18.2 ns/op	       0 B/op	       0 allocs/op

Branch with atomics:
$ go test -count=2 -cpu=4,8,16,32,64 -v -bench=BenchmarkCaches/RistrettoZipfRead
goos: linux
goarch: amd64
BenchmarkCaches/RistrettoZipfRead-4          	20000000	       108 ns/op	       8 B/op	       0 allocs/op
BenchmarkCaches/RistrettoZipfRead-4          	20000000	       106 ns/op	       8 B/op	       0 allocs/op
BenchmarkCaches/RistrettoZipfRead-8          	20000000	        55.0 ns/op	       8 B/op	       0 allocs/op
BenchmarkCaches/RistrettoZipfRead-8          	20000000	        57.3 ns/op	       8 B/op	       0 allocs/op
BenchmarkCaches/RistrettoZipfRead-16         	50000000	        33.1 ns/op	       8 B/op	       0 allocs/op
BenchmarkCaches/RistrettoZipfRead-16         	50000000	        31.9 ns/op	       8 B/op	       0 allocs/op
BenchmarkCaches/RistrettoZipfRead-32         	50000000	        22.8 ns/op	       8 B/op	       0 allocs/op
BenchmarkCaches/RistrettoZipfRead-32         	50000000	        22.9 ns/op	       8 B/op	       0 allocs/op
BenchmarkCaches/RistrettoZipfRead-64         	50000000	        21.8 ns/op	       8 B/op	       0 allocs/op
BenchmarkCaches/RistrettoZipfRead-64         	50000000	        21.8 ns/op	       8 B/op	       0 allocs/op

Master:
$ go test -count=2 -cpu=4,8,16,32,64 -v -bench=BenchmarkCaches/RistrettoZipfRead
goos: linux
goarch: amd64
BenchmarkCaches/RistrettoZipfRead-4          	10000000	       123 ns/op	       8 B/op	       0 allocs/op
BenchmarkCaches/RistrettoZipfRead-4          	20000000	       118 ns/op	       8 B/op	       0 allocs/op
BenchmarkCaches/RistrettoZipfRead-8          	20000000	        64.0 ns/op	       8 B/op	       0 allocs/op
BenchmarkCaches/RistrettoZipfRead-8          	20000000	        63.6 ns/op	       8 B/op	       0 allocs/op
BenchmarkCaches/RistrettoZipfRead-16         	30000000	        45.3 ns/op	      12 B/op	       0 allocs/op
BenchmarkCaches/RistrettoZipfRead-16         	30000000	        43.8 ns/op	       8 B/op	       0 allocs/op
BenchmarkCaches/RistrettoZipfRead-32         	50000000	        31.4 ns/op	      11 B/op	       0 allocs/op
BenchmarkCaches/RistrettoZipfRead-32         	50000000	        31.6 ns/op	       9 B/op	       0 allocs/op
BenchmarkCaches/RistrettoZipfRead-64         	50000000	        24.7 ns/op	       9 B/op	       0 allocs/op
BenchmarkCaches/RistrettoZipfRead-64         	50000000	        25.0 ns/op	       9 B/op	       0 allocs/op
PASS
ok  	_/home/mrjn/benchmarks/cachebench	171.174s


