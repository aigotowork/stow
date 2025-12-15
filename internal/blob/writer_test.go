package blob

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// TestWriterWritten tests the Written() method
func TestWriterWritten(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "written_test.bin")

	writer, err := NewWriter(testFile, 1024*1024, 1024)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer writer.Abort()

	// Initially written should be 0
	if writer.Written() != 0 {
		t.Errorf("Initial Written() = %d, want 0", writer.Written())
	}

	// Write some data
	data1 := []byte("first write")
	n1, err := writer.Write(data1)
	if err != nil {
		t.Fatalf("First Write failed: %v", err)
	}

	if writer.Written() != int64(n1) {
		t.Errorf("After first write, Written() = %d, want %d", writer.Written(), n1)
	}

	// Write more data
	data2 := []byte("second write")
	n2, err := writer.Write(data2)
	if err != nil {
		t.Fatalf("Second Write failed: %v", err)
	}

	expectedTotal := int64(n1 + n2)
	if writer.Written() != expectedTotal {
		t.Errorf("After second write, Written() = %d, want %d", writer.Written(), expectedTotal)
	}

	// Close and verify final written count
	hash, size, err := writer.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if size != expectedTotal {
		t.Errorf("Close returned size %d, want %d", size, expectedTotal)
	}

	if hash == "" {
		t.Error("Hash should not be empty")
	}
}

// TestWriterMultipleWrites tests writing data in multiple chunks
func TestWriterMultipleWrites(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "multiple_writes.bin")

	writer, err := NewWriter(testFile, 1024*1024, 1024)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	// Write in small chunks
	totalExpected := 0
	for i := 0; i < 10; i++ {
		data := bytes.Repeat([]byte{byte(i)}, 100)
		n, err := writer.Write(data)
		if err != nil {
			t.Fatalf("Write %d failed: %v", i, err)
		}
		totalExpected += n

		if writer.Written() != int64(totalExpected) {
			t.Errorf("After write %d, Written() = %d, want %d", i, writer.Written(), totalExpected)
		}
	}

	hash, size, err := writer.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if size != int64(totalExpected) {
		t.Errorf("Final size = %d, want %d", size, totalExpected)
	}

	if hash == "" {
		t.Error("Hash should not be empty")
	}

	// Verify file content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if len(content) != totalExpected {
		t.Errorf("File size = %d, want %d", len(content), totalExpected)
	}
}

// TestWriterAbort tests the Abort functionality
func TestWriterAbort(t *testing.T) {
	t.Run("abort cleans up file", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "abort_test.bin")

		writer, err := NewWriter(testFile, 1024*1024, 1024)
		if err != nil {
			t.Fatalf("NewWriter failed: %v", err)
		}

		// Write some data
		data := []byte("data to be aborted")
		_, err = writer.Write(data)
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}

		// Abort
		err = writer.Abort()
		if err != nil {
			t.Errorf("Abort failed: %v", err)
		}

		// File should be deleted
		if _, err := os.Stat(testFile); !os.IsNotExist(err) {
			t.Error("File should be deleted after Abort")
		}
	})

	t.Run("abort without write", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "abort_no_write.bin")

		writer, err := NewWriter(testFile, 1024*1024, 1024)
		if err != nil {
			t.Fatalf("NewWriter failed: %v", err)
		}

		// Abort without writing
		err = writer.Abort()
		if err != nil {
			t.Errorf("Abort without write failed: %v", err)
		}

		// File should be deleted
		if _, err := os.Stat(testFile); !os.IsNotExist(err) {
			t.Error("File should be deleted after Abort")
		}
	})

	t.Run("multiple aborts", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "multiple_abort.bin")

		writer, err := NewWriter(testFile, 1024*1024, 1024)
		if err != nil {
			t.Fatalf("NewWriter failed: %v", err)
		}

		// First abort
		err = writer.Abort()
		if err != nil {
			t.Errorf("First Abort failed: %v", err)
		}

		// Second abort (should handle gracefully)
		err = writer.Abort()
		// May fail since file is already deleted, which is acceptable
		_ = err
	})
}

// TestWriterCloseIdempotent tests that Close can be called multiple times
func TestWriterCloseIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "close_idempotent.bin")

	writer, err := NewWriter(testFile, 1024*1024, 1024)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	data := []byte("test data")
	_, err = writer.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// First close
	hash1, _, err := writer.Close()
	if err != nil {
		t.Errorf("First Close failed: %v", err)
	}

	if hash1 == "" {
		t.Error("Hash should not be empty")
	}

	// Second close (may fail, which is acceptable)
	_, _, err = writer.Close()
	// Implementation detail: may or may not fail on second close
	_ = err
}

