package stow

// StoreOption is a function that configures a Store.
type StoreOption func(*storeOptions)

// storeOptions holds configuration options for opening a store.
type storeOptions struct {
	logger Logger
}

// WithStoreLogger sets a custom logger for the store.
func WithStoreLogger(logger Logger) StoreOption {
	return func(o *storeOptions) {
		o.logger = logger
	}
}

// PutOption is a function that configures a Put operation.
type PutOption func(*putOptions)

// putOptions holds options for Put operations.
type putOptions struct {
	forceFile   bool
	forceInline bool
	fileName    string
	mimeType    string
}

// WithForceFile forces the data to be stored as a file, even if it's small.
//
// Example:
//
//	ns.Put("key", smallData, WithForceFile())
func WithForceFile() PutOption {
	return func(o *putOptions) {
		o.forceFile = true
	}
}

// WithForceInline forces the data to be stored inline (in the JSONL file),
// even if it exceeds the blob threshold.
//
// Use this when you want to ensure data is always stored inline for faster
// access, at the cost of larger JSONL files.
//
// Example:
//
//	ns.Put("config", largeConfig, WithForceInline())
func WithForceInline() PutOption {
	return func(o *putOptions) {
		o.forceInline = true
	}
}

// WithFileName specifies a custom file name for blob storage.
// The actual file name will be "{name}_{hash}.{ext}".
//
// Example:
//
//	ns.Put("avatar", imageData, WithFileName("avatar.jpg"))
//	// Stored as: _blobs/avatar_abc123.jpg
func WithFileName(name string) PutOption {
	return func(o *putOptions) {
		o.fileName = name
	}
}

// WithMimeType specifies the MIME type for blob storage.
//
// Example:
//
//	ns.Put("doc", pdfData, WithMimeType("application/pdf"))
func WithMimeType(mime string) PutOption {
	return func(o *putOptions) {
		o.mimeType = mime
	}
}
