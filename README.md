# Ristretto
Higher performance, concurrent cache library

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

