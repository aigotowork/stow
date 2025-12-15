package core

// Record represents a single JSONL record.
// Each record consists of metadata (_meta) and the actual data.
//
// Example JSON structure:
//
//	{
//	  "_meta": {"k": "user:alice", "v": 1, "op": "put", "ts": "2025-12-14T18:09:00Z"},
//	  "data": {"name": "Alice", "age": 30}
//	}
type Record struct {
	// Meta contains metadata about this record
	Meta *Meta `json:"_meta"`

	// Data contains the actual payload
	// For "put" operations, this is the data
	// For "delete" operations, this is nil
	Data map[string]interface{} `json:"data"`
}

// NewRecord creates a new Record with the given metadata and data.
func NewRecord(meta *Meta, data map[string]interface{}) *Record {
	return &Record{
		Meta: meta,
		Data: data,
	}
}

// NewPutRecord creates a new put record.
func NewPutRecord(key string, version int, data map[string]interface{}) *Record {
	return &Record{
		Meta: NewMeta(key, version, OpPut),
		Data: data,
	}
}

// NewDeleteRecord creates a new delete record.
func NewDeleteRecord(key string, version int) *Record {
	return &Record{
		Meta: NewMeta(key, version, OpDelete),
		Data: nil,
	}
}

// IsValid checks if the record is valid.
func (r *Record) IsValid() bool {
	if r.Meta == nil {
		return false
	}

	// Key must not be empty
	if r.Meta.Key == "" {
		return false
	}

	// Version must be positive
	if r.Meta.Version < 1 {
		return false
	}

	// Operation must be put or delete
	if r.Meta.Operation != OpPut && r.Meta.Operation != OpDelete {
		return false
	}

	// Put operations must have data
	if r.Meta.IsPut() && r.Data == nil {
		return false
	}

	// Delete operations should not have data (but we'll allow it)

	return true
}
