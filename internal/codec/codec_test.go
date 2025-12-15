package codec

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/aigotowork/stow/internal/blob"
)

// ========== Struct Tag Tests ==========

func TestParseStowTag(t *testing.T) {
	tests := []struct {
		tag      string
		expected TagInfo
	}{
		{
			tag: "",
			expected: TagInfo{
				IsFile:    false,
				Name:      "",
				NameField: "",
				MimeType:  "",
			},
		},
		{
			tag: "file",
			expected: TagInfo{
				IsFile:    true,
				Name:      "",
				NameField: "",
				MimeType:  "",
			},
		},
		{
			tag: "file,name:avatar.jpg",
			expected: TagInfo{
				IsFile:    true,
				Name:      "avatar.jpg",
				NameField: "",
				MimeType:  "",
			},
		},
		{
			tag: "file,name_field:FileName",
			expected: TagInfo{
				IsFile:    true,
				Name:      "",
				NameField: "FileName",
				MimeType:  "",
			},
		},
		{
			tag: "file,mime:image/jpeg",
			expected: TagInfo{
				IsFile:    true,
				Name:      "",
				NameField: "",
				MimeType:  "image/jpeg",
			},
		},
		{
			tag: "file,name:avatar.jpg,mime:image/jpeg",
			expected: TagInfo{
				IsFile:    true,
				Name:      "avatar.jpg",
				NameField: "",
				MimeType:  "image/jpeg",
			},
		},
		{
			tag: "file,name_field:FileName,mime:image/jpeg",
			expected: TagInfo{
				IsFile:    true,
				Name:      "",
				NameField: "FileName",
				MimeType:  "image/jpeg",
			},
		},
	}

	for _, tt := range tests {
		result := ParseStowTag(tt.tag)
		if result.IsFile != tt.expected.IsFile {
			t.Errorf("ParseStowTag(%q).IsFile = %v, want %v", tt.tag, result.IsFile, tt.expected.IsFile)
		}
		if result.Name != tt.expected.Name {
			t.Errorf("ParseStowTag(%q).Name = %q, want %q", tt.tag, result.Name, tt.expected.Name)
		}
		if result.NameField != tt.expected.NameField {
			t.Errorf("ParseStowTag(%q).NameField = %q, want %q", tt.tag, result.NameField, tt.expected.NameField)
		}
		if result.MimeType != tt.expected.MimeType {
			t.Errorf("ParseStowTag(%q).MimeType = %q, want %q", tt.tag, result.MimeType, tt.expected.MimeType)
		}
	}
}

// ========== Marshal/Unmarshal Tests ==========

func TestMarshalSimpleStruct(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, err := blob.NewManager(blobDir, 1024*1024, 1024)
	if err != nil {
		t.Fatalf("Failed to create blob manager: %v", err)
	}

	marshaler := NewMarshaler(bm)

	type Config struct {
		Host string
		Port int
	}

	config := Config{
		Host: "localhost",
		Port: 8080,
	}

	data, blobRefs, err := marshaler.Marshal(config, MarshalOptions{
		BlobThreshold: 1024,
	})

	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	if len(blobRefs) != 0 {
		t.Error("Simple struct should not create blob references")
	}

	if data["Host"] != "localhost" {
		t.Errorf("Host mismatch: got %v", data["Host"])
	}

	if data["Port"] != 8080 {
		t.Errorf("Port mismatch: got %v", data["Port"])
	}
}

func TestMarshalWithBytes(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	marshaler := NewMarshaler(bm)

	type Document struct {
		Title   string
		Content []byte
	}

	largeContent := make([]byte, 2048)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	doc := Document{
		Title:   "Test Doc",
		Content: largeContent,
	}

	data, blobRefs, err := marshaler.Marshal(doc, MarshalOptions{
		BlobThreshold: 1024,
	})

	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	if len(blobRefs) != 1 {
		t.Errorf("Should create 1 blob reference, got %d", len(blobRefs))
	}

	// Content should be replaced with blob reference
	contentField, ok := data["Content"].(map[string]interface{})
	if !ok {
		t.Fatal("Content should be a blob reference map")
	}

	if contentField["$blob"] != true {
		t.Error("Content should be marked as blob")
	}
}

func TestMarshalWithSmallBytes(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	marshaler := NewMarshaler(bm)

	type Document struct {
		Title   string
		Content []byte
	}

	smallContent := []byte("Small content")

	doc := Document{
		Title:   "Test Doc",
		Content: smallContent,
	}

	data, blobRefs, err := marshaler.Marshal(doc, MarshalOptions{
		BlobThreshold: 1024,
	})

	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	if len(blobRefs) != 0 {
		t.Error("Small content should not create blob")
	}

	// Content should be inline
	_, ok := data["Content"].([]byte)
	if !ok {
		t.Error("Small content should remain as []byte")
	}
}

