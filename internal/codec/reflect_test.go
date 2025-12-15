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

// ========== ToMap/FromMap Tests ==========

func TestToMapFromMapScalar(t *testing.T) {
	// Test scalar types wrapped in $value
	tests := []struct {
		name  string
		value interface{}
	}{
		{"int", 42},
		{"string", "hello"},
		{"bool", true},
		{"float", 3.14},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ToMap should wrap scalar in $value
			m, err := ToMap(tt.value)
			if err != nil {
				t.Fatalf("ToMap failed: %v", err)
			}

			if len(m) != 1 {
				t.Fatalf("Expected map with 1 key, got %d", len(m))
			}

			if _, ok := m["$value"]; !ok {
				t.Fatal("Expected $value key in map")
			}

			// FromMap should unwrap scalar from $value
			var result interface{}
			err = FromMap(m, &result)
			if err != nil {
				t.Fatalf("FromMap failed: %v", err)
			}

			if result != tt.value {
				t.Errorf("Expected %v, got %v", tt.value, result)
			}
		})
	}
}

func TestFromMapWithSlice(t *testing.T) {
	// Test converting []interface{} to []T
	type TestStruct struct {
		Numbers []int
		Strings []string
	}

	data := map[string]interface{}{
		"Numbers": []interface{}{1, 2, 3, 4, 5},
		"Strings": []interface{}{"a", "b", "c"},
	}

	var result TestStruct
	err := FromMap(data, &result)
	if err != nil {
		t.Fatalf("FromMap failed: %v", err)
	}

	expectedNumbers := []int{1, 2, 3, 4, 5}
	if len(result.Numbers) != len(expectedNumbers) {
		t.Fatalf("Numbers length mismatch: expected %d, got %d", len(expectedNumbers), len(result.Numbers))
	}
	for i, n := range expectedNumbers {
		if result.Numbers[i] != n {
			t.Errorf("Numbers[%d]: expected %d, got %d", i, n, result.Numbers[i])
		}
	}

	expectedStrings := []string{"a", "b", "c"}
	if len(result.Strings) != len(expectedStrings) {
		t.Fatalf("Strings length mismatch: expected %d, got %d", len(expectedStrings), len(result.Strings))
	}
	for i, s := range expectedStrings {
		if result.Strings[i] != s {
			t.Errorf("Strings[%d]: expected %q, got %q", i, s, result.Strings[i])
		}
	}
}

func TestFromMapWithSliceOfStructs(t *testing.T) {
	// This is the actual bug case: []interface{} containing maps -> []Group
	type Group struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	type Schedule struct {
		Groups []Group `json:"groups"`
	}

	data := map[string]interface{}{
		"groups": []interface{}{
			map[string]interface{}{"id": 1, "name": "Group 1"},
			map[string]interface{}{"id": 2, "name": "Group 2"},
			map[string]interface{}{"id": 3, "name": "Group 3"},
		},
	}

	var result Schedule
	err := FromMap(data, &result)
	if err != nil {
		t.Fatalf("FromMap failed: %v", err)
	}

	expectedGroups := []Group{
		{ID: 1, Name: "Group 1"},
		{ID: 2, Name: "Group 2"},
		{ID: 3, Name: "Group 3"},
	}

	if len(result.Groups) != len(expectedGroups) {
		t.Fatalf("Groups length mismatch: expected %d, got %d", len(expectedGroups), len(result.Groups))
	}

	for i, expected := range expectedGroups {
		if result.Groups[i].ID != expected.ID {
			t.Errorf("Groups[%d].ID: expected %d, got %d", i, expected.ID, result.Groups[i].ID)
		}
		if result.Groups[i].Name != expected.Name {
			t.Errorf("Groups[%d].Name: expected %q, got %q", i, expected.Name, result.Groups[i].Name)
		}
	}
}

