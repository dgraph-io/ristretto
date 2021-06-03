# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project will adhere to [Semantic Versioning](http://semver.org/spec/v2.0.0.html) starting v1.0.0.

## Unreleased

## [0.1.0] - 2021-06-03

[0.1.0]: https://github.com/dgraph-io/ristretto/compare/v0.1.0..v0.0.3
This release contains bug fixes and improvements to Ristretto. It also contains
major updates to the z package. The z package contains types such as Tree (B+
tree), Buffer, Mmap file, etc. All these types are used in Badger and Dgraph to
improve performance and reduce memory requirements.

### Changed
- Make item public. Add a new onReject call for rejected items. ([#180][])

### Added
- Use z.Buffer backing for B+ tree ([#268][])
- expose GetTTL function ([#270][])
- docs(README): Ristretto is production-ready. ([#267][])
- Add IterateKV ([#265][])
- feat(super-flags): Add GetPath method in superflags ([#258][])
- add GetDuration to SuperFlag ([#248][])
- add Has, GetFloat64, and GetInt64 to SuperFlag ([#247][])
- move SuperFlag to Ristretto ([#246][])
- add SuperFlagHelp tool to generate flag help text ([#251][])
- allow empty defaults in SuperFlag ([#254][])
- add mmaped b+ tree ([#207][])
- Add API to allow the MaxCost of an existing cache to be updated. ([#200][])
- Add OnExit handler which can be used for manual memory management ([#183][])
- Add life expectancy histogram ([#182][])
- Add mechanism to wait for items to be processed. ([#184][])

### Fixed
- change expiration type from int64 to time.Time ([#277][])
- fix(buffer): make buffer capacity atleast defaultCapacity ([#273][])
- Fixes for z.PersistentTree ([#272][])
- Initialize persistent tree correctly ([#271][])
- use xxhash v2 ([#266][])
- update comments to correctly reflect counter space usage ([#189][])
- enable riscv64 builds ([#264][])
- Switch from log to glog ([#263][])
- Use Fibonacci for latency numbers
- cache: fix race when clearning a cache ([#261][])
- Check for keys without values in superflags ([#259][])
- chore(perf): using tags instead of runtime callers to improve the performance of leak detection ([#255][])
- fix(Flags): panic on user errors ([#256][])
- fix SuperFlagHelp newline ([#252][])
- fix(arm): Fix crashing under ARMv6 due to memory mis-alignment ([#239][])
- Fix incorrect unit test coverage depiction ([#245][])
- chore(histogram): adding percentile in histogram ([#241][])
- fix(windows): use filepath instead of path ([#244][])
- fix(MmapFile): Close the fd before deleting the file ([#242][])
- Fixes CGO_ENABLED=0 compilation error ([#240][])
- fix(build): fix build on non-amd64 architectures ([#238][])
- fix(b+tree): Do not double the size of btree ([#237][])
- fix(jemalloc): Fix the stats of jemalloc ([#236][])
- Don't print stuff, only return strings.
- Bring memclrNoHeapPointers to z ([#235][])
- increase number of buffers from 32 to 64 in allocator ([#234][])
- Set minSize to 1MB.
- Opt(btree): Use Go memory instead of mmap files
- Opt(btree): Lightweight stats calculation
- Put padding internally to z.Buffer
- Chore(z): Add SetTmpDir API to set the temp directory ([#233][])
- Add a BufferFrom
- Bring z.Allocator and z.AllocatorPool back
- Fix(z.Allocator): Make Allocator use Go memory
- Updated ZeroOut to use a simple for loop.  ([#231][])
- Add concurrency back
- Add a test to check concurrency of Allocator.
- Fix(buffer): Expose padding by z.Buffer's APIs and fix test ([#222][])
- AllocateSlice should Truncate if the file is not big enough ([#226][])
- Zero out allocations for structs now that we're reusing Allocators.
- Fix the ristretto substring
- Deal with nil z.AllocatorPool
- Create an AllocatorPool class.
- chore(btree): clean NewTree API ([#225][])
- fix(MmapFile): Don't error out if fileSize > sz ([#224][])
- feat(btree): allow option to reset btree and mmaping it to specified file. ([#223][])
- Use mremap on Linux instead of munmap+mmap ([#221][])
- Reuse pages in B+ tree ([#220][])
- fix(allocator): make nil allocator return go byte slice ([#217][])
- fix(buffer): Make padding internal to z.buffer ([#216][])
- chore(buffer): add a parent directory field in z.Buffer ([#215][])
- Make Allocator concurrent
- Fix infinite loop in allocator ([#214][])
- Add trim func
- Use allocator pool. Turn off freelist.
- Add freelists to Allocator to reuse.
- make DeleteBelow delete values that are less than lo ([#211][])
- Avoid an unnecessary Load procedure in IncrementOffset.
- Add Stats method in Btree.
- chore(script): fix local test script ([#210][])
- fix(btree): Increase buffer size if needed. ([#209][])
- chore(btree): add occupancy ratio, search benchmark and compact bug fix ([#208][])
- Add licenses, remove prints, and fix a bug in compact
- Add IncrementOffset API for z.buffers ([#206][])
- Show count when printing histogram ([#201][])
- Zbuffer: Add LenNoPadding and make padding 8 bytes ([#204][])
- Allocate Go memory in case allocator is nil.
- Add leak detection via leak build flag and fix a leak during cache.Close.
- Add some APIs for allocator and buffer
- Sync before truncation or close.
- Handle nil MmapFile for Sync.
- Public methods must not panic after Close() ([#202][])
- Check for RD_ONLY correctly.
- Modify MmapFile APIs
- Add a bunch of APIs around MmapFile
- Move APIs for mmapfile creation over to z package.
- Add ZeroOut func
- Add SliceOffsets
- z: Add TotalSize method on bloom filter ([#197][])
- Add Msync func
- Buffer: Use 256 GB mmap size instead of MaxInt64 ([#198][])
- Add a simple test to check next2Pow
- Improve memory performance ([#195][])
- Have a way to automatically mmap a growing buffer ([#196][])
- Introduce Mmapped buffers and Merge Sort ([#194][])
- Add a way to access an allocator via reference.
- Use jemalloc.a to ensure compilation with the Go binary
- Fix up a build issue with ReadMemStats
- Add ReadMemStats function ([#193][])
- Allocator helps allocate memory to be used by unsafe structs ([#192][])
- Improve histogram output
- Move Closer from y to z ([#191][])
- Add histogram.Mean() method ([#188][])
- Introduce Calloc: Manual Memory Management via jemalloc ([#186][])

[#180]: https://github.com/dgraph-io/ristretto/pull/180
[#268]: https://github.com/dgraph-io/ristretto/pull/268
[#270]: https://github.com/dgraph-io/ristretto/pull/270
[#267]: https://github.com/dgraph-io/ristretto/pull/267
[#265]: https://github.com/dgraph-io/ristretto/pull/265
[#258]: https://github.com/dgraph-io/ristretto/pull/258
[#248]: https://github.com/dgraph-io/ristretto/pull/248
[#247]: https://github.com/dgraph-io/ristretto/pull/247
[#246]: https://github.com/dgraph-io/ristretto/pull/246
[#251]: https://github.com/dgraph-io/ristretto/pull/251
[#254]: https://github.com/dgraph-io/ristretto/pull/254
[#207]: https://github.com/dgraph-io/ristretto/pull/207
[#200]: https://github.com/dgraph-io/ristretto/pull/200
[#183]: https://github.com/dgraph-io/ristretto/pull/183
[#182]: https://github.com/dgraph-io/ristretto/pull/182
[#184]: https://github.com/dgraph-io/ristretto/pull/184
[#277]: https://github.com/dgraph-io/ristretto/pull/277
[#273]: https://github.com/dgraph-io/ristretto/pull/273
[#272]: https://github.com/dgraph-io/ristretto/pull/272
[#271]: https://github.com/dgraph-io/ristretto/pull/271
[#266]: https://github.com/dgraph-io/ristretto/pull/266
[#189]: https://github.com/dgraph-io/ristretto/pull/189
[#264]: https://github.com/dgraph-io/ristretto/pull/264
[#263]: https://github.com/dgraph-io/ristretto/pull/263
[#261]: https://github.com/dgraph-io/ristretto/pull/261
[#259]: https://github.com/dgraph-io/ristretto/pull/259
[#255]: https://github.com/dgraph-io/ristretto/pull/255
[#256]: https://github.com/dgraph-io/ristretto/pull/256
[#252]: https://github.com/dgraph-io/ristretto/pull/252
[#239]: https://github.com/dgraph-io/ristretto/pull/239
[#245]: https://github.com/dgraph-io/ristretto/pull/245
[#241]: https://github.com/dgraph-io/ristretto/pull/241
[#244]: https://github.com/dgraph-io/ristretto/pull/244
[#242]: https://github.com/dgraph-io/ristretto/pull/242
[#240]: https://github.com/dgraph-io/ristretto/pull/240
[#238]: https://github.com/dgraph-io/ristretto/pull/238
[#237]: https://github.com/dgraph-io/ristretto/pull/237
[#236]: https://github.com/dgraph-io/ristretto/pull/236
[#235]: https://github.com/dgraph-io/ristretto/pull/235
[#234]: https://github.com/dgraph-io/ristretto/pull/234
[#233]: https://github.com/dgraph-io/ristretto/pull/233
[#231]: https://github.com/dgraph-io/ristretto/pull/231
[#222]: https://github.com/dgraph-io/ristretto/pull/222
[#226]: https://github.com/dgraph-io/ristretto/pull/226
[#225]: https://github.com/dgraph-io/ristretto/pull/225
[#224]: https://github.com/dgraph-io/ristretto/pull/224
[#223]: https://github.com/dgraph-io/ristretto/pull/223
[#221]: https://github.com/dgraph-io/ristretto/pull/221
[#220]: https://github.com/dgraph-io/ristretto/pull/220
[#217]: https://github.com/dgraph-io/ristretto/pull/217
[#216]: https://github.com/dgraph-io/ristretto/pull/216
[#215]: https://github.com/dgraph-io/ristretto/pull/215
[#214]: https://github.com/dgraph-io/ristretto/pull/214
[#211]: https://github.com/dgraph-io/ristretto/pull/211
[#210]: https://github.com/dgraph-io/ristretto/pull/210
[#209]: https://github.com/dgraph-io/ristretto/pull/209
[#208]: https://github.com/dgraph-io/ristretto/pull/208
[#206]: https://github.com/dgraph-io/ristretto/pull/206
[#201]: https://github.com/dgraph-io/ristretto/pull/201
[#204]: https://github.com/dgraph-io/ristretto/pull/204
[#202]: https://github.com/dgraph-io/ristretto/pull/202
[#197]: https://github.com/dgraph-io/ristretto/pull/197
[#199]: https://github.com/dgraph-io/ristretto/pull/199
[#198]: https://github.com/dgraph-io/ristretto/pull/198
[#195]: https://github.com/dgraph-io/ristretto/pull/195
[#196]: https://github.com/dgraph-io/ristretto/pull/196
[#194]: https://github.com/dgraph-io/ristretto/pull/194
[#193]: https://github.com/dgraph-io/ristretto/pull/193
[#192]: https://github.com/dgraph-io/ristretto/pull/192
[#191]: https://github.com/dgraph-io/ristretto/pull/191
[#188]: https://github.com/dgraph-io/ristretto/pull/188
[#186]: https://github.com/dgraph-io/ristretto/pull/186

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
