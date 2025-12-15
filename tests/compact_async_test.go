package stow_test

import (
	"testing"
	"time"

	"github.com/aigotowork/stow"
)

func TestCompactAsync(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	// Create multiple versions of keys
	for i := 0; i < 10; i++ {
		ns.MustPut("key1", map[string]interface{}{"version": i})
		ns.MustPut("key2", map[string]interface{}{"version": i})
		ns.MustPut("key3", map[string]interface{}{"version": i})
	}

	// Check history before compact
	history, _ := ns.GetHistory("key1")
	t.Logf("Before compact: key1 has %d versions", len(history))
	if len(history) != 10 {
		t.Errorf("Expected 10 versions, got %d", len(history))
	}

	// Compact asynchronously
	ns.CompactAsync("key1", "key2", "key3")

	// Wait a bit for compaction to complete
	time.Sleep(500 * time.Millisecond)

	// Verify data is still accessible
	var result map[string]interface{}
	ns.MustGet("key1", &result)
	if result["version"] != float64(9) {
		t.Errorf("Expected version 9, got %v", result["version"])
	}

	// Check history after compact (should be reduced)
	history, _ = ns.GetHistory("key1")
	t.Logf("After compact: key1 has %d versions", len(history))
	if len(history) > 5 {
		t.Errorf("Expected <= 5 versions after compact, got %d", len(history))
	}
}

func TestCompactAllAsync(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	// Create multiple keys with multiple versions
	for i := 0; i < 10; i++ {
		ns.MustPut("key1", map[string]interface{}{"value": i})
		ns.MustPut("key2", map[string]interface{}{"value": i})
		ns.MustPut("key3", map[string]interface{}{"value": i})
		ns.MustPut("key4", map[string]interface{}{"value": i})
		ns.MustPut("key5", map[string]interface{}{"value": i})
	}

	// Get all keys
	keys, _ := ns.List()
	t.Logf("Created %d keys", len(keys))

	// Compact all asynchronously
	ns.CompactAllAsync()

	// Wait for compaction to complete
	time.Sleep(1 * time.Second)

	// Verify all data is still accessible
	for _, key := range keys {
		var result map[string]interface{}
		err := ns.Get(key, &result)
		if err != nil {
			t.Errorf("Failed to get key %s after compact: %v", key, err)
		}
		if result["value"] != float64(9) {
			t.Errorf("Key %s: expected value 9, got %v", key, result["value"])
		}
	}
}

func TestCompactAsyncDoesNotBlockReads(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	// Create many versions
	for i := 0; i < 20; i++ {
		ns.MustPut("key1", map[string]interface{}{"value": i})
	}

	// Start async compact
	ns.CompactAsync("key1")

	// Give a tiny bit of time for goroutine to start
	time.Sleep(10 * time.Millisecond)

	// Try to read (should not block)
	var result map[string]interface{}
	err := ns.Get("key1", &result)
	if err != nil {
		t.Errorf("Get should not block during async compact: %v", err)
	}

	// Value should be 19 (latest)
	if result["value"] != float64(19) {
		t.Errorf("Expected value 19, got %v", result["value"])
	}

	// Wait for compact to finish
	time.Sleep(500 * time.Millisecond)
}

func TestCompactAsyncDoesNotBlockWrites(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	// Create many versions for key1
	for i := 0; i < 20; i++ {
		ns.MustPut("key1", map[string]interface{}{"value": i})
	}

	// Start async compact for key1
	ns.CompactAsync("key1")

	// Wait a moment for compact to start
	time.Sleep(10 * time.Millisecond)

	// Write to a different key (should definitely not block)
	err := ns.Put("key2", map[string]interface{}{"value": 100})
	if err != nil {
		t.Errorf("Put to different key should not block: %v", err)
	}

	// Wait for compact to complete
	time.Sleep(500 * time.Millisecond)

	// Verify both keys are accessible
	var result1, result2 map[string]interface{}
	err = ns.Get("key1", &result1)
	if err != nil {
		t.Fatalf("Failed to get key1: %v", err)
	}

	err = ns.Get("key2", &result2)
	if err != nil {
		t.Fatalf("Failed to get key2: %v", err)
	}

	// Verify key1 value (was created before compact)
	if result1["value"] != float64(19) {
		t.Errorf("Key1: expected value 19, got %v", result1["value"])
	}

	// Verify key2 value (was created during/after compact)
	// Note: type might be int or float64 depending on serialization path
	switch v := result2["value"].(type) {
	case int:
		if v != 100 {
			t.Errorf("Key2: expected value 100, got %d", v)
		}
	case float64:
		if v != float64(100) {
			t.Errorf("Key2: expected value 100, got %f", v)
		}
	default:
		t.Errorf("Key2: unexpected type %T", v)
	}
}