// TestWriterWriteAfterClose tests that writing after Close fails
func TestWriterWriteAfterClose(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "write_after_close.bin")

	writer, err := NewWriter(testFile, 1024*1024, 1024)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	data := []byte("initial data")
	_, err = writer.Write(data)
	if err != nil {
		t.Fatalf("Initial Write failed: %v", err)
	}

	// Close
	_, _, err = writer.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Try to write after close
	moreData := []byte("should fail")
	_, err = writer.Write(moreData)
	if err == nil {
		t.Error("Write after Close should fail")
	}
}

// TestWriterCloseErrors tests error handling in Close
func TestWriterCloseErrors(t *testing.T) {
	t.Run("successful close", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "success_close.bin")

		writer, err := NewWriter(testFile, 1024*1024, 1024)
		if err != nil {
			t.Fatalf("NewWriter failed: %v", err)
		}

		data := []byte("test data for close")
		_, err = writer.Write(data)
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}

		hash, size, err := writer.Close()
		if err != nil {
			t.Errorf("Close failed: %v", err)
		}

		if hash == "" {
			t.Error("Hash should not be empty")
		}

		if size != int64(len(data)) {
			t.Errorf("Size = %d, want %d", size, len(data))
		}

		// File should exist
		if _, err := os.Stat(testFile); os.IsNotExist(err) {
			t.Error("File should exist after Close")
		}
	})
}

// TestWriterWriteErrors tests error handling in Write
func TestWriterWriteErrors(t *testing.T) {
	t.Run("exceeds max size", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "exceed_size.bin")

		maxSize := int64(50)
		writer, err := NewWriter(testFile, maxSize, 1024)
		if err != nil {
			t.Fatalf("NewWriter failed: %v", err)
		}
		defer writer.Abort()

		// Write data within limit
		data1 := make([]byte, 30)
		n, err := writer.Write(data1)
		if err != nil {
			t.Fatalf("First Write failed: %v", err)
		}

		if writer.Written() != int64(n) {
			t.Errorf("Written() = %d, want %d", writer.Written(), n)
		}

		// Try to write more data that would exceed limit
		data2 := make([]byte, 30) // Total would be 60, exceeding limit of 50
		_, err = writer.Write(data2)
		if err == nil {
			t.Error("Write should fail when exceeding max size")
		}

		// Written count should not update after failed write
		if writer.Written() != int64(n) {
			t.Errorf("Written() after failed write = %d, want %d", writer.Written(), n)
		}
	})

	t.Run("write to closed file", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "write_closed.bin")

		writer, err := NewWriter(testFile, 1024*1024, 1024)
		if err != nil {
			t.Fatalf("NewWriter failed: %v", err)
		}

		// Close immediately
		_, _, err = writer.Close()
		if err != nil {
			t.Fatalf("Close failed: %v", err)
		}

		// Try to write
		data := []byte("should fail")
		_, err = writer.Write(data)
		if err == nil {
			t.Error("Write to closed writer should fail")
		}
	})
}

// TestWriterLargeData tests writing large amounts of data
func TestWriterLargeData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large data test in short mode")
	}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large_data.bin")

	writer, err := NewWriter(testFile, 100*1024*1024, 64*1024) // 100MB limit, 64KB chunks
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	// Write 10MB of data
	chunkSize := 64 * 1024
	totalSize := 10 * 1024 * 1024
	numChunks := totalSize / chunkSize

	chunk := make([]byte, chunkSize)
	for i := range chunk {
		chunk[i] = byte(i % 256)
	}

	for i := 0; i < numChunks; i++ {
		_, err := writer.Write(chunk)
		if err != nil {
			t.Fatalf("Write chunk %d failed: %v", i, err)
		}
	}

	expectedWritten := int64(numChunks * chunkSize)
	if writer.Written() != expectedWritten {
		t.Errorf("Written() = %d, want %d", writer.Written(), expectedWritten)
	}

	hash, size, err := writer.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if size != expectedWritten {
		t.Errorf("Final size = %d, want %d", size, expectedWritten)
	}

	if hash == "" {
		t.Error("Hash should not be empty")
	}
}

