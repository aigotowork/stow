package codec

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
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

// TestUnmarshalStructWithMapField tests unmarshaling a struct that contains
// a map field with pointer-to-struct values (e.g., map[string]*Track).
// This ensures the fix for "cannot assign map[string]interface{} to map[string]*Track" works correctly.
func TestUnmarshalStructWithMapField(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	unmarshaler := NewUnmarshaler(bm)

	// Define test structs
	type Track struct {
		Title    string
		Duration int
	}

	type Album struct {
		Name   string
		Tracks map[string]*Track
	}

	// Create test data with map[string]interface{} containing nested maps
	data := map[string]interface{}{
		"Name": "Greatest Hits",
		"Tracks": map[string]interface{}{
			"track1": map[string]interface{}{
				"Title":    "Song One",
				"Duration": 180,
			},
			"track2": map[string]interface{}{
				"Title":    "Song Two",
				"Duration": 240,
			},
		},
	}

	// Unmarshal into target struct
	var album Album
	err := unmarshaler.Unmarshal(data, &album)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify album name
	if album.Name != "Greatest Hits" {
		t.Errorf("Album name mismatch: got %q, want %q", album.Name, "Greatest Hits")
	}

	// Verify map was created
	if album.Tracks == nil {
		t.Fatal("Tracks map should not be nil")
	}

	// Verify map length
	if len(album.Tracks) != 2 {
		t.Errorf("Tracks map length mismatch: got %d, want 2", len(album.Tracks))
	}

	// Verify track1
	track1, ok := album.Tracks["track1"]
	if !ok {
		t.Fatal("track1 should exist in map")
	}
	if track1 == nil {
		t.Fatal("track1 should not be nil")
	}
	if track1.Title != "Song One" {
		t.Errorf("track1 title mismatch: got %q, want %q", track1.Title, "Song One")
	}
	if track1.Duration != 180 {
		t.Errorf("track1 duration mismatch: got %d, want 180", track1.Duration)
	}

	// Verify track2
	track2, ok := album.Tracks["track2"]
	if !ok {
		t.Fatal("track2 should exist in map")
	}
	if track2 == nil {
		t.Fatal("track2 should not be nil")
	}
	if track2.Title != "Song Two" {
		t.Errorf("track2 title mismatch: got %q, want %q", track2.Title, "Song Two")
	}
	if track2.Duration != 240 {
		t.Errorf("track2 duration mismatch: got %d, want 240", track2.Duration)
	}
}

// TestUnmarshalStructWithMapOfSimpleTypes tests unmarshaling a map with simple value types
func TestUnmarshalStructWithMapOfSimpleTypes(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	unmarshaler := NewUnmarshaler(bm)

	type Config struct {
		Name     string
		Settings map[string]string
	}

	data := map[string]interface{}{
		"Name": "MyConfig",
		"Settings": map[string]interface{}{
			"host":    "localhost",
			"port":    "8080",
			"timeout": "30s",
		},
	}

	var config Config
	err := unmarshaler.Unmarshal(data, &config)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if config.Name != "MyConfig" {
		t.Errorf("Name mismatch: got %q", config.Name)
	}

	if config.Settings == nil {
		t.Fatal("Settings map should not be nil")
	}

	if len(config.Settings) != 3 {
		t.Errorf("Settings map length mismatch: got %d, want 3", len(config.Settings))
	}

	if config.Settings["host"] != "localhost" {
		t.Errorf("host setting mismatch: got %q", config.Settings["host"])
	}

	if config.Settings["port"] != "8080" {
		t.Errorf("port setting mismatch: got %q", config.Settings["port"])
	}

	if config.Settings["timeout"] != "30s" {
		t.Errorf("timeout setting mismatch: got %q", config.Settings["timeout"])
	}
}

