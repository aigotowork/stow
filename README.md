# Stow - Embedded Transparent File-based KV Storage Engine

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.21-blue)](https://go.dev/dl/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Tests](https://img.shields.io/badge/Tests-69%20Passing-brightgreen)](tests/)

**Stow** is a production-ready embedded key-value storage engine written in Go, positioned as a transparent file-based solution between plain JSON files and SQLite databases.

## Features

- **Transparent Storage**: Data stored in human-readable JSONL format
- **Editable**: Supports external editing with automatic refresh
- **Media-Friendly**: Smart blob routing for large files (>4KB default)
- **Struct Tags**: Full support for JSON tags and custom `stow:"file"` / `stow:"inline"` tags
- **Version History**: Built-in version tracking for all changes
- **Concurrent Safe**: Thread-safe operations with fine-grained locking
- **Async Operations**: Non-blocking compact and GC operations
- **Compact & GC**: Automatic compression and garbage collection
- **High Performance**: 4.7M reads/s (cache hit), 22K reads/s (cache miss)
- **Well Tested**: 69 comprehensive tests with 100% pass rate
- **Simple**: Single-process design, no distributed coordination

## Quick Start

```go
package main

import (
    "github.com/aigotowork/stow"
)

func main() {
    // Open or create a store
    store := stow.MustOpen("/data/myapp")

    // Get or create a namespace
    ns := store.MustGetNamespace("config")

    // Store data
    ns.MustPut("server", map[string]interface{}{
        "host": "localhost",
        "port": 8080,
    })

    // Retrieve data
    var config map[string]interface{}
    ns.MustGet("server", &config)

    // List all keys
    keys, _ := ns.List()

    // Clean up
    store.Close()
}
```

## Installation

```bash
go get github.com/aigotowork/stow
```

## Comparison

| Feature | Plain JSON | Stow | SQLite | Redis |
|---------|-----------|------|--------|-------|
| Human Readable | ✅ | ✅ | ❌ | ❌ |
| Editable | ✅ | ✅ | ❌ | ❌ |
| Version History | ❌ | ✅ | ❌ | ❌ |
| Transaction Safe | ❌ | ✅ | ✅ | ✅ |
| Concurrent Access | ❌ | ✅ | ✅ | ✅ |
| Binary Blob Support | ❌ | ✅ | ✅ | ⚠️ |
| Queries | ❌ | ❌ | ✅ | ⚠️ |
| Distributed | ❌ | ❌ | ❌ | ✅ |
| Embedded | ✅ | ✅ | ✅ | ❌ |
| Memory Usage | Low | Low | Medium | High |
| Write Speed | Fast | Fast | Medium | Very Fast |
| Read Speed (cached) | Fast | Very Fast | Fast | Very Fast |

Stow fills the sweet spot between plain JSON files and full databases, offering transparency with reliability.

## Core Concepts

### Namespace

A namespace is an isolated storage space with its own configuration and directory. Think of it as a separate database.

```go
store := stow.MustOpen("/data")
users := store.MustGetNamespace("users")
configs := store.MustGetNamespace("configs")
```

### JSONL Format

Data is stored in newline-delimited JSON format, with each line representing one version:

```json
{"_meta":{"k":"server","v":1,"op":"put","ts":"2025-12-14T18:09:00Z"},"data":{"host":"localhost","port":8080}}
{"_meta":{"k":"server","v":2,"op":"put","ts":"2025-12-14T18:10:00Z"},"data":{"host":"localhost","port":8081}}
```

### Blob Storage

Large binary data is automatically stored as separate files in the `_blobs/` directory:

```go
type Document struct {
    Title   string
    Content []byte `stow:"file"`  // Always store as blob
}

type User struct {
    Name      string
    Avatar    []byte `stow:"inline"`  // Force inline storage
    ProfilePic []byte  // Auto-decision based on size
}

// Store document
ns.MustPut("readme", Document{
    Title:   "README",
    Content: readmeData,  // Stored as _blobs/xxx.bin
})

// Force file storage via option
ns.Put("large-data", data, stow.WithForceFile())

// Force inline storage via option
ns.Put("small-data", data, stow.WithForceInline())
```

**Storage Priority**: `PutOption` > `Struct Tag` > `Type Detection` > `Size Threshold`

## Advanced Features

### Version History

```go
// Get all versions
history, _ := ns.GetHistory("server")

// Get specific version
var oldConfig map[string]interface{}
ns.GetVersion("server", 1, &oldConfig)
```

### Compression

```go
// Manual compaction (synchronous)
ns.Compact("server")

// Async compaction (non-blocking)
ns.CompactAsync("server")

// Compact all keys (async)
ns.CompactAllAsync()
```

### Garbage Collection

```go
// Clean up unreferenced blobs
result, _ := ns.GC()
fmt.Printf("Removed %d blobs, reclaimed %d bytes\n",
    result.RemovedBlobs, result.ReclaimedSize)
```

### External Editing

```go
// User manually edits the .jsonl file
// Then refresh to reload
ns.Refresh("server")

// Or refresh all
ns.RefreshAll()
```

## Configuration

```go
config := stow.NamespaceConfig{
    BlobThreshold:      4 * 1024,        // 4KB
    MaxFileSize:        100 * 1024 * 1024, // 100MB
    CacheTTL:           5 * time.Minute,
    AutoCompact:        true,
    CompactThreshold:   20,              // 20 lines
    CompactKeepRecords: 3,               // Keep last 3 versions
}

ns, _ := store.CreateNamespace("mydata", config)
```

## Directory Structure

```
/basedir/
├── namespace_A/
│   ├── _config.json           # Namespace configuration
│   ├── server.jsonl           # Key: "server"
│   ├── user_alice.jsonl       # Key: "user:alice" (sanitized)
│   └── _blobs/                # Binary files
│       ├── avatar_abc123.jpg
│       └── resume_def456.pdf
│
└── namespace_B/
    ├── _config.json
    └── ...
```

## Documentation

- [Design Document](design.md) - Complete technical specification
- [API Documentation](https://pkg.go.dev/github.com/aigotowork/stow)
- [Test Report](tests/TEST_COMPLETION_REPORT.md) - Comprehensive test coverage report

### Examples

Practical examples to get you started quickly:

- **[Basic Example](examples/basic/)** - Simple KV operations, CRUD workflow
- **[File Storage](examples/file-storage/)** - Document management, text files, images
- **[Blog System](examples/blog/)** - Nested structures, comments, categories
- **[Struct Tags](examples/struct-tags/)** - JSON tags, stow tags, storage options

Run examples:
```bash
go run examples/basic/main.go
go run examples/file-storage/main.go
go run examples/blog/main.go
go run examples/struct-tags/main.go
```

## Development Status

✅ **Production Ready** - All core features implemented and tested

### Implementation Progress

- [x] Project initialization
- [x] Foundation layer (types, errors, utils)
- [x] Core data layer (JSONL, blob, index)
- [x] Serialization layer (codec with struct tag support)
- [x] Namespace engine (KV operations)
- [x] Advanced features (history, compact, GC)
- [x] Async operations (non-blocking compact/GC)
- [x] Nested struct support
- [x] Comprehensive tests (69 tests, 100% passing)
- [x] Performance benchmarks
- [x] Practical examples

### Test Coverage

**Unit & Integration Tests**: 69 tests
- ✅ Basic KV operations (Put, Get, Delete, List)
- ✅ Blob storage (automatic routing, force file/inline)
- ✅ Version history (GetHistory, GetVersion)
- ✅ Compact & GC (orphaned blob cleanup)
- ✅ Multi-namespace isolation
- ✅ Data persistence across sessions
- ✅ Concurrent operations (race detector verified)
- ✅ Edge cases (unicode, large data, nil values)

**Benchmark Results** (Apple M4):
```
BenchmarkPut_SmallData     251 ops/s      (3.98 ms/op)
BenchmarkPut_LargeData     100 ops/s      (10.0 ms/op)
BenchmarkGet_CacheHit      4.7M ops/s     (214 ns/op)
BenchmarkGet_CacheMiss     22K ops/s      (56 μs/op)
BenchmarkList (100 keys)   120K ops/s     (8.4 μs/op)
BenchmarkCompact           206 ops/s      (4.9 ms/op)
BenchmarkGC                8.7K ops/s     (115 μs/op)
```

Run tests:
```bash
# All tests
go test ./... -v

# With race detector
go test ./tests -race -v

# Benchmarks
go test ./tests -bench=. -benchmem
```

## Performance Characteristics

Stow is optimized for common use cases:

- **Extremely Fast Reads**: 4.7 million cached reads per second
- **Efficient Caching**: In-memory cache with configurable TTL
- **Smart Blob Routing**: Automatically handles large binary data
- **Concurrent Safe**: Fine-grained locking allows parallel operations
- **Async Maintenance**: Non-blocking compact and GC operations
- **Low Overhead**: Minimal memory footprint for metadata

**Best Use Cases**:
- Application configuration storage
- User preference management
- Document storage with media files
- Small to medium datasets (<100K keys per namespace)
- Single-process applications requiring transparency

**Not Recommended For**:
- High-frequency transactional workloads (>10K writes/sec)
- Distributed systems requiring consistency
- Large datasets requiring complex queries
- Real-time applications with <1ms latency requirements

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

Inspired by the need for a transparent, editable, yet structured storage solution that bridges the gap between plain text files and full-featured databases.
