package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMetaCreation(t *testing.T) {
	meta := NewMeta("test-key", 1, OpPut)

	if meta.Key != "test-key" {
		t.Errorf("Key mismatch: got %q, want %q", meta.Key, "test-key")
	}

	if meta.Version != 1 {
		t.Errorf("Version mismatch: got %d, want %d", meta.Version, 1)
	}

	if meta.Operation != OpPut {
		t.Errorf("Operation mismatch: got %q, want %q", meta.Operation, OpPut)
	}

	if meta.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestMetaOperationChecks(t *testing.T) {
	putMeta := NewMeta("key", 1, OpPut)
	if !putMeta.IsPut() {
		t.Error("IsPut() should return true for put operation")
	}
	if putMeta.IsDelete() {
		t.Error("IsDelete() should return false for put operation")
	}

	deleteMeta := NewMeta("key", 2, OpDelete)
	if deleteMeta.IsPut() {
		t.Error("IsPut() should return false for delete operation")
	}
	if !deleteMeta.IsDelete() {
		t.Error("IsDelete() should return true for delete operation")
	}
}

func TestRecordCreation(t *testing.T) {
	data := map[string]interface{}{
		"name": "Alice",
		"age":  30,
	}

	record := NewPutRecord("user:1", 1, data)

	if !record.IsValid() {
		t.Error("Record should be valid")
	}

	if record.Meta.Key != "user:1" {
		t.Errorf("Key mismatch: got %q, want %q", record.Meta.Key, "user:1")
	}

	if record.Meta.Operation != OpPut {
		t.Errorf("Operation mismatch: got %q, want %q", record.Meta.Operation, OpPut)
	}

	if record.Data["name"] != "Alice" {
		t.Error("Data mismatch")
	}
}

func TestDeleteRecordCreation(t *testing.T) {
	record := NewDeleteRecord("user:1", 2)

	if !record.IsValid() {
		t.Error("Delete record should be valid")
	}

	if record.Meta.Operation != OpDelete {
		t.Errorf("Operation should be delete, got %q", record.Meta.Operation)
	}

	if record.Data != nil {
		t.Error("Delete record should have nil data")
	}
}

func TestRecordValidation(t *testing.T) {
	tests := []struct {
		name    string
		record  *Record
		isValid bool
	}{
		{
			name: "valid put record",
			record: &Record{
				Meta: &Meta{Key: "key", Version: 1, Operation: OpPut, Timestamp: time.Now()},
				Data: map[string]interface{}{"value": "test"},
			},
			isValid: true,
		},
		{
			name: "valid delete record",
			record: &Record{
				Meta: &Meta{Key: "key", Version: 2, Operation: OpDelete, Timestamp: time.Now()},
				Data: nil,
			},
			isValid: true,
		},
		{
			name: "nil meta",
			record: &Record{
				Meta: nil,
				Data: map[string]interface{}{},
			},
			isValid: false,
		},
		{
			name: "empty key",
			record: &Record{
				Meta: &Meta{Key: "", Version: 1, Operation: OpPut, Timestamp: time.Now()},
				Data: map[string]interface{}{},
			},
			isValid: false,
		},
		{
			name: "invalid version",
			record: &Record{
				Meta: &Meta{Key: "key", Version: 0, Operation: OpPut, Timestamp: time.Now()},
				Data: map[string]interface{}{},
			},
			isValid: false,
		},
		{
			name: "invalid operation",
			record: &Record{
				Meta: &Meta{Key: "key", Version: 1, Operation: "invalid", Timestamp: time.Now()},
				Data: map[string]interface{}{},
			},
			isValid: false,
		},
		{
			name: "put record with nil data",
			record: &Record{
				Meta: &Meta{Key: "key", Version: 1, Operation: OpPut, Timestamp: time.Now()},
				Data: nil,
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.record.IsValid()
			if result != tt.isValid {
				t.Errorf("IsValid() = %v, want %v", result, tt.isValid)
			}
		})
	}
}

func TestEncoderDecode(t *testing.T) {
	encoder := NewEncoder()

	// Create test record
	data := map[string]interface{}{
		"name": "Bob",
		"age":  25.0,
	}
	record := NewPutRecord("test-key", 1, data)

	// Encode
	encoded, err := encoder.Encode(record)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Should end with newline
	if encoded[len(encoded)-1] != '\n' {
		t.Error("Encoded data should end with newline")
	}

	// Decode
	decoder := NewDecoder()
	decoded, err := decoder.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Verify
	if decoded.Meta.Key != "test-key" {
		t.Errorf("Key mismatch: got %q, want %q", decoded.Meta.Key, "test-key")
	}

	if decoded.Data["name"] != "Bob" {
		t.Error("Data mismatch")
	}
}

func TestDecoderReadAll(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.jsonl")

	// Create test file with multiple records
	f, _ := os.Create(testFile)
	encoder := NewEncoder()

	records := []*Record{
		NewPutRecord("key1", 1, map[string]interface{}{"value": "first"}),
		NewPutRecord("key1", 2, map[string]interface{}{"value": "second"}),
		NewDeleteRecord("key1", 3),
	}

	for _, record := range records {
		data, _ := encoder.Encode(record)
		f.Write(data)
	}
	f.Close()

	// Read all
	decoder := NewDecoder()
	readRecords, err := decoder.ReadAll(testFile)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if len(readRecords) != 3 {
		t.Fatalf("Expected 3 records, got %d", len(readRecords))
	}

	// Verify last record is delete
	if !readRecords[2].Meta.IsDelete() {
		t.Error("Last record should be delete")
	}
}

func TestDecoderReadLastValid(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.jsonl")

	// Create test file
	f, _ := os.Create(testFile)
	encoder := NewEncoder()

	records := []*Record{
		NewPutRecord("key1", 1, map[string]interface{}{"value": "first"}),
		NewPutRecord("key1", 2, map[string]interface{}{"value": "second"}),
		NewPutRecord("key1", 3, map[string]interface{}{"value": "third"}),
	}

	for _, record := range records {
		data, _ := encoder.Encode(record)
		f.Write(data)
	}
	f.Close()

	// Read last valid
	decoder := NewDecoder()
	lastRecord, err := decoder.ReadLastValid(testFile)
	if err != nil {
		t.Fatalf("ReadLastValid failed: %v", err)
	}

	if lastRecord == nil {
		t.Fatal("Last record should not be nil")
	}

	if lastRecord.Meta.Version != 3 {
		t.Errorf("Expected version 3, got %d", lastRecord.Meta.Version)
	}

	if lastRecord.Data["value"] != "third" {
		t.Errorf("Expected value 'third', got %v", lastRecord.Data["value"])
	}
}

func TestDecoderReadLastValidWithDelete(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.jsonl")

	// Create test file ending with delete
	f, _ := os.Create(testFile)
	encoder := NewEncoder()

	records := []*Record{
		NewPutRecord("key1", 1, map[string]interface{}{"value": "first"}),
		NewDeleteRecord("key1", 2),
	}

	for _, record := range records {
		data, _ := encoder.Encode(record)
		f.Write(data)
	}
	f.Close()

	// Read last valid
	decoder := NewDecoder()
	lastRecord, err := decoder.ReadLastValid(testFile)
	if err != nil {
		t.Fatalf("ReadLastValid failed: %v", err)
	}

	// Should return nil because last operation is delete
	if lastRecord != nil {
		t.Error("Last record should be nil when last operation is delete")
	}
}

func TestDecoderGetLatestVersion(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.jsonl")

	// Create test file
	f, _ := os.Create(testFile)
	encoder := NewEncoder()

	records := []*Record{
		NewPutRecord("key1", 1, map[string]interface{}{}),
		NewPutRecord("key1", 5, map[string]interface{}{}),
		NewPutRecord("key1", 3, map[string]interface{}{}),
	}

	for _, record := range records {
		data, _ := encoder.Encode(record)
		f.Write(data)
	}
	f.Close()

	// Get latest version
	decoder := NewDecoder()
	version, err := decoder.GetLatestVersion(testFile)
	if err != nil {
		t.Fatalf("GetLatestVersion failed: %v", err)
	}

	if version != 5 {
		t.Errorf("Expected version 5, got %d", version)
	}
}

func TestDecoderGetLatestVersionNonExistent(t *testing.T) {
	decoder := NewDecoder()
	version, err := decoder.GetLatestVersion("/nonexistent/file.jsonl")
	if err != nil {
		t.Fatalf("GetLatestVersion should not error for non-existent file: %v", err)
	}

	if version != 0 {
		t.Errorf("Expected version 0 for non-existent file, got %d", version)
	}
}

func TestAppendRecord(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.jsonl")

	// Append multiple records
	records := []*Record{
		NewPutRecord("key1", 1, map[string]interface{}{"value": "first"}),
		NewPutRecord("key1", 2, map[string]interface{}{"value": "second"}),
	}

	for _, record := range records {
		err := AppendRecord(testFile, record)
		if err != nil {
			t.Fatalf("AppendRecord failed: %v", err)
		}
	}

	// Verify file exists and contains both records
	decoder := NewDecoder()
	readRecords, err := decoder.ReadAll(testFile)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if len(readRecords) != 2 {
		t.Fatalf("Expected 2 records, got %d", len(readRecords))
	}
}

func TestCountLines(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.jsonl")

	// Create file with known number of lines
	f, _ := os.Create(testFile)
	f.WriteString("line1\n")
	f.WriteString("line2\n")
	f.WriteString("line3\n")
	f.Close()

	// Count lines
	count, err := CountLines(testFile)
	if err != nil {
		t.Fatalf("CountLines failed: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 lines, got %d", count)
	}
}

func TestDecoderReadVersion(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.jsonl")

	// Create test file
	f, _ := os.Create(testFile)
	encoder := NewEncoder()

	records := []*Record{
		NewPutRecord("key1", 1, map[string]interface{}{"value": "v1"}),
		NewPutRecord("key1", 2, map[string]interface{}{"value": "v2"}),
		NewPutRecord("key1", 3, map[string]interface{}{"value": "v3"}),
	}

	for _, record := range records {
		data, _ := encoder.Encode(record)
		f.Write(data)
	}
	f.Close()

	// Read version 2
	decoder := NewDecoder()
	record, err := decoder.ReadVersion(testFile, 2)
	if err != nil {
		t.Fatalf("ReadVersion failed: %v", err)
	}

	if record.Data["value"] != "v2" {
		t.Errorf("Expected value 'v2', got %v", record.Data["value"])
	}
}
