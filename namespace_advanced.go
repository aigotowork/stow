package stow

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aigotowork/stow/internal/blob"
	"github.com/aigotowork/stow/internal/core"
	"github.com/aigotowork/stow/internal/fsutil"
)

// GetHistory returns all versions of a key.
func (ns *namespace) GetHistory(key string) ([]Version, error) {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	// Get file path
	filePath, err := ns.getFilePath(key, false)
	if err != nil {
		return nil, err
	}

	// Read all records
	records, err := ns.decoder.ReadAll(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read records: %w", err)
	}

	// Convert to Version list
	var versions []Version
	for _, record := range records {
		versions = append(versions, Version{
			Version:   record.Meta.Version,
			Timestamp: record.Meta.Timestamp,
			Operation: record.Meta.Operation,
			Size:      calculateRecordSize(record),
		})
	}

	// Reverse to get newest first
	for i, j := 0, len(versions)-1; i < j; i, j = i+1, j-1 {
		versions[i], versions[j] = versions[j], versions[i]
	}

	return versions, nil
}

// GetVersion retrieves a specific version.
func (ns *namespace) GetVersion(key string, version int, target interface{}) error {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	// Get file path
	filePath, err := ns.getFilePath(key, false)
	if err != nil {
		return err
	}

	// Read specific version
	record, err := ns.decoder.ReadVersion(filePath, version)
	if err != nil {
		return fmt.Errorf("failed to read version: %w", err)
	}

	if record.Meta.IsDelete() {
		return fmt.Errorf("version %d is a delete operation", version)
	}

	// Unmarshal into target
	return ns.unmarshaler.Unmarshal(record.Data, target)
}

// Compact compresses specified keys.
func (ns *namespace) Compact(keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	ns.mu.Lock()
	defer ns.mu.Unlock()

	for _, key := range keys {
		if err := ns.compactKey(key); err != nil {
			ns.logger.Error("failed to compact key", Field{"key", key}, Field{"error", err})
			// Continue with other keys
		}
	}

	return nil
}

// CompactAsync asynchronously compresses specified keys in the background.
// This method returns immediately and does not block.
// Use this for large-scale compaction operations that don't need to complete immediately.
func (ns *namespace) CompactAsync(keys ...string) {
	if len(keys) == 0 {
		return
	}

	go func() {
		for _, key := range keys {
			ns.compactKeySafe(key)
		}
	}()
}

// CompactAllAsync asynchronously compacts all keys in the namespace.
// This method returns immediately and does not block.
func (ns *namespace) CompactAllAsync() {
	go func() {
		ns.mu.RLock()
		allKeys := ns.keyMapper.ListAll()
		ns.mu.RUnlock()

		for _, key := range allKeys {
			ns.compactKeySafe(key)
		}
	}()
}

// CompactAll compacts all keys in the namespace.
func (ns *namespace) CompactAll() error {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	allKeys := ns.keyMapper.ListAll()

	for _, key := range allKeys {
		if err := ns.compactKey(key); err != nil {
			ns.logger.Error("failed to compact key", Field{"key", key}, Field{"error", err})
			// Continue with other keys
		}
	}

	return nil
}

// compactKey compacts a single key (caller must hold lock).
func (ns *namespace) compactKey(key string) error {
	// Get file path
	filePath, err := ns.getFilePath(key, false)
	if err != nil {
		return err
	}

	// Read last N records
	records, err := ns.decoder.ReadLastNRecords(filePath, ns.config.CompactKeepRecords)
	if err != nil {
		return fmt.Errorf("failed to read records: %w", err)
	}

	if len(records) == 0 {
		return nil
	}

	// Write to temporary file
	tmpPath := filePath + ".tmp"

	// Create temp file
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	// Write kept records
	for _, record := range records {
		data, err := ns.encoder.Encode(record)
		if err != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
			return fmt.Errorf("failed to encode record: %w", err)
		}

		if _, err := tmpFile.Write(data); err != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
			return fmt.Errorf("failed to write to temp file: %w", err)
		}
	}

	// Sync and close
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename
	if err := fsutil.SafeRename(tmpPath, filePath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	// Clear cache for this key
	ns.cache.Delete(key)

	return nil
}

// compactKeySafe compacts a single key using key-level locking (safe for async operations).
func (ns *namespace) compactKeySafe(key string) {
	// Acquire key-level lock
	keyLock := ns.getKeyLock(key)
	keyLock.Lock()
	defer keyLock.Unlock()

	// Get file path (need read lock for keyMapper)
	ns.mu.RLock()
	filePath, err := ns.getFilePath(key, false)
	ns.mu.RUnlock()
	if err != nil {
		ns.logger.Warn("failed to get file path for compact", Field{"key", key}, Field{"error", err})
		return
	}

	// Read last N records
	records, err := ns.decoder.ReadLastNRecords(filePath, ns.config.CompactKeepRecords)
	if err != nil {
		ns.logger.Error("failed to read records for compact", Field{"key", key}, Field{"error", err})
		return
	}

	if len(records) == 0 {
		return
	}

	// Write to temporary file
	tmpPath := filePath + ".tmp"

	// Create temp file
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		ns.logger.Error("failed to create temp file for compact", Field{"key", key}, Field{"error", err})
		return
	}

	// Write kept records
	for _, record := range records {
		data, err := ns.encoder.Encode(record)
		if err != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
			ns.logger.Error("failed to encode record for compact", Field{"key", key}, Field{"error", err})
			return
		}

		if _, err := tmpFile.Write(data); err != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
			ns.logger.Error("failed to write to temp file for compact", Field{"key", key}, Field{"error", err})
			return
		}
	}

	// Sync and close
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		ns.logger.Error("failed to sync temp file for compact", Field{"key", key}, Field{"error", err})
		return
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		ns.logger.Error("failed to close temp file for compact", Field{"key", key}, Field{"error", err})
		return
	}

	// Atomic rename
	if err := fsutil.SafeRename(tmpPath, filePath); err != nil {
		ns.logger.Error("failed to rename temp file for compact", Field{"key", key}, Field{"error", err})
		return
	}

	// Clear cache for this key
	ns.cache.Delete(key)

	ns.logger.Info("key compacted successfully", Field{"key", key}, Field{"records_kept", len(records)})
}

