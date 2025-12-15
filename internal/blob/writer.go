package blob

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"os"
)

// Writer is a chunked writer that writes data in chunks and computes hash simultaneously.
// It also enforces a maximum file size limit.
type Writer struct {
	file      *os.File
	hash      hash.Hash
	written   int64
	maxSize   int64
	chunkSize int64
}

// NewWriter creates a new chunked writer.
//
// Parameters:
//   - path: destination file path
//   - maxSize: maximum file size in bytes (0 for unlimited)
//   - chunkSize: chunk size for writing (typically 64KB)
func NewWriter(path string, maxSize, chunkSize int64) (*Writer, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}

	return &Writer{
		file:      f,
		hash:      sha256.New(),
		written:   0,
		maxSize:   maxSize,
		chunkSize: chunkSize,
	}, nil
}

// Write writes data to the file and updates the hash.
// It enforces the maximum file size limit.
func (w *Writer) Write(p []byte) (int, error) {
	// Check size limit
	if w.maxSize > 0 && w.written+int64(len(p)) > w.maxSize {
		return 0, fmt.Errorf("file size exceeds limit of %d bytes", w.maxSize)
	}

	// Write to file
	n, err := w.file.Write(p)
	if err != nil {
		return n, fmt.Errorf("failed to write to file: %w", err)
	}

	// Update hash
	w.hash.Write(p[:n])
	w.written += int64(n)

	return n, nil
}

// WriteFrom reads from a reader and writes to the file in chunks.
// This is more efficient than using io.Copy for large files.
func (w *Writer) WriteFrom(r io.Reader) error {
	buf := make([]byte, w.chunkSize)

	for {
		n, err := r.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("failed to read from source: %w", err)
		}
	}

	return nil
}

// Close closes the writer and returns the final hash.
// It syncs the file to disk before closing.
func (w *Writer) Close() (string, int64, error) {
	// Sync to disk
	if err := w.file.Sync(); err != nil {
		w.file.Close()
		return "", 0, fmt.Errorf("failed to sync file: %w", err)
	}

	// Close file
	if err := w.file.Close(); err != nil {
		return "", 0, fmt.Errorf("failed to close file: %w", err)
	}

	// Get final hash
	hashBytes := w.hash.Sum(nil)
	hashStr := fmt.Sprintf("%x", hashBytes)

	return hashStr, w.written, nil
}

// Abort closes the writer and removes the file.
// Used when an error occurs and we need to clean up.
func (w *Writer) Abort() error {
	path := w.file.Name()
	w.file.Close()
	return os.Remove(path)
}

// Written returns the number of bytes written so far.
func (w *Writer) Written() int64 {
	return w.written
}
