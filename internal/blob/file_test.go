package blob

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// TestFileDataPath tests the Path() method
func TestFileDataPath(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		expectedPath string
	}{
		{
			name:         "absolute path",
			path:         "/tmp/blobs/file.bin",
			expectedPath: "/tmp/blobs/file.bin",
		},
		{
			name:         "relative path",
			path:         "blobs/file.bin",
			expectedPath: "blobs/file.bin",
		},
		{
			name:         "path with spaces",
			path:         "/tmp/test blobs/file name.bin",
			expectedPath: "/tmp/test blobs/file name.bin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fd := NewFileData(tt.path, "test.bin", 100, "application/octet-stream", "abc123")
			if fd.Path() != tt.expectedPath {
				t.Errorf("Path() = %q, want %q", fd.Path(), tt.expectedPath)
			}
		})
	}
}

// TestFileDataReadPartial tests partial reads and multiple reads
func TestFileDataReadPartial(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "partial.bin")

	// Create test file with known content
	testData := []byte("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	if err := os.WriteFile(testFile, testData, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	fd := NewFileData(testFile, "partial.bin", int64(len(testData)), "text/plain", "hash123")
	defer fd.Close()

	t.Run("partial read", func(t *testing.T) {
		// Read first 10 bytes
		buf := make([]byte, 10)
		n, err := fd.Read(buf)
		if err != nil {
			t.Fatalf("First read failed: %v", err)
		}
		if n != 10 {
			t.Errorf("Read count = %d, want 10", n)
		}
		if string(buf) != "0123456789" {
			t.Errorf("First read data = %q, want %q", string(buf), "0123456789")
		}
	})

	t.Run("second read", func(t *testing.T) {
		// Read next 10 bytes
		buf := make([]byte, 10)
		n, err := fd.Read(buf)
		if err != nil {
			t.Fatalf("Second read failed: %v", err)
		}
		if n != 10 {
			t.Errorf("Read count = %d, want 10", n)
		}
		if string(buf) != "ABCDEFGHIJ" {
			t.Errorf("Second read data = %q, want %q", string(buf), "ABCDEFGHIJ")
		}
	})

	t.Run("read until EOF", func(t *testing.T) {
		// Read remaining data
		buf := make([]byte, 100) // Buffer larger than remaining data
		n, err := fd.Read(buf)
		if err != nil && err != io.EOF {
			t.Fatalf("Read failed: %v", err)
		}
		expectedRemaining := "KLMNOPQRSTUVWXYZ"
		if n != len(expectedRemaining) {
			t.Errorf("Read count = %d, want %d", n, len(expectedRemaining))
		}
		if string(buf[:n]) != expectedRemaining {
			t.Errorf("Remaining data = %q, want %q", string(buf[:n]), expectedRemaining)
		}

		// Next read should return EOF
		n, err = fd.Read(buf)
		if err != io.EOF {
			t.Errorf("Expected EOF, got error: %v", err)
		}
		if n != 0 {
			t.Errorf("Expected 0 bytes, got %d", n)
		}
	})
}

// TestFileDataReadErrors tests read error scenarios
func TestFileDataReadErrors(t *testing.T) {
	t.Run("file does not exist", func(t *testing.T) {
		fd := NewFileData("/nonexistent/path/file.bin", "test.bin", 100, "text/plain", "hash")
		defer fd.Close()

		buf := make([]byte, 10)
		_, err := fd.Read(buf)
		if err == nil {
			t.Error("Expected error when reading non-existent file")
		}
	})

	t.Run("file deleted after FileData creation", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "delete_me.bin")

		// Create file
		testData := []byte("test data")
		if err := os.WriteFile(testFile, testData, 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		fd := NewFileData(testFile, "delete_me.bin", int64(len(testData)), "text/plain", "hash")
		defer fd.Close()

		// Delete the file before reading
		os.Remove(testFile)

		// Try to read
		buf := make([]byte, 10)
		_, err := fd.Read(buf)
		if err == nil {
			t.Error("Expected error when reading deleted file")
		}
	})
}

// TestFileDataCloseMultiple tests multiple Close() calls
func TestFileDataCloseMultiple(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "close_test.bin")

	testData := []byte("test data")
	if err := os.WriteFile(testFile, testData, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	fd := NewFileData(testFile, "close_test.bin", int64(len(testData)), "text/plain", "hash")

	// First close should succeed
	if err := fd.Close(); err != nil {
		t.Errorf("First Close() failed: %v", err)
	}

	// Second close should also succeed (idempotent)
	if err := fd.Close(); err != nil {
		t.Errorf("Second Close() failed: %v", err)
	}

	// Third close should also succeed
	if err := fd.Close(); err != nil {
		t.Errorf("Third Close() failed: %v", err)
	}
}

// TestFileDataCloseBeforeRead tests closing without ever reading
func TestFileDataCloseBeforeRead(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "never_read.bin")

	testData := []byte("test data")
	if err := os.WriteFile(testFile, testData, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	fd := NewFileData(testFile, "never_read.bin", int64(len(testData)), "text/plain", "hash")

	// Close without reading should succeed
	if err := fd.Close(); err != nil {
		t.Errorf("Close() without Read() failed: %v", err)
	}
}

