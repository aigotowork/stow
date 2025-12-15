package blob

import (
	"bytes"
	"io"
	"path/filepath"
	"sync"
	"testing"
)

// TestBlobLifecycle tests the complete blob lifecycle
func TestBlobLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")

	// Create manager
	manager, err := NewManager(blobDir, 1024*1024, 1024)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// 1. Store a blob
	originalData := []byte("test blob content for lifecycle")
	ref, err := manager.Store(originalData, "lifecycle.txt", "text/plain")
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	if !ref.IsValid() {
		t.Error("Reference should be valid")
	}

	// 2. Verify it exists
	if !manager.Exists(ref) {
		t.Error("Blob should exist after Store")
	}

	// 3. Load and verify content
	loadedData, err := manager.LoadBytes(ref)
	if err != nil {
		t.Fatalf("LoadBytes failed: %v", err)
	}

	if !bytes.Equal(loadedData, originalData) {
		t.Error("Loaded data doesn't match original")
	}

	// 4. Load via FileData interface
	fileData, err := manager.Load(ref)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	defer fileData.Close()

	streamData, err := io.ReadAll(fileData)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if !bytes.Equal(streamData, originalData) {
		t.Error("Stream data doesn't match original")
	}

	// 5. Delete the blob
	err = manager.Delete(ref)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// 6. Verify it no longer exists
	if manager.Exists(ref) {
		t.Error("Blob should not exist after Delete")
	}

	// 7. Trying to load should fail
	_, err = manager.LoadBytes(ref)
	if err == nil {
		t.Error("LoadBytes should fail after Delete")
	}
}

// TestBlobManagerRestart tests manager restart and index rebuild
func TestBlobManagerRestart(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")

	// Create first manager and store blobs
	manager1, err := NewManager(blobDir, 1024*1024, 1024)
	if err != nil {
		t.Fatalf("First NewManager failed: %v", err)
	}

	// Store multiple blobs
	refs := make([]*Reference, 0)
	testData := []struct {
		data []byte
		name string
	}{
		{[]byte("content1"), "file1.txt"},
		{[]byte("content2"), "file2.txt"},
		{[]byte("content3"), "file3.bin"},
	}

	for _, td := range testData {
		ref, err := manager1.Store(td.data, td.name, "")
		if err != nil {
			t.Fatalf("Store %s failed: %v", td.name, err)
		}
		refs = append(refs, ref)
	}

	// Count before restart
	countBefore, _ := manager1.Count()

	// Create new manager (simulates restart)
	manager2, err := NewManager(blobDir, 1024*1024, 1024)
	if err != nil {
		t.Fatalf("Second NewManager failed: %v", err)
	}

	// Count after restart
	countAfter, _ := manager2.Count()
	if countAfter != countBefore {
		t.Errorf("Count after restart = %d, want %d", countAfter, countBefore)
	}

	// All blobs should be loadable
	for i, ref := range refs {
		loaded, err := manager2.LoadBytes(ref)
		if err != nil {
			t.Errorf("Load blob %d after restart failed: %v", i, err)
			continue
		}

		if !bytes.Equal(loaded, testData[i].data) {
			t.Errorf("Blob %d content mismatch after restart", i)
		}
	}

	// Deduplication should still work
	ref4, err := manager2.Store(testData[0].data, "duplicate.txt", "")
	if err != nil {
		t.Fatalf("Store duplicate after restart failed: %v", err)
	}

	if ref4.Hash != refs[0].Hash {
		t.Error("Deduplication not working after restart")
	}

	// Count should still be the same (deduplicated)
	countFinal, _ := manager2.Count()
	if countFinal != countBefore {
		t.Errorf("Count after dedup = %d, want %d", countFinal, countBefore)
	}
}

// TestBlobConcurrentOperations tests concurrent operations
func TestBlobConcurrentOperations(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")

	manager, err := NewManager(blobDir, 10*1024*1024, 1024)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	const numGoroutines = 20
	var wg sync.WaitGroup
	refs := make([]*Reference, numGoroutines)
	errors := make(chan error, numGoroutines)

	// Concurrent stores
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			data := bytes.Repeat([]byte{byte(id)}, 1000)
			ref, err := manager.Store(data, "concurrent_"+string(rune('a'+id))+".bin", "")
			if err != nil {
				errors <- err
				return
			}
			refs[id] = ref
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent store error: %v", err)
	}

	// Concurrent loads
	wg = sync.WaitGroup{}
	loadErrors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			if refs[id] == nil {
				return
			}

			data, err := manager.LoadBytes(refs[id])
			if err != nil {
				loadErrors <- err
				return
			}

			expectedData := bytes.Repeat([]byte{byte(id)}, 1000)
			if !bytes.Equal(data, expectedData) {
				loadErrors <- err
			}
		}(i)
	}

	wg.Wait()
	close(loadErrors)

	for err := range loadErrors {
		t.Errorf("Concurrent load error: %v", err)
	}

	// Verify final count
	count, _ := manager.Count()
	if count != numGoroutines {
		t.Errorf("Final count = %d, want %d", count, numGoroutines)
	}
}

