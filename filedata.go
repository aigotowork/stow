package stow

import "io"

// IFileData represents a file data handle for streaming large blobs.
// It provides streaming access to blob files without loading the entire
// content into memory.
//
// Users must call Close() when done to release resources.
//
// Example usage:
//
//	var user struct {
//	    Name   string
//	    Resume IFileData
//	}
//	ns.Get("alice", &user)
//	defer user.Resume.Close()
//	io.Copy(os.Stdout, user.Resume)
type IFileData interface {
	// ReadCloser provides streaming read access
	io.ReadCloser

	// Name returns the original file name (e.g., "resume.pdf")
	Name() string

	// Size returns the file size in bytes
	Size() int64

	// MimeType returns the MIME type (e.g., "application/pdf")
	MimeType() string

	// Path returns the absolute path to the blob file
	Path() string

	// Hash returns the SHA256 hash of the file content
	Hash() string
}
