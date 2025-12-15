// Package index provides key indexing and caching functionality.
package index

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// Compile regex once for performance
	consecutiveUnderscores = regexp.MustCompile(`_+`)
)

// SanitizeKey sanitizes a key by removing invalid file name characters.
// Invalid characters are replaced with underscores.
// Consecutive underscores are compressed to a single underscore.
//
// Invalid characters: / \ : * ? " < > |
//
// Example:
//   - "user/data:v1" -> "user_data_v1"
//   - "file<name>" -> "file_name"
//   - "a//b::c" -> "a_b_c" (consecutive underscores compressed)
func SanitizeKey(key string) string {
	// List of invalid characters for file names
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}

	result := key
	for _, char := range invalidChars {
		result = strings.ReplaceAll(result, char, "_")
	}

	// Compress consecutive underscores to a single underscore
	result = consecutiveUnderscores.ReplaceAllString(result, "_")

	// Trim leading/trailing spaces and underscores
	result = strings.Trim(result, " _")

	// Ensure it's not empty
	if result == "" {
		result = "unnamed"
	}

	return result
}

// GenerateFileName generates a file name from a key.
// If addHash is true, appends a hash suffix to avoid collisions.
//
// Format:
//   - Without hash: {sanitized_key}.jsonl
//   - With hash: {sanitized_key}_{hash}.jsonl
func GenerateFileName(key string, addHash bool) string {
	sanitized := SanitizeKey(key)

	if !addHash {
		return sanitized + ".jsonl"
	}

	// Generate short hash of the original key
	hash := hashString(key)
	return fmt.Sprintf("%s_%s.jsonl", sanitized, hash)
}

// hashString generates a short hash of a string.
// Uses first 6 characters of SHA256 hash.
func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h[:3]) // 6 hex characters
}

// ExtractKeyFromFileName extracts the sanitized key portion from a file name.
// This doesn't return the original key, just the sanitized part.
//
// Example:
//   - "user_data_v1.jsonl" -> "user_data_v1"
//   - "user_data_v1_abc123.jsonl" -> "user_data_v1"
func ExtractKeyFromFileName(fileName string) string {
	// Remove .jsonl extension
	name := strings.TrimSuffix(fileName, ".jsonl")

	// Check if it has a hash suffix (pattern: _{6 hex chars})
	parts := strings.Split(name, "_")
	if len(parts) > 1 {
		lastPart := parts[len(parts)-1]
		// If last part looks like a hash (6 hex characters), remove it
		if len(lastPart) == 6 && isHexString(lastPart) {
			parts = parts[:len(parts)-1]
			name = strings.Join(parts, "_")
		}
	}

	return name
}

// isHexString checks if a string contains only hexadecimal characters.
func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// KeyConflict checks if two keys would generate the same file name after sanitization.
func KeyConflict(key1, key2 string) bool {
	return SanitizeKey(key1) == SanitizeKey(key2) && key1 != key2
}

// NeedsHashSuffix determines if a key needs a hash suffix to avoid conflicts.
// This is determined by checking if the sanitized key differs from the original.
func NeedsHashSuffix(key string) bool {
	// Check if sanitization changed the key
	sanitized := SanitizeKey(key)

	// Remove invalid chars from original and compare
	cleaned := key
	for _, char := range []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"} {
		cleaned = strings.ReplaceAll(cleaned, char, "")
	}

	cleaned = strings.Trim(cleaned, " _")

	return sanitized != cleaned
}

// IsValidKey checks if a key is valid (not empty and not too long).
func IsValidKey(key string) bool {
	if key == "" {
		return false
	}

	// Check length (max 255 characters for most file systems)
	// But leave room for .jsonl and hash suffix
	if len(key) > 200 {
		return false
	}

	return true
}

// CleanPath cleans a file path and returns just the base name.
func CleanPath(path string) string {
	return filepath.Base(path)
}
