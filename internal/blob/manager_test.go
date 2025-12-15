package blob

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestManagerTotalSize tests the TotalSize method
func TestManagerTotalSize(t *testing.T) {
	t.Run("empty manager", func(t *testing.T) {
		tmpDir := t.TempDir()
		blobDir := filepath.Join(tmpDir, "_blobs")

		manager, err := NewManager(blobDir, 1024*1024, 1024)
		if err != nil {
			t.Fatalf("NewManager failed: %v", err)
		}

		size, err := manager.TotalSize()
		if err != nil {
			t.Errorf("TotalSize failed: %v", err)
		}

		// Empty directory should have size 0 or small (depending on filesystem)
		if size < 0 {
			t.Errorf("TotalSize = %d, should be >= 0", size)
		}
	})

	t.Run("single blob", func(t *testing.T) {
		tmpDir := t.TempDir()
		blobDir := filepath.Join(tmpDir, "_blobs")

		manager, err := NewManager(blobDir, 1024*1024, 1024)
		if err != nil {
			t.Fatalf("NewManager failed: %v", err)
		}

		// Store a blob
		data := []byte("test data for size calculation")
		ref, err := manager.Store(data, "test.txt", "text/plain")
		if err != nil {
			t.Fatalf("Store failed: %v", err)
		}

		size, err := manager.TotalSize()
		if err != nil {
			t.Errorf("TotalSize failed: %v", err)
		}

		// Size should be at least the size of our data
		if size < ref.Size {
			t.Errorf("TotalSize = %d, should be >= %d", size, ref.Size)
		}

		// Size should match file size
		if size != ref.Size {
			t.Logf("TotalSize = %d, ref.Size = %d (may differ due to filesystem overhead)", size, ref.Size)
		}
	})

	t.Run("multiple blobs", func(t *testing.T) {
		tmpDir := t.TempDir()
		blobDir := filepath.Join(tmpDir, "_blobs")

		manager, err := NewManager(blobDir, 1024*1024, 1024)
		if err != nil {
			t.Fatalf("NewManager failed: %v", err)
		}

		// Store multiple blobs
		var totalExpected int64
		for i := 0; i < 5; i++ {
			data := bytes.Repeat([]byte("x"), 1000+i*100)
			ref, err := manager.Store(data, "file"+string(rune('1'+i))+".bin", "application/octet-stream")
			if err != nil {
				t.Fatalf("Store %d failed: %v", i, err)
			}
			totalExpected += ref.Size
		}

		size, err := manager.TotalSize()
		if err != nil {
			t.Errorf("TotalSize failed: %v", err)
		}

		if size != totalExpected {
			t.Errorf("TotalSize = %d, want %d", size, totalExpected)
		}
	})
}

// TestTotalSizeAfterOperations tests TotalSize updates after operations
func TestTotalSizeAfterOperations(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")

	manager, err := NewManager(blobDir, 1024*1024, 1024)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Initial size (empty)
	initialSize, _ := manager.TotalSize()

	// Store a blob
	data := []byte("test data")
	ref, err := manager.Store(data, "test.bin", "")
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Size after store
	sizeAfterStore, _ := manager.TotalSize()
	if sizeAfterStore <= initialSize {
		t.Errorf("Size after store (%d) should be > initial size (%d)", sizeAfterStore, initialSize)
	}

	// Delete the blob
	err = manager.Delete(ref)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Size after delete
	sizeAfterDelete, _ := manager.TotalSize()
	if sizeAfterDelete >= sizeAfterStore {
		t.Errorf("Size after delete (%d) should be < size after store (%d)", sizeAfterDelete, sizeAfterStore)
	}
}

