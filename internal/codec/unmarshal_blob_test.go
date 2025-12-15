package codec

import (
	"io"
	"path/filepath"
	"testing"

	"github.com/aigotowork/stow/internal/blob"
)

// ========== unmarshalToMap Tests (0% coverage) ==========

func TestUnmarshalToMapWithBlobs(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, err := blob.NewManager(blobDir, 1024*1024, 1024)
	if err != nil {
		t.Fatalf("Failed to create blob manager: %v", err)
	}

	// Create a test blob
	blobContent := []byte("test blob content")
	ref, err := bm.Store(blobContent, "test.txt", "")
	if err != nil {
		t.Fatalf("Failed to store blob: %v", err)
	}

	unmarshaler := NewUnmarshaler(bm)

	// Create data with blob reference
	data := map[string]interface{}{
		"regular_key": "regular value",
		"blob_key": map[string]interface{}{
			"$blob": true,
			"loc":   ref.Location,
			"hash":  ref.Hash,
			"size":  ref.Size,
		},
		"number_key": float64(123),
	}

	var result map[string]interface{}
	err = unmarshaler.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Check regular value
	if result["regular_key"] != "regular value" {
		t.Errorf("regular_key mismatch: got %v", result["regular_key"])
	}

	// Check number value
	if result["number_key"] != float64(123) {
		t.Errorf("number_key mismatch: got %v", result["number_key"])
	}

	// Check blob was loaded
	blobValue, ok := result["blob_key"].([]byte)
	if !ok {
		t.Fatalf("blob_key should be []byte, got %T", result["blob_key"])
	}

	if string(blobValue) != string(blobContent) {
		t.Errorf("blob content mismatch: got %q, want %q", string(blobValue), string(blobContent))
	}
}

func TestUnmarshalToMapWithMissingBlob(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	unmarshaler := NewUnmarshaler(bm)

	// Mock logger to capture warnings
	mockLogger := &MockLogger{warnings: make([]string, 0)}
	unmarshaler.SetLogger(mockLogger)

	// Create data with non-existent blob reference
	data := map[string]interface{}{
		"key1": "value1",
		"blob_key": map[string]interface{}{
			"$blob": true,
			"loc":   "_blobs/nonexistent.bin",
			"hash":  "abc123",
			"size":  int64(100),
		},
	}

	var result map[string]interface{}
	err := unmarshaler.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal should not error: %v", err)
	}

	// Check regular value exists
	if result["key1"] != "value1" {
		t.Errorf("key1 mismatch: got %v", result["key1"])
	}

	// Blob key should be skipped (not present in result)
	if _, ok := result["blob_key"]; ok {
		t.Error("blob_key should not be present when blob loading fails")
	}

	// Check that warning was logged
	if len(mockLogger.warnings) == 0 {
		t.Error("Expected warning to be logged for missing blob")
	}
}

func TestUnmarshalToMapMultipleBlobs(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	// Create multiple test blobs
	blob1Content := []byte("blob 1 content")
	ref1, _ := bm.Store(blob1Content, "blob1.txt", "")

	blob2Content := []byte("blob 2 content")
	ref2, _ := bm.Store(blob2Content, "blob2.txt", "")

	unmarshaler := NewUnmarshaler(bm)

	// Create data with multiple blob references
	data := map[string]interface{}{
		"blob1": map[string]interface{}{
			"$blob": true,
			"loc":   ref1.Location,
			"hash":  ref1.Hash,
			"size":  ref1.Size,
		},
		"blob2": map[string]interface{}{
			"$blob": true,
			"loc":   ref2.Location,
			"hash":  ref2.Hash,
			"size":  ref2.Size,
		},
		"regular": "value",
	}

	var result map[string]interface{}
	err := unmarshaler.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Check both blobs were loaded
	blob1Value, ok1 := result["blob1"].([]byte)
	blob2Value, ok2 := result["blob2"].([]byte)

	if !ok1 || !ok2 {
		t.Fatal("Both blob keys should be []byte")
	}

	if string(blob1Value) != string(blob1Content) {
		t.Errorf("blob1 content mismatch")
	}

	if string(blob2Value) != string(blob2Content) {
		t.Errorf("blob2 content mismatch")
	}
}

func TestUnmarshalToMapNilMap(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	unmarshaler := NewUnmarshaler(bm)

	data := map[string]interface{}{
		"key1": "value1",
		"key2": float64(42),
	}

	// Start with nil map (should be initialized)
	var result map[string]interface{}
	err := unmarshaler.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result map should be initialized")
	}

	if result["key1"] != "value1" {
		t.Errorf("key1 mismatch: got %v", result["key1"])
	}
}

func TestUnmarshalToMapEmptyData(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	unmarshaler := NewUnmarshaler(bm)

	data := map[string]interface{}{}

	var result map[string]interface{}
	err := unmarshaler.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d items", len(result))
	}
}

// ========== loadBlobAsFileData Tests (0% coverage) ==========

