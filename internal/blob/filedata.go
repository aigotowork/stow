package blob

import (
	"fmt"
	"os"
)

// FileData implements the IFileData interface for streaming blob file access.
// It provides lazy loading - the file is only opened when Read() is called.
type FileData struct {
	path     string
	name     string
	size     int64
	mimeType string
	hash     string
	file     *os.File
}

// NewFileData creates a new FileData handle.
// The file is not opened until Read() is called.
func NewFileData(path, name string, size int64, mimeType, hash string) *FileData {
	return &FileData{
		path:     path,
		name:     name,
		size:     size,
		mimeType: mimeType,
		hash:     hash,
		file:     nil,
	}
}

// Read implements io.Reader.
// It lazily opens the file on the first Read() call.
func (f *FileData) Read(p []byte) (int, error) {
	if f.file == nil {
		file, err := os.Open(f.path)
		if err != nil {
			return 0, fmt.Errorf("failed to open blob file: %w", err)
		}
		f.file = file
	}

	n, err := f.file.Read(p)
	return n, err
}

// Close implements io.Closer.
// It closes the underlying file if it was opened.
func (f *FileData) Close() error {
	if f.file != nil {
		err := f.file.Close()
		f.file = nil
		return err
	}
	return nil
}

// Name returns the original file name.
func (f *FileData) Name() string {
	return f.name
}

// Size returns the file size in bytes.
func (f *FileData) Size() int64 {
	return f.size
}

// MimeType returns the MIME type.
func (f *FileData) MimeType() string {
	return f.mimeType
}

// Path returns the absolute path to the blob file.
func (f *FileData) Path() string {
	return f.path
}

// Hash returns the SHA256 hash of the file content.
func (f *FileData) Hash() string {
	return f.hash
}

// Ensure FileData implements the IFileData interface at compile time.
// This will be checked when we import stow package.
// For now, we just document the interface compatibility.
