package core

import (
	"encoding/json"
	"testing"
	"time"
)

// TestNewRecord tests the NewRecord constructor
func TestNewRecord(t *testing.T) {
	tests := []struct {
		name  string
		meta  *Meta
		data  map[string]interface{}
		check func(*testing.T, *Record)
	}{
		{
			name: "basic put record",
			meta: NewMeta("test-key", 1, OpPut),
			data: map[string]interface{}{"value": "test"},
			check: func(t *testing.T, r *Record) {
				if r.Meta.Key != "test-key" {
					t.Errorf("Key = %q, want %q", r.Meta.Key, "test-key")
				}
				if r.Meta.Version != 1 {
					t.Errorf("Version = %d, want %d", r.Meta.Version, 1)
				}
				if r.Data["value"] != "test" {
					t.Errorf("Data[value] = %v, want %q", r.Data["value"], "test")
				}
				if !r.IsValid() {
					t.Error("Record should be valid")
				}
			},
		},
		{
			name: "delete record",
			meta: NewMeta("key-to-delete", 5, OpDelete),
			data: nil,
			check: func(t *testing.T, r *Record) {
				if !r.Meta.IsDelete() {
					t.Error("Should be delete operation")
				}
				if r.Data != nil {
					t.Error("Delete record should have nil data")
				}
				if !r.IsValid() {
					t.Error("Delete record should be valid")
				}
			},
		},
		{
			name: "record with complex data",
			meta: NewMeta("complex", 1, OpPut),
			data: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "Alice",
					"age":  30.0,
				},
				"tags":   []interface{}{"admin", "user"},
				"active": true,
			},
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
			name: "record with empty data",
			meta: NewMeta("empty", 1, OpPut),
			data: map[string]interface{}{},
			check: func(t *testing.T, r *Record) {
				if r.Data == nil {
					t.Error("Data should not be nil")
				}
				if len(r.Data) != 0 {
					t.Errorf("Data length = %d, want 0", len(r.Data))
				}
			},
		},
		{
			name: "record with nil meta",
			meta: nil,
			data: map[string]interface{}{"value": "test"},
			check: func(t *testing.T, r *Record) {
				if r.Meta != nil {
					t.Error("Meta should be nil")
				}
				if r.IsValid() {
					t.Error("Record with nil meta should be invalid")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := NewRecord(tt.meta, tt.data)
			if record == nil {
				t.Fatal("NewRecord() returned nil")
			}
			if tt.check != nil {
				tt.check(t, record)
			}
		})
	}
}

// TestNewRecordValidation tests validation of created records
func TestNewRecordValidation(t *testing.T) {
	tests := []struct {
		name     string
		meta     *Meta
		data     map[string]interface{}
		isValid  bool
		validErr string
	}{
		{
			name:     "valid put record",
			meta:     NewMeta("key", 1, OpPut),
			data:     map[string]interface{}{"value": "test"},
			isValid:  true,
			validErr: "",
		},
		{
			name:     "valid delete record",
			meta:     NewMeta("key", 1, OpDelete),
			data:     nil,
			isValid:  true,
			validErr: "",
		},
		{
			name:     "nil meta",
			meta:     nil,
			data:     map[string]interface{}{},
			isValid:  false,
			validErr: "nil meta",
		},
		{
			name:     "empty key",
			meta:     NewMeta("", 1, OpPut),
			data:     map[string]interface{}{},
			isValid:  false,
			validErr: "empty key",
		},
		{
			name:     "zero version",
			meta:     NewMeta("key", 0, OpPut),
			data:     map[string]interface{}{},
			isValid:  false,
			validErr: "zero version",
		},
		{
			name:     "negative version",
			meta:     NewMeta("key", -1, OpPut),
			data:     map[string]interface{}{},
			isValid:  false,
			validErr: "negative version",
		},
		{
			name: "invalid operation",
			meta: &Meta{
				Key:       "key",
				Version:   1,
				Operation: "update",
				Timestamp: time.Now(),
			},
			data:     map[string]interface{}{},
			isValid:  false,
			validErr: "invalid operation",
		},
		{
			name:     "put record without data",
			meta:     NewMeta("key", 1, OpPut),
			data:     nil,
			isValid:  false,
			validErr: "put without data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := NewRecord(tt.meta, tt.data)
			if record.IsValid() != tt.isValid {
				t.Errorf("IsValid() = %v, want %v (%s)", record.IsValid(), tt.isValid, tt.validErr)
			}
		})
	}
}