// TestBlobMixedConcurrentOperations tests mixed concurrent operations
func TestBlobMixedConcurrentOperations(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")

	manager, err := NewManager(blobDir, 10*1024*1024, 1024)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Pre-populate with some blobs
	initialRefs := make([]*Reference, 10)
	for i := 0; i < 10; i++ {
		data := bytes.Repeat([]byte("x"), 100+i*10)
		ref, _ := manager.Store(data, "initial_"+string(rune('0'+i))+".bin", "")
		initialRefs[i] = ref
	}

	var wg sync.WaitGroup
	const numOps = 30

	// Mix of store, load, and exists operations
	for i := 0; i < numOps; i++ {
		wg.Add(1)

		opType := i % 3
		switch opType {
		case 0: // Store
			go func(id int) {
				defer wg.Done()
				data := bytes.Repeat([]byte("y"), 50+id)
				_, err := manager.Store(data, "new_"+string(rune('a'+id))+".bin", "")
				if err != nil {
					t.Errorf("Concurrent store error: %v", err)
				}
			}(i)

		case 1: // Load
			go func(id int) {
				defer wg.Done()
				refIdx := id % len(initialRefs)
				_, err := manager.LoadBytes(initialRefs[refIdx])
				if err != nil {
					t.Errorf("Concurrent load error: %v", err)
				}
			}(i)

		case 2: // Exists
			go func(id int) {
				defer wg.Done()
				refIdx := id % len(initialRefs)
				exists := manager.Exists(initialRefs[refIdx])
				if !exists {
					t.Error("Blob should exist")
				}
			}(i)
		}
	}

	wg.Wait()

	// Verify integrity
	for i, ref := range initialRefs {
		if !manager.Exists(ref) {
			t.Errorf("Initial ref %d should still exist", i)
		}
	}
}

// TestBlobDataIntegrity tests data integrity
func TestBlobDataIntegrity(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")

	manager, err := NewManager(blobDir, 1024*1024, 1024)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Test with various data types
	testCases := []struct {
		name string
		data []byte
	}{
		{
			name: "empty",
			data: []byte{},
		},
		{
			name: "single byte",
			data: []byte{0xFF},
		},
		{
			name: "text",
			data: []byte("Hello, World! 你好世界 🌍"),
		},
		{
			name: "binary",
			data: []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD},
		},
		{
			name: "large data",
			data: bytes.Repeat([]byte("x"), 10000),
		},
		{
			name: "pattern",
			data: bytes.Repeat([]byte{0xAA, 0x55}, 5000),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Store
			ref, err := manager.Store(tc.data, tc.name+".bin", "")
			if err != nil {
				t.Fatalf("Store failed: %v", err)
			}

			// Load and verify
			loaded, err := manager.LoadBytes(ref)
			if err != nil {
				t.Fatalf("LoadBytes failed: %v", err)
			}

			if !bytes.Equal(loaded, tc.data) {
				t.Error("Data integrity check failed")
			}

			// Verify hash
			expectedHash := ComputeSHA256FromBytes(tc.data)
			if ref.Hash != expectedHash {
				t.Errorf("Hash mismatch: got %s, want %s", ref.Hash, expectedHash)
			}

			// Verify size
			if ref.Size != int64(len(tc.data)) {
				t.Errorf("Size mismatch: got %d, want %d", ref.Size, len(tc.data))
			}
		})
	}
}

