package core

import (
	"encoding/json"
	"testing"
	"time"
)

// TestNewMeta tests the NewMeta constructor
func TestNewMeta(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		version   int
		operation string
		check     func(*testing.T, *Meta)
	}{
		{
			name:      "put operation",
			key:       "test-key",
			version:   1,
			operation: OpPut,
			check: func(t *testing.T, m *Meta) {
				if m.Key != "test-key" {
					t.Errorf("Key = %q, want %q", m.Key, "test-key")
				}
				if m.Version != 1 {
					t.Errorf("Version = %d, want %d", m.Version, 1)
				}
				if m.Operation != OpPut {
					t.Errorf("Operation = %q, want %q", m.Operation, OpPut)
				}
				if m.Timestamp.IsZero() {
					t.Error("Timestamp should not be zero")
				}
				if !m.IsPut() {
					t.Error("IsPut() should return true")
				}
				if m.IsDelete() {
					t.Error("IsDelete() should return false")
				}
			},
		},
		{
			name:      "delete operation",
			key:       "key-to-delete",
			version:   5,
			operation: OpDelete,
			check: func(t *testing.T, m *Meta) {
				if m.Operation != OpDelete {
					t.Errorf("Operation = %q, want %q", m.Operation, OpDelete)
				}
				if m.IsPut() {
					t.Error("IsPut() should return false")
				}
				if !m.IsDelete() {
					t.Error("IsDelete() should return true")
				}
			},
		},
		{
			name:      "high version number",
			key:       "key",
			version:   9999,
			operation: OpPut,
			check: func(t *testing.T, m *Meta) {
				if m.Version != 9999 {
					t.Errorf("Version = %d, want %d", m.Version, 9999)
				}
			},
		},
		{
			name:      "special characters in key",
			key:       "user:alice:profile",
			version:   1,
			operation: OpPut,
			check: func(t *testing.T, m *Meta) {
				if m.Key != "user:alice:profile" {
					t.Errorf("Key = %q, want %q", m.Key, "user:alice:profile")
				}
			},
		},
		{
			name:      "unicode key",
			key:       "用户:测试",
			version:   1,
			operation: OpPut,
			check: func(t *testing.T, m *Meta) {
				if m.Key != "用户:测试" {
					t.Errorf("Key = %q, want %q", m.Key, "用户:测试")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := NewMeta(tt.key, tt.version, tt.operation)
			if meta == nil {
				t.Fatal("NewMeta() returned nil")
			}
			if tt.check != nil {
				tt.check(t, meta)
			}
		})
	}
}

// TestMetaTimestamp tests timestamp functionality
func TestMetaTimestamp(t *testing.T) {
	before := time.Now().UTC()
	time.Sleep(1 * time.Millisecond)
	meta := NewMeta("key", 1, OpPut)
	time.Sleep(1 * time.Millisecond)
	after := time.Now().UTC()

	t.Run("timestamp is set", func(t *testing.T) {
		if meta.Timestamp.IsZero() {
			t.Error("Timestamp should not be zero")
		}
	})

	t.Run("timestamp is in UTC", func(t *testing.T) {
		if meta.Timestamp.Location() != time.UTC {
			t.Errorf("Timestamp should be in UTC, got %v", meta.Timestamp.Location())
		}
	})

	t.Run("timestamp is between before and after", func(t *testing.T) {
		if meta.Timestamp.Before(before) {
			t.Error("Timestamp is too early")
		}
		if meta.Timestamp.After(after) {
			t.Error("Timestamp is too late")
		}
	})
}

