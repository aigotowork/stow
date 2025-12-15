package core

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// TestEncodeDecodeRoundTrip tests full encode-decode cycle
func TestEncodeDecodeRoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		record *Record
	}{
		{
			name: "simple put record",
			record: NewPutRecord("user:1", 1, map[string]interface{}{
				"name": "Alice",
				"age":  30.0,
			}),
		},
		{
			name:   "delete record",
			record: NewDeleteRecord("user:2", 5),
		},
		{
			name: "complex nested data",
			record: NewPutRecord("profile:1", 1, map[string]interface{}{
				"user": map[string]interface{}{
					"name": "Bob",
					"contacts": map[string]interface{}{
						"email": "bob@example.com",
						"phone": "123-456-7890",
					},
				},
				"settings": map[string]interface{}{
					"theme":         "dark",
					"notifications": true,
				},
				"tags": []interface{}{"admin", "verified"},
			}),
		},
		{
			name: "record with unicode",
			record: NewPutRecord("国际化", 1, map[string]interface{}{
				"text":  "Hello 世界 🌍",
				"emoji": "😀🎉🚀",
			}),
		},
	}

	encoder := NewEncoder()
	decoder := NewDecoder()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			encoded, err := encoder.Encode(tt.record)
			if err != nil {
				t.Fatalf("Encode() error = %v", err)
			}

			// Decode
			decoded, err := decoder.Decode(encoded)
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}

			// Verify metadata
			if decoded.Meta.Key != tt.record.Meta.Key {
				t.Errorf("Key mismatch: got %q, want %q", decoded.Meta.Key, tt.record.Meta.Key)
			}
			if decoded.Meta.Version != tt.record.Meta.Version {
				t.Errorf("Version mismatch: got %d, want %d", decoded.Meta.Version, tt.record.Meta.Version)
			}
			if decoded.Meta.Operation != tt.record.Meta.Operation {
				t.Errorf("Operation mismatch: got %q, want %q", decoded.Meta.Operation, tt.record.Meta.Operation)
			}

			// Verify validity
			if !decoded.IsValid() {
				t.Error("Decoded record should be valid")
			}
		})
	}
}

// TestRecordPersistence tests writing and reading records from files
func TestRecordPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "persistence.jsonl")

	// Create and write records
	records := []*Record{
		NewPutRecord("key1", 1, map[string]interface{}{"value": "v1"}),
		NewPutRecord("key1", 2, map[string]interface{}{"value": "v2"}),
		NewPutRecord("key2", 1, map[string]interface{}{"value": "v2_1"}),
		NewDeleteRecord("key1", 3),
	}

	for _, record := range records {
		err := AppendRecord(testFile, record)
		if err != nil {
			t.Fatalf("AppendRecord() error = %v", err)
		}
	}

	// Read all records
	decoder := NewDecoder()
	readRecords, err := decoder.ReadAll(testFile)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	// Verify count
	if len(readRecords) != len(records) {
		t.Fatalf("Read %d records, want %d", len(readRecords), len(records))
	}

	// Verify each record
	for i, original := range records {
		read := readRecords[i]
		if read.Meta.Key != original.Meta.Key {
			t.Errorf("Record[%d].Key = %q, want %q", i, read.Meta.Key, original.Meta.Key)
		}
		if read.Meta.Version != original.Meta.Version {
			t.Errorf("Record[%d].Version = %d, want %d", i, read.Meta.Version, original.Meta.Version)
		}
		if read.Meta.Operation != original.Meta.Operation {
			t.Errorf("Record[%d].Operation = %q, want %q", i, read.Meta.Operation, original.Meta.Operation)
		}
	}
}