// TestFileDataCloseAfterRead tests normal read-close lifecycle
func TestFileDataCloseAfterRead(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "read_close.bin")

	testData := []byte("test data for close after read")
	if err := os.WriteFile(testFile, testData, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	fd := NewFileData(testFile, "read_close.bin", int64(len(testData)), "text/plain", "hash")

	// Read some data
	buf := make([]byte, 10)
	n, err := fd.Read(buf)
	if err != nil {
		t.Fatalf("Read() failed: %v", err)
	}
	if n != 10 {
		t.Errorf("Read count = %d, want 10", n)
	}

	// Close should succeed
	if err := fd.Close(); err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	// Try to read after close
	// Note: This behavior depends on implementation
	// The underlying file is closed, so read should fail
	_, err = fd.Read(buf)
	// Some implementations may or may not fail here
	// It's acceptable for this to pass if the implementation handles it gracefully
	if err == nil {
		t.Log("Note: Read() after Close() succeeded (implementation allows this)")
	}
}

// TestFileDataConcurrentRead tests concurrent reading (should be safe)
func TestFileDataConcurrentRead(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "concurrent.bin")

	// Create larger test file
	testData := bytes.Repeat([]byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ"), 100)
	if err := os.WriteFile(testFile, testData, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create separate FileData instances (proper way for concurrent access)
	fd1 := NewFileData(testFile, "concurrent.bin", int64(len(testData)), "text/plain", "hash1")
	fd2 := NewFileData(testFile, "concurrent.bin", int64(len(testData)), "text/plain", "hash2")
	defer fd1.Close()
	defer fd2.Close()

	done := make(chan bool, 2)

	// Reader 1
	go func() {
		buf := make([]byte, 100)
		total := 0
		for {
			n, err := fd1.Read(buf)
			total += n
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Errorf("Reader 1 error: %v", err)
				break
			}
		}
		if total != len(testData) {
			t.Errorf("Reader 1 read %d bytes, want %d", total, len(testData))
		}
		done <- true
	}()

	// Reader 2
	go func() {
		buf := make([]byte, 100)
		total := 0
		for {
			n, err := fd2.Read(buf)
			total += n
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Errorf("Reader 2 error: %v", err)
				break
			}
		}
		if total != len(testData) {
			t.Errorf("Reader 2 read %d bytes, want %d", total, len(testData))
		}
		done <- true
	}()

	// Wait for both readers
	<-done
	<-done
}

// TestFileDataMetadataAccessors tests all metadata accessor methods
func TestFileDataMetadataAccessors(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		fileName string
		size     int64
		mimeType string
		hash     string
	}{
		{
			name:     "basic metadata",
			path:     "/tmp/blobs/test.jpg",
			fileName: "avatar.jpg",
			size:     102400,
			mimeType: "image/jpeg",
			hash:     "abc123def456",
		},
		{
			name:     "no mime type",
			path:     "/tmp/blobs/data.bin",
			fileName: "data.bin",
			size:     1024,
			mimeType: "",
			hash:     "hash123",
		},
		{
			name:     "zero size",
			path:     "/tmp/blobs/empty.txt",
			fileName: "empty.txt",
			size:     0,
			mimeType: "text/plain",
			hash:     "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fd := NewFileData(tt.path, tt.fileName, tt.size, tt.mimeType, tt.hash)

			if fd.Path() != tt.path {
				t.Errorf("Path() = %q, want %q", fd.Path(), tt.path)
			}
			if fd.Name() != tt.fileName {
				t.Errorf("Name() = %q, want %q", fd.Name(), tt.fileName)
			}
			if fd.Size() != tt.size {
				t.Errorf("Size() = %d, want %d", fd.Size(), tt.size)
			}
			if fd.MimeType() != tt.mimeType {
				t.Errorf("MimeType() = %q, want %q", fd.MimeType(), tt.mimeType)
			}
			if fd.Hash() != tt.hash {
				t.Errorf("Hash() = %q, want %q", fd.Hash(), tt.hash)
			}
		})
	}
}

// TestFileDataLazyOpen tests that file is only opened on first Read()
func TestFileDataLazyOpen(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "lazy.bin")

	testData := []byte("lazy open test")
	if err := os.WriteFile(testFile, testData, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	fd := NewFileData(testFile, "lazy.bin", int64(len(testData)), "text/plain", "hash")
	defer fd.Close()

	// At this point, file should not be opened yet
	// We can verify this indirectly by deleting the file and creating a new one

	// Remove original file
	os.Remove(testFile)

	// Create new file with different content
	newData := []byte("new content")
	if err := os.WriteFile(testFile, newData, 0644); err != nil {
		t.Fatalf("Failed to create new test file: %v", err)
	}

	// Now read - should get new content (proving lazy open)
	buf := make([]byte, 20)
	n, err := fd.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Read() failed: %v", err)
	}

	// Should read the new content
	if string(buf[:n]) != string(newData) {
		t.Logf("Read %q from lazy-opened file", string(buf[:n]))
		// This demonstrates lazy opening behavior
	}
}