// TestBuildIndexComplete tests buildIndex with various scenarios
func TestBuildIndexComplete(t *testing.T) {
	t.Run("empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		blobDir := filepath.Join(tmpDir, "_blobs")

		manager, err := NewManager(blobDir, 1024*1024, 1024)
		if err != nil {
			t.Fatalf("NewManager failed: %v", err)
		}

		// Check indexes are empty
		count, _ := manager.Count()
		if count != 0 {
			t.Errorf("Count = %d, want 0 for empty directory", count)
		}
	})

	t.Run("existing blobs", func(t *testing.T) {
		tmpDir := t.TempDir()
		blobDir := filepath.Join(tmpDir, "_blobs")
		os.MkdirAll(blobDir, 0755)

		// Create some blob files manually
		testFiles := []struct {
			name string
			data []byte
		}{
			{"file1_abc123.txt", []byte("content1")},
			{"file2_def456.bin", []byte("content2")},
			{"file3_789abc.dat", []byte("content3")},
		}

		for _, tf := range testFiles {
			path := filepath.Join(blobDir, tf.name)
			if err := os.WriteFile(path, tf.data, 0644); err != nil {
				t.Fatalf("Failed to create test file %s: %v", tf.name, err)
			}
		}

		// Create manager (should build index)
		manager, err := NewManager(blobDir, 1024*1024, 1024)
		if err != nil {
			t.Fatalf("NewManager failed: %v", err)
		}

		// Verify files are indexed
		count, err := manager.Count()
		if err != nil {
			t.Fatalf("Count failed: %v", err)
		}

		if count != len(testFiles) {
			t.Errorf("Count = %d, want %d", count, len(testFiles))
		}

		// Verify ListAll returns all files
		files, err := manager.ListAll()
		if err != nil {
			t.Fatalf("ListAll failed: %v", err)
		}

		if len(files) != len(testFiles) {
			t.Errorf("ListAll returned %d files, want %d", len(files), len(testFiles))
		}
	})

	t.Run("skip temporary files", func(t *testing.T) {
		tmpDir := t.TempDir()
		blobDir := filepath.Join(tmpDir, "_blobs")
		os.MkdirAll(blobDir, 0755)

		// Create regular and temporary files
		regularFile := filepath.Join(blobDir, "regular_abc123.bin")
		tmpFile1 := filepath.Join(blobDir, "tmp_12345")
		tmpFile2 := filepath.Join(blobDir, "tmp_67890")

		os.WriteFile(regularFile, []byte("regular"), 0644)
		os.WriteFile(tmpFile1, []byte("temp1"), 0644)
		os.WriteFile(tmpFile2, []byte("temp2"), 0644)

		// Create manager
		manager, err := NewManager(blobDir, 1024*1024, 1024)
		if err != nil {
			t.Fatalf("NewManager failed: %v", err)
		}

		// Should only count regular file
		count, _ := manager.Count()
		if count != 1 {
			t.Errorf("Count = %d, want 1 (should skip tmp_ files)", count)
		}

		// ListAll should also skip tmp files
		files, _ := manager.ListAll()
		if len(files) != 1 {
			t.Errorf("ListAll returned %d files, want 1", len(files))
		}
	})

	t.Run("hash index correctness", func(t *testing.T) {
		tmpDir := t.TempDir()
		blobDir := filepath.Join(tmpDir, "_blobs")

		manager, err := NewManager(blobDir, 1024*1024, 1024)
		if err != nil {
			t.Fatalf("NewManager failed: %v", err)
		}

		// Store a blob
		data := []byte("test content for hash index")
		ref, err := manager.Store(data, "test.txt", "text/plain")
		if err != nil {
			t.Fatalf("Store failed: %v", err)
		}

		// Create new manager instance (should rebuild index)
		manager2, err := NewManager(blobDir, 1024*1024, 1024)
		if err != nil {
			t.Fatalf("Second NewManager failed: %v", err)
		}

		// Try to load using the reference
		loaded, err := manager2.LoadBytes(ref)
		if err != nil {
			t.Fatalf("Load from rebuilt index failed: %v", err)
		}

		if !bytes.Equal(loaded, data) {
			t.Error("Loaded data mismatch after index rebuild")
		}

		// Verify deduplication still works after rebuild
		ref2, err := manager2.Store(data, "duplicate.txt", "text/plain")
		if err != nil {
			t.Fatalf("Store duplicate failed: %v", err)
		}

		if ref2.Hash != ref.Hash {
			t.Error("Deduplication not working after index rebuild")
		}

		// Should still have only 1 file
		count, _ := manager2.Count()
		if count != 1 {
			t.Errorf("Count = %d, want 1 (deduplicated)", count)
		}
	})
}

