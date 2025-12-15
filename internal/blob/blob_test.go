package blob

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestComputeSHA256(t *testing.T) {
	data := []byte("Hello, Stow!")
	reader := bytes.NewReader(data)

	hash, err := ComputeSHA256(reader)
	if err != nil {
		t.Fatalf("ComputeSHA256 failed: %v", err)
	}

	// Verify hash is not empty
	if hash == "" {
		t.Error("Hash should not be empty")
	}

	// Verify hash is consistent
	reader2 := bytes.NewReader(data)
	hash2, _ := ComputeSHA256(reader2)

	if hash != hash2 {
		t.Error("Hash should be consistent for same data")
	}

	// Verify different data produces different hash
	differentData := []byte("Different data")
	reader3 := bytes.NewReader(differentData)
	hash3, _ := ComputeSHA256(reader3)

	if hash == hash3 {
		t.Error("Different data should produce different hash")
	}
}

func TestComputeSHA256FromBytes(t *testing.T) {
	data := []byte("Test data")
	hash1 := ComputeSHA256FromBytes(data)
	hash2 := ComputeSHA256FromBytes(data)

	if hash1 != hash2 {
		t.Error("Hash should be consistent")
	}

	// Verify length (SHA256 produces 64 hex characters)
	if len(hash1) != 64 {
		t.Errorf("Hash length should be 64, got %d", len(hash1))
	}
}

func TestHashPrefix(t *testing.T) {
	hash := "abcdef1234567890"

	tests := []struct {
		n        int
		expected string
	}{
		{0, "abcdef1234567890"},
		{4, "abcd"},
		{8, "abcdef12"},
		{100, "abcdef1234567890"}, // Should return full hash if n > length
	}

	for _, tt := range tests {
		result := HashPrefix(hash, tt.n)
		if result != tt.expected {
			t.Errorf("HashPrefix(%q, %d) = %q, want %q", hash, tt.n, result, tt.expected)
		}
	}
}

func TestShortHash(t *testing.T) {
	hash := "abcdef1234567890abcdef1234567890"
	short := ShortHash(hash)

	if len(short) != DefaultHashPrefixLength {
		t.Errorf("ShortHash length should be %d, got %d", DefaultHashPrefixLength, len(short))
	}

	if short != hash[:DefaultHashPrefixLength] {
		t.Error("ShortHash should return first N characters")
	}
}

func TestBlobReferenceCreation(t *testing.T) {
	ref := NewReference("_blobs/test.jpg", "abc123", 1024, "image/jpeg", "test.jpg")

	if !ref.IsValid() {
		t.Error("Reference should be valid")
	}

	if !ref.IsBlob {
		t.Error("IsBlob should be true")
	}

	if ref.Location != "_blobs/test.jpg" {
		t.Errorf("Location mismatch: got %q", ref.Location)
	}

	if ref.Size != 1024 {
		t.Errorf("Size mismatch: got %d", ref.Size)
	}
}

func TestBlobReferenceValidation(t *testing.T) {
	tests := []struct {
		name  string
		ref   *Reference
		valid bool
	}{
		{
			name:  "valid reference",
			ref:   &Reference{IsBlob: true, Location: "_blobs/file.jpg", Hash: "abc123", Size: 100},
			valid: true,
		},
		{
			name:  "invalid - IsBlob false",
			ref:   &Reference{IsBlob: false, Location: "_blobs/file.jpg", Hash: "abc123", Size: 100},
			valid: false,
		},
		{
			name:  "invalid - empty location",
			ref:   &Reference{IsBlob: true, Location: "", Hash: "abc123", Size: 100},
			valid: false,
		},
		{
			name:  "invalid - empty hash",
			ref:   &Reference{IsBlob: true, Location: "_blobs/file.jpg", Hash: "", Size: 100},
			valid: false,
		},
		{
			name:  "invalid - negative size",
			ref:   &Reference{IsBlob: true, Location: "_blobs/file.jpg", Hash: "abc123", Size: -1},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.ref.IsValid()
			if result != tt.valid {
				t.Errorf("IsValid() = %v, want %v", result, tt.valid)
			}
		})
	}
}