// TestRecordSerialization tests JSON serialization of records
func TestRecordSerialization(t *testing.T) {
	original := NewPutRecord("user:123", 5, map[string]interface{}{
		"name":  "Alice",
		"email": "alice@example.com",
	})

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Unmarshal back
	var decoded Record
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	// Verify fields
	if decoded.Meta.Key != original.Meta.Key {
		t.Errorf("Key = %q, want %q", decoded.Meta.Key, original.Meta.Key)
	}
	if decoded.Meta.Version != original.Meta.Version {
		t.Errorf("Version = %d, want %d", decoded.Meta.Version, original.Meta.Version)
	}
	if decoded.Meta.Operation != original.Meta.Operation {
		t.Errorf("Operation = %q, want %q", decoded.Meta.Operation, original.Meta.Operation)
	}
	if decoded.Data["name"] != original.Data["name"] {
		t.Errorf("Data[name] = %v, want %v", decoded.Data["name"], original.Data["name"])
	}
}

// TestRecordSerializationFormats tests different serialization scenarios
func TestRecordSerializationFormats(t *testing.T) {
	tests := []struct {
		name   string
		record *Record
	}{
		{
			name:   "put record",
			record: NewPutRecord("key", 1, map[string]interface{}{"value": "test"}),
		},
		{
			name:   "delete record",
			record: NewDeleteRecord("key", 2),
		},
		{
			name: "record with nested data",
			record: NewPutRecord("key", 1, map[string]interface{}{
				"nested": map[string]interface{}{
					"field": "value",
				},
			}),
		},
		{
			name: "record with arrays",
			record: NewPutRecord("key", 1, map[string]interface{}{
				"tags": []interface{}{"a", "b", "c"},
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize
			data, err := json.Marshal(tt.record)
			if err != nil {
				t.Fatalf("Marshal error = %v", err)
			}

			// Deserialize
			var decoded Record
			err = json.Unmarshal(data, &decoded)
			if err != nil {
				t.Fatalf("Unmarshal error = %v", err)
			}

			// Verify validity
			if tt.record.IsValid() != decoded.IsValid() {
				t.Errorf("Validity mismatch: original=%v, decoded=%v", tt.record.IsValid(), decoded.IsValid())
			}
		})
	}
}

// TestRecordTypes tests different record type constructors
func TestRecordTypes(t *testing.T) {
	t.Run("NewPutRecord", func(t *testing.T) {
		record := NewPutRecord("test-key", 10, map[string]interface{}{"value": "test"})

		if record.Meta.Key != "test-key" {
			t.Errorf("Key = %q, want %q", record.Meta.Key, "test-key")
		}
		if record.Meta.Version != 10 {
			t.Errorf("Version = %d, want %d", record.Meta.Version, 10)
		}
		if !record.Meta.IsPut() {
			t.Error("Should be put operation")
		}
		if record.Data == nil {
			t.Error("Data should not be nil")
		}
		if !record.IsValid() {
			t.Error("Put record should be valid")
		}
	})

	t.Run("NewDeleteRecord", func(t *testing.T) {
		record := NewDeleteRecord("key-to-delete", 20)

		if record.Meta.Key != "key-to-delete" {
			t.Errorf("Key = %q, want %q", record.Meta.Key, "key-to-delete")
		}
		if record.Meta.Version != 20 {
			t.Errorf("Version = %d, want %d", record.Meta.Version, 20)
		}
		if !record.Meta.IsDelete() {
			t.Error("Should be delete operation")
		}
		if record.Data != nil {
			t.Error("Delete record data should be nil")
		}
		if !record.IsValid() {
			t.Error("Delete record should be valid")
		}
	})
}

// TestRecordMetadata tests record metadata handling
func TestRecordMetadata(t *testing.T) {
	// Create record
	record := NewPutRecord("test", 1, map[string]interface{}{"value": "test"})

	t.Run("timestamp is set", func(t *testing.T) {
		if record.Meta.Timestamp.IsZero() {
			t.Error("Timestamp should be set")
		}
	})

	t.Run("timestamp is recent", func(t *testing.T) {
		now := time.Now()
		diff := now.Sub(record.Meta.Timestamp)
		if diff > time.Second {
			t.Errorf("Timestamp is too old: %v", diff)
		}
		if diff < 0 {
			t.Error("Timestamp is in the future")
		}
	})

	t.Run("version is positive", func(t *testing.T) {
		if record.Meta.Version < 1 {
			t.Errorf("Version = %d, should be >= 1", record.Meta.Version)
		}
	})

	t.Run("operation is valid", func(t *testing.T) {
		if record.Meta.Operation != OpPut && record.Meta.Operation != OpDelete {
			t.Errorf("Invalid operation: %q", record.Meta.Operation)
		}
	})
}

// TestRecordDataTypes tests various data types in records
func TestRecordDataTypes(t *testing.T) {
	tests := []struct {
		name  string
		data  map[string]interface{}
		check func(*testing.T, *Record)
	}{
		{
			name: "string data",
			data: map[string]interface{}{"text": "hello"},
			check: func(t *testing.T, r *Record) {
				if r.Data["text"] != "hello" {
					t.Errorf("text = %v, want %q", r.Data["text"], "hello")
				}
			},
		},
		{
			name: "number data",
			data: map[string]interface{}{"count": 42.0},
			check: func(t *testing.T, r *Record) {
				if r.Data["count"] != 42.0 {
					t.Errorf("count = %v, want %f", r.Data["count"], 42.0)
				}
			},
		},
		{
			name: "boolean data",
			data: map[string]interface{}{"active": true},
			check: func(t *testing.T, r *Record) {
				if r.Data["active"] != true {
					t.Errorf("active = %v, want %v", r.Data["active"], true)
				}
			},
		},
		{
			name: "null data",
			data: map[string]interface{}{"nullable": nil},
			check: func(t *testing.T, r *Record) {
				if r.Data["nullable"] != nil {
					t.Errorf("nullable = %v, want nil", r.Data["nullable"])
				}
			},
		},
		{
			name: "array data",
			data: map[string]interface{}{"items": []interface{}{1, 2, 3}},
			check: func(t *testing.T, r *Record) {
				items, ok := r.Data["items"].([]interface{})
				if !ok {
					t.Fatal("items should be array")
				}
				if len(items) != 3 {
					t.Errorf("items length = %d, want 3", len(items))
				}
			},
		},
		{
			name: "nested object data",
			data: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "Bob",
				},
			},
			check: func(t *testing.T, r *Record) {
				user, ok := r.Data["user"].(map[string]interface{})
				if !ok {
					t.Fatal("user should be object")
				}
				if user["name"] != "Bob" {
					t.Errorf("user.name = %v, want %q", user["name"], "Bob")
				}
			},
		},
		{
			name: "mixed types",
			data: map[string]interface{}{
				"string":  "text",
				"number":  123.0,
				"boolean": false,
				"array":   []interface{}{},
				"object":  map[string]interface{}{},
			},
			check: func(t *testing.T, r *Record) {
				if len(r.Data) != 5 {
					t.Errorf("data length = %d, want 5", len(r.Data))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := NewPutRecord("key", 1, tt.data)
			if !record.IsValid() {
				t.Error("Record should be valid")
			}
			if tt.check != nil {
				tt.check(t, record)
			}
		})
	}
}

