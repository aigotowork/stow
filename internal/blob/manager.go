package blob

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aigotowork/stow/internal/fsutil"
)

// Manager manages blob file storage and retrieval.
// It handles file naming, storage, and indexing.
type Manager struct {
	blobDir   string // Path to _blobs/ directory
	maxSize   int64  // Maximum file size
	chunkSize int64  // Chunk size for writing

	// Name index: maps clean file names to actual file names with hash
	// Example: "avatar.jpg" -> ["avatar_abc123.jpg", "avatar_def456.jpg"]
	nameIndex map[string][]string

	// Hash index: maps content hash to relative file path (for deduplication)
	// Example: "abc123..." -> "avatar_abc123.jpg"
	hashIndex map[string]string

	mu sync.RWMutex
}

// NewManager creates a new blob manager.
func NewManager(blobDir string, maxSize, chunkSize int64) (*Manager, error) {
	// Ensure blob directory exists
	if err := fsutil.EnsureDir(blobDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create blob directory: %w", err)
	}

	m := &Manager{
		blobDir:   blobDir,
		maxSize:   maxSize,
		chunkSize: chunkSize,
		nameIndex: make(map[string][]string),
		hashIndex: make(map[string]string),
	}

	// Build initial index
	if err := m.buildIndex(); err != nil {
		return nil, fmt.Errorf("failed to build blob index: %w", err)
	}

	return m, nil
}

// Store stores data as a blob file and returns a reference.
//
// Parameters:
//   - data: the data to store (io.Reader or []byte)
//   - name: optional file name (e.g., "avatar.jpg")
//   - mimeType: optional MIME type
//
// Returns a Reference that should be stored in the JSONL record.
func (m *Manager) Store(data interface{}, name, mimeType string) (*Reference, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Convert data to reader
	var reader io.Reader
	switch v := data.(type) {
	case io.Reader:
		reader = v
	case []byte:
		reader = bytes.NewReader(v)
	default:
		return nil, fmt.Errorf("unsupported data type: %T", data)
	}

	// Generate temporary file path
	tmpPath := filepath.Join(m.blobDir, fmt.Sprintf("tmp_%d", os.Getpid()))

	// Create writer
	writer, err := NewWriter(tmpPath, m.maxSize, m.chunkSize)
	if err != nil {
		return nil, err
	}

	// Write data
	if err := writer.WriteFrom(reader); err != nil {
		writer.Abort()
		return nil, fmt.Errorf("failed to write blob: %w", err)
	}

	// Close and get hash
	hash, size, err := writer.Close()
	if err != nil {
		os.Remove(tmpPath)
		return nil, err
	}

	// Use short hash for indexing (consistent with filename extraction)
	shortHash := ShortHash(hash)

	// Check if this content already exists (deduplication by content hash)
	var finalPath string
	var fileName string

	if existingFile, exists := m.hashIndex[shortHash]; exists {
		// Content already exists, reuse the existing file
		fileName = existingFile
		finalPath = filepath.Join(m.blobDir, fileName)

		// Remove temp file since we're reusing existing
		os.Remove(tmpPath)
	} else {
		// New content, generate final file name
		fileName = m.generateFileName(name, hash)
		finalPath = filepath.Join(m.blobDir, fileName)

		// Rename temp file to final name
		if err := fsutil.SafeRename(tmpPath, finalPath); err != nil {
			os.Remove(tmpPath)
			return nil, fmt.Errorf("failed to rename blob file: %w", err)
		}

		// Update hash index with new file (using short hash as key)
		m.hashIndex[shortHash] = fileName
	}

	// Update name index
	if name != "" {
		cleanName := m.extractCleanName(name)
		m.nameIndex[cleanName] = append(m.nameIndex[cleanName], fileName)
	}

	// Create reference (with full hash)
	location := filepath.Join("_blobs", fileName)
	ref := NewReference(location, hash, size, mimeType, name)

	return ref, nil
}

// Load loads a blob file from a reference.
// Returns a FileData handle for streaming access.
func (m *Manager) Load(ref *Reference) (*FileData, error) {
	if ref == nil || !ref.IsValid() {
		return nil, fmt.Errorf("invalid blob reference")
	}

	// Get absolute path
	path := m.resolveRefPath(ref)

	// Check if file exists
	if !fsutil.FileExists(path) {
		return nil, fmt.Errorf("blob file not found: %s", path)
	}

	// Create FileData handle
	fileData := NewFileData(path, ref.Name, ref.Size, ref.MimeType, ref.Hash)
	return fileData, nil
}

// LoadBytes loads a blob file and returns its contents as bytes.
// This loads the entire file into memory - use Load() for streaming large files.
func (m *Manager) LoadBytes(ref *Reference) ([]byte, error) {
	fileData, err := m.Load(ref)
	if err != nil {
		return nil, err
	}
	defer fileData.Close()

	return io.ReadAll(fileData)
}

// Exists checks if a blob file exists.
func (m *Manager) Exists(ref *Reference) bool {
	if ref == nil || !ref.IsValid() {
		return false
	}

	path := m.resolveRefPath(ref)
	return fsutil.FileExists(path)
}

