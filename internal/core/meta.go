// Package core provides core data structures for JSONL records.
package core

import "time"

// Meta represents metadata for a JSONL record.
// It contains information about the key, version, operation, and timestamp.
type Meta struct {
	// Key is the original key (before sanitization)
	Key string `json:"k"`

	// Version is the incremental version number
	Version int `json:"v"`

	// Operation is either "put" or "delete"
	Operation string `json:"op"`

	// Timestamp is when this record was created
	Timestamp time.Time `json:"ts"`
}

// Operation types
const (
	OpPut    = "put"
	OpDelete = "delete"
)

// NewMeta creates a new Meta with the given parameters.
func NewMeta(key string, version int, operation string) *Meta {
	return &Meta{
		Key:       key,
		Version:   version,
		Operation: operation,
		Timestamp: time.Now().UTC(),
	}
}

// IsPut returns true if this is a put operation.
func (m *Meta) IsPut() bool {
	return m.Operation == OpPut
}

// IsDelete returns true if this is a delete operation.
func (m *Meta) IsDelete() bool {
	return m.Operation == OpDelete
}