// TestVersionEvolution tests multiple versions of the same key
func TestVersionEvolution(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "versions.jsonl")

	// Create multiple versions
	versions := []*Record{
		NewPutRecord("user:alice", 1, map[string]interface{}{"name": "Alice", "age": 25.0}),
		NewPutRecord("user:alice", 2, map[string]interface{}{"name": "Alice", "age": 26.0}),
		NewPutRecord("user:alice", 3, map[string]interface{}{"name": "Alice Smith", "age": 26.0}),
		NewPutRecord("user:alice", 4, map[string]interface{}{"name": "Alice Smith", "age": 27.0}),
	}

	for _, record := range versions {
		AppendRecord(testFile, record)
	}

	decoder := NewDecoder()

	t.Run("GetLatestVersion", func(t *testing.T) {
		latestVersion, err := decoder.GetLatestVersion(testFile)
		if err != nil {
			t.Fatalf("GetLatestVersion() error = %v", err)
		}
		if latestVersion != 4 {
			t.Errorf("GetLatestVersion() = %d, want %d", latestVersion, 4)
		}
	})

	t.Run("ReadVersion", func(t *testing.T) {
		// Read version 2
		record, err := decoder.ReadVersion(testFile, 2)
		if err != nil {
			t.Fatalf("ReadVersion() error = %v", err)
		}
		if record.Meta.Version != 2 {
			t.Errorf("Version = %d, want %d", record.Meta.Version, 2)
		}
		if record.Data["age"] != 26.0 {
			t.Errorf("Age = %v, want %v", record.Data["age"], 26.0)
		}
	})

	t.Run("ReadLastValid", func(t *testing.T) {
		record, err := decoder.ReadLastValid(testFile)
		if err != nil {
			t.Fatalf("ReadLastValid() error = %v", err)
		}
		if record.Meta.Version != 4 {
			t.Errorf("Version = %d, want %d", record.Meta.Version, 4)
		}
		if record.Data["age"] != 27.0 {
			t.Errorf("Age = %v, want %v", record.Data["age"], 27.0)
		}
	})
}

// TestConcurrentReads tests concurrent reading from the same file
func TestConcurrentReads(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "concurrent_reads.jsonl")

	// Write test data
	encoder := NewEncoder()
	f, _ := os.Create(testFile)
	for i := 1; i <= 100; i++ {
		record := NewPutRecord("key", i, map[string]interface{}{"value": i})
		data, _ := encoder.Encode(record)
		f.Write(data)
	}
	f.Close()

	// Concurrent reads
	const numReaders = 10
	var wg sync.WaitGroup
	errors := make(chan error, numReaders)

	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()

			decoder := NewDecoder()

			// Test ReadAll
			records, err := decoder.ReadAll(testFile)
			if err != nil {
				errors <- err
				return
			}
			if len(records) != 100 {
				t.Errorf("Reader %d: expected 100 records, got %d", readerID, len(records))
			}

			// Test ReadLastValid
			_, err = decoder.ReadLastValid(testFile)
			if err != nil {
				errors <- err
				return
			}

			// Test GetLatestVersion
			_, err = decoder.GetLatestVersion(testFile)
			if err != nil {
				errors <- err
				return
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent read error: %v", err)
	}
}

// TestConcurrentWrites tests concurrent writing to the same file
func TestConcurrentWrites(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "concurrent_writes.jsonl")

	const numWriters = 10
	const recordsPerWriter = 10
	var wg sync.WaitGroup
	errors := make(chan error, numWriters)

	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()

			for j := 0; j < recordsPerWriter; j++ {
				record := NewPutRecord("key", writerID*100+j+1, map[string]interface{}{
					"writer": writerID,
					"index":  j,
				})

				err := AppendRecord(testFile, record)
				if err != nil {
					errors <- err
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent write error: %v", err)
	}

	// Verify all records were written
	decoder := NewDecoder()
	records, err := decoder.ReadAll(testFile)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	expectedCount := numWriters * recordsPerWriter
	if len(records) != expectedCount {
		t.Errorf("Expected %d records, got %d", expectedCount, len(records))
	}
}

