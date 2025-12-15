package stow

import "time"

// NamespaceConfig holds configuration for a namespace.
type NamespaceConfig struct {
	// BlobThreshold is the size threshold (in bytes) above which data is stored as a blob file.
	// Default: 4KB
	BlobThreshold int64 `json:"blob_threshold"`

	// MaxFileSize is the maximum size (in bytes) for a single blob file.
	// Default: 100MB
	MaxFileSize int64 `json:"max_file_size"`

	// BlobChunkSize is the chunk size (in bytes) for writing blob files.
	// Default: 64KB
	BlobChunkSize int64 `json:"blob_chunk_size"`

	// CacheTTL is the time-to-live for cached data.
	// Default: 5 minutes
	CacheTTL time.Duration `json:"cache_ttl"`

	// CacheTTLJitter is the random jitter factor for cache TTL (0.0-1.0).
	// Actual TTL = CacheTTL * (1 ± jitter)
	// This helps avoid thundering herd when many keys expire simultaneously.
	// Default: 0.2 (±20%)
	CacheTTLJitter float64 `json:"cache_ttl_jitter"`

	// DisableCache disables the in-memory cache entirely.
	// Every Get() will read from disk.
	// Default: false
	DisableCache bool `json:"disable_cache"`

	// CompactStrategy determines when to trigger compaction.
	// Default: CompactStrategyLineCount
	CompactStrategy CompactStrategy `json:"compact_strategy"`

	// CompactThreshold is the threshold for triggering compaction.
	// - For LineCount strategy: number of lines
	// - For FileSize strategy: file size in bytes
	// Default: 20 lines
	CompactThreshold int `json:"compact_threshold"`

	// CompactKeepRecords is the number of recent records to keep after compaction.
	// Default: 3
	CompactKeepRecords int `json:"compact_keep_records"`

	// AutoCompact enables automatic compaction after Put operations.
	// Default: true
	AutoCompact bool `json:"auto_compact"`

	// LockTimeout is the timeout for acquiring locks.
	// Default: 30 seconds
	LockTimeout time.Duration `json:"lock_timeout"`
}

// DefaultNamespaceConfig returns the default configuration for a namespace.
func DefaultNamespaceConfig() NamespaceConfig {
	return NamespaceConfig{
		BlobThreshold:      4 * 1024,         // 4KB
		MaxFileSize:        100 * 1024 * 1024, // 100MB
		BlobChunkSize:      64 * 1024,        // 64KB
		CacheTTL:           5 * time.Minute,
		CacheTTLJitter:     0.2,
		DisableCache:       false,
		CompactStrategy:    CompactStrategyLineCount,
		CompactThreshold:   20,
		CompactKeepRecords: 3,
		AutoCompact:        true,
		LockTimeout:        30 * time.Second,
	}
}

// Validate checks if the configuration is valid.
func (c *NamespaceConfig) Validate() error {
	if c.BlobThreshold < 0 {
		return ErrInvalidConfig
	}
	if c.MaxFileSize <= 0 {
		return ErrInvalidConfig
	}
	if c.BlobChunkSize <= 0 {
		return ErrInvalidConfig
	}
	if c.CacheTTL < 0 {
		return ErrInvalidConfig
	}
	if c.CacheTTLJitter < 0 || c.CacheTTLJitter > 1 {
		return ErrInvalidConfig
	}
	if c.CompactThreshold < 0 {
		return ErrInvalidConfig
	}
	if c.CompactKeepRecords < 1 {
		return ErrInvalidConfig
	}
	if c.LockTimeout <= 0 {
		return ErrInvalidConfig
	}
	return nil
}
