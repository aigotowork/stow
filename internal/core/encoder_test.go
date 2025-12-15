package core

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestEncodeToString tests the EncodeToString function
func TestEncodeToString(t *testing.T) {
	tests := []struct {
		name    string
		record  *Record
		wantErr bool
		check   func(*testing.T, string)
	}{
		{
			name: "valid put record",
			record: NewPutRecord("test-key", 1, map[string]interface{}{
				"name": "Alice",
				"age":  30.0,
			}),
			wantErr: false,
			check: func(t *testing.T, result string) {
				if !strings.HasSuffix(result, "\n") {
					t.Error("Encoded string should end with newline")
				}
				// Verify it's valid JSON
				var decoded Record
				err := json.Unmarshal([]byte(result), &decoded)
				if err != nil {
					t.Errorf("Result is not valid JSON: %v", err)
				}
				if decoded.Meta.Key != "test-key" {
					t.Errorf("Decoded key = %q, want %q", decoded.Meta.Key, "test-key")
				}
			},
		},
		{
			name:    "valid delete record",
			record:  NewDeleteRecord("key-to-delete", 5),
			wantErr: false,
			check: func(t *testing.T, result string) {
				if !strings.Contains(result, `"op":"delete"`) {
					t.Error("Should contain delete operation")
				}
			},
		},
		{
			name: "record with special characters",
			record: NewPutRecord("key", 1, map[string]interface{}{
				"text": "Hello\nWorld\t\"quotes\"",
			}),
			wantErr: false,
			check: func(t *testing.T, result string) {
				// JSON should escape special characters
				if !strings.Contains(result, `\n`) || !strings.Contains(result, `\t`) {
					t.Error("Special characters should be escaped")
				}
			},
		},
		{
			name: "record with nested data",
			record: NewPutRecord("key", 1, map[string]interface{}{
				"user": map[string]interface{}{
					"name": "Bob",
					"profile": map[string]interface{}{
						"age": 25.0,
					},
				},
			}),
			wantErr: false,
			check: func(t *testing.T, result string) {
				// Should contain nested structure
				if !strings.Contains(result, `"user"`) {
					t.Error("Should contain nested user data")
				}
			},
		},
		{
			name: "record with empty data",
			record: NewPutRecord("key", 1, map[string]interface{}{
				// Empty but not nil
			}),
			wantErr: false,
			check: func(t *testing.T, result string) {
				if !strings.Contains(result, `"data":{}`) {
					t.Error("Should encode empty data as {}")
				}
			},
		},
		{
			name: "record with various data types",
			record: NewPutRecord("key", 1, map[string]interface{}{
				"string":  "text",
				"number":  42.5,
				"boolean": true,
				"null":    nil,
				"array":   []interface{}{1, 2, 3},
			}),
			wantErr: false,
			check: func(t *testing.T, result string) {
				// All types should be encoded
				if !strings.Contains(result, `"string":"text"`) {
					t.Error("Should contain string field")
				}
			},
		},
		{
			name:    "nil record",
			record:  nil,
			wantErr: true,
		},
		{
			name: "invalid record - empty key",
			record: &Record{
				Meta: NewMeta("", 1, OpPut),
				Data: map[string]interface{}{"value": "test"},
			},
			wantErr: true,
		},
		{
			name: "invalid record - zero version",
			record: &Record{
				Meta: NewMeta("key", 0, OpPut),
				Data: map[string]interface{}{"value": "test"},
			},
			wantErr: true,
		},
		{
			name: "invalid record - invalid operation",
			record: &Record{
				Meta: &Meta{
					Key:       "key",
					Version:   1,
					Operation: "invalid-op",
				},
				Data: map[string]interface{}{},
			},
			wantErr: true,
		},
		{
			name: "invalid record - nil meta",
			record: &Record{
				Meta: nil,
				Data: map[string]interface{}{},
			},
			wantErr: true,
		},
		{
			name: "invalid record - put without data",
			record: &Record{
				Meta: NewMeta("key", 1, OpPut),
				Data: nil,
			},
			wantErr: true,
		},
	}

	encoder := NewEncoder()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := encoder.EncodeToString(tt.record)

			if (err != nil) != tt.wantErr {
				t.Errorf("EncodeToString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

// TestEncodeToStringRoundTrip tests encoding and then decoding
func TestEncodeToStringRoundTrip(t *testing.T) {
	originalData := map[string]interface{}{
		"name":   "Alice",
		"age":    30.0,
		"email":  "alice@example.com",
		"active": true,
	}

	original := NewPutRecord("user:alice", 5, originalData)

	encoder := NewEncoder()
	encoded, err := encoder.EncodeToString(original)
	if err != nil {
		t.Fatalf("EncodeToString() error = %v", err)
	}

	decoder := NewDecoder()
	decoded, err := decoder.DecodeString(encoded)
	if err != nil {
		t.Fatalf("DecodeString() error = %v", err)
	}

	// Verify all fields match
	if decoded.Meta.Key != original.Meta.Key {
		t.Errorf("Key = %q, want %q", decoded.Meta.Key, original.Meta.Key)
	}
	if decoded.Meta.Version != original.Meta.Version {
		t.Errorf("Version = %d, want %d", decoded.Meta.Version, original.Meta.Version)
	}
	if decoded.Meta.Operation != original.Meta.Operation {
		t.Errorf("Operation = %q, want %q", decoded.Meta.Operation, original.Meta.Operation)
	}

	// Verify data
	if decoded.Data["name"] != originalData["name"] {
		t.Errorf("Data[name] = %v, want %v", decoded.Data["name"], originalData["name"])
	}
	if decoded.Data["age"] != originalData["age"] {
		t.Errorf("Data[age] = %v, want %v", decoded.Data["age"], originalData["age"])
	}
}

// TestEncodeMultipleFormats tests encoding consistency
func TestEncodeMultipleFormats(t *testing.T) {
	record := NewPutRecord("test", 1, map[string]interface{}{"value": "test"})

	encoder := NewEncoder()

	// Encode to bytes
	bytes1, err1 := encoder.Encode(record)
	if err1 != nil {
		t.Fatalf("Encode() error = %v", err1)
	}

	// Encode to string
	str, err2 := encoder.EncodeToString(record)
	if err2 != nil {
		t.Fatalf("EncodeToString() error = %v", err2)
	}

	// They should be identical
	if string(bytes1) != str {
		t.Error("Encode() and EncodeToString() should produce identical output")
	}
}

// TestEncodeLargeData tests encoding large data structures
func TestEncodeLargeData(t *testing.T) {
	// Create large data map
	largeData := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		largeData[string(rune('a'+i%26))+string(rune('0'+i%10))] = strings.Repeat("x", 100)
	}

	record := NewPutRecord("large-key", 1, largeData)

	encoder := NewEncoder()
	result, err := encoder.EncodeToString(record)
	if err != nil {
		t.Fatalf("EncodeToString() failed for large data: %v", err)
	}

	if len(result) == 0 {
		t.Error("Result should not be empty for large data")
	}

	// Verify it can be decoded back
	decoder := NewDecoder()
	decoded, err := decoder.DecodeString(result)
	if err != nil {
		t.Fatalf("DecodeString() failed for large data: %v", err)
	}

	if len(decoded.Data) != len(largeData) {
		t.Errorf("Decoded data has %d fields, want %d", len(decoded.Data), len(largeData))
	}
}

// TestEncodeUnicodeData tests encoding Unicode characters
func TestEncodeUnicodeData(t *testing.T) {
	unicodeData := map[string]interface{}{
		"chinese":  "你好世界",
		"emoji":    "😀🎉🚀",
		"arabic":   "مرحبا",
		"japanese": "こんにちは",
	}

	record := NewPutRecord("unicode", 1, unicodeData)

	encoder := NewEncoder()
	result, err := encoder.EncodeToString(record)
	if err != nil {
		t.Fatalf("EncodeToString() failed for unicode: %v", err)
	}

	// Decode and verify
	decoder := NewDecoder()
	decoded, err := decoder.DecodeString(result)
	if err != nil {
		t.Fatalf("DecodeString() failed: %v", err)
	}

	if decoded.Data["chinese"] != unicodeData["chinese"] {
		t.Errorf("Unicode data mismatch: got %v, want %v", decoded.Data["chinese"], unicodeData["chinese"])
	}
	if decoded.Data["emoji"] != unicodeData["emoji"] {
		t.Errorf("Emoji data mismatch: got %v, want %v", decoded.Data["emoji"], unicodeData["emoji"])
	}
}

// TestEncodeEdgeCases tests various edge cases
func TestEncodeEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		record *Record
	}{
		{
			name: "record with empty string values",
			record: NewPutRecord("key", 1, map[string]interface{}{
				"empty": "",
			}),
		},
		{
			name: "record with zero numbers",
			record: NewPutRecord("key", 1, map[string]interface{}{
				"zero": 0.0,
			}),
		},
		{
			name: "record with false boolean",
			record: NewPutRecord("key", 1, map[string]interface{}{
				"bool": false,
			}),
		},
		{
			name: "record with null value",
			record: NewPutRecord("key", 1, map[string]interface{}{
				"null": nil,
			}),
		},
		{
			name: "record with empty array",
			record: NewPutRecord("key", 1, map[string]interface{}{
				"array": []interface{}{},
			}),
		},
		{
			name: "record with nested empty objects",
			record: NewPutRecord("key", 1, map[string]interface{}{
				"nested": map[string]interface{}{},
			}),
		},
	}

	encoder := NewEncoder()
	decoder := NewDecoder()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			result, err := encoder.EncodeToString(tt.record)
			if err != nil {
				t.Fatalf("EncodeToString() error = %v", err)
			}

			// Decode back
			decoded, err := decoder.DecodeString(result)
			if err != nil {
				t.Fatalf("DecodeString() error = %v", err)
			}

			// Should have same number of data fields
			if len(decoded.Data) != len(tt.record.Data) {
				t.Errorf("Data field count mismatch: got %d, want %d", len(decoded.Data), len(tt.record.Data))
			}
		})
	}
}