// TestReadWriteMixed tests mixed concurrent reads and writes
func TestReadWriteMixed(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "mixed.jsonl")

	// Write initial data
	for i := 1; i <= 50; i++ {
		record := NewPutRecord("key", i, map[string]interface{}{"value": i})
		AppendRecord(testFile, record)
	}

	const numOperations = 20
	var wg sync.WaitGroup
	errors := make(chan error, numOperations)

	// Launch readers and writers
	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		if i%2 == 0 {
			// Reader
			go func(id int) {
				defer wg.Done()
				decoder := NewDecoder()
				_, err := decoder.ReadAll(testFile)
				if err != nil {
					errors <- err
				}
			}(i)
		} else {
			// Writer
			go func(id int) {
				defer wg.Done()
				record := NewPutRecord("key", 50+id, map[string]interface{}{"value": 50 + id})
				err := AppendRecord(testFile, record)
				if err != nil {
					errors <- err
				}
			}(i)
		}
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Mixed operation error: %v", err)
	}
}

// TestDataIntegrity tests data integrity after round-trip
func TestDataIntegrity(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "integrity.jsonl")

	// Create record with various data types
	originalData := map[string]interface{}{
		"string":  "test string with special chars: \n\t\"quotes\"",
		"number":  42.5,
		"integer": 100.0,
		"boolean": true,
		"null":    nil,
		"array":   []interface{}{1, 2, 3, "four"},
		"nested": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": "deep value",
			},
		},
	}

	original := NewPutRecord("integrity-test", 1, originalData)

	// Write
	err := AppendRecord(testFile, original)
	if err != nil {
		t.Fatalf("AppendRecord() error = %v", err)
	}

	// Read back
	decoder := NewDecoder()
	records, err := decoder.ReadAll(testFile)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(records))
	}

	read := records[0]

	// Verify each field
	if read.Data["string"] != originalData["string"] {
		t.Error("String field mismatch")
	}
	if read.Data["number"] != originalData["number"] {
		t.Error("Number field mismatch")
	}
	if read.Data["boolean"] != originalData["boolean"] {
		t.Error("Boolean field mismatch")
	}

	// Verify nested structure
	nested := read.Data["nested"].(map[string]interface{})
	level2 := nested["level2"].(map[string]interface{})
	if level2["level3"] != "deep value" {
		t.Error("Nested value mismatch")
	}
}

// TestFileOperationsSequence tests a sequence of file operations
func TestFileOperationsSequence(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "sequence.jsonl")

	decoder := NewDecoder()

	// Initially file doesn't exist
	version, err := decoder.GetLatestVersion(testFile)
	if err != nil {
		t.Fatalf("GetLatestVersion() on non-existent file error = %v", err)
	}
	if version != 0 {
		t.Errorf("Initial version = %d, want 0", version)
	}

	// Write first record
	AppendRecord(testFile, NewPutRecord("key", 1, map[string]interface{}{"value": "v1"}))

	// Check version
	version, _ = decoder.GetLatestVersion(testFile)
	if version != 1 {
		t.Errorf("Version after first write = %d, want 1", version)
	}

	// Write more records
	AppendRecord(testFile, NewPutRecord("key", 2, map[string]interface{}{"value": "v2"}))
	AppendRecord(testFile, NewPutRecord("key", 3, map[string]interface{}{"value": "v3"}))

	// Check version
	version, _ = decoder.GetLatestVersion(testFile)
	if version != 3 {
		t.Errorf("Version after writes = %d, want 3", version)
	}

	// Read last valid
	record, _ := decoder.ReadLastValid(testFile)
	if record.Meta.Version != 3 {
		t.Errorf("Last valid version = %d, want 3", record.Meta.Version)
	}

	// Write delete record
	AppendRecord(testFile, NewDeleteRecord("key", 4))

	// Read last valid should return nil
	record, _ = decoder.ReadLastValid(testFile)
	if record != nil {
		t.Error("Last valid after delete should be nil")
	}

	// But latest version should still be 4
	version, _ = decoder.GetLatestVersion(testFile)
	if version != 4 {
		t.Errorf("Version after delete = %d, want 4", version)
	}
}