func TestMarshalWithForceFile(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	marshaler := NewMarshaler(bm)

	type Document struct {
		Content []byte
	}

	smallContent := []byte("Small content")

	doc := Document{
		Content: smallContent,
	}

	data, blobRefs, err := marshaler.Marshal(doc, MarshalOptions{
		BlobThreshold: 1024,
		ForceFile:     true,
		FileName:      "test.txt",
	})

	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	if len(blobRefs) != 1 {
		t.Error("ForceFile should create blob even for small content")
	}

	// Content should be blob reference
	contentField, ok := data["Content"].(map[string]interface{})
	if !ok {
		t.Fatal("Content should be a blob reference map")
	}

	if contentField["$blob"] != true {
		t.Error("Content should be marked as blob")
	}
}

func TestUnmarshalSimpleStruct(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	unmarshaler := NewUnmarshaler(bm)

	type Config struct {
		Host string
		Port int
	}

	data := map[string]interface{}{
		"Host": "localhost",
		"Port": float64(8080),
	}

	var config Config
	err := unmarshaler.Unmarshal(data, &config)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if config.Host != "localhost" {
		t.Errorf("Host mismatch: got %q", config.Host)
	}

	if config.Port != 8080 {
		t.Errorf("Port mismatch: got %d", config.Port)
	}
}

func TestMarshalUnmarshalRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	marshaler := NewMarshaler(bm)
	unmarshaler := NewUnmarshaler(bm)

	type User struct {
		Name  string
		Email string
		Age   int
		Bio   []byte
	}

	originalUser := User{
		Name:  "Alice",
		Email: "alice@example.com",
		Age:   30,
		Bio:   make([]byte, 2048),
	}

	// Fill bio with data
	for i := range originalUser.Bio {
		originalUser.Bio[i] = byte(i % 256)
	}

	// Marshal
	data, blobRefs, err := marshaler.Marshal(originalUser, MarshalOptions{
		BlobThreshold: 1024,
	})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	if len(blobRefs) != 1 {
		t.Errorf("Should create 1 blob reference, got %d", len(blobRefs))
	}

	// Unmarshal
	var restoredUser User
	err = unmarshaler.Unmarshal(data, &restoredUser)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify
	if restoredUser.Name != originalUser.Name {
		t.Errorf("Name mismatch: got %q", restoredUser.Name)
	}

	if restoredUser.Email != originalUser.Email {
		t.Errorf("Email mismatch: got %q", restoredUser.Email)
	}

	if restoredUser.Age != originalUser.Age {
		t.Errorf("Age mismatch: got %d", restoredUser.Age)
	}

	if !bytes.Equal(restoredUser.Bio, originalUser.Bio) {
		t.Error("Bio content mismatch")
	}
}

func TestMarshalBytes(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	marshaler := NewMarshaler(bm)

	largeData := make([]byte, 2048)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	result, ref, err := marshaler.MarshalBytes(largeData, MarshalOptions{
		BlobThreshold: 1024,
		FileName:      "test.bin",
		MimeType:      "application/octet-stream",
	})

	if err != nil {
		t.Fatalf("MarshalBytes failed: %v", err)
	}

	if ref == nil {
		t.Fatal("Should create blob reference")
	}

	// Result should be blob reference map
	refMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result should be a map")
	}

	if refMap["$blob"] != true {
		t.Error("Should be marked as blob")
	}
}

func TestMarshalBytesInline(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	marshaler := NewMarshaler(bm)

	smallData := []byte("Small data")

	result, ref, err := marshaler.MarshalBytes(smallData, MarshalOptions{
		BlobThreshold: 1024,
	})

	if err != nil {
		t.Fatalf("MarshalBytes failed: %v", err)
	}

	if ref != nil {
		t.Error("Should not create blob reference for small data")
	}

	// Result should be the data itself
	resultBytes, ok := result.([]byte)
	if !ok {
		t.Fatal("Result should be []byte")
	}

	if !bytes.Equal(resultBytes, smallData) {
		t.Error("Data mismatch")
	}
}