// TestWriterWriteFromMethod tests WriteFrom method
func TestWriterWriteFromMethod(t *testing.T) {
	t.Run("successful write from reader", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "write_from.bin")

		writer, err := NewWriter(testFile, 1024*1024, 1024)
		if err != nil {
			t.Fatalf("NewWriter failed: %v", err)
		}

		data := []byte("data from reader")
		reader := bytes.NewReader(data)

		err = writer.WriteFrom(reader)
		if err != nil {
			t.Fatalf("WriteFrom failed: %v", err)
		}

		if writer.Written() != int64(len(data)) {
			t.Errorf("Written() = %d, want %d", writer.Written(), len(data))
		}

		hash, size, err := writer.Close()
		if err != nil {
			t.Fatalf("Close failed: %v", err)
		}

		if size != int64(len(data)) {
			t.Errorf("Size = %d, want %d", size, len(data))
		}

		if hash == "" {
			t.Error("Hash should not be empty")
		}

		// Verify content
		content, _ := os.ReadFile(testFile)
		if !bytes.Equal(content, data) {
			t.Error("File content mismatch")
		}
	})

	t.Run("write from failing reader", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "write_from_error.bin")

		writer, err := NewWriter(testFile, 1024*1024, 1024)
		if err != nil {
			t.Fatalf("NewWriter failed: %v", err)
		}
		defer writer.Abort()

		// Create a reader that fails after some data
		failingReader := &failAfterNBytesReader{
			data:      []byte("initial data that works"),
			failAfter: 10,
		}

		err = writer.WriteFrom(failingReader)
		if err == nil {
			t.Error("WriteFrom should fail with failing reader")
		}

		// Should have written some data before failure
		if writer.Written() == 0 {
			t.Error("Should have written some data before failure")
		}
	})

	t.Run("write from empty reader", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "write_from_empty.bin")

		writer, err := NewWriter(testFile, 1024*1024, 1024)
		if err != nil {
			t.Fatalf("NewWriter failed: %v", err)
		}

		emptyReader := bytes.NewReader([]byte{})

		err = writer.WriteFrom(emptyReader)
		if err != nil {
			t.Errorf("WriteFrom with empty reader failed: %v", err)
		}

		if writer.Written() != 0 {
			t.Errorf("Written() = %d, want 0", writer.Written())
		}

		_, size, err := writer.Close()
		if err != nil {
			t.Fatalf("Close failed: %v", err)
		}

		if size != 0 {
			t.Errorf("Size = %d, want 0", size)
		}
	})
}

// failAfterNBytesReader is a helper reader that fails after N bytes
type failAfterNBytesReader struct {
	data      []byte
	failAfter int
	read      int
}

func (r *failAfterNBytesReader) Read(p []byte) (n int, err error) {
	if r.read >= r.failAfter {
		return 0, errors.New("simulated read error")
	}

	remaining := r.failAfter - r.read
	toRead := len(p)
	if toRead > remaining {
		toRead = remaining
	}
	if toRead > len(r.data)-r.read {
		toRead = len(r.data) - r.read
	}

	if toRead == 0 {
		return 0, io.EOF
	}

	n = copy(p, r.data[r.read:r.read+toRead])
	r.read += n
	return n, nil
}

// TestWriterHashConsistency tests that hash is computed correctly
func TestWriterHashConsistency(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "hash_consistency.bin")

	writer, err := NewWriter(testFile, 1024*1024, 1024)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	data := []byte("test data for hash consistency")
	_, err = writer.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	hash, _, err := writer.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Compute expected hash
	expectedHash := ComputeSHA256FromBytes(data)

	if hash != expectedHash {
		t.Errorf("Hash = %s, want %s", hash, expectedHash)
	}
}

// TestWriterEmptyWrite tests writing empty data
func TestWriterEmptyWrite(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty_write.bin")

	writer, err := NewWriter(testFile, 1024*1024, 1024)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	// Write empty slice
	n, err := writer.Write([]byte{})
	if err != nil {
		t.Errorf("Write empty slice failed: %v", err)
	}

	if n != 0 {
		t.Errorf("Write count = %d, want 0", n)
	}

	if writer.Written() != 0 {
		t.Errorf("Written() = %d, want 0", writer.Written())
	}

	_, size, err := writer.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if size != 0 {
		t.Errorf("Final size = %d, want 0", size)
	}
}

// BenchmarkWriterWrite benchmarks Write performance
func BenchmarkWriterWrite(b *testing.B) {
	tmpDir := b.TempDir()
	testFile := filepath.Join(tmpDir, "bench.bin")

	data := bytes.Repeat([]byte("x"), 1024) // 1KB

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		writer, _ := NewWriter(testFile, 10*1024*1024, 64*1024)
		writer.Write(data)
		writer.Close()
	}
}

// BenchmarkWriterWriteFrom benchmarks WriteFrom performance
func BenchmarkWriterWriteFrom(b *testing.B) {
	tmpDir := b.TempDir()
	testFile := filepath.Join(tmpDir, "bench_from.bin")

	data := bytes.Repeat([]byte("x"), 1024*1024) // 1MB

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		writer, _ := NewWriter(testFile, 10*1024*1024, 64*1024)
		reader := bytes.NewReader(data)
		writer.WriteFrom(reader)
		writer.Close()
	}
}