// TestMetaOperationChecksMethods tests IsPut and IsDelete methods comprehensively
func TestMetaOperationChecksMethods(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		isPut     bool
		isDelete  bool
	}{
		{
			name:      "put operation",
			operation: OpPut,
			isPut:     true,
			isDelete:  false,
		},
		{
			name:      "delete operation",
			operation: OpDelete,
			isPut:     false,
			isDelete:  true,
		},
		{
			name:      "invalid operation",
			operation: "invalid",
			isPut:     false,
			isDelete:  false,
		},
		{
			name:      "empty operation",
			operation: "",
			isPut:     false,
			isDelete:  false,
		},
		{
			name:      "case sensitive - PUT",
			operation: "PUT",
			isPut:     false,
			isDelete:  false,
		},
		{
			name:      "case sensitive - DELETE",
			operation: "DELETE",
			isPut:     false,
			isDelete:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &Meta{
				Key:       "key",
				Version:   1,
				Operation: tt.operation,
				Timestamp: time.Now(),
			}

			if meta.IsPut() != tt.isPut {
				t.Errorf("IsPut() = %v, want %v", meta.IsPut(), tt.isPut)
			}
			if meta.IsDelete() != tt.isDelete {
				t.Errorf("IsDelete() = %v, want %v", meta.IsDelete(), tt.isDelete)
			}
		})
	}
}

// TestMetaSerialization tests JSON serialization of Meta
func TestMetaSerialization(t *testing.T) {
	original := NewMeta("test-key", 5, OpPut)

	// Marshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Unmarshal
	var decoded Meta
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	// Verify all fields
	if decoded.Key != original.Key {
		t.Errorf("Key = %q, want %q", decoded.Key, original.Key)
	}
	if decoded.Version != original.Version {
		t.Errorf("Version = %d, want %d", decoded.Version, original.Version)
	}
	if decoded.Operation != original.Operation {
		t.Errorf("Operation = %q, want %q", decoded.Operation, original.Operation)
	}

	// Timestamps should be equal (within reasonable precision)
	if !decoded.Timestamp.Equal(original.Timestamp) {
		t.Errorf("Timestamp = %v, want %v", decoded.Timestamp, original.Timestamp)
	}
}

// TestMetaJSONFormat tests the JSON format of Meta
func TestMetaJSONFormat(t *testing.T) {
	meta := &Meta{
		Key:       "test-key",
		Version:   1,
		Operation: OpPut,
		Timestamp: time.Date(2025, 12, 14, 18, 9, 0, 0, time.UTC),
	}

	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	jsonStr := string(data)

	// Check for expected fields with short names
	expectedFields := []string{`"k":"test-key"`, `"v":1`, `"op":"put"`, `"ts":"`}
	for _, field := range expectedFields {
		if !containsSubstring(jsonStr, field) {
			t.Errorf("JSON should contain %q, got: %s", field, jsonStr)
		}
	}
}

// Helper function to check substring
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && contains(s, substr))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestMetaValidation tests validation scenarios
func TestMetaValidation(t *testing.T) {
	tests := []struct {
		name        string
		meta        *Meta
		shouldValid bool
		reason      string
	}{
		{
			name:        "valid put meta",
			meta:        NewMeta("key", 1, OpPut),
			shouldValid: true,
		},
		{
			name:        "valid delete meta",
			meta:        NewMeta("key", 1, OpDelete),
			shouldValid: true,
		},
		{
			name: "empty key",
			meta: &Meta{
				Key:       "",
				Version:   1,
				Operation: OpPut,
				Timestamp: time.Now(),
			},
			shouldValid: false,
			reason:      "empty key",
		},
		{
			name: "zero version",
			meta: &Meta{
				Key:       "key",
				Version:   0,
				Operation: OpPut,
				Timestamp: time.Now(),
			},
			shouldValid: false,
			reason:      "zero version",
		},
		{
			name: "negative version",
			meta: &Meta{
				Key:       "key",
				Version:   -1,
				Operation: OpPut,
				Timestamp: time.Now(),
			},
			shouldValid: false,
			reason:      "negative version",
		},
		{
			name: "invalid operation",
			meta: &Meta{
				Key:       "key",
				Version:   1,
				Operation: "update",
				Timestamp: time.Now(),
			},
			shouldValid: false,
			reason:      "invalid operation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a record with this meta to test validity
			var data map[string]interface{}
			if tt.meta.Operation == OpPut {
				data = map[string]interface{}{"value": "test"}
			}
			record := NewRecord(tt.meta, data)

			if record.IsValid() != tt.shouldValid {
				t.Errorf("Record.IsValid() = %v, want %v (%s)", record.IsValid(), tt.shouldValid, tt.reason)
			}
		})
	}
}