func TestFromMapWithSliceWrappedInValue(t *testing.T) {
	// Test $value wrapped slice (the original bug scenario)
	type Group struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	data := map[string]interface{}{
		"$value": []interface{}{
			map[string]interface{}{"id": 1, "name": "Group 1"},
			map[string]interface{}{"id": 2, "name": "Group 2"},
		},
	}

	var result []Group
	err := FromMap(data, &result)
	if err != nil {
		t.Fatalf("FromMap failed: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("Expected 2 groups, got %d", len(result))
	}

	if result[0].ID != 1 || result[0].Name != "Group 1" {
		t.Errorf("Group 0: expected {1, Group 1}, got {%d, %s}", result[0].ID, result[0].Name)
	}
	if result[1].ID != 2 || result[1].Name != "Group 2" {
		t.Errorf("Group 1: expected {2, Group 2}, got {%d, %s}", result[1].ID, result[1].Name)
	}
}

func TestFromMapWithArray(t *testing.T) {
	// Test converting []interface{} to [N]T
	type TestStruct struct {
		FixedNumbers [5]int
	}

	data := map[string]interface{}{
		"FixedNumbers": []interface{}{1, 2, 3, 4, 5},
	}

	var result TestStruct
	err := FromMap(data, &result)
	if err != nil {
		t.Fatalf("FromMap failed: %v", err)
	}

	expectedNumbers := [5]int{1, 2, 3, 4, 5}
	for i, n := range expectedNumbers {
		if result.FixedNumbers[i] != n {
			t.Errorf("FixedNumbers[%d]: expected %d, got %d", i, n, result.FixedNumbers[i])
		}
	}
}

func TestFromMapWithArrayLengthMismatch(t *testing.T) {
	// Test that array length mismatch returns error
	type TestStruct struct {
		FixedNumbers [5]int
	}

	data := map[string]interface{}{
		"FixedNumbers": []interface{}{1, 2, 3}, // Only 3 elements, need 5
	}

	var result TestStruct
	err := FromMap(data, &result)
	if err == nil {
		t.Fatal("Expected error for array length mismatch, got nil")
	}
}

func TestFromMapWithPointer(t *testing.T) {
	// Test converting to pointer types
	type TestStruct struct {
		IntPtr    *int
		StringPtr *string
		BoolPtr   *bool
	}

	intVal := 42
	strVal := "hello"
	boolVal := true

	data := map[string]interface{}{
		"IntPtr":    intVal,
		"StringPtr": strVal,
		"BoolPtr":   boolVal,
	}

	var result TestStruct
	err := FromMap(data, &result)
	if err != nil {
		t.Fatalf("FromMap failed: %v", err)
	}

	if result.IntPtr == nil {
		t.Fatal("IntPtr is nil")
	}
	if *result.IntPtr != intVal {
		t.Errorf("IntPtr: expected %d, got %d", intVal, *result.IntPtr)
	}

	if result.StringPtr == nil {
		t.Fatal("StringPtr is nil")
	}
	if *result.StringPtr != strVal {
		t.Errorf("StringPtr: expected %q, got %q", strVal, *result.StringPtr)
	}

	if result.BoolPtr == nil {
		t.Fatal("BoolPtr is nil")
	}
	if *result.BoolPtr != boolVal {
		t.Errorf("BoolPtr: expected %v, got %v", boolVal, *result.BoolPtr)
	}
}

func TestFromMapWithNestedSlices(t *testing.T) {
	// Test nested slices: [][]int
	type TestStruct struct {
		Matrix [][]int
	}

	data := map[string]interface{}{
		"Matrix": []interface{}{
			[]interface{}{1, 2, 3},
			[]interface{}{4, 5, 6},
			[]interface{}{7, 8, 9},
		},
	}

	var result TestStruct
	err := FromMap(data, &result)
	if err != nil {
		t.Fatalf("FromMap failed: %v", err)
	}

	expected := [][]int{
		{1, 2, 3},
		{4, 5, 6},
		{7, 8, 9},
	}

	if len(result.Matrix) != len(expected) {
		t.Fatalf("Matrix length mismatch: expected %d, got %d", len(expected), len(result.Matrix))
	}

	for i, row := range expected {
		if len(result.Matrix[i]) != len(row) {
			t.Fatalf("Matrix[%d] length mismatch: expected %d, got %d", i, len(row), len(result.Matrix[i]))
		}
		for j, val := range row {
			if result.Matrix[i][j] != val {
				t.Errorf("Matrix[%d][%d]: expected %d, got %d", i, j, val, result.Matrix[i][j])
			}
		}
	}
}

func TestFromMapWithMapContainingSlice(t *testing.T) {
	// Test map containing slices
	type TestStruct struct {
		Data map[string][]int
	}

	data := map[string]interface{}{
		"Data": map[string]interface{}{
			"a": []interface{}{1, 2, 3},
			"b": []interface{}{4, 5, 6},
			"c": []interface{}{7, 8, 9},
		},
	}

	var result TestStruct
	err := FromMap(data, &result)
	if err != nil {
		t.Fatalf("FromMap failed: %v", err)
	}

	expected := map[string][]int{
		"a": {1, 2, 3},
		"b": {4, 5, 6},
		"c": {7, 8, 9},
	}

	if len(result.Data) != len(expected) {
		t.Fatalf("Data length mismatch: expected %d, got %d", len(expected), len(result.Data))
	}

	for key, expectedSlice := range expected {
		resultSlice, ok := result.Data[key]
		if !ok {
			t.Errorf("Key %q not found in result", key)
			continue
		}

		if len(resultSlice) != len(expectedSlice) {
			t.Errorf("Data[%q] length mismatch: expected %d, got %d", key, len(expectedSlice), len(resultSlice))
			continue
		}

		for i, val := range expectedSlice {
			if resultSlice[i] != val {
				t.Errorf("Data[%q][%d]: expected %d, got %d", key, i, val, resultSlice[i])
			}
		}
	}
}

func TestFromMapWithNilSlice(t *testing.T) {
	// Test that nil values create zero values
	type TestStruct struct {
		Numbers []int
	}

	data := map[string]interface{}{
		"Numbers": nil,
	}

	var result TestStruct
	err := FromMap(data, &result)
	if err != nil {
		t.Fatalf("FromMap failed: %v", err)
	}

	if result.Numbers != nil {
		t.Errorf("Expected nil slice, got %v", result.Numbers)
	}
}

func TestFromMapWithEmptySlice(t *testing.T) {
	// Test empty slice
	type TestStruct struct {
		Numbers []int
	}

	data := map[string]interface{}{
		"Numbers": []interface{}{},
	}

	var result TestStruct
	err := FromMap(data, &result)
	if err != nil {
		t.Fatalf("FromMap failed: %v", err)
	}

	if len(result.Numbers) != 0 {
		t.Errorf("Expected empty slice, got %d elements", len(result.Numbers))
	}
}

func TestToMapFromMapRoundTrip(t *testing.T) {
	// Test complete round trip with complex nested structure
	type Inner struct {
		Value int `json:"value"`
	}

	type Outer struct {
		Name   string   `json:"name"`
		Items  []Inner  `json:"items"`
		Tags   []string `json:"tags"`
		Counts []int    `json:"counts"`
	}

	original := Outer{
		Name: "Test",
		Items: []Inner{
			{Value: 1},
			{Value: 2},
			{Value: 3},
		},
		Tags:   []string{"tag1", "tag2", "tag3"},
		Counts: []int{10, 20, 30},
	}

	// Convert to map
	m, err := ToMap(original)
	if err != nil {
		t.Fatalf("ToMap failed: %v", err)
	}

	// Convert back to struct
	var result Outer
	err = FromMap(m, &result)
	if err != nil {
		t.Fatalf("FromMap failed: %v", err)
	}

	// Verify
	if result.Name != original.Name {
		t.Errorf("Name mismatch: expected %q, got %q", original.Name, result.Name)
	}

	if len(result.Items) != len(original.Items) {
		t.Fatalf("Items length mismatch: expected %d, got %d", len(original.Items), len(result.Items))
	}
	for i, item := range original.Items {
		if result.Items[i].Value != item.Value {
			t.Errorf("Items[%d].Value: expected %d, got %d", i, item.Value, result.Items[i].Value)
		}
	}

	if len(result.Tags) != len(original.Tags) {
		t.Fatalf("Tags length mismatch: expected %d, got %d", len(original.Tags), len(result.Tags))
	}
	for i, tag := range original.Tags {
		if result.Tags[i] != tag {
			t.Errorf("Tags[%d]: expected %q, got %q", i, tag, result.Tags[i])
		}
	}

	if len(result.Counts) != len(original.Counts) {
		t.Fatalf("Counts length mismatch: expected %d, got %d", len(original.Counts), len(result.Counts))
	}
	for i, count := range original.Counts {
		if result.Counts[i] != count {
			t.Errorf("Counts[%d]: expected %d, got %d", i, count, result.Counts[i])
		}
	}
}

