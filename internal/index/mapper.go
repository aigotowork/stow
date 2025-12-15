package index

import (
	"fmt"
	"sync"
)

// FileInfo holds information about a mapped file.
type FileInfo struct {
	FileName    string // The actual file name (e.g., "user_data_v1_abc123.jsonl")
	OriginalKey string // The original key before sanitization
}

// KeyMapper maps keys to file names, handling key collisions.
// It maintains a mapping from sanitized keys to actual file names.
//
// Structure: cleanKey -> []{fileName, originalKey}
//
// Example:
//   - "user_data_v1" -> [
//       {fileName: "user_data_v1.jsonl", originalKey: "user/data:v1"},
//       {fileName: "user_data_v1_abc123.jsonl", originalKey: "user_data:v1"}
//     ]
type KeyMapper struct {
	index map[string][]FileInfo
	mu    sync.RWMutex
}

// NewKeyMapper creates a new KeyMapper.
func NewKeyMapper() *KeyMapper {
	return &KeyMapper{
		index: make(map[string][]FileInfo),
	}
}

// Add adds a key-to-file mapping.
// If the mapping already exists, it updates it.
func (km *KeyMapper) Add(originalKey, fileName string) {
	km.mu.Lock()
	defer km.mu.Unlock()

	cleanKey := SanitizeKey(originalKey)

	// Check if this exact mapping already exists
	files := km.index[cleanKey]
	for i, info := range files {
		if info.OriginalKey == originalKey {
			// Update existing mapping
			km.index[cleanKey][i].FileName = fileName
			return
		}
	}

	// Add new mapping
	km.index[cleanKey] = append(km.index[cleanKey], FileInfo{
		FileName:    fileName,
		OriginalKey: originalKey,
	})
}

// Find finds candidate file names for a given key.
// Returns a list of FileInfo that might contain the key.
// The caller should read each file and check the _meta.k field to find the correct one.
func (km *KeyMapper) Find(key string) []FileInfo {
	km.mu.RLock()
	defer km.mu.RUnlock()

	cleanKey := SanitizeKey(key)
	files := km.index[cleanKey]

	// Return a copy to avoid race conditions
	result := make([]FileInfo, len(files))
	copy(result, files)

	return result
}

// FindExact finds the exact file name for a given key.
// Returns empty string if not found.
func (km *KeyMapper) FindExact(key string) string {
	km.mu.RLock()
	defer km.mu.RUnlock()

	cleanKey := SanitizeKey(key)
	files := km.index[cleanKey]

	for _, info := range files {
		if info.OriginalKey == key {
			return info.FileName
		}
	}

	return ""
}

// Remove removes a key from the index.
func (km *KeyMapper) Remove(key string) {
	km.mu.Lock()
	defer km.mu.Unlock()

	cleanKey := SanitizeKey(key)
	files := km.index[cleanKey]

	// Remove the entry with matching original key
	var newFiles []FileInfo
	for _, info := range files {
		if info.OriginalKey != key {
			newFiles = append(newFiles, info)
		}
	}

	if len(newFiles) == 0 {
		delete(km.index, cleanKey)
	} else {
		km.index[cleanKey] = newFiles
	}
}

// RemoveByFileName removes a mapping by file name.
func (km *KeyMapper) RemoveByFileName(fileName string) {
	km.mu.Lock()
	defer km.mu.Unlock()

	// Search through all entries
	for cleanKey, files := range km.index {
		var newFiles []FileInfo
		for _, info := range files {
			if info.FileName != fileName {
				newFiles = append(newFiles, info)
			}
		}

		if len(newFiles) == 0 {
			delete(km.index, cleanKey)
		} else if len(newFiles) != len(files) {
			km.index[cleanKey] = newFiles
		}
	}
}

// ListAll returns all keys in the mapper.
func (km *KeyMapper) ListAll() []string {
	km.mu.RLock()
	defer km.mu.RUnlock()

	var keys []string
	for _, files := range km.index {
		for _, info := range files {
			keys = append(keys, info.OriginalKey)
		}
	}

	return keys
}

// Count returns the total number of keys.
func (km *KeyMapper) Count() int {
	km.mu.RLock()
	defer km.mu.RUnlock()

	count := 0
	for _, files := range km.index {
		count += len(files)
	}

	return count
}

// Clear clears all mappings.
func (km *KeyMapper) Clear() {
	km.mu.Lock()
	defer km.mu.Unlock()

	km.index = make(map[string][]FileInfo)
}

// HasConflict checks if a clean key has multiple original keys mapped to it.
func (km *KeyMapper) HasConflict(key string) bool {
	km.mu.RLock()
	defer km.mu.RUnlock()

	cleanKey := SanitizeKey(key)
	return len(km.index[cleanKey]) > 1
}

// GetConflicts returns all keys that conflict with the given key.
func (km *KeyMapper) GetConflicts(key string) []string {
	km.mu.RLock()
	defer km.mu.RUnlock()

	cleanKey := SanitizeKey(key)
	files := km.index[cleanKey]

	var conflicts []string
	for _, info := range files {
		if info.OriginalKey != key {
			conflicts = append(conflicts, info.OriginalKey)
		}
	}

	return conflicts
}

// Stats returns statistics about the mapper.
func (km *KeyMapper) Stats() map[string]interface{} {
	km.mu.RLock()
	defer km.mu.RUnlock()

	conflictCount := 0
	for _, files := range km.index {
		if len(files) > 1 {
			conflictCount++
		}
	}

	return map[string]interface{}{
		"total_keys":      km.Count(),
		"unique_clean_keys": len(km.index),
		"conflicts":       conflictCount,
	}
}

// String returns a string representation of the mapper (for debugging).
func (km *KeyMapper) String() string {
	km.mu.RLock()
	defer km.mu.RUnlock()

	return fmt.Sprintf("KeyMapper{keys: %d, cleanKeys: %d}", km.Count(), len(km.index))
}
