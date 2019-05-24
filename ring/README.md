# ring

Striped, lossy ring buffers meant for MPSC work.

## benchmarks

Benchmarks are ran on an Intel i7-8700K (3.7GHz 6-core).

| Stripes | Size     | Parallel | Ops/Second      | Allocs/Op |
|:-------:|:--------:|:--------:|:----------------|:----------|
| 1       | 128      |          | 131,140,000     | 0     |
| 16      | 128      |          | 97,960,000      | 0     |
| 1       | 128      | ✔        | 32,440,000      | 0     |
| **16**  | **128**  | **✔**    | **47,720,000**  | **0** |

### sync.Pool benchmarks (new lossy version with unbounded stripes)

| Stripes | Size    | Parallel | Ops/Second      | Allocs/Op |
|:-------:|:-------:|:--------:|:----------------|:----------|
| 16      | 128     |          | 59,550,000      | 0     |
| ~       | 128     |          | 46,950,000      | 0     |
| 16      | 128     | ✔        | 30,670,000      | 0     |
| **~**   | **128** | **✔**    | **282,660,000** | **0** |