// TestUnmarshalStructWithSliceField tests unmarshaling a struct that contains
// slice fields (e.g., []string, []int).
// This ensures the fix for "cannot assign []interface{} to []string" works correctly.
func TestUnmarshalStructWithSliceField(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	unmarshaler := NewUnmarshaler(bm)

	type Playlist struct {
		Name     string
		TrackIDs []string
		Ratings  []int
	}

	// Create test data with []interface{} containing values
	data := map[string]interface{}{
		"Name":     "My Favorites",
		"TrackIDs": []interface{}{"track1", "track2", "track3"},
		"Ratings":  []interface{}{5, 4, 5},
	}

	// Unmarshal into target struct
	var playlist Playlist
	err := unmarshaler.Unmarshal(data, &playlist)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify playlist name
	if playlist.Name != "My Favorites" {
		t.Errorf("Playlist name mismatch: got %q, want %q", playlist.Name, "My Favorites")
	}

	// Verify TrackIDs slice
	if playlist.TrackIDs == nil {
		t.Fatal("TrackIDs slice should not be nil")
	}

	if len(playlist.TrackIDs) != 3 {
		t.Errorf("TrackIDs length mismatch: got %d, want 3", len(playlist.TrackIDs))
	}

	expectedIDs := []string{"track1", "track2", "track3"}
	for i, expected := range expectedIDs {
		if playlist.TrackIDs[i] != expected {
			t.Errorf("TrackIDs[%d] mismatch: got %q, want %q", i, playlist.TrackIDs[i], expected)
		}
	}

	// Verify Ratings slice
	if playlist.Ratings == nil {
		t.Fatal("Ratings slice should not be nil")
	}

	if len(playlist.Ratings) != 3 {
		t.Errorf("Ratings length mismatch: got %d, want 3", len(playlist.Ratings))
	}

	expectedRatings := []int{5, 4, 5}
	for i, expected := range expectedRatings {
		if playlist.Ratings[i] != expected {
			t.Errorf("Ratings[%d] mismatch: got %d, want %d", i, playlist.Ratings[i], expected)
		}
	}
}

// TestUnmarshalStructWithSliceOfStructs tests unmarshaling a slice of structs
func TestUnmarshalStructWithSliceOfStructs(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	unmarshaler := NewUnmarshaler(bm)

	type Item struct {
		ID   string
		Name string
	}

	type Collection struct {
		Title string
		Items []Item
	}

	// Create test data with []interface{} containing nested maps
	data := map[string]interface{}{
		"Title": "My Collection",
		"Items": []interface{}{
			map[string]interface{}{
				"ID":   "item1",
				"Name": "First Item",
			},
			map[string]interface{}{
				"ID":   "item2",
				"Name": "Second Item",
			},
		},
	}

	// Unmarshal into target struct
	var collection Collection
	err := unmarshaler.Unmarshal(data, &collection)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify collection title
	if collection.Title != "My Collection" {
		t.Errorf("Collection title mismatch: got %q", collection.Title)
	}

	// Verify items slice
	if collection.Items == nil {
		t.Fatal("Items slice should not be nil")
	}

	if len(collection.Items) != 2 {
		t.Errorf("Items length mismatch: got %d, want 2", len(collection.Items))
	}

	// Verify first item
	if collection.Items[0].ID != "item1" {
		t.Errorf("Items[0].ID mismatch: got %q", collection.Items[0].ID)
	}
	if collection.Items[0].Name != "First Item" {
		t.Errorf("Items[0].Name mismatch: got %q", collection.Items[0].Name)
	}

	// Verify second item
	if collection.Items[1].ID != "item2" {
		t.Errorf("Items[1].ID mismatch: got %q", collection.Items[1].ID)
	}
	if collection.Items[1].Name != "Second Item" {
		t.Errorf("Items[1].Name mismatch: got %q", collection.Items[1].Name)
	}
}

// ========== Advanced Type Conversion Tests ==========

// TestUnmarshalNumericTypeConversion tests numeric type conversions
func TestUnmarshalNumericTypeConversion(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	unmarshaler := NewUnmarshaler(bm)

	type Numbers struct {
		Int8Val   int8
		Int16Val  int16
		Int32Val  int32
		Int64Val  int64
		Uint8Val  uint8
		Uint16Val uint16
		Uint32Val uint32
		Uint64Val uint64
		Float32Val float32
		Float64Val float64
	}

	// JSON unmarshaling typically gives us float64 for numbers
	data := map[string]interface{}{
		"Int8Val":    float64(42),
		"Int16Val":   float64(1000),
		"Int32Val":   float64(100000),
		"Int64Val":   float64(1000000),
		"Uint8Val":   float64(200),
		"Uint16Val":  float64(50000),
		"Uint32Val":  float64(3000000),
		"Uint64Val":  float64(9000000),
		"Float32Val": float64(3.14),
		"Float64Val": float64(2.71828),
	}

	var numbers Numbers
	err := unmarshaler.Unmarshal(data, &numbers)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify conversions
	if numbers.Int8Val != 42 {
		t.Errorf("Int8Val mismatch: got %d", numbers.Int8Val)
	}
	if numbers.Int16Val != 1000 {
		t.Errorf("Int16Val mismatch: got %d", numbers.Int16Val)
	}
	if numbers.Float32Val != 3.14 {
		t.Errorf("Float32Val mismatch: got %f", numbers.Float32Val)
	}
}

