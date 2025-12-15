package fsutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WalkFunc is the type of the function called by Walk for each file or directory.
// The path argument contains the full path.
// If there's an error reading the directory, the error is passed to the function.
// The function can return filepath.SkipDir to skip the directory.
type WalkFunc func(path string, info os.FileInfo, err error) error

// Walk walks the file tree rooted at root, calling fn for each file or directory.
// This is a wrapper around filepath.Walk with some additional checks.
func Walk(root string, fn WalkFunc) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		return fn(path, info, err)
	})
}

// WalkFilesWithExt walks the directory and calls fn for each file with the specified extension.
// Extension should include the dot (e.g., ".txt", ".jsonl").
func WalkFilesWithExt(root string, ext string, fn func(path string) error) error {
	return Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check extension
		if filepath.Ext(path) == ext {
			return fn(path)
		}

		return nil
	})
}

// FindFiles finds all files matching a pattern in the directory tree.
// Pattern can contain wildcards like "*.jsonl".
// Returns absolute paths.
func FindFiles(root string, pattern string) ([]string, error) {
	var matches []string

	err := Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if filename matches pattern
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err != nil {
			return fmt.Errorf("invalid pattern: %w", err)
		}

		if matched {
			matches = append(matches, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return matches, nil
}

// DirSize calculates the total size of all files in a directory recursively.
func DirSize(root string) (int64, error) {
	var size int64

	err := Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			size += info.Size()
		}

		return nil
	})

	return size, err
}

// CountFiles counts the number of files (not directories) in a directory recursively.
func CountFiles(root string) (int, error) {
	var count int

	err := Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			count++
		}

		return nil
	})

	return count, err
}

// IsHidden checks if a file or directory is hidden.
// On Unix systems, files starting with "." are hidden.
func IsHidden(path string) bool {
	name := filepath.Base(path)
	return strings.HasPrefix(name, ".")
}

// FilterHidden filters out hidden files and directories from a list of paths.
func FilterHidden(paths []string) []string {
	var filtered []string
	for _, path := range paths {
		if !IsHidden(path) {
			filtered = append(filtered, path)
		}
	}
	return filtered
}
