Ristretto has these requirements:

- Concurrent.
- Memory-bounded (limit to configurable max memory usage).
- Scale well as the number of cores and goroutines increase.
- Scale well under non-random key access distribution (e.g. Zipf).
- High cache hit ratio

There also an additional nice-to-have which is to minimize Go GC. Other caches
like BigCache, FreeCache, etc. do this already.

See the [blog post][blog] for more details.

[blog]: https://blog.dgraph.io/post/caching-in-go/

Ristretto is divided up into a few parts:

1. Tiny-LFU
1. Memory Allocator
1. Cache using BP8 and Tiny-LFU.

### Tiny-LFU

This can be adapted from Damian Gryski's implementation [here][tiny]. Work with
Damian and Ben Manes to ensure that this implementation is correct and contains
most low-hanging fruits.

[tiny]: https://github.com/dgryski/go-tinylfu

### Memory Allocator

Memory Allocator can be inspired by Google's TC Malloc. We allocate memory based
on the size of the value in classes. Each class has pre-determined size and the
value would take up a full slot in that class. The class keeps track of empty
slots, potentially using bits and atomics to ensure threads are not acquiring
locks to allocate or deallocate a slot. Note that Go doesn't expose
thread-locality like C, so this is important.

There's plenty of material online about TCMalloc. See [this][tc1].

[tc1]: https://www.jamesgolick.com/2013/5/19/how-tcmalloc-works.html

### Cache using BP-Wrapper

Initial version of the cache can just have a RW mutex lock. It
can be based on a basic LRU algorithm initially. The main innovation would be to
use [BP-Wrapper][wrapper] to ensure that the cache performs great under load.

[wrapper]: https://drive.google.com/open?id=0B8oWjCpZGTn3YVhCc2dZSU5DNFBJRlBZa0s1STdaUWR5emxj

We can optimize this and do benchmarks. Once those show clear benefits, we can
then integrate Tiny-LFU to increase the hit ratios and memory allocator to
decrease Go GC time spent on the cache.

As per Ben:

> I use [striped][] & [lossy][] ring buffer for reads, a bounded [ring][] buffer
> for writes, schedule with a state machine, and guard executions with a
> try-lock. Since the LRU operations are performed under a non-blocking lock,
> the policy operations are simple and the buffers are optimized as
> multi-producer / single-consumer.

[striped]: https://github.com/ben-manes/caffeine/blob/master/caffeine/src/main/java/com/github/benmanes/caffeine/cache/StripedBuffer.java
[lossy]: https://github.com/ben-manes/caffeine/blob/master/caffeine/src/main/java/com/github/benmanes/caffeine/cache/BoundedBuffer.java
[ring]: https://github.com/ben-manes/caffeine/blob/master/caffeine/src/main/java/com/github/benmanes/caffeine/cache/MpscGrowableArrayQueue.java

To implement these kind of buffers, an easy way in Go would be to use channels.
We can further stripe them to divide up the accesses to avoid contention.

A lossy ring buffer can be a channel with a non-blocking push. A bounded ring
buffer would be a channel with buffer and so on.

This would allow our reads to acquire read-only lock on the
cache, while doing a best-effort push into one of the channels.