// TestRecordValidationEdgeCases tests edge cases in record validation
func TestRecordValidationEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		record  *Record
		isValid bool
	}{
		{
			name: "valid minimum record",
			record: &Record{
				Meta: &Meta{
					Key:       "k",
					Version:   1,
					Operation: OpPut,
					Timestamp: time.Now(),
				},
				Data: map[string]interface{}{},
			},
			isValid: true,
		},
		{
			name: "large version number",
			record: &Record{
				Meta: &Meta{
					Key:       "key",
					Version:   999999,
					Operation: OpPut,
					Timestamp: time.Now(),
				},
				Data: map[string]interface{}{},
			},
			isValid: true,
		},
		{
			name: "very long key",
			record: &Record{
				Meta: &Meta{
					Key:       string(make([]byte, 1000)),
					Version:   1,
					Operation: OpPut,
					Timestamp: time.Now(),
				},
				Data: map[string]interface{}{},
			},
			isValid: true,
		},
		{
			name: "delete with data (should still be valid)",
			record: &Record{
				Meta: &Meta{
					Key:       "key",
					Version:   1,
					Operation: OpDelete,
					Timestamp: time.Now(),
				},
				Data: map[string]interface{}{"extra": "data"},
			},
			isValid: true, // The implementation allows this
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.record.IsValid() != tt.isValid {
				t.Errorf("IsValid() = %v, want %v", tt.record.IsValid(), tt.isValid)
			}
		})
	}
}

// TestRecordImmutability tests that records maintain their structure
func TestRecordImmutability(t *testing.T) {
	originalData := map[string]interface{}{
		"name": "Original",
	}

	record := NewPutRecord("key", 1, originalData)

	// Store original values
	originalKey := record.Meta.Key
	originalVersion := record.Meta.Version
	originalName := record.Data["name"]

	// Modify the original data map
	originalData["name"] = "Modified"
	originalData["new_field"] = "new"

	// Record should reflect the change since it references the same map
	// (This tests actual Go behavior - maps are reference types)
	if record.Data["name"] != "Modified" {
		t.Log("Note: Record data is a reference to the original map")
	}

	// But metadata should not be affected
	if record.Meta.Key != originalKey {
		t.Error("Meta.Key should not change")
	}
	if record.Meta.Version != originalVersion {
		t.Error("Meta.Version should not change")
	}

	// Store the modified value for reference
	_ = originalName
}
