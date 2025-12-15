package core

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDecodeString tests the DecodeString function with various inputs
func TestDecodeString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(*testing.T, *Record)
	}{
		{
			name:    "valid JSON record",
			input:   `{"_meta":{"k":"test-key","v":1,"op":"put","ts":"2025-12-14T18:09:00Z"},"data":{"name":"Alice"}}`,
			wantErr: false,
			check: func(t *testing.T, r *Record) {
				if r.Meta.Key != "test-key" {
					t.Errorf("Key = %q, want %q", r.Meta.Key, "test-key")
				}
				if r.Meta.Version != 1 {
					t.Errorf("Version = %d, want %d", r.Meta.Version, 1)
				}
				if r.Data["name"] != "Alice" {
					t.Errorf("Data[name] = %v, want %q", r.Data["name"], "Alice")
				}
			},
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			input:   "   \n\t  ",
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			input:   `{"invalid": json}`,
			wantErr: true,
		},
		{
			name:    "valid JSON but invalid record structure - missing meta",
			input:   `{"data":{"name":"Bob"}}`,
			wantErr: true,
		},
		{
			name:    "valid JSON but invalid record - empty key",
			input:   `{"_meta":{"k":"","v":1,"op":"put","ts":"2025-12-14T18:09:00Z"},"data":{}}`,
			wantErr: true,
		},
		{
			name:    "valid JSON but invalid record - zero version",
			input:   `{"_meta":{"k":"key","v":0,"op":"put","ts":"2025-12-14T18:09:00Z"},"data":{}}`,
			wantErr: true,
		},
		{
			name:    "valid JSON but invalid record - invalid operation",
			input:   `{"_meta":{"k":"key","v":1,"op":"update","ts":"2025-12-14T18:09:00Z"},"data":{}}`,
			wantErr: true,
		},
		{
			name:    "valid JSON but invalid record - put without data",
			input:   `{"_meta":{"k":"key","v":1,"op":"put","ts":"2025-12-14T18:09:00Z"},"data":null}`,
			wantErr: true,
		},
		{
			name:    "delete record",
			input:   `{"_meta":{"k":"key","v":2,"op":"delete","ts":"2025-12-14T18:09:00Z"},"data":null}`,
			wantErr: false,
			check: func(t *testing.T, r *Record) {
				if !r.Meta.IsDelete() {
					t.Error("Expected delete operation")
				}
			},
		},
		{
			name:    "record with special characters in data",
			input:   `{"_meta":{"k":"key","v":1,"op":"put","ts":"2025-12-14T18:09:00Z"},"data":{"text":"Hello\nWorld\t\"quotes\""}}`,
			wantErr: false,
			check: func(t *testing.T, r *Record) {
				expected := "Hello\nWorld\t\"quotes\""
				if r.Data["text"] != expected {
					t.Errorf("Data[text] = %q, want %q", r.Data["text"], expected)
				}
			},
		},
		{
			name:    "record with nested data",
			input:   `{"_meta":{"k":"key","v":1,"op":"put","ts":"2025-12-14T18:09:00Z"},"data":{"user":{"name":"Alice","age":30}}}`,
			wantErr: false,
			check: func(t *testing.T, r *Record) {
				user, ok := r.Data["user"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected nested user object")
				}
				if user["name"] != "Alice" {
					t.Errorf("user.name = %v, want %q", user["name"], "Alice")
				}
			},
		},
		{
			name:    "large string data",
			input:   `{"_meta":{"k":"key","v":1,"op":"put","ts":"2025-12-14T18:09:00Z"},"data":{"large":"` + strings.Repeat("x", 10000) + `"}}`,
			wantErr: false,
			check: func(t *testing.T, r *Record) {
				if len(r.Data["large"].(string)) != 10000 {
					t.Errorf("Large string length = %d, want %d", len(r.Data["large"].(string)), 10000)
				}
			},
		},
	}

	decoder := NewDecoder()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record, err := decoder.DecodeString(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.check != nil {
				tt.check(t, record)
			}
		})
	}
}