func TestMarshalReader(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	marshaler := NewMarshaler(bm)

	data := []byte("Test reader data")
	reader := bytes.NewReader(data)

	result, ref, err := marshaler.MarshalReader(reader, MarshalOptions{
		FileName: "test.txt",
		MimeType: "text/plain",
	})

	if err != nil {
		t.Fatalf("MarshalReader failed: %v", err)
	}

	if ref == nil {
		t.Fatal("Should create blob reference")
	}

	// Result should be blob reference map
	refMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result should be a map")
	}

	if refMap["$blob"] != true {
		t.Error("Should be marked as blob")
	}

	if refMap["name"] != "test.txt" {
		t.Errorf("Name mismatch: got %v", refMap["name"])
	}
}

func TestToMapWithNilValue(t *testing.T) {
	result, err := ToMap(nil)
	if err != nil {
		t.Fatalf("ToMap(nil) should not error: %v", err)
	}

	// Nil should be wrapped in $value
	if v, ok := result["$value"]; !ok || v != nil {
		t.Error("ToMap(nil) should return {\"$value\": nil}")
	}
}

func TestToMapWithMap(t *testing.T) {
	input := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}

	result, err := ToMap(input)
	if err != nil {
		t.Fatalf("ToMap failed: %v", err)
	}

	if result["key1"] != "value1" {
		t.Error("key1 mismatch")
	}

	if result["key2"] != 42 {
		t.Error("key2 mismatch")
	}
}

func TestToMapWithStruct(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}

	person := Person{
		Name: "Bob",
		Age:  25,
	}

	result, err := ToMap(person)
	if err != nil {
		t.Fatalf("ToMap failed: %v", err)
	}

	if result["Name"] != "Bob" {
		t.Errorf("Name mismatch: got %v", result["Name"])
	}

	if result["Age"] != 25 {
		t.Errorf("Age mismatch: got %v", result["Age"])
	}
}

func TestMarshalSimpleValue(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	marshaler := NewMarshaler(bm)

	// Test string
	result, blobRefs, err := marshaler.MarshalSimple("test string", MarshalOptions{})
	if err != nil {
		t.Fatalf("MarshalSimple failed: %v", err)
	}

	if len(blobRefs) != 0 {
		t.Error("String should not create blob")
	}

	if result != "test string" {
		t.Errorf("Value mismatch: got %v", result)
	}

	// Test number
	result, blobRefs, err = marshaler.MarshalSimple(42, MarshalOptions{})
	if err != nil {
		t.Fatalf("MarshalSimple failed: %v", err)
	}

	if result != 42 {
		t.Errorf("Value mismatch: got %v", result)
	}

	// Test map
	input := map[string]interface{}{"key": "value"}
	result, blobRefs, err = marshaler.MarshalSimple(input, MarshalOptions{})
	if err != nil {
		t.Fatalf("MarshalSimple failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result should be a map")
	}

	if resultMap["key"] != "value" {
		t.Error("Map value mismatch")
	}
}

func TestUnmarshalWithMissingBlob(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	unmarshaler := NewUnmarshaler(bm)

	type Document struct {
		Title   string
		Content []byte
	}

	// Create data with blob reference that doesn't exist
	data := map[string]interface{}{
		"Title": "Test",
		"Content": map[string]interface{}{
			"$blob": true,
			"loc":   "_blobs/nonexistent.bin",
			"hash":  "abc123",
			"size":  int64(100),
		},
	}

	var doc Document
	err := unmarshaler.Unmarshal(data, &doc)

	// Should not return error, but field should be zero value
	if err != nil {
		t.Fatalf("Unmarshal should handle missing blob gracefully: %v", err)
	}

	if doc.Title != "Test" {
		t.Error("Title should be unmarshaled correctly")
	}

	// Content should be zero value (nil or empty)
	if doc.Content != nil {
		t.Error("Missing blob should result in zero value")
	}
}

func TestStoreBytesAsBlob(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	marshaler := NewMarshaler(bm)

	data := []byte("Test blob storage")
	ref, err := marshaler.StoreBytesAsBlob(data, "test.txt", "text/plain")

	if err != nil {
		t.Fatalf("StoreBytesAsBlob failed: %v", err)
	}

	if ref == nil {
		t.Fatal("Should return blob reference")
	}

	if !ref.IsValid() {
		t.Error("Reference should be valid")
	}

	// Verify blob was actually stored
	blobPath := filepath.Join(blobDir, filepath.Base(ref.Location))
	if _, err := os.Stat(blobPath); os.IsNotExist(err) {
		t.Error("Blob file was not created")
	}

	// Verify content
	stored, err := os.ReadFile(blobPath)
	if err != nil {
		t.Fatalf("Failed to read blob: %v", err)
	}

	if !bytes.Equal(stored, data) {
		t.Error("Stored content mismatch")
	}
}
