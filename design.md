# Design

Ristretto has five requirements to consider when making design decisions.

1. Concurrent: safe for concurrent use out-of-the-box.
2. Memory Bounded: able to set a hard limit on memory usage.
3. Horizontal Scaling: throughput performance should increase linearly as 
   hardware capabilities (number of cores and threads) increase.
4. Contention Mitigation: perform well under non-random key access distributions
   (e.g. [Zipf](https://en.wikipedia.org/wiki/Zipf%27s_law)).
5. Hit Ratio: at least as high as non-concurrent
   [implementations](https://github.com/dgryski/go-tinylfu) of TinyLFU.

These requirements have emerged from a thorough
[analysis](https://blog.dgraph.io/post/caching-in-go/) of the current
state-of-the-art in Go cache libraries.

We will point back to these requirements as we describe Ristretto's components,
so you can get a better idea of why we made the decisions we did.

## Store

At its core, a cache is a hash map with rules about what goes in and what goes 
out. The `store` interface is the concurrent hash map essential to the actual
storage of key-value items. Everything else in Ristretto pertains to metadata
about what goes in and what goes out.

Currently we just keep an `interface{}` (pointer) for keys and values. In the
future, if GC becomes an issue, we may look into directly managing memory and
avoiding Go's GC altogether.

Due to our **Concurrent** and **Contention Mitigation** requirement, it's 
important that the `store` implementation is fast and avoids mutex contention
like the plague. We've seen good results with `sync.Map` (because of their
internal usage of thread-local storage) and sharded mutex-maps.

As for the other requirements, there's not much that `store` needs to do. Since
we have other components managing Ristretto's metadata, it's not even important
for `store` to be memory bounded. The rest is handled elsewhere.

## Admission

The admission policy contains the answer to the question, "What goes in to the
cache?" For this, we looked to the new and fascinating paper written by Gil
Einziger, Roy Friedman, and Ben Manes called [TinyLFU: A Highly Efficient Cache
Admission Policy](https://arxiv.org/abs/1512.00727). 

The paper explains it best, but TinyLFU is an eviction-agnostic admission policy
designed to improve hit ratios with very little memory overhead. If I had to
describe it in one sentence, I would say: TinyLFU uses 4-bit counters to 
approximate item value and only lets in new items if they are more valuable than
the average item currently in the cache.

With help from Ben Manes and single-threaded implementations of TinyLFU, we
implemented TinyLFU in Ristretto using a
[Count-Min](https://en.wikipedia.org/wiki/Count%E2%80%93min_sketch) Sketch (in
the `sketch` file) and use it in the `policy.Add()` function to reject or accept
new items.

## Eviction

Ristretto uses a Sampled LFU eviction policy. You can see this in the
`policy.Add()` function. Most caches use a LRU eviction policy, which is
probably considered the safest and oldest approach. The problem with a LRU
policy is the concurrent mutations of a doubly-linked list *for every cache 
access*. Not only is it expensive memory-wise, but contention can become an
issue very quickly.

Since we were interested in using the TinyLFU admission policy, we tried using
the same (tiny) counters for eviction. Amazingly, the hit ratios were within 1%
of exact LRU/LFU policies for a variety of workloads. This means we get the
benefits of an admission policy, conservative memory usage, and lower contention
in the same little package. The only metadata we keep for each item is the 4 bit
hit counter.

### BP-Wrapper

One of the hardest things a cache has to do is effectively manage item metadata.
Because of our **Contention Mitigation** and **Horizontal Scaling**
requirements, this became even harder. Luckily, a paper written by Xiaoning
Ding, Song Jiang, and Xiaodong Zhang called [BP-Wrapper: A System Framework
Making Any Replacement Algorithms (Almost) Lock Contention
Free](https://ieeexplore.ieee.org/document/4812418) gives us a solution here.

The paper describes two ways to mitigate contention: *prefetching* and
*batching*. We only use batching.

Batching works pretty much how you'd think. Rather than acquiring a mutex lock
for every metadata mutation, we wait for a ring buffer to fill up before we
acquire a mutex (the "drain" process) and process the mutations. As described in
the paper, this lowers contention considerably with little overhead.

#### Get Buffers

This might seem like a perfect use case of Go channels, and it is.
Unfortunately, the throughput performance of channels prevented us from using
them. Instead, we use a `sync.Pool` to implement "striped, lossy" ring buffers
(in the `ring` file) that have great performance with little loss of data. The
performance benefits of using a `sync.Pool` over anything else (slices, striped
mutexes, etc.) are mostly due to the internal usage of thread-local storage
(similar to the `sync.Map` performance). It would be nice to have access to
these runtime primitives, but the Go team have rejected it in the name of
simplicity.

#### Set Buffers