// Delete removes a blob file.
func (m *Manager) Delete(ref *Reference) error {
	if ref == nil || !ref.IsValid() {
		return fmt.Errorf("invalid blob reference")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	path := m.resolveRefPath(ref)

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete blob: %w", err)
	}

	// Update hash index (use short hash)
	if ref.Hash != "" {
		shortHash := ShortHash(ref.Hash)
		delete(m.hashIndex, shortHash)
	}

	// Update name index
	if ref.Name != "" {
		cleanName := m.extractCleanName(ref.Name)
		fileName := filepath.Base(path)
		m.removeFromIndex(cleanName, fileName)
	}

	return nil
}

// ListAll returns all blob files in the directory.
func (m *Manager) ListAll() ([]string, error) {
	files, err := fsutil.ListFiles(m.blobDir)
	if err != nil {
		return nil, fmt.Errorf("failed to list blobs: %w", err)
	}

	var blobs []string
	for _, file := range files {
		// Skip temporary files
		if strings.Contains(filepath.Base(file), "tmp_") {
			continue
		}
		blobs = append(blobs, file)
	}

	return blobs, nil
}

// TotalSize calculates the total size of all blob files.
func (m *Manager) TotalSize() (int64, error) {
	return fsutil.DirSize(m.blobDir)
}

// Count returns the number of blob files.
func (m *Manager) Count() (int, error) {
	blobs, err := m.ListAll()
	if err != nil {
		return 0, err
	}
	return len(blobs), nil
}

// buildIndex builds the name and hash indexes by scanning the blob directory.
func (m *Manager) buildIndex() error {
	files, err := fsutil.ListFiles(m.blobDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		fileName := filepath.Base(file)

		// Skip temporary files
		if strings.Contains(fileName, "tmp_") {
			continue
		}

		// Extract hash from file name
		// Format: {name}_{shorthash}.{ext} or {shorthash}.bin
		hash := m.extractHashFromFileName(fileName)
		if hash != "" {
			// Add to hash index
			m.hashIndex[hash] = fileName
		}

		// Extract clean name from file name
		// Example: "avatar_abc123.jpg" -> "avatar.jpg"
		cleanName := m.extractCleanNameFromFileName(fileName)
		if cleanName != "" {
			m.nameIndex[cleanName] = append(m.nameIndex[cleanName], fileName)
		}
	}

	return nil
}

// generateFileName generates a file name for a blob.
// Format: {name}_{hash}.{ext} or {hash}.bin
func (m *Manager) generateFileName(name, hash string) string {
	shortHash := ShortHash(hash)

	if name == "" {
		// No name specified, use hash only
		return shortHash + ".bin"
	}

	// Extract extension
	ext := filepath.Ext(name)
	baseName := strings.TrimSuffix(name, ext)

	// Sanitize base name (remove path separators and special characters)
	baseName = sanitizeFileName(baseName)

	// Generate file name: {name}_{hash}{.ext}
	if ext != "" {
		return fmt.Sprintf("%s_%s%s", baseName, shortHash, ext)
	}

	return fmt.Sprintf("%s_%s", baseName, shortHash)
}

// extractCleanName extracts the clean name from a user-provided name.
// Example: "avatar.jpg" -> "avatar.jpg"
func (m *Manager) extractCleanName(name string) string {
	return filepath.Base(name)
}

// extractCleanNameFromFileName extracts clean name from a blob file name.
// Example: "avatar_abc123.jpg" -> "avatar.jpg"
func (m *Manager) extractCleanNameFromFileName(fileName string) string {
	// Find the last underscore
	lastUnderscore := strings.LastIndex(fileName, "_")
	if lastUnderscore == -1 {
		return ""
	}

	// Everything before the underscore is the base name
	baseName := fileName[:lastUnderscore]

	// Get extension
	ext := filepath.Ext(fileName)

	return baseName + ext
}

// extractHashFromFileName extracts the hash portion from a blob file name.
// Example: "avatar_abc123.jpg" -> "abc123" (short hash)
// Example: "abc123.bin" -> "abc123"
func (m *Manager) extractHashFromFileName(fileName string) string {
	// Remove extension
	nameWithoutExt := strings.TrimSuffix(fileName, filepath.Ext(fileName))

	// Find the last underscore
	lastUnderscore := strings.LastIndex(nameWithoutExt, "_")
	if lastUnderscore == -1 {
		// No underscore, the entire name is the hash (e.g., "abc123.bin")
		return nameWithoutExt
	}

	// Everything after the underscore is the short hash
	return nameWithoutExt[lastUnderscore+1:]
}

// resolveRefPath resolves a reference to an absolute file path.
func (m *Manager) resolveRefPath(ref *Reference) string {
	// ref.Location is like "_blobs/file_abc123.jpg"
	// We need to convert it to absolute path

	// Extract just the file name
	fileName := filepath.Base(ref.Location)

	return filepath.Join(m.blobDir, fileName)
}

// removeFromIndex removes a file name from the name index.
func (m *Manager) removeFromIndex(cleanName, fileName string) {
	files, ok := m.nameIndex[cleanName]
	if !ok {
		return
	}

	// Remove fileName from the list
	var newFiles []string
	for _, f := range files {
		if f != fileName {
			newFiles = append(newFiles, f)
		}
	}

	if len(newFiles) == 0 {
		delete(m.nameIndex, cleanName)
	} else {
		m.nameIndex[cleanName] = newFiles
	}
}

// sanitizeFileName removes invalid characters from a file name.
func sanitizeFileName(name string) string {
	// Replace invalid characters with underscore
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := name

	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}

	return result
}
