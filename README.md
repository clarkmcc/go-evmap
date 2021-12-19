# go-evmap
#### Note: this is not a production-ready data structure by any-means. It is currently a work-in-progress exploration of a left-right-backed concurrent map.

A Go implementation of Rust's [evmap](https://github.com/jonhoo/evmap). This implementation is more of a naive implementation that does not support writer/reader handles and iterators, but this also means that the implementation is extremely simple (<200 lines). It has no direct dependencies.

## Usage
```go
cache := eventual.NewMap[string, int]()
reader := cache.Reader()

// Insert a key
cache.Insert("foo", 0)
reader.Has("foo") // false

// Explicitly expose the current state of the map to the reads
cache.Refresh()
reader.Has("foo") // true
```

## Why?
This data structure is optimized for high-read, low-write workloads where readers never have to coordinate with writers. This lack of coordination comes at a cost, "The trade-off exposed by this module is one of eventual consistency: writes are not visible to readers except following explicit synchronization. Specifically, readers only see the operations that preceded the last call to `Refresh` by a writer. This lets writers decide how stale they are willing to let reads get. They can refresh the map after every write to emulate a regular map, or they can refresh only occasionally to reduce the synchronization overhead at the cost of stale reads." ([evmap readme](https://github.com/jonhoo/evmap))

## Features
* Readers never block writers
* Writers never block readers
* Reads and writes are completely thread-safe
* 100% test coverage
* Utilizes Go 1.18 generics

## Caveats
* Readers do not observe writes as they occur (eventual consistency)
* Writers block other writers (writes are guarded by a mutex).

## Help Needed
I do not have the expertise to benchmark this. I've implemented a crude benchmark in [map_bench_test.go](./map_bench_test.go) but the results are all across the board.