// TestUnmarshalEmptyCollections tests empty maps and slices
func TestUnmarshalEmptyCollections(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	unmarshaler := NewUnmarshaler(bm)

	type Container struct {
		EmptyMap   map[string]string
		EmptySlice []string
		Name       string
	}

	data := map[string]interface{}{
		"EmptyMap":   map[string]interface{}{},
		"EmptySlice": []interface{}{},
		"Name":       "test",
	}

	var container Container
	err := unmarshaler.Unmarshal(data, &container)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if container.EmptyMap == nil {
		t.Error("EmptyMap should not be nil")
	}
	if len(container.EmptyMap) != 0 {
		t.Errorf("EmptyMap should have length 0, got %d", len(container.EmptyMap))
	}

	if container.EmptySlice == nil {
		t.Error("EmptySlice should not be nil")
	}
	if len(container.EmptySlice) != 0 {
		t.Errorf("EmptySlice should have length 0, got %d", len(container.EmptySlice))
	}
}

// TestUnmarshalNilValues tests nil value handling
func TestUnmarshalNilValues(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	unmarshaler := NewUnmarshaler(bm)

	type Optional struct {
		Name        string
		Description *string
		Count       *int
	}

	data := map[string]interface{}{
		"Name":        "test",
		"Description": nil,
		"Count":       nil,
	}

	var optional Optional
	err := unmarshaler.Unmarshal(data, &optional)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if optional.Name != "test" {
		t.Errorf("Name mismatch: got %q", optional.Name)
	}

	if optional.Description != nil {
		t.Error("Description should be nil")
	}

	if optional.Count != nil {
		t.Error("Count should be nil")
	}
}

// TestUnmarshalNestedMaps tests nested map structures
func TestUnmarshalNestedMaps(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	unmarshaler := NewUnmarshaler(bm)

	type Config struct {
		Settings map[string]map[string]string
	}

	data := map[string]interface{}{
		"Settings": map[string]interface{}{
			"database": map[string]interface{}{
				"host": "localhost",
				"port": "5432",
			},
			"cache": map[string]interface{}{
				"host": "redis",
				"port": "6379",
			},
		},
	}

	var config Config
	err := unmarshaler.Unmarshal(data, &config)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if config.Settings == nil {
		t.Fatal("Settings should not be nil")
	}

	if len(config.Settings) != 2 {
		t.Errorf("Settings length mismatch: got %d, want 2", len(config.Settings))
	}

	if config.Settings["database"]["host"] != "localhost" {
		t.Errorf("database host mismatch: got %q", config.Settings["database"]["host"])
	}

	if config.Settings["cache"]["port"] != "6379" {
		t.Errorf("cache port mismatch: got %q", config.Settings["cache"]["port"])
	}
}

// TestUnmarshalNestedSlices tests nested slice structures
func TestUnmarshalNestedSlices(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	unmarshaler := NewUnmarshaler(bm)

	type Matrix struct {
		Data [][]int
	}

	data := map[string]interface{}{
		"Data": []interface{}{
			[]interface{}{1, 2, 3},
			[]interface{}{4, 5, 6},
			[]interface{}{7, 8, 9},
		},
	}

	var matrix Matrix
	err := unmarshaler.Unmarshal(data, &matrix)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if matrix.Data == nil {
		t.Fatal("Data should not be nil")
	}

	if len(matrix.Data) != 3 {
		t.Errorf("Data length mismatch: got %d, want 3", len(matrix.Data))
	}

	if len(matrix.Data[0]) != 3 {
		t.Errorf("Data[0] length mismatch: got %d, want 3", len(matrix.Data[0]))
	}

	if matrix.Data[1][1] != 5 {
		t.Errorf("Data[1][1] mismatch: got %d, want 5", matrix.Data[1][1])
	}
}

