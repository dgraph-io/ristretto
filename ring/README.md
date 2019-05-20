# ring

Striped, lossy ring buffers meant for MPSC work.

## benchmarks

Benchmarks are ran on an Intel i7-8700K (3.7GHz 6-core).

| Striped | Parallel | Ops/Second  | Allocs/Op |
|:-------:|:--------:|:------------|:----------|
|         |          | 131,140,000 | 0 |
|         | ✔        | 32,440,000  | 0 |
| ✔       |          | 97,960,000  | 0 |
| ✔       | ✔        | 47,720,000  | 0 |