func TestLoadBlobAsFileData(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	// Create a test blob
	blobContent := []byte("test file data content")
	ref, err := bm.Store(blobContent, "test.txt", "")
	if err != nil {
		t.Fatalf("Failed to store blob: %v", err)
	}

	unmarshaler := NewUnmarshaler(bm)

	// Load blob as file data (io.ReadCloser)
	fileData, err := unmarshaler.loadBlobAsFileData(ref)
	if err != nil {
		t.Fatalf("loadBlobAsFileData failed: %v", err)
	}
	defer fileData.Close()

	// Read content
	content, err := io.ReadAll(fileData)
	if err != nil {
		t.Fatalf("Failed to read file data: %v", err)
	}

	if string(content) != string(blobContent) {
		t.Errorf("Content mismatch: got %q, want %q", string(content), string(blobContent))
	}
}

func TestLoadBlobAsFileDataNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	unmarshaler := NewUnmarshaler(bm)

	// Create a reference to non-existent blob
	ref := &blob.Reference{
		IsBlob:   true,
		Location: "_blobs/nonexistent.bin",
		Hash:     "abc123",
		Size:     100,
	}

	// Should fail
	_, err := unmarshaler.loadBlobAsFileData(ref)
	if err == nil {
		t.Error("loadBlobAsFileData should fail for non-existent blob")
	}
}

func TestUnmarshalStructWithInterfaceField(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	// Create a test blob
	blobContent := []byte("interface field blob content")
	ref, err := bm.Store(blobContent, "interface.txt", "")
	if err != nil {
		t.Fatalf("Failed to store blob: %v", err)
	}

	unmarshaler := NewUnmarshaler(bm)

	type DocWithInterface struct {
		Title string
		Data  interface{} // This should receive io.ReadCloser
	}

	// Create data with blob reference in interface field
	data := map[string]interface{}{
		"Title": "Test Doc",
		"Data": map[string]interface{}{
			"$blob": true,
			"loc":   ref.Location,
			"hash":  ref.Hash,
			"size":  ref.Size,
		},
	}

	var doc DocWithInterface
	err = unmarshaler.Unmarshal(data, &doc)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if doc.Title != "Test Doc" {
		t.Errorf("Title mismatch: got %q", doc.Title)
	}

	// Data should be loaded as io.ReadCloser since field is interface{}
	reader, ok := doc.Data.(io.ReadCloser)
	if !ok {
		t.Fatalf("Data should be io.ReadCloser, got %T", doc.Data)
	}
	defer reader.Close()

	// Read content
	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read content: %v", err)
	}

	if string(content) != string(blobContent) {
		t.Errorf("Content mismatch: got %q, want %q", string(content), string(blobContent))
	}
}

func TestUnmarshalStructWithInterfaceFieldMissingBlob(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	unmarshaler := NewUnmarshaler(bm)

	// Mock logger
	mockLogger := &MockLogger{warnings: make([]string, 0)}
	unmarshaler.SetLogger(mockLogger)

	type DocWithInterface struct {
		Title string
		Data  interface{}
	}

	// Create data with non-existent blob reference
	data := map[string]interface{}{
		"Title": "Test Doc",
		"Data": map[string]interface{}{
			"$blob": true,
			"loc":   "_blobs/missing.bin",
			"hash":  "abc123",
			"size":  int64(100),
		},
	}

	var doc DocWithInterface
	err := unmarshaler.Unmarshal(data, &doc)
	if err != nil {
		t.Fatalf("Unmarshal should not error: %v", err)
	}

	// Data should be zero value (nil) due to missing blob
	if doc.Data != nil {
		t.Errorf("Data should be nil for missing blob, got %v", doc.Data)
	}

	// Warning should be logged
	if len(mockLogger.warnings) == 0 {
		t.Error("Expected warning for missing blob")
	}
}

// ========== Integration Tests ==========

func TestUnmarshalBlobRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	marshaler := NewMarshaler(bm)
	unmarshaler := NewUnmarshaler(bm)

	// Marshal data with blobs
	type Doc struct {
		Title   string
		Content []byte
	}

	originalContent := make([]byte, 2048)
	for i := range originalContent {
		originalContent[i] = byte(i % 256)
	}

	original := Doc{
		Title:   "Test Document",
		Content: originalContent,
	}

	data, _, err := marshaler.Marshal(original, MarshalOptions{
		BlobThreshold: 1024,
	})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal back
	var result Doc
	err = unmarshaler.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if result.Title != original.Title {
		t.Errorf("Title mismatch: got %q", result.Title)
	}

	if string(result.Content) != string(original.Content) {
		t.Error("Content mismatch after round trip")
	}
}

func TestUnmarshalMapBlobRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	// Create a blob manually
	blobContent := []byte("map blob content")
	ref, _ := bm.Store(blobContent, "map_blob.txt", "")

	unmarshaler := NewUnmarshaler(bm)

	// Create map with blob reference
	data := map[string]interface{}{
		"name":     "test",
		"version":  float64(1),
		"blobData": ref.ToMap(),
	}

	var result map[string]interface{}
	err := unmarshaler.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Check blob was loaded correctly
	blobValue, ok := result["blobData"].([]byte)
	if !ok {
		t.Fatalf("blobData should be []byte, got %T", result["blobData"])
	}

	if string(blobValue) != string(blobContent) {
		t.Errorf("Blob content mismatch: got %q, want %q", string(blobValue), string(blobContent))
	}
}

// Note: MockLogger is defined in codec_test.go and is reused here