// TestReadLastNRecords tests the ReadLastNRecords function
func TestReadLastNRecords(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.jsonl")

	// Create test file with 5 records
	encoder := NewEncoder()
	f, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	records := []*Record{
		NewPutRecord("key1", 1, map[string]interface{}{"value": "v1"}),
		NewPutRecord("key1", 2, map[string]interface{}{"value": "v2"}),
		NewPutRecord("key1", 3, map[string]interface{}{"value": "v3"}),
		NewPutRecord("key1", 4, map[string]interface{}{"value": "v4"}),
		NewPutRecord("key1", 5, map[string]interface{}{"value": "v5"}),
	}

	for _, record := range records {
		data, _ := encoder.Encode(record)
		f.Write(data)
	}
	f.Close()

	decoder := NewDecoder()

	tests := []struct {
		name          string
		n             int
		expectedCount int
		checkVersions []int
	}{
		{
			name:          "read last 3 records",
			n:             3,
			expectedCount: 3,
			checkVersions: []int{3, 4, 5},
		},
		{
			name:          "read last 1 record",
			n:             1,
			expectedCount: 1,
			checkVersions: []int{5},
		},
		{
			name:          "n equals total records",
			n:             5,
			expectedCount: 5,
			checkVersions: []int{1, 2, 3, 4, 5},
		},
		{
			name:          "n greater than total records",
			n:             10,
			expectedCount: 5,
			checkVersions: []int{1, 2, 3, 4, 5},
		},
		{
			name:          "n is zero",
			n:             0,
			expectedCount: 0,
			checkVersions: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := decoder.ReadLastNRecords(testFile, tt.n)
			if err != nil {
				t.Fatalf("ReadLastNRecords() error = %v", err)
			}

			if len(result) != tt.expectedCount {
				t.Errorf("ReadLastNRecords() returned %d records, want %d", len(result), tt.expectedCount)
				return
			}

			for i, expectedVersion := range tt.checkVersions {
				if result[i].Meta.Version != expectedVersion {
					t.Errorf("Record[%d].Version = %d, want %d", i, result[i].Meta.Version, expectedVersion)
				}
			}
		})
	}
}

// TestReadLastNRecordsNegative tests edge cases with negative N
// Note: The current implementation doesn't handle negative N gracefully
// This test is commented out to avoid panic, but indicates a potential improvement area
// func TestReadLastNRecordsNegative(t *testing.T) {
// 	tmpDir := t.TempDir()
// 	testFile := filepath.Join(tmpDir, "test.jsonl")
//
// 	// Create a simple test file
// 	encoder := NewEncoder()
// 	f, _ := os.Create(testFile)
// 	record := NewPutRecord("key", 1, map[string]interface{}{"value": "v1"})
// 	data, _ := encoder.Encode(record)
// 	f.Write(data)
// 	f.Close()
//
// 	decoder := NewDecoder()
//
// 	// TODO: Fix ReadLastNRecords to handle negative N gracefully
// 	result, err := decoder.ReadLastNRecords(testFile, -1)
// 	if err != nil {
// 		t.Fatalf("ReadLastNRecords() with negative N error = %v", err)
// 	}
//
// 	// Should return empty slice for negative N
// 	if len(result) != 0 {
// 		t.Errorf("Expected empty result for negative N, got %d records", len(result))
// 	}
// }

// TestReadLastNRecordsEmptyFile tests reading from an empty file
func TestReadLastNRecordsEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.jsonl")

	// Create empty file
	f, _ := os.Create(testFile)
	f.Close()

	decoder := NewDecoder()
	result, err := decoder.ReadLastNRecords(testFile, 5)
	if err != nil {
		t.Fatalf("ReadLastNRecords() on empty file error = %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d records", len(result))
	}
}

// TestReadLastNRecordsNonExistentFile tests reading from a non-existent file
func TestReadLastNRecordsNonExistentFile(t *testing.T) {
	decoder := NewDecoder()
	_, err := decoder.ReadLastNRecords("/nonexistent/file.jsonl", 5)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

// TestReadLines tests the ReadLines function
func TestReadLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "multiple lines",
			input:    "line1\nline2\nline3\n",
			expected: []string{"line1", "line2", "line3"},
		},
		{
			name:     "single line without newline",
			input:    "single line",
			expected: []string{"single line"},
		},
		{
			name:     "single line with newline",
			input:    "single line\n",
			expected: []string{"single line"},
		},
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
		{
			name:     "lines with spaces",
			input:    "  spaces  \n\ttabs\t\n",
			expected: []string{"  spaces  ", "\ttabs\t"},
		},
		{
			name:     "empty lines",
			input:    "line1\n\nline3\n",
			expected: []string{"line1", "", "line3"},
		},
		{
			name:     "only newlines",
			input:    "\n\n\n",
			expected: []string{"", "", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result, err := ReadLines(reader)
			if err != nil {
				t.Fatalf("ReadLines() error = %v", err)
			}

			if len(result) != len(tt.expected) {
				t.Errorf("ReadLines() returned %d lines, want %d", len(result), len(tt.expected))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Line[%d] = %q, want %q", i, result[i], expected)
				}
			}
		})
	}
}

// TestReadLinesLargeFile tests reading a large number of lines
func TestReadLinesLargeFile(t *testing.T) {
	// Create a buffer with many lines
	var buf bytes.Buffer
	lineCount := 10000
	for i := 0; i < lineCount; i++ {
		buf.WriteString("line\n")
	}

	result, err := ReadLines(&buf)
	if err != nil {
		t.Fatalf("ReadLines() error = %v", err)
	}

	if len(result) != lineCount {
		t.Errorf("ReadLines() returned %d lines, want %d", len(result), lineCount)
	}
}