// TestLargeFile tests operations on large files
func TestLargeFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large file test in short mode")
	}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.jsonl")

	// Write many records
	const numRecords = 1000
	encoder := NewEncoder()
	f, _ := os.Create(testFile)

	for i := 1; i <= numRecords; i++ {
		record := NewPutRecord("key", i, map[string]interface{}{
			"index": i,
			"data":  "some data here",
		})
		data, _ := encoder.Encode(record)
		f.Write(data)
	}
	f.Close()

	decoder := NewDecoder()

	t.Run("ReadAll", func(t *testing.T) {
		records, err := decoder.ReadAll(testFile)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}
		if len(records) != numRecords {
			t.Errorf("Read %d records, want %d", len(records), numRecords)
		}
	})

	t.Run("ReadLastValid", func(t *testing.T) {
		record, err := decoder.ReadLastValid(testFile)
		if err != nil {
			t.Fatalf("ReadLastValid() error = %v", err)
		}
		if record.Meta.Version != numRecords {
			t.Errorf("Last version = %d, want %d", record.Meta.Version, numRecords)
		}
	})

	t.Run("CountLines", func(t *testing.T) {
		count, err := CountLines(testFile)
		if err != nil {
			t.Fatalf("CountLines() error = %v", err)
		}
		if count != numRecords {
			t.Errorf("Line count = %d, want %d", count, numRecords)
		}
	})

	t.Run("ReadLastNRecords", func(t *testing.T) {
		records, err := decoder.ReadLastNRecords(testFile, 100)
		if err != nil {
			t.Fatalf("ReadLastNRecords() error = %v", err)
		}
		if len(records) != 100 {
			t.Errorf("Read %d records, want 100", len(records))
		}
		// Should be versions 901-1000
		if records[0].Meta.Version != numRecords-99 {
			t.Errorf("First record version = %d, want %d", records[0].Meta.Version, numRecords-99)
		}
	})
}

// BenchmarkEncode benchmarks record encoding
func BenchmarkEncode(b *testing.B) {
	record := NewPutRecord("benchmark-key", 1, map[string]interface{}{
		"name":  "Benchmark User",
		"email": "bench@example.com",
		"age":   25.0,
	})

	encoder := NewEncoder()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = encoder.Encode(record)
	}
}

// BenchmarkDecode benchmarks record decoding
func BenchmarkDecode(b *testing.B) {
	record := NewPutRecord("benchmark-key", 1, map[string]interface{}{
		"name":  "Benchmark User",
		"email": "bench@example.com",
		"age":   25.0,
	})

	encoder := NewEncoder()
	data, _ := encoder.Encode(record)

	decoder := NewDecoder()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = decoder.Decode(data)
	}
}

// BenchmarkAppendRecord benchmarks appending records to a file
func BenchmarkAppendRecord(b *testing.B) {
	tmpDir := b.TempDir()
	testFile := filepath.Join(tmpDir, "benchmark.jsonl")

	record := NewPutRecord("key", 1, map[string]interface{}{"value": "test"})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = AppendRecord(testFile, record)
	}
}

// BenchmarkReadLastValid benchmarks reading the last valid record
func BenchmarkReadLastValid(b *testing.B) {
	tmpDir := b.TempDir()
	testFile := filepath.Join(tmpDir, "benchmark.jsonl")

	// Create file with records
	encoder := NewEncoder()
	f, _ := os.Create(testFile)
	for i := 1; i <= 100; i++ {
		record := NewPutRecord("key", i, map[string]interface{}{"value": i})
		data, _ := encoder.Encode(record)
		f.Write(data)
	}
	f.Close()

	decoder := NewDecoder()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = decoder.ReadLastValid(testFile)
	}
}

// BenchmarkReadAll benchmarks reading all records
func BenchmarkReadAll(b *testing.B) {
	tmpDir := b.TempDir()
	testFile := filepath.Join(tmpDir, "benchmark.jsonl")

	// Create file with records
	encoder := NewEncoder()
	f, _ := os.Create(testFile)
	for i := 1; i <= 100; i++ {
		record := NewPutRecord("key", i, map[string]interface{}{"value": i})
		data, _ := encoder.Encode(record)
		f.Write(data)
	}
	f.Close()

	decoder := NewDecoder()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = decoder.ReadAll(testFile)
	}
}