// TestBuildIndexWithInvalidFiles tests index building with various invalid files
func TestBuildIndexWithInvalidFiles(t *testing.T) {
	t.Run("files with no underscore", func(t *testing.T) {
		tmpDir := t.TempDir()
		blobDir := filepath.Join(tmpDir, "_blobs")
		os.MkdirAll(blobDir, 0755)

		// Create files without hash pattern
		noUnderscoreFile := filepath.Join(blobDir, "noundersc ore.txt")
		os.WriteFile(noUnderscoreFile, []byte("content"), 0644)

		manager, err := NewManager(blobDir, 1024*1024, 1024)
		if err != nil {
			t.Fatalf("NewManager failed: %v", err)
		}

		// Should still count the file
		count, _ := manager.Count()
		if count != 1 {
			t.Errorf("Count = %d, want 1", count)
		}
	})

	t.Run("mixed valid and invalid files", func(t *testing.T) {
		tmpDir := t.TempDir()
		blobDir := filepath.Join(tmpDir, "_blobs")
		os.MkdirAll(blobDir, 0755)

		// Create valid and invalid files
		validFile := filepath.Join(blobDir, "valid_abc123.bin")
		invalidFile1 := filepath.Join(blobDir, "no_hash_pattern.txt")
		invalidFile2 := filepath.Join(blobDir, "tmp_temp")
		validFile2 := filepath.Join(blobDir, "another_def456.dat")

		os.WriteFile(validFile, []byte("valid1"), 0644)
		os.WriteFile(invalidFile1, []byte("invalid1"), 0644)
		os.WriteFile(invalidFile2, []byte("temp"), 0644)
		os.WriteFile(validFile2, []byte("valid2"), 0644)

		manager, err := NewManager(blobDir, 1024*1024, 1024)
		if err != nil {
			t.Fatalf("NewManager failed: %v", err)
		}

		// Should count valid files and skip tmp files
		count, _ := manager.Count()
		// invalidFile1 is counted because it's not a tmp file
		if count != 3 {
			t.Errorf("Count = %d, want 3 (valid files + invalid non-tmp)", count)
		}
	})
}

// TestNewManagerValidation tests manager creation validation
func TestNewManagerValidation(t *testing.T) {
	t.Run("creates directory if not exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		blobDir := filepath.Join(tmpDir, "new_blobs_dir")

		// Directory doesn't exist yet
		if _, err := os.Stat(blobDir); !os.IsNotExist(err) {
			t.Fatal("Directory should not exist yet")
		}

		// Create manager
		manager, err := NewManager(blobDir, 1024*1024, 1024)
		if err != nil {
			t.Fatalf("NewManager failed: %v", err)
		}

		// Directory should now exist
		if _, err := os.Stat(blobDir); os.IsNotExist(err) {
			t.Error("Directory was not created")
		}

		// Should be usable
		data := []byte("test")
		_, err = manager.Store(data, "test.bin", "")
		if err != nil {
			t.Errorf("Store failed: %v", err)
		}
	})

	t.Run("works with existing directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		blobDir := filepath.Join(tmpDir, "_blobs")

		// Pre-create directory
		if err := os.MkdirAll(blobDir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		// Create manager
		_, err := NewManager(blobDir, 1024*1024, 1024)
		if err != nil {
			t.Fatalf("NewManager with existing directory failed: %v", err)
		}
	})
}

