package core

import (
	"encoding/json"
	"fmt"
)

// Encoder encodes Records to JSONL format.
type Encoder struct{}

// NewEncoder creates a new Encoder.
func NewEncoder() *Encoder {
	return &Encoder{}
}

// Encode encodes a Record to a single line of JSON.
// Returns the JSON bytes with a newline appended.
//
// Example output:
//
//	{"_meta":{"k":"key","v":1,"op":"put","ts":"2025-12-14T18:09:00Z"},"data":{"field":"value"}}\n
func (e *Encoder) Encode(record *Record) ([]byte, error) {
	if record == nil {
		return nil, fmt.Errorf("record is nil")
	}

	if !record.IsValid() {
		return nil, fmt.Errorf("invalid record")
	}

	// Marshal to JSON
	data, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal record: %w", err)
	}

	// Append newline
	data = append(data, '\n')

	return data, nil
}

// EncodeToString encodes a Record to a JSON string with newline.
func (e *Encoder) EncodeToString(record *Record) (string, error) {
	data, err := e.Encode(record)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