func TestIsBlobReference(t *testing.T) {
	// Valid blob reference
	data1 := map[string]interface{}{
		"$blob": true,
		"loc":   "_blobs/file.jpg",
	}

	if !IsBlobReference(data1) {
		t.Error("Should recognize blob reference")
	}

	// Not a blob reference
	data2 := map[string]interface{}{
		"name": "test",
		"age":  30,
	}

	if IsBlobReference(data2) {
		t.Error("Should not recognize non-blob data as reference")
	}

	// $blob is false
	data3 := map[string]interface{}{
		"$blob": false,
	}

	if IsBlobReference(data3) {
		t.Error("Should not recognize when $blob is false")
	}
}

func TestFromMap(t *testing.T) {
	data := map[string]interface{}{
		"$blob": true,
		"loc":   "_blobs/test.jpg",
		"hash":  "abc123",
		"size":  float64(1024),
		"mime":  "image/jpeg",
		"name":  "test.jpg",
	}

	ref, ok := FromMap(data)
	if !ok {
		t.Fatal("FromMap should succeed")
	}

	if ref.Location != "_blobs/test.jpg" {
		t.Errorf("Location mismatch: got %q", ref.Location)
	}

	if ref.Hash != "abc123" {
		t.Errorf("Hash mismatch: got %q", ref.Hash)
	}

	if ref.Size != 1024 {
		t.Errorf("Size mismatch: got %d", ref.Size)
	}

	if ref.MimeType != "image/jpeg" {
		t.Errorf("MimeType mismatch: got %q", ref.MimeType)
	}
}

func TestToMap(t *testing.T) {
	ref := NewReference("_blobs/test.jpg", "abc123", 1024, "image/jpeg", "test.jpg")
	m := ref.ToMap()

	if m["$blob"] != true {
		t.Error("$blob should be true")
	}

	if m["loc"] != "_blobs/test.jpg" {
		t.Error("Location mismatch")
	}

	if m["hash"] != "abc123" {
		t.Error("Hash mismatch")
	}

	if m["size"] != int64(1024) {
		t.Error("Size mismatch")
	}
}

func TestWriter(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.bin")

	// Create writer
	writer, err := NewWriter(testFile, 1024*1024, 1024)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	// Write data
	data := []byte("Hello, Stow!")
	n, err := writer.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if n != len(data) {
		t.Errorf("Write count mismatch: got %d, want %d", n, len(data))
	}

	// Close and get hash
	hash, size, err := writer.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if size != int64(len(data)) {
		t.Errorf("Size mismatch: got %d, want %d", size, len(data))
	}

	if hash == "" {
		t.Error("Hash should not be empty")
	}

	// Verify file exists
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("File was not created")
	}
}

func TestWriterMaxSize(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.bin")

	// Create writer with small max size
	maxSize := int64(10)
	writer, err := NewWriter(testFile, maxSize, 1024)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	// Try to write more than max size
	data := make([]byte, 20)
	_, err = writer.Write(data)
	if err == nil {
		t.Error("Write should fail when exceeding max size")
	}

	writer.Abort()
}

func TestWriterWriteFrom(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.bin")

	writer, _ := NewWriter(testFile, 1024*1024, 64)

	// Write from reader
	data := []byte("Test data from reader")
	reader := bytes.NewReader(data)

	err := writer.WriteFrom(reader)
	if err != nil {
		t.Fatalf("WriteFrom failed: %v", err)
	}

	hash, size, _ := writer.Close()

	if size != int64(len(data)) {
		t.Errorf("Size mismatch: got %d, want %d", size, len(data))
	}

	if hash == "" {
		t.Error("Hash should not be empty")
	}

	// Verify file content
	content, _ := os.ReadFile(testFile)
	if !bytes.Equal(content, data) {
		t.Error("File content mismatch")
	}
}

func TestBlobManager(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")

	// Create manager
	manager, err := NewManager(blobDir, 1024*1024, 1024)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Test storing bytes
	data := []byte("Test blob data")
	ref, err := manager.Store(data, "test.bin", "application/octet-stream")
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	if !ref.IsValid() {
		t.Error("Returned reference should be valid")
	}

	// Test loading
	loaded, err := manager.LoadBytes(ref)
	if err != nil {
		t.Fatalf("LoadBytes failed: %v", err)
	}

	if !bytes.Equal(loaded, data) {
		t.Error("Loaded data mismatch")
	}

	// Test exists
	if !manager.Exists(ref) {
		t.Error("Blob should exist")
	}

	// Test count
	count, _ := manager.Count()
	if count != 1 {
		t.Errorf("Expected 1 blob, got %d", count)
	}

	// Test delete
	err = manager.Delete(ref)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if manager.Exists(ref) {
		t.Error("Blob should not exist after delete")
	}
}

