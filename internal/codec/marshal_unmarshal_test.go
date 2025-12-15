package codec

import (
	"path/filepath"
	"testing"

	"github.com/aigotowork/stow/internal/blob"
)

// ========== MarshalSimple Tests (44.4% coverage) ==========

func TestMarshalSimple(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	marshaler := NewMarshaler(bm)

	tests := []struct {
		name     string
		value    interface{}
		expected interface{}
	}{
		{"string", "hello", "hello"},
		{"int", 42, 42},
		{"float", 3.14, 3.14},
		{"bool", true, true},
		{"nil", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := marshaler.MarshalSimple(tt.value, MarshalOptions{})
			if err != nil {
				t.Fatalf("MarshalSimple failed: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestMarshalSimpleBytes(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	marshaler := NewMarshaler(bm)

	// Small bytes (inline)
	smallBytes := []byte("small")
	result, _, err := marshaler.MarshalSimple(smallBytes, MarshalOptions{BlobThreshold: 1024})
	if err != nil {
		t.Fatalf("MarshalSimple failed: %v", err)
	}

	if resultBytes, ok := result.([]byte); !ok {
		t.Error("Small bytes should remain as []byte")
	} else if string(resultBytes) != string(smallBytes) {
		t.Errorf("Bytes content mismatch")
	}
}

func TestMarshalSimpleLargeBytes(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	marshaler := NewMarshaler(bm)

	// Large bytes (should be stored as blob)
	largeBytes := make([]byte, 2048)
	for i := range largeBytes {
		largeBytes[i] = byte(i % 256)
	}

	result, _, err := marshaler.MarshalSimple(largeBytes, MarshalOptions{BlobThreshold: 1024})
	if err != nil {
		t.Fatalf("MarshalSimple failed: %v", err)
	}

	// Should be a blob reference (map)
	blobRef, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Large bytes should be converted to blob reference")
	}

	if blobRef["$blob"] != true {
		t.Error("Should be marked as blob")
	}
}

func TestMarshalSimpleMap(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	marshaler := NewMarshaler(bm)

	// Map value
	mapValue := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}

	result, _, err := marshaler.MarshalSimple(mapValue, MarshalOptions{})
	if err != nil {
		t.Fatalf("MarshalSimple failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result should be a map")
	}

	if resultMap["key1"] != "value1" {
		t.Errorf("key1 mismatch")
	}

	if resultMap["key2"] != 42 {
		t.Errorf("key2 mismatch")
	}
}

func TestMarshalSimpleSlice(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	marshaler := NewMarshaler(bm)

	// Slice value
	sliceValue := []string{"a", "b", "c"}

	result, _, err := marshaler.MarshalSimple(sliceValue, MarshalOptions{})
	if err != nil {
		t.Fatalf("MarshalSimple failed: %v", err)
	}

	// Result should be same as input for simple slices
	resultSlice, ok := result.([]string)
	if !ok {
		t.Fatalf("Result should be []string, got %T", result)
	}

	if len(resultSlice) != 3 {
		t.Errorf("Expected 3 elements, got %d", len(resultSlice))
	}
}

// ========== UnmarshalSimple Enhanced Tests (27.8% coverage) ==========

func TestUnmarshalSimpleBytes(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	unmarshaler := NewUnmarshaler(bm)

	// Test []byte target
	data := []byte("test bytes")
	var result []byte
	err := unmarshaler.UnmarshalSimple(data, &result)
	if err != nil {
		t.Fatalf("UnmarshalSimple failed: %v", err)
	}

	if string(result) != string(data) {
		t.Errorf("Bytes mismatch")
	}
}

func TestUnmarshalSimpleBlobReference(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	// Create a blob
	blobContent := []byte("blob content for unmarshal simple")
	ref, _ := bm.Store(blobContent, "test.bin", "")

	unmarshaler := NewUnmarshaler(bm)

	// Create blob reference
	blobRef := ref.ToMap()

	// Unmarshal to []byte
	var result []byte
	err := unmarshaler.UnmarshalSimple(blobRef, &result)
	if err != nil {
		t.Fatalf("UnmarshalSimple failed: %v", err)
	}

	if string(result) != string(blobContent) {
		t.Errorf("Blob content mismatch")
	}
}

func TestUnmarshalSimpleNonPointer(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	unmarshaler := NewUnmarshaler(bm)

	// Should fail with non-pointer target
	var result string
	err := unmarshaler.UnmarshalSimple("test", result)
	if err == nil {
		t.Error("UnmarshalSimple should fail with non-pointer target")
	}
}

func TestUnmarshalSimpleIncompatibleType(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	// Create a blob
	blobContent := []byte("blob")
	ref, _ := bm.Store(blobContent, "test.bin", "")

	unmarshaler := NewUnmarshaler(bm)

	// Try to unmarshal blob to incompatible type (string)
	var result string
	err := unmarshaler.UnmarshalSimple(ref.ToMap(), &result)
	if err == nil {
		t.Error("UnmarshalSimple should fail for incompatible target type")
	}
}

func TestUnmarshalSimpleMissingBlob(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	unmarshaler := NewUnmarshaler(bm)

	// Non-existent blob reference
	blobRef := map[string]interface{}{
		"$blob": true,
		"loc":   "_blobs/nonexistent.bin",
		"hash":  "abc",
		"size":  int64(100),
	}

	var result []byte
	err := unmarshaler.UnmarshalSimple(blobRef, &result)
	if err == nil {
		t.Error("UnmarshalSimple should fail for missing blob")
	}
}

// ========== shouldStoreAsBlob Tests (69.2% coverage) ==========

func TestShouldStoreAsBlobThreshold(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	marshaler := NewMarshaler(bm)

	// Small bytes below threshold
	smallBytes := []byte("small")
	shouldStore, _ := marshaler.shouldStoreAsBlob(smallBytes, MarshalOptions{BlobThreshold: 1024})
	if shouldStore {
		t.Error("Small bytes should not be stored as blob")
	}

	// Large bytes above threshold
	largeBytes := make([]byte, 2048)
	shouldStore, _ = marshaler.shouldStoreAsBlob(largeBytes, MarshalOptions{BlobThreshold: 1024})
	if !shouldStore {
		t.Error("Large bytes should be stored as blob")
	}
}

func TestShouldStoreAsBlobNil(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	marshaler := NewMarshaler(bm)

	// Nil value
	shouldStore, _ := marshaler.shouldStoreAsBlob(nil, MarshalOptions{})
	if shouldStore {
		t.Error("Nil value should not be stored as blob")
	}
}