// GC performs garbage collection on blob files using streaming to minimize memory usage.
func (ns *namespace) GC() (GCResult, error) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	startTime := time.Now()

	// Collect all blob references from JSONL files (streaming mode)
	referencedBlobs := make(map[string]bool)

	files, err := fsutil.FindFiles(ns.path, "*.jsonl")
	if err != nil {
		return GCResult{}, fmt.Errorf("failed to find JSONL files: %w", err)
	}

	for _, filePath := range files {
		// Skip files in _blobs directory
		if strings.Contains(filePath, "_blobs") {
			continue
		}

		// Stream through the file line by line
		if err := ns.streamBlobRefs(filePath, referencedBlobs); err != nil {
			continue // Skip files that can't be read
		}
	}

	// Find all blob files
	allBlobs, err := ns.blobManager.ListAll()
	if err != nil {
		return GCResult{}, fmt.Errorf("failed to list blobs: %w", err)
	}

	// Find unreferenced blobs
	var removed int
	var reclaimedSize int64

	for _, blobPath := range allBlobs {
		blobName := filepath.Base(blobPath)
		relativePath := filepath.Join("_blobs", blobName)

		if !referencedBlobs[relativePath] {
			// This blob is not referenced, delete it
			size := fsutil.FileSize(blobPath)
			if err := os.Remove(blobPath); err != nil {
				ns.logger.Warn("failed to remove blob", Field{"path", blobPath}, Field{"error", err})
				continue
			}

			removed++
			reclaimedSize += size
		}
	}

	duration := time.Since(startTime)

	return GCResult{
		RemovedBlobs:  removed,
		ReclaimedSize: reclaimedSize,
		Duration:      duration,
	}, nil
}

// streamBlobRefs streams through a JSONL file and extracts blob references without loading all data.
// Only collects references from the MOST RECENT non-deleted record for each key.
func (ns *namespace) streamBlobRefs(filePath string, refs map[string]bool) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Map to store the latest record for each key
	latestRecords := make(map[string]*core.Record)

	// Use bufio.Scanner for line-by-line JSONL reading
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue // Skip empty lines
		}

		// Decode one record at a time
		var record core.Record
		if err := json.Unmarshal(line, &record); err != nil {
			// Skip invalid lines but continue
			continue
		}

		// Store the latest record for this key
		key := record.Meta.Key
		if existing, ok := latestRecords[key]; !ok || record.Meta.Version > existing.Meta.Version {
			latestRecords[key] = &record
		}
	}

	// Now collect blob refs only from the latest non-deleted records
	for _, record := range latestRecords {
		if !record.Meta.IsDelete() {
			collectBlobRefs(record.Data, refs)
		}
	}

	return scanner.Err()
}

// Refresh invalidates cache for specified keys.
func (ns *namespace) Refresh(keys ...string) error {
	ns.cache.DeleteMultiple(keys)
	return nil
}

// RefreshAll invalidates cache for all keys.
func (ns *namespace) RefreshAll() error {
	ns.cache.Clear()
	return nil
}

// Stats returns namespace statistics.
func (ns *namespace) Stats() (NamespaceStats, error) {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	stats := NamespaceStats{
		KeyCount:  ns.keyMapper.Count(),
		BlobCount: 0,
		TotalSize: 0,
		BlobSize:  0,
	}

	// Count blobs
	blobCount, err := ns.blobManager.Count()
	if err == nil {
		stats.BlobCount = blobCount
	}

	// Calculate sizes
	dirSize, err := fsutil.DirSize(ns.path)
	if err == nil {
		stats.TotalSize = dirSize
	}

	blobSize, err := ns.blobManager.TotalSize()
	if err == nil {
		stats.BlobSize = blobSize
	}

	return stats, nil
}

// Helper functions

// calculateRecordSize estimates the size of a record.
func calculateRecordSize(record *core.Record) int64 {
	// Rough estimate based on JSON serialization
	data, err := json.Marshal(record)
	if err != nil {
		return 0
	}
	return int64(len(data))
}

// collectBlobRefs collects all blob references from a data map.
func collectBlobRefs(data map[string]interface{}, refs map[string]bool) {
	for _, value := range data {
		switch v := value.(type) {
		case map[string]interface{}:
			// Check if it's a blob reference
			if ref, ok := blob.FromMap(v); ok {
				refs[ref.Location] = true
			} else {
				// Recursively check nested maps
				collectBlobRefs(v, refs)
			}
		}
	}
}