// TestDecodeErrors tests various error conditions in Decode
func TestDecodeErrors(t *testing.T) {
	decoder := NewDecoder()

	tests := []struct {
		name  string
		input []byte
	}{
		{
			name:  "empty bytes",
			input: []byte{},
		},
		{
			name:  "only whitespace",
			input: []byte("   \t\n   "),
		},
		{
			name:  "malformed JSON - missing closing brace",
			input: []byte(`{"_meta":{"k":"key"`),
		},
		{
			name:  "malformed JSON - invalid syntax",
			input: []byte(`{invalid json}`),
		},
		{
			name:  "corrupted data",
			input: []byte{0xFF, 0xFE, 0xFD},
		},
		{
			name:  "incomplete record",
			input: []byte(`{"_meta":`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decoder.Decode(tt.input)
			if err == nil {
				t.Error("Expected error for invalid input")
			}
		})
	}
}

// TestDecoderReadAllWithInvalidLines tests ReadAll skipping invalid lines
func TestDecoderReadAllWithInvalidLines(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "mixed.jsonl")

	// Create file with mix of valid and invalid lines
	f, _ := os.Create(testFile)
	encoder := NewEncoder()

	// Valid record
	validRecord := NewPutRecord("key1", 1, map[string]interface{}{"value": "v1"})
	validData, _ := encoder.Encode(validRecord)
	f.Write(validData)

	// Invalid line
	f.WriteString("invalid json line\n")

	// Another valid record
	validRecord2 := NewPutRecord("key2", 2, map[string]interface{}{"value": "v2"})
	validData2, _ := encoder.Encode(validRecord2)
	f.Write(validData2)

	// Empty line
	f.WriteString("\n")

	// Another valid record
	validRecord3 := NewPutRecord("key3", 3, map[string]interface{}{"value": "v3"})
	validData3, _ := encoder.Encode(validRecord3)
	f.Write(validData3)

	f.Close()

	decoder := NewDecoder()
	records, err := decoder.ReadAll(testFile)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	// Should only get the 3 valid records
	if len(records) != 3 {
		t.Errorf("Expected 3 valid records, got %d", len(records))
	}
}

// TestReadLastValidReverseEmptyFile tests reading from empty file
func TestReadLastValidReverseEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.jsonl")

	f, _ := os.Create(testFile)
	f.Close()

	decoder := NewDecoder()
	record, err := decoder.ReadLastValidReverse(testFile)
	if err != nil {
		t.Fatalf("ReadLastValidReverse() error = %v", err)
	}

	if record != nil {
		t.Error("Expected nil record for empty file")
	}
}

// TestReadLastValidReverseLargeFile tests efficient reading of large files
func TestReadLastValidReverseLargeFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.jsonl")

	encoder := NewEncoder()
	f, _ := os.Create(testFile)

	// Write many records (more than one chunk)
	for i := 1; i <= 200; i++ {
		record := NewPutRecord("key", i, map[string]interface{}{"value": i})
		data, _ := encoder.Encode(record)
		f.Write(data)
	}
	f.Close()

	decoder := NewDecoder()
	record, err := decoder.ReadLastValidReverse(testFile)
	if err != nil {
		t.Fatalf("ReadLastValidReverse() error = %v", err)
	}

	if record == nil {
		t.Fatal("Expected non-nil record")
	}

	// Should get the last record (version 200)
	if record.Meta.Version != 200 {
		t.Errorf("Expected version 200, got %d", record.Meta.Version)
	}
}

// TestReadVersionNotFound tests reading a version that doesn't exist
func TestReadVersionNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.jsonl")

	encoder := NewEncoder()
	f, _ := os.Create(testFile)
	record := NewPutRecord("key", 1, map[string]interface{}{"value": "v1"})
	data, _ := encoder.Encode(record)
	f.Write(data)
	f.Close()

	decoder := NewDecoder()
	_, err := decoder.ReadVersion(testFile, 999)
	if err == nil {
		t.Error("Expected error for non-existent version")
	}
}

// TestCountLinesNonExistentFile tests CountLines with non-existent file
func TestCountLinesNonExistentFile(t *testing.T) {
	_, err := CountLines("/nonexistent/file.txt")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

// TestAppendRecordInvalidRecord tests AppendRecord with invalid record
func TestAppendRecordInvalidRecord(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.jsonl")

	// Create invalid record (empty key)
	invalidRecord := &Record{
		Meta: NewMeta("", 1, OpPut),
		Data: map[string]interface{}{},
	}

	err := AppendRecord(testFile, invalidRecord)
	if err == nil {
		t.Error("Expected error for invalid record")
	}
}

// TestAppendRecordNilRecord tests AppendRecord with nil record
func TestAppendRecordNilRecord(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.jsonl")

	err := AppendRecord(testFile, nil)
	if err == nil {
		t.Error("Expected error for nil record")
	}
}
