package fsutil

import (
	"fmt"
	"os"
	"path/filepath"
)

// EnsureDir ensures that a directory exists. Creates it if it doesn't exist.
// Creates parent directories as needed (like mkdir -p).
func EnsureDir(path string, perm os.FileMode) error {
	// Check if directory already exists
	info, err := os.Stat(path)
	if err == nil {
		// Path exists, check if it's a directory
		if !info.IsDir() {
			return fmt.Errorf("path exists but is not a directory: %s", path)
		}
		return nil
	}

	// Path doesn't exist or error occurred
	if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat directory: %w", err)
	}

	// Create directory with parents
	if err := os.MkdirAll(path, perm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return nil
}

// FileExists checks if a file exists and is not a directory.
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// DirExists checks if a directory exists.
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// FileSize returns the size of a file in bytes.
// Returns 0 if the file doesn't exist or is a directory.
func FileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return 0
	}
	return info.Size()
}

// RemoveAll removes a path and all its contents, like rm -rf.
// It doesn't return an error if the path doesn't exist.
func RemoveAll(path string) error {
	err := os.RemoveAll(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove: %w", err)
	}
	return nil
}

// ListFiles returns all regular files in a directory (non-recursive).
// Directories and symlinks are excluded.
func ListFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}

	return files, nil
}

// ListDirs returns all subdirectories in a directory (non-recursive).
// Files are excluded.
func ListDirs(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, filepath.Join(dir, entry.Name()))
		}
	}

	return dirs, nil
}

// CleanPath cleans and normalizes a file path.
func CleanPath(path string) string {
	return filepath.Clean(path)
}

// AbsPath returns the absolute path, resolving any relative components.
func AbsPath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}
	return absPath, nil
}