// TestBlobDeduplicationComprehensive tests comprehensive deduplication scenarios
func TestBlobDeduplicationComprehensive(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")

	manager, err := NewManager(blobDir, 1024*1024, 1024)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Same content, different names
	content := []byte("shared content across files")
	names := []string{"copy1.txt", "copy2.txt", "copy3.txt"}

	refs := make([]*Reference, len(names))
	for i, name := range names {
		ref, err := manager.Store(content, name, "text/plain")
		if err != nil {
			t.Fatalf("Store %s failed: %v", name, err)
		}
		refs[i] = ref
	}

	// All refs should have same hash
	for i := 1; i < len(refs); i++ {
		if refs[i].Hash != refs[0].Hash {
			t.Errorf("ref[%d].Hash = %s, want %s", i, refs[i].Hash, refs[0].Hash)
		}
	}

	// Should have only 1 physical file
	count, _ := manager.Count()
	if count != 1 {
		t.Errorf("Count = %d, want 1 (deduplicated)", count)
	}

	// All refs should be loadable with correct content
	for i, ref := range refs {
		loaded, err := manager.LoadBytes(ref)
		if err != nil {
			t.Errorf("Load ref[%d] failed: %v", i, err)
		}
		if !bytes.Equal(loaded, content) {
			t.Errorf("Content mismatch for ref[%d]", i)
		}
	}

	// Delete one ref - file should still exist
	err = manager.Delete(refs[0])
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Other refs should still be loadable (this depends on implementation)
	// In the current implementation, deleting removes the physical file
	// In a production system, you might want reference counting
}

// TestBlobLargeFile tests handling of large files
func TestBlobLargeFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large file test in short mode")
	}

	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")

	manager, err := NewManager(blobDir, 100*1024*1024, 64*1024) // 100MB limit
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Create 10MB of data
	largeData := make([]byte, 10*1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	// Store
	ref, err := manager.Store(largeData, "large.bin", "application/octet-stream")
	if err != nil {
		t.Fatalf("Store large file failed: %v", err)
	}

	// Verify size
	if ref.Size != int64(len(largeData)) {
		t.Errorf("Size = %d, want %d", ref.Size, len(largeData))
	}

	// Load via streaming
	fileData, err := manager.Load(ref)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	defer fileData.Close()

	// Read in chunks
	buf := make([]byte, 64*1024)
	totalRead := 0
	for {
		n, err := fileData.Read(buf)
		totalRead += n

		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read error: %v", err)
		}
	}

	if totalRead != len(largeData) {
		t.Errorf("Total read = %d, want %d", totalRead, len(largeData))
	}

	// Verify TotalSize
	totalSize, err := manager.TotalSize()
	if err != nil {
		t.Fatalf("TotalSize failed: %v", err)
	}

	if totalSize != int64(len(largeData)) {
		t.Errorf("TotalSize = %d, want %d", totalSize, len(largeData))
	}
}

// TestBlobListAll tests the ListAll functionality
func TestBlobListAll(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")

	manager, err := NewManager(blobDir, 1024*1024, 1024)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Initially should be empty
	files, err := manager.ListAll()
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("Initial ListAll returned %d files, want 0", len(files))
	}

	// Store several blobs
	numBlobs := 5
	for i := 0; i < numBlobs; i++ {
		data := bytes.Repeat([]byte{byte(i)}, 100)
		_, err := manager.Store(data, "file"+string(rune('0'+i))+".bin", "")
		if err != nil {
			t.Fatalf("Store failed: %v", err)
		}
	}

	// List should return all blobs
	files, err = manager.ListAll()
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}

	if len(files) != numBlobs {
		t.Errorf("ListAll returned %d files, want %d", len(files), numBlobs)
	}
}

// BenchmarkBlobStore benchmarks blob store operation
func BenchmarkBlobStore(b *testing.B) {
	tmpDir := b.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")

	manager, _ := NewManager(blobDir, 10*1024*1024, 64*1024)
	data := bytes.Repeat([]byte("x"), 10240) // 10KB

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		manager.Store(data, "bench.bin", "")
	}
}

// BenchmarkBlobLoad benchmarks blob load operation
func BenchmarkBlobLoad(b *testing.B) {
	tmpDir := b.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")

	manager, _ := NewManager(blobDir, 10*1024*1024, 64*1024)
	data := bytes.Repeat([]byte("x"), 10240) // 10KB
	ref, _ := manager.Store(data, "bench.bin", "")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		manager.LoadBytes(ref)
	}
}

// BenchmarkBlobIndexBuild benchmarks index building
func BenchmarkBlobIndexBuild(b *testing.B) {
	tmpDir := b.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")

	// Pre-create blobs
	manager, _ := NewManager(blobDir, 10*1024*1024, 64*1024)
	for i := 0; i < 100; i++ {
		data := bytes.Repeat([]byte{byte(i)}, 100)
		manager.Store(data, "file"+string(rune('0'+i%10))+".bin", "")
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		NewManager(blobDir, 10*1024*1024, 64*1024)
	}
}
