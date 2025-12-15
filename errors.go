// Package stow provides an embedded transparent file-based KV storage engine.
package stow

import "errors"

// Common errors returned by Stow operations.
var (
	// ErrNotFound is returned when a key is not found in the namespace.
	ErrNotFound = errors.New("key not found")

	// ErrKeyConflict is returned when key sanitization results in a conflict.
	ErrKeyConflict = errors.New("key conflict after sanitization")

	// ErrFileTooLarge is returned when a file exceeds the MaxFileSize limit.
	ErrFileTooLarge = errors.New("file exceeds MaxFileSize limit")

	// ErrDiskFull is returned when there is insufficient disk space.
	ErrDiskFull = errors.New("disk space insufficient")

	// ErrPermissionDenied is returned when permission is denied for file operations.
	ErrPermissionDenied = errors.New("permission denied")

	// ErrInvalidConfig is returned when configuration validation fails.
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrNamespaceNotFound is returned when a namespace does not exist.
	ErrNamespaceNotFound = errors.New("namespace not found")

	// ErrNamespaceExists is returned when attempting to create an existing namespace.
	ErrNamespaceExists = errors.New("namespace already exists")

	// ErrCorruptedData is returned when data is corrupted or cannot be parsed.
	ErrCorruptedData = errors.New("data corrupted")

	// ErrLockTimeout is returned when lock acquisition times out.
	ErrLockTimeout = errors.New("lock acquisition timeout")
)
