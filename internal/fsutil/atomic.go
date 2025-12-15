// Package fsutil provides file system utilities for safe and atomic file operations.
package fsutil

import (
	"fmt"
	"os"
	"path/filepath"
)

// AtomicWriteFile writes data to a file atomically.
// It writes to a temporary file first, syncs it to disk, then renames it to the target path.
// This ensures that the file is either fully written or not written at all, even if the process crashes.
//
// Steps:
// 1. Write to {path}.tmp
// 2. Sync to disk
// 3. Rename to {path}
// 4. Sync parent directory
func AtomicWriteFile(path string, data []byte, perm os.FileMode) error {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := EnsureDir(dir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Create temporary file in the same directory
	tmpPath := path + ".tmp"

	// Write to temporary file
	f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	// Write data
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmpPath) // Clean up temp file
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	// Sync to disk
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	// Close the file
	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Rename to target path (atomic operation)
	if err := SafeRename(tmpPath, path); err != nil {
		os.Remove(tmpPath) // Clean up temp file
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	// Sync parent directory to ensure rename is persisted
	if err := syncDir(dir); err != nil {
		// Log warning but don't fail - the file is already renamed
		// This is a best-effort operation
		return nil
	}

	return nil
}

// SafeRename renames a file safely.
// On Unix systems, os.Rename is atomic if src and dst are on the same filesystem.
func SafeRename(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}

// syncDir syncs a directory to disk.
// This ensures that directory metadata (like new file entries) is persisted.
func syncDir(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer f.Close()

	return f.Sync()
}