// TestEncodeDeleteRecord tests encoding delete records specifically
func TestEncodeDeleteRecord(t *testing.T) {
	deleteRecord := NewDeleteRecord("key-to-delete", 10)

	encoder := NewEncoder()
	result, err := encoder.EncodeToString(deleteRecord)
	if err != nil {
		t.Fatalf("EncodeToString() error = %v", err)
	}

	// Verify it contains delete operation
	if !strings.Contains(result, `"op":"delete"`) {
		t.Error("Should contain delete operation")
	}

	// Verify it can be decoded
	decoder := NewDecoder()
	decoded, err := decoder.DecodeString(result)
	if err != nil {
		t.Fatalf("DecodeString() error = %v", err)
	}

	if !decoded.Meta.IsDelete() {
		t.Error("Decoded record should be a delete operation")
	}
	if decoded.Meta.Version != 10 {
		t.Errorf("Version = %d, want %d", decoded.Meta.Version, 10)
	}
}

// TestEncodeConsistency tests that encoding the same record multiple times produces the same result
func TestEncodeConsistency(t *testing.T) {
	// Note: This test may fail if timestamps are regenerated on each encode
	// For this to work, the timestamp should be part of the record, not generated during encode
	meta := NewMeta("consistent-key", 1, OpPut)
	record := NewRecord(meta, map[string]interface{}{"value": "test"})

	encoder := NewEncoder()

	result1, err1 := encoder.EncodeToString(record)
	if err1 != nil {
		t.Fatalf("First encode error = %v", err1)
	}

	result2, err2 := encoder.EncodeToString(record)
	if err2 != nil {
		t.Fatalf("Second encode error = %v", err2)
	}

	// Results should be identical since we're encoding the same record
	if result1 != result2 {
		t.Error("Encoding the same record should produce identical results")
	}
}

// TestNewEncoder tests the NewEncoder constructor
func TestNewEncoder(t *testing.T) {
	encoder := NewEncoder()
	if encoder == nil {
		t.Fatal("NewEncoder() should not return nil")
	}
}

// TestEncodeNilData tests encoding with nil data field
func TestEncodeNilData(t *testing.T) {
	encoder := NewEncoder()

	// For delete records, nil data is valid
	deleteRecord := NewDeleteRecord("key", 1)
	_, err := encoder.Encode(deleteRecord)
	if err != nil {
		t.Errorf("Encode() should succeed for delete record with nil data: %v", err)
	}

	// For put records, nil data is invalid
	putRecord := &Record{
		Meta: NewMeta("key", 1, OpPut),
		Data: nil,
	}
	_, err = encoder.Encode(putRecord)
	if err == nil {
		t.Error("Encode() should fail for put record with nil data")
	}
}
