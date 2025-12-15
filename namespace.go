package stow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/aigotowork/stow/internal/blob"
	"github.com/aigotowork/stow/internal/codec"
	"github.com/aigotowork/stow/internal/core"
	"github.com/aigotowork/stow/internal/fsutil"
	"github.com/aigotowork/stow/internal/index"
)

// namespace implements the Namespace interface.
type namespace struct {
	name   string
	path   string
	config NamespaceConfig
	logger Logger

	// Core components
	blobManager *blob.Manager
	keyMapper   *index.KeyMapper
	cache       *index.Cache
	marshaler   *codec.Marshaler
	unmarshaler *codec.Unmarshaler
	decoder     *core.Decoder
	encoder     *core.Encoder

	// Concurrency control
	mu       sync.RWMutex    // For metadata operations (keyMapper, config, etc.)
	keyLocks sync.Map        // Per-key locks: key → *sync.Mutex

	// Statistics
	stats NamespaceStats
}

// openNamespace opens or creates a namespace.
func openNamespace(path, name string, config NamespaceConfig, logger Logger) (*namespace, error) {
	// Ensure namespace directory exists
	if err := fsutil.EnsureDir(path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create namespace directory: %w", err)
	}

	// Ensure _blobs directory exists
	blobDir := filepath.Join(path, "_blobs")
	if err := fsutil.EnsureDir(blobDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create blobs directory: %w", err)
	}

	// Create blob manager
	blobManager, err := blob.NewManager(blobDir, config.MaxFileSize, config.BlobChunkSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create blob manager: %w", err)
	}

	// Scan directory and build key mapper
	scanner := index.NewScanner()
	keyMapper, err := scanner.ScanNamespace(path)
	if err != nil {
		return nil, fmt.Errorf("failed to scan namespace: %w", err)
	}

	// Create cache
	cache := index.NewCache(config.CacheTTL, config.CacheTTLJitter)

	// Create codec components
	marshaler := codec.NewMarshaler(blobManager)
	unmarshaler := codec.NewUnmarshaler(blobManager)

	ns := &namespace{
		name:        name,
		path:        path,
		config:      config,
		logger:      logger,
		blobManager: blobManager,
		keyMapper:   keyMapper,
		cache:       cache,
		marshaler:   marshaler,
		unmarshaler: unmarshaler,
		decoder:     core.NewDecoder(),
		encoder:     core.NewEncoder(),
	}

	// Try to load config from file
	if err := ns.loadConfig(); err != nil {
		// Config doesn't exist, save default config
		if err := ns.saveConfig(); err != nil {
			logger.Warn("failed to save default config", Field{"error", err})
		}
	}

	return ns, nil
}

// getKeyLock returns a mutex for the given key.
// If the mutex doesn't exist, it creates one.
func (ns *namespace) getKeyLock(key string) *sync.Mutex {
	// Try to load existing lock
	if lock, ok := ns.keyLocks.Load(key); ok {
		return lock.(*sync.Mutex)
	}

	// Create new lock
	newLock := &sync.Mutex{}
	actual, loaded := ns.keyLocks.LoadOrStore(key, newLock)
	if loaded {
		// Another goroutine created the lock, use that one
		return actual.(*sync.Mutex)
	}

	return newLock
}

