package stow

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/aigotowork/stow/internal/fsutil"
)

// store implements the Store interface.
type store struct {
	basePath   string
	namespaces map[string]*namespace
	mu         sync.RWMutex
	logger     Logger
}

// openStore opens or creates a store.
func openStore(basePath string, opts ...StoreOption) (Store, error) {
	// Apply options
	options := &storeOptions{
		logger: NewDefaultLogger(),
	}

	for _, opt := range opts {
		opt(options)
	}

	// Convert to absolute path
	absPath, err := fsutil.AbsPath(basePath)
	if err != nil {
		return nil, fmt.Errorf("invalid base path: %w", err)
	}

	// Ensure base directory exists
	if err := fsutil.EnsureDir(absPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	s := &store{
		basePath:   absPath,
		namespaces: make(map[string]*namespace),
		logger:     options.logger,
	}

	return s, nil
}

// CreateNamespace creates a new namespace.
func (s *store) CreateNamespace(name string, config NamespaceConfig) (Namespace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if already exists in memory
	if _, exists := s.namespaces[name]; exists {
		return nil, ErrNamespaceExists
	}

	// Check if directory already exists
	nsPath := filepath.Join(s.basePath, name)
	if fsutil.DirExists(nsPath) {
		return nil, ErrNamespaceExists
	}

	// Validate config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Create namespace
	ns, err := openNamespace(nsPath, name, config, s.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create namespace: %w", err)
	}

	// Cache it
	s.namespaces[name] = ns

	return ns, nil
}

// GetNamespace returns an existing namespace or creates it with default config.
func (s *store) GetNamespace(name string) (Namespace, error) {
	s.mu.RLock()
	// Check cache first
	if ns, exists := s.namespaces[name]; exists {
		s.mu.RUnlock()
		return ns, nil
	}
	s.mu.RUnlock()

	// Not in cache, need write lock
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check after acquiring write lock
	if ns, exists := s.namespaces[name]; exists {
		return ns, nil
	}

	// Try to open or create namespace
	nsPath := filepath.Join(s.basePath, name)
	config := DefaultNamespaceConfig()

	ns, err := openNamespace(nsPath, name, config, s.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to open namespace: %w", err)
	}

	// Cache it
	s.namespaces[name] = ns

	return ns, nil
}

// MustGetNamespace is like GetNamespace but panics on error.
func (s *store) MustGetNamespace(name string) Namespace {
	ns, err := s.GetNamespace(name)
	if err != nil {
		panic(err)
	}
	return ns
}

// ListNamespaces returns all namespace names.
func (s *store) ListNamespaces() ([]string, error) {
	dirs, err := fsutil.ListDirs(s.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	var names []string
	for _, dir := range dirs {
		name := filepath.Base(dir)
		// Skip hidden directories
		if fsutil.IsHidden(name) {
			continue
		}
		names = append(names, name)
	}

	return names, nil
}

// DeleteNamespace deletes a namespace and all its data.
func (s *store) DeleteNamespace(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove from cache
	delete(s.namespaces, name)

	// Delete directory
	nsPath := filepath.Join(s.basePath, name)
	if err := fsutil.RemoveAll(nsPath); err != nil {
		return fmt.Errorf("failed to delete namespace: %w", err)
	}

	return nil
}

// Close closes the store and all open namespaces.
func (s *store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Close all namespaces
	for _, ns := range s.namespaces {
		// Namespace doesn't have Close method in current design
		// If needed, add it to the interface
		_ = ns
	}

	// Clear cache
	s.namespaces = make(map[string]*namespace)

	return nil
}
