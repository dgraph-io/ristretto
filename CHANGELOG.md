# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project will adhere to [Semantic Versioning](http://semver.org/spec/v2.0.0.html) starting v1.0.0.

## Unreleased

## [0.1.0] - 2021-06-03

[0.1.0]: https://github.com/dgraph-io/ristretto/compare/v0.1.0..v0.0.3

### Changed
18e2797 Make item public. Add a new onReject call for rejected items. (#180)

### Added
8405ab9 Use z.Buffer backing for B+ tree (#268)
0f08db7 expose GetTTL function (#270)
b837fdf docs(README): Ristretto is production-ready. (#267)
221ca9b Add IterateKV (#265)
a4346e5 feat(super-flags): Add GetPath method in superflags (#258)
58fa1b4 add GetDuration to SuperFlag (#248)
a29b033 add Has, GetFloat64, and GetInt64 to SuperFlag (#247)
1fb8d28 move SuperFlag to Ristretto (#246)
024fba8 add SuperFlagHelp tool to generate flag help text (#251)
8ec1dc1 allow empty defaults in SuperFlag (#254)
1d4870a add mmaped b+ tree (#207)
9739cfa Add API to allow the MaxCost of an existing cache to be updated. (#200)
623d8ef Add OnExit handler which can be used for manual memory management (#183)
1940d54 Add life expectancy histogram (#182)
bf86548 Add mechanism to wait for items to be processed. (#184)

### Fixed
9d4946d change expiration type from int64 to time.Time (#277)
0bf2acd fix(buffer): make buffer capacity atleast defaultCapacity (#273)
6429872 Fixes for z.PersistentTree (#272)
ad070f2 Initialize persistent tree correctly (#271)
5946b62 use xxhash v2 (#266)
e0a933c update comments to correctly reflect counter space usage (#189)
cd8cd61 enable riscv64 builds (#264)
59dd468 Switch from log to glog (#263)
62d2e17 Use Fibonacci for latency numbers
74754f6 cache: fix race when clearning a cache (#261)
ecb55b4 Check for keys without values in superflags (#259)
3836124 chore(perf): using tags instead of runtime callers to improve the performance of leak detection (#255)
9b320d0 fix(Flags): panic on user errors (#256)
61bbb40 fix SuperFlagHelp newline (#252)
9c8fa18 fix(arm): Fix crashing under ARMv6 due to memory mis-alignment (#239)
426327c Fix incorrect unit test coverage depiction (#245)
13024c7 chore(histogram): adding percentile in histogram (#241)
bb5d392 fix(windows): use filepath instead of path (#244)
b1486d8 fix(MmapFile): Close the fd before deleting the file (#242)
d7c5d7a Fixes CGO_ENABLED=0 compilation error (#240)
e860a6c fix(build): fix build on non-amd64 architectures (#238)
6a5070b fix(b+tree): Do not double the size of btree (#237)
c72a155 fix(jemalloc): Fix the stats of jemalloc (#236)
bafef75 Don't print stuff, only return strings.
b7ca2e9 Bring memclrNoHeapPointers to z (#235)
bc9300e increase number of buffers from 32 to 64 in allocator (#234)
67fef61 Set minSize to 1MB.
d04b4c2 Opt(btree): Use Go memory instead of mmap files
afb2200 Opt(btree): Lightweight stats calculation
766bca5 Put padding internally to z.Buffer
bd7dd13 Chore(z): Add SetTmpDir API to set the temp directory (#233)
0074940 Add a BufferFrom
6497cc6 Bring z.Allocator and z.AllocatorPool back
68b18eb Fix(z.Allocator): Make Allocator use Go memory
729b324 Updated ZeroOut to use a simple for loop.  (#231)
eeefcb8 Add concurrency back
110f2c6 Add a test to check concurrency of Allocator.
3e25d09 Fix(buffer): Expose padding by z.Buffer's APIs and fix test (#222)
261a957 AllocateSlice should Truncate if the file is not big enough (#226)
24ae56e Zero out allocations for structs now that we're reusing Allocators.
1040b7d Fix the ristretto substring
692243c Deal with nil z.AllocatorPool
32c2982 Create an AllocatorPool class.
1caec3b chore(btree): clean NewTree API (#225)
f3ca035 fix(MmapFile): Don't error out if fileSize > sz (#224)
af58718 feat(btree): allow option to reset btree and mmaping it to specified file. (#223)
f30e50e Use mremap on Linux instead of munmap+mmap (#221)
a2c5a34 Reuse pages in B+ tree (#220)
732f879 fix(allocator): make nil allocator return go byte slice (#217)
d8d5371 fix(buffer): Make padding internal to z.buffer (#216)
93dc830 chore(buffer): add a parent directory field in z.Buffer (#215)
4dcfe40 Make Allocator concurrent
cd75c35 Fix infinite loop in allocator (#214)
4f21aeb Add trim func
0ca62b6 Use allocator pool. Turn off freelist.
d0f9132 Add freelists to Allocator to reuse.
0eff948 make DeleteBelow delete values that are less than lo (#211)
e2057c1 Avoid an unnecessary Load procedure in IncrementOffset.
5dc1199 Add Stats method in Btree.
1c00afa chore(script): fix local test script (#210)
2652d61 fix(btree): Increase buffer size if needed. (#209)
78a6c82 chore(btree): add occupancy ratio, search benchmark and compact bug fix (#208)
72c2139 Add licenses, remove prints, and fix a bug in compact
f32a016 Add IncrementOffset API for z.buffers (#206)
385d3ac Show count when printing histogram (#201)
f071429 Zbuffer: Add LenNoPadding and make padding 8 bytes (#204)
28aba7a Allocate Go memory in case allocator is nil.
6d6fac6 Add leak detection via leak build flag and fix a leak during cache.Close.
0af15dd Add some APIs for allocator and buffer
ba670c7 Sync before truncation or close.
079c5f0 Handle nil MmapFile for Sync.
88ad187 Public methods must not panic after Close() (#202)
0310ffe Check for RD_ONLY correctly.
7b37336 Modify MmapFile APIs
8795246 Add a bunch of APIs around MmapFile
b807f09 Move APIs for mmapfile creation over to z package.
db2bdec Add ZeroOut func
4b068f2 Add SliceOffsets
e1609c8 z: Add TotalSize method on bloom filter (#197)
646c5f3 Add Msync func
2878aeb Update CODEOWNERS (#199)
163c5d4 Buffer: Use 256GB mmap size instead of MaxInt64 (#198)
9dda05d Add a simple test to check next2Pow
5f615bf Improve memory performance (#195)
a1c354a Have a way to automatically mmap a growing buffer (#196)
148048a Introduce Mmapped buffers and Merge Sort (#194)
0f2ad8c Add a way to access an allocator via reference.
5635329 Use jemalloc.a to ensure compilation with the Go binary
034d03c Fix up a build issue with ReadMemStats
578ecae Add ReadMemStats function (#193)
41ebdbf Allocator helps allocate memory to be used by unsafe structs (#192)
96070d1 Improve histogram output
4dec277 Move Closer from y to z (#191)
9d26abc Add histogram.Mean() method (#188)
834a9bc Delete .travis.yml (#185)
2ce4f8f Introduce Calloc: Manual Memory Management via jemalloc (#186)

## [0.0.3] - 2020-07-06

[0.0.3]: https://github.com/dgraph-io/ristretto/compare/v0.0.2..v0.0.3

### Changed

### Added

### Fixed

- z: use MemHashString and xxhash.Sum64String ([#153][])
- Check conflict key before updating expiration map. ([#154][])
- Fix race condition in Cache.Clear ([#133][])
- Improve handling of updated items ([#168][])
- Fix droppedSets count while updating the item ([#171][])

## [0.0.2] - 2020-02-24

[0.0.2]: https://github.com/dgraph-io/ristretto/compare/v0.0.1..v0.0.2

### Added

- Sets with TTL. ([#122][])

### Fixed

- Fix the way metrics are handled for deletions. ([#111][])
- Support nil `*Cache` values in `Clear` and `Close`. ([#119][]) 
- Delete item immediately. ([#113][])
- Remove key from policy after TTL eviction. ([#130][])

[#111]: https://github.com/dgraph-io/ristretto/issues/111
[#113]: https://github.com/dgraph-io/ristretto/issues/113
[#119]: https://github.com/dgraph-io/ristretto/issues/119
[#122]: https://github.com/dgraph-io/ristretto/issues/122
[#130]: https://github.com/dgraph-io/ristretto/issues/130

## 0.0.1

First release. Basic cache functionality based on a LFU policy.