// Put stores a key-value pair.
func (ns *namespace) Put(key string, value interface{}, opts ...PutOption) error {
	// Validate key
	if !index.IsValidKey(key) {
		return fmt.Errorf("invalid key: %s", key)
	}

	// Acquire key-level lock
	keyLock := ns.getKeyLock(key)
	keyLock.Lock()
	defer keyLock.Unlock()

	// Apply options
	options := &putOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// Marshal value
	marshalOpts := codec.MarshalOptions{
		BlobThreshold: ns.config.BlobThreshold,
		ForceFile:     options.forceFile,
		ForceInline:   options.forceInline,
		FileName:      options.fileName,
		MimeType:      options.mimeType,
	}

	data, blobRefs, err := ns.marshaler.Marshal(value, marshalOpts)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	// Get file path (need read lock for keyMapper)
	ns.mu.RLock()
	filePath, err := ns.getFilePath(key, true)
	ns.mu.RUnlock()
	if err != nil {
		return err
	}

	// Get current version
	version := ns.getNextVersion(filePath)

	// Create record
	record := core.NewPutRecord(key, version, data)

	// Append to file
	if err := core.AppendRecord(filePath, record); err != nil {
		// Clean up blobs on failure
		for _, ref := range blobRefs {
			ns.blobManager.Delete(ref)
		}
		return fmt.Errorf("failed to append record: %w", err)
	}

	// Update key mapper (need write lock for metadata)
	ns.mu.Lock()
	fileName := filepath.Base(filePath)
	ns.keyMapper.Add(key, fileName)
	ns.mu.Unlock()

	// Update cache (no lock needed, cache is thread-safe)
	ns.cache.Set(key, data)

	// Auto compact if enabled
	if ns.config.AutoCompact {
		go ns.compactIfNeeded(key, filePath)
	}

	return nil
}

// MustPut is like Put but panics on error.
func (ns *namespace) MustPut(key string, value interface{}, opts ...PutOption) {
	if err := ns.Put(key, value, opts...); err != nil {
		panic(err)
	}
}

// Get retrieves a value by key.
func (ns *namespace) Get(key string, target interface{}) error {
	// Check cache first (no lock needed, cache is thread-safe)
	if !ns.config.DisableCache {
		if cached, ok := ns.cache.Get(key); ok {
			data, ok := cached.(map[string]interface{})
			if ok {
				return ns.unmarshaler.Unmarshal(data, target)
			}
		}
	}

	// Get file path (need read lock for keyMapper)
	ns.mu.RLock()
	filePath, err := ns.getFilePath(key, false)
	ns.mu.RUnlock()
	if err != nil {
		return err
	}

	// Check if file exists
	if !fsutil.FileExists(filePath) {
		return ErrNotFound
	}

	// Read last valid record (no lock needed, file reads are safe)
	record, err := ns.decoder.ReadLastValid(filePath)
	if err != nil {
		return fmt.Errorf("failed to read record: %w", err)
	}

	if record == nil || record.Meta.IsDelete() {
		return ErrNotFound
	}

	// Update cache
	if !ns.config.DisableCache {
		ns.cache.Set(key, record.Data)
	}

	// Unmarshal into target
	return ns.unmarshaler.Unmarshal(record.Data, target)
}

// MustGet is like Get but panics on error.
func (ns *namespace) MustGet(key string, target interface{}) {
	if err := ns.Get(key, target); err != nil {
		panic(err)
	}
}

// GetRaw returns the raw record.
func (ns *namespace) GetRaw(key string) (RawItem, error) {
	// Get file path (need read lock for keyMapper)
	ns.mu.RLock()
	filePath, err := ns.getFilePath(key, false)
	ns.mu.RUnlock()
	if err != nil {
		return nil, err
	}

	// Read last valid record (no lock needed, file reads are safe)
	record, err := ns.decoder.ReadLastValid(filePath)
	if err != nil {
		return nil, err
	}

	if record == nil || record.Meta.IsDelete() {
		return nil, ErrNotFound
	}

	return &rawItem{record: record, unmarshaler: ns.unmarshaler}, nil
}

// Delete marks a key as deleted.
func (ns *namespace) Delete(key string) error {
	// Acquire key-level lock
	keyLock := ns.getKeyLock(key)
	keyLock.Lock()
	defer keyLock.Unlock()

	// Get file path (need read lock for keyMapper)
	ns.mu.RLock()
	filePath, err := ns.getFilePath(key, false)
	ns.mu.RUnlock()
	if err != nil {
		return err
	}

	// Get next version
	version := ns.getNextVersion(filePath)

	// Create delete record
	record := core.NewDeleteRecord(key, version)

	// Append to file
	if err := core.AppendRecord(filePath, record); err != nil {
		return fmt.Errorf("failed to append delete record: %w", err)
	}

	// Clear cache (no lock needed, cache is thread-safe)
	ns.cache.Delete(key)

	return nil
}

// MustDelete is like Delete but panics on error.
func (ns *namespace) MustDelete(key string) {
	if err := ns.Delete(key); err != nil {
		panic(err)
	}
}

