package fsutil

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// ========== AtomicWriteFile Tests ==========

func TestAtomicWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Test basic write
	data := []byte("Hello, Stow!")
	err := AtomicWriteFile(testFile, data, 0644)
	if err != nil {
		t.Fatalf("AtomicWriteFile failed: %v", err)
	}

	// Verify file exists
	if !FileExists(testFile) {
		t.Fatal("File was not created")
	}

	// Verify content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != string(data) {
		t.Fatalf("Content mismatch: got %q, want %q", string(content), string(data))
	}

	// Test overwrite
	newData := []byte("Updated content")
	err = AtomicWriteFile(testFile, newData, 0644)
	if err != nil {
		t.Fatalf("AtomicWriteFile (overwrite) failed: %v", err)
	}

	content, _ = os.ReadFile(testFile)
	if string(content) != string(newData) {
		t.Fatalf("Overwrite failed: got %q, want %q", string(content), string(newData))
	}
}

func TestAtomicWriteFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name string
		perm os.FileMode
	}{
		{"read-only", 0444},
		{"read-write", 0644},
		{"full", 0755},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, "perm_"+tt.name+".txt")
			err := AtomicWriteFile(testFile, []byte("test"), tt.perm)
			if err != nil {
				t.Fatalf("AtomicWriteFile failed: %v", err)
			}

			info, err := os.Stat(testFile)
			if err != nil {
				t.Fatalf("Failed to stat file: %v", err)
			}

			// Check permissions (masking out type bits)
			gotPerm := info.Mode().Perm()
			if gotPerm != tt.perm {
				t.Errorf("Permissions = %o, want %o", gotPerm, tt.perm)
			}
		})
	}
}

func TestAtomicWriteFileParentDirCreation(t *testing.T) {
	tmpDir := t.TempDir()

	// Test with nested non-existent parent directories
	testFile := filepath.Join(tmpDir, "parent", "child", "test.txt")
	err := AtomicWriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("AtomicWriteFile failed to create parent dirs: %v", err)
	}

	if !FileExists(testFile) {
		t.Error("File was not created with parent directories")
	}
}

func TestAtomicWriteFileEmptyContent(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.txt")

	// Test writing empty content
	err := AtomicWriteFile(testFile, []byte{}, 0644)
	if err != nil {
		t.Fatalf("AtomicWriteFile failed with empty content: %v", err)
	}

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if len(content) != 0 {
		t.Errorf("Expected empty file, got %d bytes", len(content))
	}
}

func TestAtomicWriteFileLargeContent(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.txt")

	// Create 10MB of data
	data := make([]byte, 10*1024*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	err := AtomicWriteFile(testFile, data, 0644)
	if err != nil {
		t.Fatalf("AtomicWriteFile failed with large content: %v", err)
	}

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if len(content) != len(data) {
		t.Errorf("Content length mismatch: got %d, want %d", len(content), len(data))
	}
}

func TestAtomicWriteFileSpecialCharsInName(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		filename string
	}{
		{"spaces", "file with spaces.txt"},
		{"unicode", "文件名.txt"},
		{"dots", "file.name.with.dots.txt"},
		{"underscores", "file_with_underscores.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, tt.filename)
			err := AtomicWriteFile(testFile, []byte("test"), 0644)
			if err != nil {
				t.Fatalf("AtomicWriteFile failed with filename %q: %v", tt.filename, err)
			}

			if !FileExists(testFile) {
				t.Errorf("File %q was not created", tt.filename)
			}
		})
	}
}