// TestUnmarshalMapWithSliceValues tests map containing slices
func TestUnmarshalMapWithSliceValues(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	unmarshaler := NewUnmarshaler(bm)

	type Groups struct {
		Members map[string][]string
	}

	data := map[string]interface{}{
		"Members": map[string]interface{}{
			"admin":     []interface{}{"alice", "bob"},
			"developer": []interface{}{"charlie", "david", "eve"},
			"guest":     []interface{}{"frank"},
		},
	}

	var groups Groups
	err := unmarshaler.Unmarshal(data, &groups)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if groups.Members == nil {
		t.Fatal("Members should not be nil")
	}

	if len(groups.Members["admin"]) != 2 {
		t.Errorf("admin group length mismatch: got %d, want 2", len(groups.Members["admin"]))
	}

	if groups.Members["admin"][0] != "alice" {
		t.Errorf("admin[0] mismatch: got %q", groups.Members["admin"][0])
	}

	if len(groups.Members["developer"]) != 3 {
		t.Errorf("developer group length mismatch: got %d, want 3", len(groups.Members["developer"]))
	}
}

// TestUnmarshalSliceWithMapElements tests slice containing maps
func TestUnmarshalSliceWithMapElements(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	unmarshaler := NewUnmarshaler(bm)

	type Records struct {
		Items []map[string]string
	}

	data := map[string]interface{}{
		"Items": []interface{}{
			map[string]interface{}{
				"id":   "1",
				"name": "Item 1",
			},
			map[string]interface{}{
				"id":   "2",
				"name": "Item 2",
			},
		},
	}

	var records Records
	err := unmarshaler.Unmarshal(data, &records)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if records.Items == nil {
		t.Fatal("Items should not be nil")
	}

	if len(records.Items) != 2 {
		t.Errorf("Items length mismatch: got %d, want 2", len(records.Items))
	}

	if records.Items[0]["name"] != "Item 1" {
		t.Errorf("Items[0][name] mismatch: got %q", records.Items[0]["name"])
	}
}

// TestUnmarshalComplexNesting tests deeply nested structures
func TestUnmarshalComplexNesting(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	unmarshaler := NewUnmarshaler(bm)

	type Tag struct {
		Name  string
		Value string
	}

	type Resource struct {
		ID   string
		Tags []Tag
	}

	type Environment struct {
		Name      string
		Resources map[string][]Resource
	}

	data := map[string]interface{}{
		"Name": "production",
		"Resources": map[string]interface{}{
			"compute": []interface{}{
				map[string]interface{}{
					"ID": "vm-1",
					"Tags": []interface{}{
						map[string]interface{}{
							"Name":  "environment",
							"Value": "prod",
						},
						map[string]interface{}{
							"Name":  "team",
							"Value": "backend",
						},
					},
				},
			},
		},
	}

	var env Environment
	err := unmarshaler.Unmarshal(data, &env)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if env.Name != "production" {
		t.Errorf("Name mismatch: got %q", env.Name)
	}

	if len(env.Resources["compute"]) != 1 {
		t.Errorf("compute resources length mismatch: got %d, want 1", len(env.Resources["compute"]))
	}

	resource := env.Resources["compute"][0]
	if resource.ID != "vm-1" {
		t.Errorf("Resource ID mismatch: got %q", resource.ID)
	}

	if len(resource.Tags) != 2 {
		t.Errorf("Tags length mismatch: got %d, want 2", len(resource.Tags))
	}

	if resource.Tags[0].Name != "environment" || resource.Tags[0].Value != "prod" {
		t.Errorf("Tag[0] mismatch: got %+v", resource.Tags[0])
	}
}

// TestUnmarshalWithJSONTags tests proper handling of JSON tags
func TestUnmarshalWithJSONTags(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	unmarshaler := NewUnmarshaler(bm)

	type Person struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Age       int    `json:"age"`
		Email     string `json:"email_address"`
	}

	data := map[string]interface{}{
		"first_name":    "John",
		"last_name":     "Doe",
		"age":           float64(30),
		"email_address": "john@example.com",
	}

	var person Person
	err := unmarshaler.Unmarshal(data, &person)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if person.FirstName != "John" {
		t.Errorf("FirstName mismatch: got %q", person.FirstName)
	}

	if person.LastName != "Doe" {
		t.Errorf("LastName mismatch: got %q", person.LastName)
	}

	if person.Email != "john@example.com" {
		t.Errorf("Email mismatch: got %q", person.Email)
	}
}

