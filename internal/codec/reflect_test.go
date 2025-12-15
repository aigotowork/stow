package codec

import (
	"bytes"
	"strings"
	"testing"
)

// ========== IsBlob Tests (0% coverage) ==========

func TestIsBlob(t *testing.T) {
	tests := []struct {
		name       string
		value      interface{}
		tagInfo    TagInfo
		threshold  int64
		forceFile  bool
		expected   bool
	}{
		{
			name:      "nil value",
			value:     nil,
			tagInfo:   TagInfo{},
			threshold: 1024,
			forceFile: false,
			expected:  false,
		},
		{
			name:      "small bytes below threshold",
			value:     []byte("small"),
			tagInfo:   TagInfo{},
			threshold: 1024,
			forceFile: false,
			expected:  false,
		},
		{
			name:      "large bytes above threshold",
			value:     make([]byte, 2048),
			tagInfo:   TagInfo{},
			threshold: 1024,
			forceFile: false,
			expected:  true,
		},
		{
			name:      "io.Reader",
			value:     bytes.NewReader([]byte("test")),
			tagInfo:   TagInfo{},
			threshold: 1024,
			forceFile: false,
			expected:  true,
		},
		{
			name:      "strings.Reader (implements io.Reader)",
			value:     strings.NewReader("test"),
			tagInfo:   TagInfo{},
			threshold: 1024,
			forceFile: false,
			expected:  true,
		},
		{
			name:      "file tag set",
			value:     []byte("small"),
			tagInfo:   TagInfo{IsFile: true},
			threshold: 1024,
			forceFile: false,
			expected:  true,
		},
		{
			name:      "forceFile set",
			value:     []byte("small"),
			tagInfo:   TagInfo{},
			threshold: 1024,
			forceFile: true,
			expected:  true,
		},
		{
			name:      "string value",
			value:     "test string",
			tagInfo:   TagInfo{},
			threshold: 1024,
			forceFile: false,
			expected:  false,
		},
		{
			name:      "int value",
			value:     42,
			tagInfo:   TagInfo{},
			threshold: 1024,
			forceFile: false,
			expected:  false,
		},
		{
			name:      "bytes exactly at threshold",
			value:     make([]byte, 1024),
			tagInfo:   TagInfo{},
			threshold: 1024,
			forceFile: false,
			expected:  false, // > threshold, not >=
		},
		{
			name:      "bytes at threshold + 1",
			value:     make([]byte, 1025),
			tagInfo:   TagInfo{},
			threshold: 1024,
			forceFile: false,
			expected:  true,
		},
		{
			name:      "empty bytes",
			value:     []byte{},
			tagInfo:   TagInfo{},
			threshold: 1024,
			forceFile: false,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBlob(tt.value, tt.tagInfo, tt.threshold, tt.forceFile)
			if result != tt.expected {
				t.Errorf("IsBlob() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// ========== ExtractBlobFields Tests (0% coverage) ==========

func TestExtractBlobFields(t *testing.T) {
	type TestStruct struct {
		Name        string
		SmallData   []byte
		LargeData   []byte
		FileField   []byte `stow:"file"`
		ReaderField []byte // Will be set to Reader in test
	}

	largeData := make([]byte, 2048)

	doc := TestStruct{
		Name:      "Test",
		SmallData: []byte("small"),
		LargeData: largeData,
		FileField: []byte("file data"),
	}

	blobFields, err := ExtractBlobFields(doc, 1024)
	if err != nil {
		t.Fatalf("ExtractBlobFields failed: %v", err)
	}

	// Should extract LargeData (> threshold) and FileField (has file tag)
	if len(blobFields) != 2 {
		t.Fatalf("Expected 2 blob fields, got %d", len(blobFields))
	}

	// Check field names
	fieldNames := make(map[string]bool)
	for _, field := range blobFields {
		fieldNames[field.Name] = true
	}

	if !fieldNames["LargeData"] {
		t.Error("LargeData should be extracted as blob field")
	}

	if !fieldNames["FileField"] {
		t.Error("FileField should be extracted as blob field")
	}

	if fieldNames["SmallData"] {
		t.Error("SmallData should not be extracted as blob field")
	}
}

func TestExtractBlobFieldsWithPointer(t *testing.T) {
	type Doc struct {
		Title   string
		Content []byte
	}

	largeContent := make([]byte, 2048)
	doc := &Doc{
		Title:   "Test",
		Content: largeContent,
	}

	blobFields, err := ExtractBlobFields(doc, 1024)
	if err != nil {
		t.Fatalf("ExtractBlobFields failed with pointer: %v", err)
	}

	if len(blobFields) != 1 {
		t.Fatalf("Expected 1 blob field, got %d", len(blobFields))
	}

	if blobFields[0].Name != "Content" {
		t.Errorf("Expected field name 'Content', got %q", blobFields[0].Name)
	}
}

func TestExtractBlobFieldsNonStruct(t *testing.T) {
	_, err := ExtractBlobFields("not a struct", 1024)
	if err == nil {
		t.Error("ExtractBlobFields should fail for non-struct")
	}

	_, err = ExtractBlobFields(42, 1024)
	if err == nil {
		t.Error("ExtractBlobFields should fail for int")
	}
}

func TestExtractBlobFieldsEmptyStruct(t *testing.T) {
	type Empty struct{}

	blobFields, err := ExtractBlobFields(Empty{}, 1024)
	if err != nil {
		t.Fatalf("ExtractBlobFields failed: %v", err)
	}

	if len(blobFields) != 0 {
		t.Errorf("Expected 0 blob fields for empty struct, got %d", len(blobFields))
	}
}

func TestExtractBlobFieldsWithUnexportedFields(t *testing.T) {
	type DocWithPrivate struct {
		Name        string
		privateData []byte // unexported
		PublicData  []byte
	}

	largeData := make([]byte, 2048)
	doc := DocWithPrivate{
		Name:        "Test",
		privateData: largeData, // Should be skipped
		PublicData:  largeData,
	}

	blobFields, err := ExtractBlobFields(doc, 1024)
	if err != nil {
		t.Fatalf("ExtractBlobFields failed: %v", err)
	}

	// Should only extract PublicData (privateData is unexported)
	if len(blobFields) != 1 {
		t.Fatalf("Expected 1 blob field, got %d", len(blobFields))
	}

	if blobFields[0].Name != "PublicData" {
		t.Errorf("Expected 'PublicData', got %q", blobFields[0].Name)
	}
}

func TestExtractBlobFieldsWithMultipleTags(t *testing.T) {
	type Doc struct {
		File1 []byte `stow:"file,name:doc1.pdf"`
		File2 []byte `stow:"file,mime:image/jpeg"`
		Data  []byte
	}

	smallData := []byte("small")
	doc := Doc{
		File1: smallData,
		File2: smallData,
		Data:  smallData,
	}

	blobFields, err := ExtractBlobFields(doc, 1024)
	if err != nil {
		t.Fatalf("ExtractBlobFields failed: %v", err)
	}

	// Should extract File1 and File2 (both have file tag)
	if len(blobFields) != 2 {
		t.Fatalf("Expected 2 blob fields, got %d", len(blobFields))
	}

	// Check tag info is preserved
	for _, field := range blobFields {
		if !field.TagInfo.IsFile {
			t.Errorf("Field %s should have IsFile=true", field.Name)
		}
	}
}

// ========== ResolveNameField Tests (0% coverage) ==========

func TestResolveNameField(t *testing.T) {
	type Doc struct {
		FileName string
		FileData []byte
	}

	doc := Doc{
		FileName: "avatar.jpg",
		FileData: []byte("data"),
	}

	name, err := ResolveNameField(doc, "FileName")
	if err != nil {
		t.Fatalf("ResolveNameField failed: %v", err)
	}

	if name != "avatar.jpg" {
		t.Errorf("Expected 'avatar.jpg', got %q", name)
	}
}

func TestResolveNameFieldWithPointer(t *testing.T) {
	type Doc struct {
		FileName string
	}

	doc := &Doc{
		FileName: "document.pdf",
	}

	name, err := ResolveNameField(doc, "FileName")
	if err != nil {
		t.Fatalf("ResolveNameField failed with pointer: %v", err)
	}

	if name != "document.pdf" {
		t.Errorf("Expected 'document.pdf', got %q", name)
	}
}

func TestResolveNameFieldNonString(t *testing.T) {
	type Doc struct {
		FileID int
	}

	doc := Doc{
		FileID: 123,
	}

	_, err := ResolveNameField(doc, "FileID")
	if err == nil {
		t.Error("ResolveNameField should fail for non-string field")
	}
}

func TestResolveNameFieldNotFound(t *testing.T) {
	type Doc struct {
		FileName string
	}

	doc := Doc{
		FileName: "test.txt",
	}

	_, err := ResolveNameField(doc, "NonExistentField")
	if err == nil {
		t.Error("ResolveNameField should fail for non-existent field")
	}
}

func TestResolveNameFieldUnexported(t *testing.T) {
	type Doc struct {
		fileName string // unexported
	}

	doc := Doc{
		fileName: "test.txt",
	}

	_, err := ResolveNameField(doc, "fileName")
	if err == nil {
		t.Error("ResolveNameField should fail for unexported field")
	}
}

func TestResolveNameFieldNonStruct(t *testing.T) {
	_, err := ResolveNameField("not a struct", "FileName")
	if err == nil {
		t.Error("ResolveNameField should fail for non-struct")
	}

	_, err = ResolveNameField(42, "FileName")
	if err == nil {
		t.Error("ResolveNameField should fail for int")
	}
}

func TestResolveNameFieldEmptyString(t *testing.T) {
	type Doc struct {
		FileName string
	}

	doc := Doc{
		FileName: "",
	}

	name, err := ResolveNameField(doc, "FileName")
	if err != nil {
		t.Fatalf("ResolveNameField failed: %v", err)
	}

	if name != "" {
		t.Errorf("Expected empty string, got %q", name)
	}
}

// ========== Integration Tests ==========

func TestExtractAndResolve(t *testing.T) {
	type Document struct {
		FileName string
		Title    string
		Content  []byte `stow:"file,name_field:FileName"`
	}

	doc := Document{
		FileName: "mydoc.pdf",
		Title:    "My Document",
		Content:  []byte("document content"),
	}

	// Extract blob fields
	blobFields, err := ExtractBlobFields(doc, 0)
	if err != nil {
		t.Fatalf("ExtractBlobFields failed: %v", err)
	}

	if len(blobFields) != 1 {
		t.Fatalf("Expected 1 blob field, got %d", len(blobFields))
	}

	field := blobFields[0]
	if field.Name != "Content" {
		t.Errorf("Expected field name 'Content', got %q", field.Name)
	}

	// Resolve name field
	if field.TagInfo.NameField != "FileName" {
		t.Errorf("Expected NameField 'FileName', got %q", field.TagInfo.NameField)
	}

	resolvedName, err := ResolveNameField(doc, field.TagInfo.NameField)
	if err != nil {
		t.Fatalf("ResolveNameField failed: %v", err)
	}

	if resolvedName != "mydoc.pdf" {
		t.Errorf("Expected 'mydoc.pdf', got %q", resolvedName)
	}
}
