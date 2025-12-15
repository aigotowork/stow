package index

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/aigotowork/stow/internal/core"
	"github.com/aigotowork/stow/internal/fsutil"
)

// Scanner scans a namespace directory and builds a KeyMapper.
type Scanner struct {
	decoder *core.Decoder
}

// NewScanner creates a new Scanner.
func NewScanner() *Scanner {
	return &Scanner{
		decoder: core.NewDecoder(),
	}
}

// ScanNamespace scans a namespace directory and returns a KeyMapper.
// It reads the first line of each .jsonl file to get the original key.
//
// Directory structure:
//
//	namespace/
//	  ├── key1.jsonl
//	  ├── key2_abc123.jsonl
//	  └── _blobs/
func (s *Scanner) ScanNamespace(namespacePath string) (*KeyMapper, error) {
	mapper := NewKeyMapper()

	// Find all .jsonl files
	files, err := fsutil.FindFiles(namespacePath, "*.jsonl")
	if err != nil {
		return nil, fmt.Errorf("failed to scan namespace: %w", err)
	}

	// Process each file
	for _, filePath := range files {
		// Skip files in _blobs directory
		if strings.Contains(filePath, "_blobs") {
			continue
		}

		// Read the original key from the first record
		originalKey, err := s.readKeyFromFile(filePath)
		if err != nil {
			// Skip files that can't be read or are invalid
			// In production, this should log a warning
			continue
		}

		// Add to mapper
		fileName := filepath.Base(filePath)
		mapper.Add(originalKey, fileName)
	}

	return mapper, nil
}

// readKeyFromFile reads the first record from a .jsonl file and returns the original key.
func (s *Scanner) readKeyFromFile(filePath string) (string, error) {
	// Read all records (we only need the first one, but ReadAll is simpler)
	records, err := s.decoder.ReadAll(filePath)
	if err != nil {
		return "", err
	}

	if len(records) == 0 {
		return "", fmt.Errorf("file is empty: %s", filePath)
	}

	// Get the key from the first record
	return records[0].Meta.Key, nil
}

// ScanAndValidate scans a namespace and validates the index.
// Returns the mapper and a list of issues found.
func (s *Scanner) ScanAndValidate(namespacePath string) (*KeyMapper, []string, error) {
	mapper, err := s.ScanNamespace(namespacePath)
	if err != nil {
		return nil, nil, err
	}

	var issues []string

	// Check for conflicts
	for cleanKey, files := range mapper.index {
		if len(files) > 1 {
			var keys []string
			for _, info := range files {
				keys = append(keys, info.OriginalKey)
			}
			issue := fmt.Sprintf("Key collision for '%s': %v", cleanKey, keys)
			issues = append(issues, issue)
		}
	}

	return mapper, issues, nil
}

// CountFiles counts the number of .jsonl files in a namespace.
func CountFiles(namespacePath string) (int, error) {
	files, err := fsutil.FindFiles(namespacePath, "*.jsonl")
	if err != nil {
		return 0, err
	}

	count := 0
	for _, file := range files {
		// Skip files in _blobs directory
		if !strings.Contains(file, "_blobs") {
			count++
		}
	}

	return count, nil
}

// ListKeys returns all keys in a namespace without building a full mapper.
// This is faster than ScanNamespace if you only need the key list.
func ListKeys(namespacePath string) ([]string, error) {
	scanner := NewScanner()
	files, err := fsutil.FindFiles(namespacePath, "*.jsonl")
	if err != nil {
		return nil, err
	}

	var keys []string
	for _, filePath := range files {
		// Skip files in _blobs directory
		if strings.Contains(filePath, "_blobs") {
			continue
		}

		key, err := scanner.readKeyFromFile(filePath)
		if err != nil {
			// Skip invalid files
			continue
		}

		keys = append(keys, key)
	}

	return keys, nil
}