// Exists checks if a key exists.
func (ns *namespace) Exists(key string) bool {
	err := ns.Get(key, new(interface{}))
	return err == nil
}

// List returns all keys.
func (ns *namespace) List() ([]string, error) {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	allKeys := ns.keyMapper.ListAll()

	// Filter out deleted keys
	var activeKeys []string
	for _, key := range allKeys {
		if ns.Exists(key) {
			activeKeys = append(activeKeys, key)
		}
	}

	return activeKeys, nil
}

// Helper methods

// getFilePath gets the file path for a key.
func (ns *namespace) getFilePath(key string, create bool) (string, error) {
	// Try to find existing file
	exactFile := ns.keyMapper.FindExact(key)
	if exactFile != "" {
		return filepath.Join(ns.path, exactFile), nil
	}

	if !create {
		return "", ErrNotFound
	}

	// Need to create new file
	// Check if sanitized key would conflict
	needsHash := index.NeedsHashSuffix(key) || ns.keyMapper.HasConflict(key)
	fileName := index.GenerateFileName(key, needsHash)

	return filepath.Join(ns.path, fileName), nil
}

// getNextVersion gets the next version number for a key.
func (ns *namespace) getNextVersion(filePath string) int {
	version, err := ns.decoder.GetLatestVersion(filePath)
	if err != nil {
		return 1
	}
	return version + 1
}

// compactIfNeeded checks if compaction is needed and performs it.
func (ns *namespace) compactIfNeeded(key, filePath string) {
	// Check if compaction is needed based on strategy
	needsCompact := false

	switch ns.config.CompactStrategy {
	case CompactStrategyLineCount:
		lineCount, err := core.CountLines(filePath)
		if err == nil && lineCount > ns.config.CompactThreshold {
			needsCompact = true
		}

	case CompactStrategyFileSize:
		size := fsutil.FileSize(filePath)
		if size > int64(ns.config.CompactThreshold) {
			needsCompact = true
		}
	}

	if needsCompact {
		ns.Compact(key)
	}
}

// loadConfig loads configuration from _config.json.
func (ns *namespace) loadConfig() error {
	configPath := filepath.Join(ns.path, "_config.json")

	if !fsutil.FileExists(configPath) {
		return fmt.Errorf("config file not found")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var config NamespaceConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	ns.config = config
	return nil
}

// saveConfig saves configuration to _config.json.
func (ns *namespace) saveConfig() error {
	configPath := filepath.Join(ns.path, "_config.json")

	data, err := json.MarshalIndent(ns.config, "", "  ")
	if err != nil {
		return err
	}

	return fsutil.AtomicWriteFile(configPath, data, 0644)
}

// rawItem implements RawItem interface.
type rawItem struct {
	record      *core.Record
	unmarshaler *codec.Unmarshaler
}

func (r *rawItem) Meta() MetaInfo {
	return MetaInfo{
		Key:       r.record.Meta.Key,
		Version:   r.record.Meta.Version,
		Operation: r.record.Meta.Operation,
		Timestamp: r.record.Meta.Timestamp,
	}
}

func (r *rawItem) DecodeInto(target interface{}) error {
	return r.unmarshaler.Unmarshal(r.record.Data, target)
}

func (r *rawItem) RawData() map[string]interface{} {
	return r.record.Data
}

// Fluent API methods

func (ns *namespace) WithLogger(logger Logger) Namespace {
	ns.logger = logger
	return ns
}

func (ns *namespace) WithBlobThreshold(bytes int64) Namespace {
	ns.config.BlobThreshold = bytes
	return ns
}

func (ns *namespace) WithMaxFileSize(bytes int64) Namespace {
	ns.config.MaxFileSize = bytes
	return ns
}

// Metadata methods

func (ns *namespace) Name() string {
	return ns.name
}

func (ns *namespace) Path() string {
	return ns.path
}

func (ns *namespace) GetConfig() NamespaceConfig {
	return ns.config
}

func (ns *namespace) SetConfig(config NamespaceConfig) error {
	if err := config.Validate(); err != nil {
		return err
	}

	ns.config = config
	return ns.saveConfig()
}