// TestManagerIndexConsistency tests that indexes remain consistent
func TestManagerIndexConsistency(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")

	manager, err := NewManager(blobDir, 1024*1024, 1024)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Store multiple blobs with same content
	data := []byte("shared content")

	refs := make([]*Reference, 0)
	names := []string{"file1.txt", "file2.txt", "file3.txt"}

	for _, name := range names {
		ref, err := manager.Store(data, name, "text/plain")
		if err != nil {
			t.Fatalf("Store %s failed: %v", name, err)
		}
		refs = append(refs, ref)
	}

	// All refs should have same hash (deduplication)
	for i := 1; i < len(refs); i++ {
		if refs[i].Hash != refs[0].Hash {
			t.Errorf("ref[%d].Hash = %s, want %s (deduplication failed)", i, refs[i].Hash, refs[0].Hash)
		}
	}

	// Should have only 1 physical file
	count, _ := manager.Count()
	if count != 1 {
		t.Errorf("Count = %d, want 1 (deduplicated)", count)
	}

	// All refs should be loadable
	for i, ref := range refs {
		loaded, err := manager.LoadBytes(ref)
		if err != nil {
			t.Errorf("Load ref[%d] failed: %v", i, err)
		}
		if !bytes.Equal(loaded, data) {
			t.Errorf("Loaded data mismatch for ref[%d]", i)
		}
	}
}

// TestManagerStoreErrors tests error handling in Store
func TestManagerStoreErrors(t *testing.T) {
	t.Run("unsupported data type", func(t *testing.T) {
		tmpDir := t.TempDir()
		blobDir := filepath.Join(tmpDir, "_blobs")

		manager, err := NewManager(blobDir, 1024*1024, 1024)
		if err != nil {
			t.Fatalf("NewManager failed: %v", err)
		}

		// Try to store unsupported type
		_, err = manager.Store(123, "test.bin", "")
		if err == nil {
			t.Error("Store should fail with unsupported data type")
		}
	})

	t.Run("exceeds max size", func(t *testing.T) {
		tmpDir := t.TempDir()
		blobDir := filepath.Join(tmpDir, "_blobs")

		// Create manager with small max size
		maxSize := int64(10)
		manager, err := NewManager(blobDir, maxSize, 1024)
		if err != nil {
			t.Fatalf("NewManager failed: %v", err)
		}

		// Try to store data larger than max size
		largeData := make([]byte, 100)
		_, err = manager.Store(largeData, "large.bin", "")
		if err == nil {
			t.Error("Store should fail when data exceeds max size")
		}
	})
}

// TestManagerLoadErrors tests error handling in Load
func TestManagerLoadErrors(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")

	manager, err := NewManager(blobDir, 1024*1024, 1024)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	t.Run("nil reference", func(t *testing.T) {
		_, err := manager.Load(nil)
		if err == nil {
			t.Error("Load should fail with nil reference")
		}
	})

	t.Run("invalid reference", func(t *testing.T) {
		invalidRef := &Reference{
			IsBlob:   false,
			Location: "",
			Hash:     "",
			Size:     0,
		}
		_, err := manager.Load(invalidRef)
		if err == nil {
			t.Error("Load should fail with invalid reference")
		}
	})

	t.Run("blob not found", func(t *testing.T) {
		nonExistentRef := &Reference{
			IsBlob:   true,
			Location: "_blobs/nonexistent_abc123.bin",
			Hash:     "abc123",
			Size:     100,
		}
		_, err := manager.Load(nonExistentRef)
		if err == nil {
			t.Error("Load should fail when blob file doesn't exist")
		}
	})
}

// TestManagerConcurrentStore tests concurrent Store operations
func TestManagerConcurrentStore(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")

	manager, err := NewManager(blobDir, 1024*1024, 1024)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Store blobs concurrently
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			data := bytes.Repeat([]byte("x"), 100+id)
			_, err := manager.Store(data, "file"+string(rune('0'+id))+".bin", "")
			if err != nil {
				t.Errorf("Concurrent store %d failed: %v", id, err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all blobs were stored
	count, _ := manager.Count()
	if count != 10 {
		t.Errorf("Count = %d, want 10 after concurrent stores", count)
	}
}