// TestUnmarshalSliceOfPointers tests slice with pointer elements
func TestUnmarshalSliceOfPointers(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	unmarshaler := NewUnmarshaler(bm)

	type Node struct {
		ID   string
		Name string
	}

	type Graph struct {
		Nodes []*Node
	}

	data := map[string]interface{}{
		"Nodes": []interface{}{
			map[string]interface{}{
				"ID":   "node1",
				"Name": "First Node",
			},
			map[string]interface{}{
				"ID":   "node2",
				"Name": "Second Node",
			},
		},
	}

	var graph Graph
	err := unmarshaler.Unmarshal(data, &graph)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(graph.Nodes) != 2 {
		t.Errorf("Nodes length mismatch: got %d, want 2", len(graph.Nodes))
	}

	if graph.Nodes[0] == nil {
		t.Fatal("Nodes[0] should not be nil")
	}

	if graph.Nodes[0].ID != "node1" {
		t.Errorf("Nodes[0].ID mismatch: got %q", graph.Nodes[0].ID)
	}
}

// TestUnmarshalMissingFields tests handling of missing fields in data
func TestUnmarshalMissingFields(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	unmarshaler := NewUnmarshaler(bm)

	type Complete struct {
		Required string
		Optional string
		Missing  string
		Count    int
	}

	// Data is missing "Missing" and "Count" fields
	data := map[string]interface{}{
		"Required": "present",
		"Optional": "also present",
	}

	var complete Complete
	err := unmarshaler.Unmarshal(data, &complete)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if complete.Required != "present" {
		t.Errorf("Required mismatch: got %q", complete.Required)
	}

	// Missing fields should remain at zero value
	if complete.Missing != "" {
		t.Errorf("Missing should be empty string, got %q", complete.Missing)
	}

	if complete.Count != 0 {
		t.Errorf("Count should be 0, got %d", complete.Count)
	}
}

// ========== Additional Coverage Tests ==========

// TestUnmarshalToMapTarget tests unmarshaling directly into a map target
func TestUnmarshalToMapTarget(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	unmarshaler := NewUnmarshaler(bm)

	data := map[string]interface{}{
		"key1": "value1",
		"key2": float64(42),
		"key3": true,
	}

	var result map[string]interface{}
	err := unmarshaler.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if result["key1"] != "value1" {
		t.Errorf("key1 mismatch: got %v", result["key1"])
	}

	if result["key2"] != float64(42) {
		t.Errorf("key2 mismatch: got %v", result["key2"])
	}

	if result["key3"] != true {
		t.Errorf("key3 mismatch: got %v", result["key3"])
	}
}

// TestUnmarshalSimple tests UnmarshalSimple method
func TestUnmarshalSimple(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	unmarshaler := NewUnmarshaler(bm)

	// Test simple string
	var str string
	err := unmarshaler.UnmarshalSimple("test string", &str)
	if err != nil {
		t.Fatalf("UnmarshalSimple failed: %v", err)
	}

	if str != "test string" {
		t.Errorf("String mismatch: got %q", str)
	}

	// Test simple number
	var num int
	err = unmarshaler.UnmarshalSimple(42, &num)
	if err != nil {
		t.Fatalf("UnmarshalSimple failed: %v", err)
	}

	if num != 42 {
		t.Errorf("Number mismatch: got %d", num)
	}
}

// TestUnmarshalWithLogger tests warning logging
func TestUnmarshalWithLogger(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	unmarshaler := NewUnmarshaler(bm)

	// Mock logger that implements the Logger interface
	var mockLogger Logger = &MockLogger{warnings: make([]string, 0)}
	unmarshaler.SetLogger(mockLogger)

	type Doc struct {
		Content []byte
	}

	// Create data with non-existent blob reference
	data := map[string]interface{}{
		"Content": map[string]interface{}{
			"$blob": true,
			"loc":   "_blobs/nonexistent.bin",
			"hash":  "abc123",
			"size":  int64(100),
		},
	}

	var doc Doc
	err := unmarshaler.Unmarshal(data, &doc)

	// Should not error, but should log warning
	if err != nil {
		t.Fatalf("Unmarshal should not error: %v", err)
	}

	// Content should be zero value
	if doc.Content != nil {
		t.Error("Content should be nil due to missing blob")
	}
}

