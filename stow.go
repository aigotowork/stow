/*
Package stow provides an embedded transparent file-based KV storage engine.

Stow is positioned as a storage solution between plain JSON files and SQLite databases,
offering transparency (human-readable JSONL format), editability (supports external modifications),
and media-friendly features (smart blob storage for large files).

Quick Start:

	// Open or create a store
	store := stow.MustOpen("/data/myapp")
	defer store.Close()

	// Get or create a namespace
	ns := store.MustGetNamespace("config")

	// Store data
	ns.MustPut("server", map[string]interface{}{
		"host": "localhost",
		"port": 8080,
	})

	// Retrieve data
	var config map[string]interface{}
	ns.MustGet("server", &config)

Features:

- Transparent JSONL storage format
- Smart blob routing for large files
- Version history tracking
- Concurrent-safe operations
- Automatic compaction and garbage collection
- External file editability

For more information, see the README and design documentation.
*/
package stow

// Store is the main entry point for Stow.
// It manages multiple namespaces, each in its own directory.
//
// Example:
//
//	store := stow.MustOpen("/data")
//	defer store.Close()
type Store interface {
	// CreateNamespace creates a new namespace with the given configuration.
	// Returns ErrNamespaceExists if the namespace already exists.
	CreateNamespace(name string, config NamespaceConfig) (Namespace, error)

	// GetNamespace returns an existing namespace.
	// Creates it with default config if it doesn't exist.
	GetNamespace(name string) (Namespace, error)

	// MustGetNamespace is like GetNamespace but panics on error.
	MustGetNamespace(name string) Namespace

	// ListNamespaces returns all namespace names.
	ListNamespaces() ([]string, error)

	// DeleteNamespace deletes a namespace and all its data.
	// This is a destructive operation and cannot be undone.
	DeleteNamespace(name string) error

	// Close closes the store and all open namespaces.
	Close() error
}

// Namespace represents an isolated storage space with its own configuration.
// All KV operations are performed on a namespace.
type Namespace interface {
	// ========== Basic KV Operations ==========

	// Put stores a key-value pair.
	Put(key string, value interface{}, opts ...PutOption) error

	// MustPut is like Put but panics on error.
	MustPut(key string, value interface{}, opts ...PutOption)

	// Get retrieves a value by key and deserializes it into target.
	// Returns ErrNotFound if the key doesn't exist or has been deleted.
	Get(key string, target interface{}) error

	// MustGet is like Get but panics on error.
	MustGet(key string, target interface{})

	// GetRaw returns the raw record without deserialization.
	GetRaw(key string) (RawItem, error)

	// Delete marks a key as deleted (soft delete).
	Delete(key string) error

	// MustDelete is like Delete but panics on error.
	MustDelete(key string)

	// Exists checks if a key exists (and is not deleted).
	Exists(key string) bool

	// List returns all keys in the namespace (excluding deleted keys).
	List() ([]string, error)

	// ========== Version History ==========

	// GetHistory returns all versions of a key.
	GetHistory(key string) ([]Version, error)

	// GetVersion retrieves a specific version of a key.
	GetVersion(key string, version int, target interface{}) error

	// ========== Maintenance ==========

	// Compact compresses the specified keys by keeping only recent versions.
	// If no keys specified, does nothing (use CompactAll for all keys).
	Compact(keys ...string) error

	// CompactAsync asynchronously compresses the specified keys in the background.
	// Returns immediately without waiting for completion.
	// Use this for large-scale compaction that doesn't need to complete immediately.
	CompactAsync(keys ...string)

	// CompactAll compacts all keys in the namespace.
	CompactAll() error

	// CompactAllAsync asynchronously compacts all keys in the namespace.
	// Returns immediately without waiting for completion.
	CompactAllAsync()

	// GC performs garbage collection, removing unreferenced blob files.
	GC() (GCResult, error)

	// Refresh invalidates cache for specified keys, forcing reload from disk.
	// This allows detecting external file modifications.
	Refresh(keys ...string) error

	// RefreshAll invalidates cache for all keys.
	RefreshAll() error

	// ========== Configuration ==========

	// GetConfig returns the current namespace configuration.
	GetConfig() NamespaceConfig

	// SetConfig updates the namespace configuration.
	// Some settings take effect immediately, others require restart.
	SetConfig(config NamespaceConfig) error

	// ========== Fluent API ==========

	// WithLogger sets a custom logger for this namespace (returns self for chaining).
	WithLogger(logger Logger) Namespace

	// WithBlobThreshold sets the blob threshold (returns self for chaining).
	WithBlobThreshold(bytes int64) Namespace

	// WithMaxFileSize sets the max file size (returns self for chaining).
	WithMaxFileSize(bytes int64) Namespace

	// ========== Metadata ==========

	// Name returns the namespace name.
	Name() string

	// Path returns the absolute path to the namespace directory.
	Path() string

	// Stats returns statistics about the namespace.
	Stats() (NamespaceStats, error)
}

// Open opens or creates a store at the specified base path.
//
// Example:
//
//	store, err := stow.Open("/data/myapp")
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer store.Close()
func Open(basePath string, opts ...StoreOption) (Store, error) {
	return openStore(basePath, opts...)
}

// MustOpen is like Open but panics on error.
// Useful for initialization code where errors are unrecoverable.
//
// Example:
//
//	store := stow.MustOpen("/data/myapp")
//	defer store.Close()
func MustOpen(basePath string, opts ...StoreOption) Store {
	store, err := Open(basePath, opts...)
	if err != nil {
		panic(err)
	}
	return store
}
