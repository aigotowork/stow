package stow_test

import (
	"fmt"
	"testing"

	"github.com/aigotowork/stow"
)

// BenchmarkPut_SmallData benchmarks writing small data (< 1KB).
func BenchmarkPut_SmallData(b *testing.B) {
	tmpDir := b.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("bench")

	smallData := map[string]interface{}{
		"name":  "test",
		"value": 42,
		"flag":  true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i)
		if err := ns.Put(key, smallData); err != nil {
			b.Fatalf("Put failed: %v", err)
		}
	}
}

// BenchmarkPut_LargeData benchmarks writing large data (> 1MB).
func BenchmarkPut_LargeData(b *testing.B) {
	tmpDir := b.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("bench")

	type Document struct {
		Title   string
		Content []byte
	}

	// Create 1MB of data
	largeContent := make([]byte, 1024*1024)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	largeData := Document{
		Title:   "Large Document",
		Content: largeContent,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("doc%d", i)
		if err := ns.Put(key, largeData); err != nil {
			b.Fatalf("Put failed: %v", err)
		}
	}
}

// BenchmarkGet_CacheHit benchmarks reading data that's in cache.
func BenchmarkGet_CacheHit(b *testing.B) {
	tmpDir := b.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("bench")

	// Pre-populate data
	testData := map[string]interface{}{
		"name":  "cached",
		"value": 100,
	}
	ns.MustPut("cached_key", testData)

	// Warm up cache
	var warmup map[string]interface{}
	ns.MustGet("cached_key", &warmup)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result map[string]interface{}
		if err := ns.Get("cached_key", &result); err != nil {
			b.Fatalf("Get failed: %v", err)
		}
	}
}

// BenchmarkGet_CacheMiss benchmarks reading data that's not in cache.
func BenchmarkGet_CacheMiss(b *testing.B) {
	tmpDir := b.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("bench")

	// Pre-populate multiple keys
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i)
		data := map[string]interface{}{
			"id":    i,
			"value": fmt.Sprintf("value_%d", i),
		}
		ns.MustPut(key, data)
	}

	// Clear cache by refreshing
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i)
		ns.Refresh(key)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i)
		var result map[string]interface{}
		if err := ns.Get(key, &result); err != nil {
			b.Fatalf("Get failed: %v", err)
		}
	}
}

// BenchmarkGet_WithBlob benchmarks reading data with blob fields.
func BenchmarkGet_WithBlob(b *testing.B) {
	tmpDir := b.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("bench")

	type Document struct {
		Title   string
		Content []byte
	}

	// Pre-populate documents with blobs (> 4KB threshold)
	blobContent := make([]byte, 8192) // 8KB
	for i := range blobContent {
		blobContent[i] = byte(i % 256)
	}

	// Only create 10 documents instead of 100 to speed up setup
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("doc%d", i)
		doc := Document{
			Title:   fmt.Sprintf("Document %d", i),
			Content: blobContent,
		}
		ns.MustPut(key, doc)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("doc%d", i%10)
		var result Document
		if err := ns.Get(key, &result); err != nil {
			b.Fatalf("Get failed: %v", err)
		}
	}
}

// BenchmarkList benchmarks listing 100 keys.
func BenchmarkList(b *testing.B) {
	tmpDir := b.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("bench")

	// Pre-populate 100 keys (reduced from 1000 for faster benchmarking)
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key%03d", i)
		data := map[string]interface{}{
			"id":    i,
			"value": fmt.Sprintf("value_%d", i),
		}
		ns.MustPut(key, data)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		keys, err := ns.List()
		if err != nil {
			b.Fatalf("List failed: %v", err)
		}
		if len(keys) != 100 {
			b.Fatalf("Expected 100 keys, got %d", len(keys))
		}
	}
}

// BenchmarkCompact benchmarks compacting 100 versions (10 keys × 10 versions).
func BenchmarkCompact(b *testing.B) {
	tmpDir := b.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("bench")

	// Create keys with many versions (reduced from 1000 to 100 total versions)
	const numKeys = 10
	const versionsPerKey = 10 // 10 keys × 10 versions = 100 versions

	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key%d", i)
		for v := 0; v < versionsPerKey; v++ {
			data := map[string]interface{}{
				"version": v,
				"data":    fmt.Sprintf("version %d of key %d", v, i),
			}
			ns.MustPut(key, data)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i%numKeys)
		if err := ns.Compact(key); err != nil {
			b.Fatalf("Compact failed: %v", err)
		}
	}
}

// BenchmarkGC benchmarks garbage collecting orphaned blobs.
func BenchmarkGC(b *testing.B) {
	// Skip if short mode
	if testing.Short() {
		b.Skip("Skipping GC benchmark in short mode")
	}

	tmpDir := b.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("bench")

	type Document struct {
		Content []byte
	}

	// Helper to create unique blob content (to avoid deduplication)
	createBlobContent := func(seed int) []byte {
		data := make([]byte, 5120) // > 4KB threshold
		// Make each blob truly unique by varying the pattern
		for i := range data {
			data[i] = byte((seed*7 + i*13) % 256)
		}
		return data
	}

	// Pre-setup: Create some initial orphaned blobs
	for j := 0; j < 5; j++ {
		key := fmt.Sprintf("doc%d", j)
		doc := Document{
			Content: createBlobContent(j),
		}
		ns.MustPut(key, doc)
	}

	// Update to create orphans
	for j := 0; j < 5; j++ {
		key := fmt.Sprintf("doc%d", j)
		doc := Document{
			Content: createBlobContent(j + 100),
		}
		ns.MustPut(key, doc)
	}

	b.ResetTimer()

	// Only benchmark the GC operation itself
	for i := 0; i < b.N; i++ {
		result, err := ns.GC()
		if err != nil {
			b.Fatalf("GC failed: %v", err)
		}

		// After first GC, no more orphans exist, so just measure empty GC
		if i == 0 && result.RemovedBlobs > 0 {
			b.Logf("First GC removed %d blobs, reclaimed %d bytes",
				result.RemovedBlobs, result.ReclaimedSize)
		}
	}
}
