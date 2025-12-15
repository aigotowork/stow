package stow_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aigotowork/stow"
)

// ========== Basic KV Operations ==========

func TestBasicPutGet(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := stow.Open(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer store.Close()

	ns, err := store.GetNamespace("test")
	if err != nil {
		t.Fatalf("Failed to get namespace: %v", err)
	}

	// Put
	err = ns.Put("key1", map[string]interface{}{
		"name": "test",
		"value": 42,
	})
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Get
	var result map[string]interface{}
	err = ns.Get("key1", &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if result["name"] != "test" {
		t.Errorf("Name mismatch: got %v", result["name"])
	}

	if result["value"] != 42 {
		t.Errorf("Value mismatch: got %v", result["value"])
	}
}

func TestPutGetStruct(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	type Config struct {
		Host string
		Port int
	}

	original := Config{
		Host: "localhost",
		Port: 8080,
	}

	ns.MustPut("config", original)

	var retrieved Config
	ns.MustGet("config", &retrieved)

	if retrieved.Host != original.Host {
		t.Errorf("Host mismatch: got %q", retrieved.Host)
	}

	if retrieved.Port != original.Port {
		t.Errorf("Port mismatch: got %d", retrieved.Port)
	}
}

func TestDelete(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	// Put and verify exists
	ns.MustPut("key1", map[string]interface{}{"value": "value1"})
	if !ns.Exists("key1") {
		t.Fatal("Key should exist after Put")
	}

	// Delete
	err := ns.Delete("key1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify doesn't exist
	if ns.Exists("key1") {
		t.Error("Key should not exist after Delete")
	}

	// Get should return ErrNotFound
	var result map[string]interface{}
	err = ns.Get("key1", &result)
	if err == nil {
		t.Error("Get should return error for deleted key")
	}
}

func TestList(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	// Put multiple keys
	ns.MustPut("key1", map[string]interface{}{"value": "value1"})
	ns.MustPut("key2", map[string]interface{}{"value": "value2"})
	ns.MustPut("key3", map[string]interface{}{"value": "value3"})

	// List
	keys, err := ns.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(keys) != 3 {
		t.Errorf("List should return 3 keys, got %d", len(keys))
	}

	// Verify keys
	keySet := make(map[string]bool)
	for _, key := range keys {
		keySet[key] = true
	}

	for _, expected := range []string{"key1", "key2", "key3"} {
		if !keySet[expected] {
			t.Errorf("List missing key: %s", expected)
		}
	}
}

// ========== Blob Storage Tests ==========

func TestBlobStorage(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	type Document struct {
		Title   string
		Content []byte
	}

	// Create large content (> 4KB threshold, default is 4096 bytes)
	largeContent := make([]byte, 5120)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	doc := Document{
		Title:   "Test Document",
		Content: largeContent,
	}

	// Put
	ns.MustPut("doc1", doc)

	// Verify blob file exists
	blobDir := filepath.Join(tmpDir, "test", "_blobs")
	files, err := os.ReadDir(blobDir)
	if err != nil {
		t.Fatalf("Failed to read blob dir: %v", err)
	}

	if len(files) == 0 {
		t.Error("Blob file should be created")
	}

	// Get
	var retrieved Document
	ns.MustGet("doc1", &retrieved)

	if retrieved.Title != doc.Title {
		t.Errorf("Title mismatch: got %q", retrieved.Title)
	}

	if !bytes.Equal(retrieved.Content, doc.Content) {
		t.Error("Content mismatch")
	}
}

func TestSmallDataInline(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	type Document struct {
		Title   string
		Content []byte
	}

	smallContent := []byte("Small content")

	doc := Document{
		Title:   "Test",
		Content: smallContent,
	}

	ns.MustPut("doc1", doc)

	// Verify no blob files created
	blobDir := filepath.Join(tmpDir, "test", "_blobs")
	files, _ := os.ReadDir(blobDir)

	if len(files) > 0 {
		t.Error("Small content should not create blob files")
	}

	// Get should still work
	var retrieved Document
	ns.MustGet("doc1", &retrieved)

	if !bytes.Equal(retrieved.Content, doc.Content) {
		t.Error("Content mismatch")
	}
}

// ========== Version History Tests ==========

func TestVersionHistory(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	// Put multiple versions
	ns.MustPut("config", map[string]interface{}{"version": 1})
	time.Sleep(10 * time.Millisecond)
	ns.MustPut("config", map[string]interface{}{"version": 2})
	time.Sleep(10 * time.Millisecond)
	ns.MustPut("config", map[string]interface{}{"version": 3})

	// Get history
	history, err := ns.GetHistory("config")
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}

	if len(history) != 3 {
		t.Errorf("Should have 3 versions, got %d", len(history))
	}

	// Verify order (newest first)
	if history[0].Version != 3 {
		t.Errorf("First version should be 3, got %d", history[0].Version)
	}

	// Get specific version
	var v2 map[string]interface{}
	err = ns.GetVersion("config", 2, &v2)
	if err != nil {
		t.Fatalf("GetVersion failed: %v", err)
	}

	if v2["version"] != float64(2) {
		t.Errorf("Version 2 value mismatch: got %v", v2["version"])
	}
}

func TestDeleteRecordsInHistory(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	// Put, delete, put again
	ns.MustPut("key1", "value1")
	ns.MustDelete("key1")
	ns.MustPut("key1", "value2")

	// Get should return latest value
	var result string
	ns.MustGet("key1", &result)

	if result != "value2" {
		t.Errorf("Should get latest value, got %q", result)
	}

	// History should show all operations
	history, _ := ns.GetHistory("key1")
	if len(history) < 2 {
		t.Error("History should contain put and delete operations")
	}
}

// ========== Multiple Namespaces ==========

func TestMultipleNamespaces(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns1 := store.MustGetNamespace("ns1")
	ns2 := store.MustGetNamespace("ns2")

	// Put in ns1
	ns1.MustPut("key1", "value1")

	// Put in ns2
	ns2.MustPut("key1", "different_value")

	// Verify isolation
	var v1 string
	ns1.MustGet("key1", &v1)
	if v1 != "value1" {
		t.Error("Namespace 1 data mismatch")
	}

	var v2 string
	ns2.MustGet("key1", &v2)
	if v2 != "different_value" {
		t.Error("Namespace 2 data mismatch")
	}
}

func TestListNamespaces(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	// Create multiple namespaces
	store.MustGetNamespace("ns1")
	store.MustGetNamespace("ns2")
	store.MustGetNamespace("ns3")

	// List
	namespaces, err := store.ListNamespaces()
	if err != nil {
		t.Fatalf("ListNamespaces failed: %v", err)
	}

	if len(namespaces) != 3 {
		t.Errorf("Should have 3 namespaces, got %d", len(namespaces))
	}
}

func TestDeleteNamespace(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")
	ns.MustPut("key1", "value1")

	// Delete namespace
	err := store.DeleteNamespace("test")
	if err != nil {
		t.Fatalf("DeleteNamespace failed: %v", err)
	}

	// Verify directory deleted
	nsPath := filepath.Join(tmpDir, "test")
	if _, err := os.Stat(nsPath); !os.IsNotExist(err) {
		t.Error("Namespace directory should be deleted")
	}

	// ListNamespaces should not include deleted namespace
	namespaces, _ := store.ListNamespaces()
	for _, name := range namespaces {
		if name == "test" {
			t.Error("Deleted namespace should not appear in list")
		}
	}
}

// ========== Persistence Tests ==========

func TestPersistence(t *testing.T) {
	tmpDir := t.TempDir()

	// Write data
	{
		store := stow.MustOpen(tmpDir)
		ns := store.MustGetNamespace("test")
		ns.MustPut("key1", "value1")
		ns.MustPut("key2", "value2")
		store.Close()
	}

	// Read data in new store
	{
		store := stow.MustOpen(tmpDir)
		ns := store.MustGetNamespace("test")

		var v1 string
		ns.MustGet("key1", &v1)
		if v1 != "value1" {
			t.Errorf("Persisted data mismatch: got %q", v1)
		}

		var v2 string
		ns.MustGet("key2", &v2)
		if v2 != "value2" {
			t.Errorf("Persisted data mismatch: got %q", v2)
		}

		store.Close()
	}
}

func TestPersistenceWithBlobs(t *testing.T) {
	tmpDir := t.TempDir()

	type Document struct {
		Title   string
		Content []byte
	}

	largeContent := make([]byte, 5120)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	// Write
	{
		store := stow.MustOpen(tmpDir)
		ns := store.MustGetNamespace("test")
		ns.MustPut("doc1", Document{
			Title:   "Test",
			Content: largeContent,
		})
		store.Close()
	}

	// Read
	{
		store := stow.MustOpen(tmpDir)
		ns := store.MustGetNamespace("test")

		var doc Document
		ns.MustGet("doc1", &doc)

		if doc.Title != "Test" {
			t.Error("Title mismatch")
		}

		if !bytes.Equal(doc.Content, largeContent) {
			t.Error("Blob content not persisted correctly")
		}

		store.Close()
	}
}

// ========== Key Sanitization Tests ==========

func TestKeySanitization(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	// Put with special characters in key
	keys := []string{
		"user/data:v1",
		"file<test>",
		"path\\to\\file",
		"query?param",
	}

	for i, key := range keys {
		ns.MustPut(key, i)
	}

	// Get should work with original keys
	for i, key := range keys {
		var value int
		ns.MustGet(key, &value)
		if value != i {
			t.Errorf("Key %q value mismatch: got %d", key, value)
		}
	}

	// List should return original keys
	listKeys, _ := ns.List()
	if len(listKeys) != len(keys) {
		t.Errorf("List should return %d keys, got %d", len(keys), len(listKeys))
	}
}

// ========== Stats and Maintenance ==========

func TestStats(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	// Add some data
	ns.MustPut("key1", "value1")
	ns.MustPut("key2", "value2")
	ns.MustPut("key3", make([]byte, 5120)) // > 4KB threshold

	// Get stats
	stats, err := ns.Stats()
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}

	if stats.KeyCount != 3 {
		t.Errorf("KeyCount should be 3, got %d", stats.KeyCount)
	}

	if stats.TotalSize == 0 {
		t.Error("TotalSize should not be zero")
	}

	// Should have 1 blob
	if stats.BlobCount != 1 {
		t.Errorf("BlobCount should be 1, got %d", stats.BlobCount)
	}
}

func TestCompact(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	// Create multiple versions
	for i := 0; i < 10; i++ {
		ns.MustPut("key1", i)
	}

	// Compact
	err := ns.Compact("key1")
	if err != nil {
		t.Fatalf("Compact failed: %v", err)
	}

	// Get should still work
	var value int
	ns.MustGet("key1", &value)
	if value != 9 {
		t.Errorf("Should get latest value after compact, got %d", value)
	}

	// History should be reduced
	history, _ := ns.GetHistory("key1")
	if len(history) > 5 {
		t.Error("Compact should reduce history size")
	}
}

func TestGC(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	type Document struct {
		Content []byte
	}

	// Create documents with blobs (> 4KB threshold, and with different content to avoid deduplication)
	doc1Content := make([]byte, 5120)
	for i := range doc1Content {
		doc1Content[i] = 1 // Fill with 1s
	}
	doc1 := Document{Content: doc1Content}

	doc2Content := make([]byte, 5120)
	for i := range doc2Content {
		doc2Content[i] = 2 // Fill with 2s
	}
	doc2 := Document{Content: doc2Content}

	ns.MustPut("doc1", doc1)
	ns.MustPut("doc2", doc2)

	// Update doc1 (old blob becomes unreferenced)
	doc1UpdatedContent := make([]byte, 5120)
	for i := range doc1UpdatedContent {
		doc1UpdatedContent[i] = 3 // Fill with 3s
	}
	doc1Updated := Document{Content: doc1UpdatedContent}
	ns.MustPut("doc1", doc1Updated)

	// Delete doc2 (blob becomes unreferenced)
	ns.MustDelete("doc2")

	// GC should remove unreferenced blobs
	result, err := ns.GC()
	if err != nil {
		t.Fatalf("GC failed: %v", err)
	}

	if result.RemovedBlobs == 0 {
		t.Error("GC should remove unreferenced blobs")
	}

	if result.ReclaimedSize == 0 {
		t.Error("GC should reclaim space")
	}
}

func TestRefresh(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	// Put value
	ns.MustPut("key1", "value1")

	// Get (caches value)
	var v1 string
	ns.MustGet("key1", &v1)

	// Update value
	ns.MustPut("key1", "value2")

	// Refresh cache
	ns.Refresh("key1")

	// Get should return updated value
	var v2 string
	ns.MustGet("key1", &v2)
	if v2 != "value2" {
		t.Error("Refresh should invalidate cache")
	}
}

// ========== Edge Cases ==========

func TestEmptyKey(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	// Put with empty key should fail
	err := ns.Put("", "value")
	if err == nil {
		t.Error("Put with empty key should fail")
	}
}

func TestGetNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	var result string
	err := ns.Get("nonexistent", &result)
	if err == nil {
		t.Error("Get non-existent key should return error")
	}
}

func TestUpdateValue(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	// Put initial value
	ns.MustPut("key1", "value1")

	// Update
	ns.MustPut("key1", "value2")

	// Get should return latest
	var result string
	ns.MustGet("key1", &result)
	if result != "value2" {
		t.Errorf("Should get updated value, got %q", result)
	}
}

func TestNilValue(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	// Put nil should work (stores as null)
	err := ns.Put("key1", nil)
	if err != nil {
		t.Fatalf("Put nil failed: %v", err)
	}

	// Get should return nil
	var result interface{}
	ns.MustGet("key1", &result)
	if result != nil {
		t.Error("Should get nil value")
	}
}