func TestBlobManagerStoreReader(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")

	manager, _ := NewManager(blobDir, 1024*1024, 1024)

	// Store from reader
	data := []byte("Data from reader")
	reader := bytes.NewReader(data)

	ref, err := manager.Store(reader, "test.bin", "")
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Load and verify
	loaded, _ := manager.LoadBytes(ref)
	if !bytes.Equal(loaded, data) {
		t.Error("Data mismatch")
	}
}

func TestFileData(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.bin")

	// Create test file
	testData := []byte("Test file content")
	os.WriteFile(testFile, testData, 0644)

	// Create FileData
	fileData := NewFileData(testFile, "test.bin", int64(len(testData)), "text/plain", "abc123")

	// Test metadata
	if fileData.Name() != "test.bin" {
		t.Errorf("Name mismatch: got %q", fileData.Name())
	}

	if fileData.Size() != int64(len(testData)) {
		t.Errorf("Size mismatch: got %d", fileData.Size())
	}

	if fileData.MimeType() != "text/plain" {
		t.Errorf("MimeType mismatch: got %q", fileData.MimeType())
	}

	if fileData.Hash() != "abc123" {
		t.Errorf("Hash mismatch: got %q", fileData.Hash())
	}

	// Test reading
	buf := make([]byte, len(testData))
	n, err := fileData.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Read failed: %v", err)
	}

	if n != len(testData) {
		t.Errorf("Read count mismatch: got %d, want %d", n, len(testData))
	}

	if !bytes.Equal(buf, testData) {
		t.Error("Read data mismatch")
	}

	// Test close
	err = fileData.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestBlobDeduplication(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")

	// Create manager
	manager, err := NewManager(blobDir, 1024*1024, 1024)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Store the same content twice with different names
	data := []byte("This is duplicate content")

	// First store
	ref1, err := manager.Store(data, "file1.txt", "text/plain")
	if err != nil {
		t.Fatalf("First Store failed: %v", err)
	}

	// Second store with same content but different name
	ref2, err := manager.Store(data, "file2.txt", "text/plain")
	if err != nil {
		t.Fatalf("Second Store failed: %v", err)
	}

	// Both references should have the same hash
	if ref1.Hash != ref2.Hash {
		t.Errorf("Hash mismatch: ref1=%s, ref2=%s", ref1.Hash, ref2.Hash)
	}

	// Verify only one physical file was created
	count, err := manager.Count()
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 blob file (deduplicated), got %d", count)
	}

	// Both references should be loadable
	loaded1, err := manager.LoadBytes(ref1)
	if err != nil {
		t.Fatalf("LoadBytes ref1 failed: %v", err)
	}
	if !bytes.Equal(loaded1, data) {
		t.Error("Loaded data from ref1 mismatch")
	}

	loaded2, err := manager.LoadBytes(ref2)
	if err != nil {
		t.Fatalf("LoadBytes ref2 failed: %v", err)
	}
	if !bytes.Equal(loaded2, data) {
		t.Error("Loaded data from ref2 mismatch")
	}

	// Store different content with same name (should create new file)
	differentData := []byte("This is different content")
	ref3, err := manager.Store(differentData, "file1.txt", "text/plain")
	if err != nil {
		t.Fatalf("Third Store failed: %v", err)
	}

	// Should have different hash
	if ref3.Hash == ref1.Hash {
		t.Error("Different content should have different hash")
	}

	// Now should have 2 files
	count, _ = manager.Count()
	if count != 2 {
		t.Errorf("Expected 2 blob files, got %d", count)
	}

	// Test deduplication with reader
	reader := bytes.NewReader(data)
	ref4, err := manager.Store(reader, "file3.txt", "text/plain")
	if err != nil {
		t.Fatalf("Fourth Store (reader) failed: %v", err)
	}

	// Should have same hash as ref1/ref2
	if ref4.Hash != ref1.Hash {
		t.Error("Reader-based store should deduplicate to same hash")
	}

	// Still should have only 2 files
	count, _ = manager.Count()
	if count != 2 {
		t.Errorf("Expected 2 blob files (deduplicated), got %d", count)
	}
}