func TestAtomicWriteFileConcurrent(t *testing.T) {
	tmpDir := t.TempDir()

	const goroutines = 20
	const iterations = 10

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				testFile := filepath.Join(tmpDir, "concurrent_"+strings.Repeat("a", id)+".txt")
				data := []byte("test")
				if err := AtomicWriteFile(testFile, data, 0644); err != nil {
					t.Errorf("Concurrent write failed: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify all files were created
	files, err := ListFiles(tmpDir)
	if err != nil {
		t.Fatalf("Failed to list files: %v", err)
	}

	if len(files) != goroutines {
		t.Errorf("Expected %d files, got %d", goroutines, len(files))
	}
}

func TestAtomicWriteFileErrorHandling(t *testing.T) {
	// Test writing to read-only directory (if we can create one)
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	os.Mkdir(readOnlyDir, 0755)

	// Make directory read-only
	os.Chmod(readOnlyDir, 0444)
	defer os.Chmod(readOnlyDir, 0755) // Cleanup

	testFile := filepath.Join(readOnlyDir, "test.txt")
	err := AtomicWriteFile(testFile, []byte("test"), 0644)
	if err == nil {
		t.Error("AtomicWriteFile should fail when writing to read-only directory")
	}
}

// ========== SafeRename Tests ==========

func TestSafeRename(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file
	srcFile := filepath.Join(tmpDir, "source.txt")
	os.WriteFile(srcFile, []byte("test"), 0644)

	// Rename
	dstFile := filepath.Join(tmpDir, "dest.txt")
	err := SafeRename(srcFile, dstFile)
	if err != nil {
		t.Fatalf("SafeRename failed: %v", err)
	}

	// Verify source doesn't exist
	if FileExists(srcFile) {
		t.Error("Source file should not exist after rename")
	}

	// Verify destination exists
	if !FileExists(dstFile) {
		t.Error("Destination file should exist after rename")
	}

	// Verify content
	content, _ := os.ReadFile(dstFile)
	if string(content) != "test" {
		t.Errorf("Content mismatch after rename: got %q", string(content))
	}
}

func TestSafeRenameOverwrite(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source and destination files
	srcFile := filepath.Join(tmpDir, "source.txt")
	dstFile := filepath.Join(tmpDir, "dest.txt")
	os.WriteFile(srcFile, []byte("new content"), 0644)
	os.WriteFile(dstFile, []byte("old content"), 0644)

	// Rename (should overwrite)
	err := SafeRename(srcFile, dstFile)
	if err != nil {
		t.Fatalf("SafeRename failed: %v", err)
	}

	// Verify new content
	content, _ := os.ReadFile(dstFile)
	if string(content) != "new content" {
		t.Errorf("Content after overwrite: got %q, want %q", string(content), "new content")
	}
}

func TestSafeRenameNonExistent(t *testing.T) {
	tmpDir := t.TempDir()

	srcFile := filepath.Join(tmpDir, "nonexistent.txt")
	dstFile := filepath.Join(tmpDir, "dest.txt")

	err := SafeRename(srcFile, dstFile)
	if err == nil {
		t.Error("SafeRename should fail with non-existent source")
	}
}

// ========== syncDir Tests ==========

func TestSyncDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file in the directory
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	// Sync directory
	err := syncDir(tmpDir)
	if err != nil {
		t.Fatalf("syncDir failed: %v", err)
	}
}

func TestSyncDirNonExistent(t *testing.T) {
	err := syncDir("/nonexistent/path")
	if err == nil {
		t.Error("syncDir should fail with non-existent directory")
	}
}

func TestSyncDirFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "file.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	// syncDir on a file may succeed or fail depending on OS
	// Just verify it doesn't panic
	_ = syncDir(testFile)
}

// ========== Integration Tests ==========

func TestAtomicWriteFileFullFlow(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "flow_test.txt")

	// Write initial content
	err := AtomicWriteFile(testFile, []byte("v1"), 0644)
	if err != nil {
		t.Fatalf("Initial write failed: %v", err)
	}

	// Verify temp file was cleaned up
	tmpFile := testFile + ".tmp"
	if FileExists(tmpFile) {
		t.Error("Temporary file should be cleaned up")
	}

	// Update multiple times
	for i := 2; i <= 5; i++ {
		data := []byte("v" + strings.Repeat("x", i))
		err := AtomicWriteFile(testFile, data, 0644)
		if err != nil {
			t.Fatalf("Update %d failed: %v", i, err)
		}

		// Verify content
		content, _ := os.ReadFile(testFile)
		if string(content) != string(data) {
			t.Errorf("Content mismatch at update %d", i)
		}

		// Verify no temp file
		if FileExists(tmpFile) {
			t.Errorf("Temporary file exists after update %d", i)
		}
	}
}

func TestAtomicOperationsAreAtomic(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "atomic.txt")

	// Write initial content
	AtomicWriteFile(testFile, []byte("initial"), 0644)

	// Concurrent reads and writes
	var wg sync.WaitGroup
	const readers = 10
	const writers = 5

	wg.Add(readers + writers)

	errors := make(chan string, readers)

	// Readers - should always read complete content, never partial
	for i := 0; i < readers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				content, err := os.ReadFile(testFile)
				if err != nil {
					continue
				}
				// Content should be complete, not partial
				str := string(content)
				// Check if content is either initial or a valid update
				if len(str) > 0 && str != "initial" && !strings.HasPrefix(str, "update") {
					errors <- str
					return
				}
			}
		}()
	}

	// Writers
	for i := 0; i < writers; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				data := []byte("update_" + strings.Repeat("x", id*10+j))
				AtomicWriteFile(testFile, data, 0644)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any partial reads
	for str := range errors {
		t.Errorf("Read partial/corrupt content: %q", str)
	}

	// Final verification
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read final content: %v", err)
	}

	// Should have complete content
	if len(content) == 0 {
		t.Error("File should not be empty")
	}
}