// TestMetaEdgeCases tests edge cases in Meta
func TestMetaEdgeCases(t *testing.T) {
	t.Run("very long key", func(t *testing.T) {
		longKey := string(make([]byte, 10000))
		meta := NewMeta(longKey, 1, OpPut)
		if meta.Key != longKey {
			t.Error("Should handle very long keys")
		}
	})

	t.Run("large version number", func(t *testing.T) {
		meta := NewMeta("key", 999999999, OpPut)
		if meta.Version != 999999999 {
			t.Error("Should handle large version numbers")
		}
	})

	t.Run("special characters in key", func(t *testing.T) {
		specialKey := "key:with:colons/and/slashes-and-dashes_and_underscores"
		meta := NewMeta(specialKey, 1, OpPut)
		if meta.Key != specialKey {
			t.Error("Should handle special characters")
		}
	})

	t.Run("unicode in key", func(t *testing.T) {
		unicodeKey := "用户:😀:测试"
		meta := NewMeta(unicodeKey, 1, OpPut)
		if meta.Key != unicodeKey {
			t.Error("Should handle unicode")
		}
	})
}

// TestMetaConstants tests operation constants
func TestMetaConstants(t *testing.T) {
	if OpPut != "put" {
		t.Errorf("OpPut = %q, want %q", OpPut, "put")
	}
	if OpDelete != "delete" {
		t.Errorf("OpDelete = %q, want %q", OpDelete, "delete")
	}
}

// TestMetaImmutability tests that operation checks don't modify meta
func TestMetaImmutability(t *testing.T) {
	meta := NewMeta("key", 1, OpPut)

	originalKey := meta.Key
	originalVersion := meta.Version
	originalOperation := meta.Operation
	originalTimestamp := meta.Timestamp

	// Call operation checks
	_ = meta.IsPut()
	_ = meta.IsDelete()

	// Verify nothing changed
	if meta.Key != originalKey {
		t.Error("IsPut/IsDelete modified Key")
	}
	if meta.Version != originalVersion {
		t.Error("IsPut/IsDelete modified Version")
	}
	if meta.Operation != originalOperation {
		t.Error("IsPut/IsDelete modified Operation")
	}
	if !meta.Timestamp.Equal(originalTimestamp) {
		t.Error("IsPut/IsDelete modified Timestamp")
	}
}

// TestMetaJSONRoundTrip tests encoding and decoding Meta
func TestMetaJSONRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		meta *Meta
	}{
		{
			name: "put meta",
			meta: NewMeta("test-key", 1, OpPut),
		},
		{
			name: "delete meta",
			meta: NewMeta("test-key", 5, OpDelete),
		},
		{
			name: "large version",
			meta: NewMeta("key", 999999, OpPut),
		},
		{
			name: "unicode key",
			meta: NewMeta("用户:测试", 1, OpPut),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			data, err := json.Marshal(tt.meta)
			if err != nil {
				t.Fatalf("Marshal error = %v", err)
			}

			// Decode
			var decoded Meta
			err = json.Unmarshal(data, &decoded)
			if err != nil {
				t.Fatalf("Unmarshal error = %v", err)
			}

			// Compare
			if decoded.Key != tt.meta.Key {
				t.Errorf("Key mismatch: got %q, want %q", decoded.Key, tt.meta.Key)
			}
			if decoded.Version != tt.meta.Version {
				t.Errorf("Version mismatch: got %d, want %d", decoded.Version, tt.meta.Version)
			}
			if decoded.Operation != tt.meta.Operation {
				t.Errorf("Operation mismatch: got %q, want %q", decoded.Operation, tt.meta.Operation)
			}
		})
	}
}

// TestMetaOperationTypes tests all operation types
func TestMetaOperationTypes(t *testing.T) {
	operations := []string{OpPut, OpDelete}

	for _, op := range operations {
		t.Run(op, func(t *testing.T) {
			meta := NewMeta("key", 1, op)
			if meta.Operation != op {
				t.Errorf("Operation = %q, want %q", meta.Operation, op)
			}

			switch op {
			case OpPut:
				if !meta.IsPut() {
					t.Error("IsPut() should be true for put operation")
				}
			case OpDelete:
				if !meta.IsDelete() {
					t.Error("IsDelete() should be true for delete operation")
				}
			}
		})
	}
}
