package stow

import "time"

// Version represents a single version record of a key.
type Version struct {
	// Version number (incremental)
	Version int `json:"version"`

	// Timestamp of this version
	Timestamp time.Time `json:"timestamp"`

	// Operation type: "put" or "delete"
	Operation string `json:"operation"`

	// Size of the data in bytes (0 for delete operations)
	Size int64 `json:"size"`
}

// MetaInfo contains metadata for a record.
type MetaInfo struct {
	// Original key (before sanitization)
	Key string `json:"k"`

	// Version number
	Version int `json:"v"`

	// Operation: "put" or "delete"
	Operation string `json:"op"`

	// Timestamp when this record was created
	Timestamp time.Time `json:"ts"`
}

// NamespaceStats contains statistics about a namespace.
type NamespaceStats struct {
	// Number of keys in the namespace
	KeyCount int `json:"key_count"`

	// Number of blob files
	BlobCount int `json:"blob_count"`

	// Total size of all data (JSONL + blobs) in bytes
	TotalSize int64 `json:"total_size"`

	// Total size of blob files in bytes
	BlobSize int64 `json:"blob_size"`

	// Last compaction time
	LastCompactAt time.Time `json:"last_compact_at,omitempty"`

	// Last garbage collection time
	LastGCAt time.Time `json:"last_gc_at,omitempty"`
}

// GCResult contains the result of a garbage collection operation.
type GCResult struct {
	// Number of blob files removed
	RemovedBlobs int `json:"removed_blobs"`

	// Total size reclaimed in bytes
	ReclaimedSize int64 `json:"reclaimed_size"`

	// Duration of the GC operation
	Duration time.Duration `json:"duration"`
}

// CompactStrategy defines when to trigger compaction.
type CompactStrategy string

const (
	// CompactStrategyLineCount triggers compaction when line count exceeds threshold
	CompactStrategyLineCount CompactStrategy = "line_count"

	// CompactStrategyFileSize triggers compaction when file size exceeds threshold
	CompactStrategyFileSize CompactStrategy = "file_size"

	// CompactStrategyManual disables automatic compaction
	CompactStrategyManual CompactStrategy = "manual"
)

// Field represents a structured logging field.
type Field struct {
	Key   string
	Value interface{}
}

// RawItem represents a raw data item without deserialization.
type RawItem interface {
	// Meta returns the metadata of this item
	Meta() MetaInfo

	// DecodeInto decodes the data into the target
	DecodeInto(target interface{}) error

	// RawData returns the raw data as a map
	RawData() map[string]interface{}
}