// MockLogger implements Logger interface for testing
type MockLogger struct {
	warnings []string
}

func (m *MockLogger) Warn(msg string, fields ...interface{}) {
	m.warnings = append(m.warnings, msg)
}

// TestScalarWrapping tests scalar value wrapping/unwrapping
func TestScalarWrapping(t *testing.T) {
	tmpDir := t.TempDir()
	blobDir := filepath.Join(tmpDir, "_blobs")
	bm, _ := blob.NewManager(blobDir, 1024*1024, 1024)

	unmarshaler := NewUnmarshaler(bm)

	// Test wrapped scalar
	data := map[string]interface{}{
		"$value": "scalar string",
	}

	var result string
	err := unmarshaler.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if result != "scalar string" {
		t.Errorf("Result mismatch: got %q", result)
	}

	// Test wrapped nil
	nilData := map[string]interface{}{
		"$value": nil,
	}

	var nilResult *string
	err = unmarshaler.Unmarshal(nilData, &nilResult)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if nilResult != nil {
		t.Error("Result should be nil")
	}
}

// TestFromMapWithNestedStructs tests FromMap with nested structures
func TestFromMapWithNestedStructs(t *testing.T) {
	type Address struct {
		Street string
		City   string
	}

	type Person struct {
		Name    string
		Address Address
	}

	data := map[string]interface{}{
		"Name": "Alice",
		"Address": map[string]interface{}{
			"Street": "123 Main St",
			"City":   "Springfield",
		},
	}

	var person Person
	err := FromMap(data, &person)
	if err != nil {
		t.Fatalf("FromMap failed: %v", err)
	}

	if person.Name != "Alice" {
		t.Errorf("Name mismatch: got %q", person.Name)
	}

	if person.Address.Street != "123 Main St" {
		t.Errorf("Street mismatch: got %q", person.Address.Street)
	}

	if person.Address.City != "Springfield" {
		t.Errorf("City mismatch: got %q", person.Address.City)
	}
}

// TestFromMapWithPointerToStruct tests FromMap with pointer to nested struct
func TestFromMapWithPointerToStruct(t *testing.T) {
	type Metadata struct {
		Version int
		Author  string
	}

	type Document struct {
		Title    string
		Metadata *Metadata
	}

	data := map[string]interface{}{
		"Title": "Test Doc",
		"Metadata": map[string]interface{}{
			"Version": float64(1),
			"Author":  "Bob",
		},
	}

	var doc Document
	err := FromMap(data, &doc)
	if err != nil {
		t.Fatalf("FromMap failed: %v", err)
	}

	if doc.Title != "Test Doc" {
		t.Errorf("Title mismatch: got %q", doc.Title)
	}

	if doc.Metadata == nil {
		t.Fatal("Metadata should not be nil")
	}

	if doc.Metadata.Version != 1 {
		t.Errorf("Version mismatch: got %d", doc.Metadata.Version)
	}

	if doc.Metadata.Author != "Bob" {
		t.Errorf("Author mismatch: got %q", doc.Metadata.Author)
	}
}

// TestIsSimpleType tests the IsSimpleType helper function
func TestIsSimpleType(t *testing.T) {
	// Test simple types
	if !IsSimpleType(reflect.TypeOf(int(0))) {
		t.Error("int should be simple type")
	}

	if !IsSimpleType(reflect.TypeOf(string(""))) {
		t.Error("string should be simple type")
	}

	if !IsSimpleType(reflect.TypeOf(float64(0))) {
		t.Error("float64 should be simple type")
	}

	if !IsSimpleType(reflect.TypeOf(bool(false))) {
		t.Error("bool should be simple type")
	}

	// Test complex types
	if IsSimpleType(reflect.TypeOf(struct{}{})) {
		t.Error("struct should not be simple type")
	}

	if IsSimpleType(reflect.TypeOf([]int{})) {
		t.Error("slice should not be simple type")
	}

	if IsSimpleType(reflect.TypeOf(map[string]int{})) {
		t.Error("map should not be simple type")
	}
}
