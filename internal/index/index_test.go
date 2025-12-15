package index

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ========== Key Sanitization Tests ==========

func TestSanitizeKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal_key", "normal_key"},
		{"user/data", "user_data"},
		{"path\\to\\file", "path_to_file"},
		{"file:v1", "file_v1"},
		{"query*", "query"},         // trailing _ trimmed
		{"what?", "what"},           // trailing _ trimmed
		{"<tag>", "tag"},            // leading/trailing _ trimmed
		{"file|pipe", "file_pipe"},
		{`"quoted"`, "quoted"},      // leading/trailing _ trimmed
		{"  spaces  ", "spaces"},
		{"___underscores___", "underscores"},
		{"user/path:v1*", "user_path_v1"}, // trailing _ trimmed
	}

	for _, tt := range tests {
		result := SanitizeKey(tt.input)
		if result != tt.expected {
			t.Errorf("SanitizeKey(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestGenerateFileName(t *testing.T) {
	// Without hash
	name1 := GenerateFileName("simple_key", false)
	if name1 != "simple_key.jsonl" {
		t.Errorf("GenerateFileName without hash failed: got %q", name1)
	}

	// With hash
	name2 := GenerateFileName("simple_key", true)
	if name2 == "simple_key.jsonl" {
		t.Error("GenerateFileName with hash should add hash suffix")
	}

	// Verify hash is deterministic
	name3 := GenerateFileName("simple_key", true)
	if name2 != name3 {
		t.Error("Hash should be deterministic")
	}

	// Verify sanitization happens
	name4 := GenerateFileName("user/data:v1", false)
	if name4 != "user_data_v1.jsonl" {
		t.Errorf("GenerateFileName should sanitize: got %q", name4)
	}
}

func TestHasConflict(t *testing.T) {
	mapper := NewKeyMapper()

	// Add first key
	mapper.Add("user/data", "user_data.jsonl")

	// At this point, no conflict yet
	hasCollision := mapper.HasConflict("user/data")
	if hasCollision {
		t.Error("Should not detect collision with only one key")
	}

	// Add second key that sanitizes to the same name
	mapper.Add("user_data", "user_data_abc123.jsonl")

	// Now there should be a conflict
	hasCollision = mapper.HasConflict("user/data")
	if !hasCollision {
		t.Error("Should detect collision after adding conflicting key")
	}

	// Check the other conflicting key too
	hasCollision = mapper.HasConflict("user_data")
	if !hasCollision {
		t.Error("Should detect collision for both conflicting keys")
	}

	// No collision for unrelated key
	hasCollision = mapper.HasConflict("other_key")
	if hasCollision {
		t.Error("Should not detect collision for different key")
	}
}

// ========== KeyMapper Tests ==========

func TestKeyMapper(t *testing.T) {
	mapper := NewKeyMapper()

	// Add mappings
	mapper.Add("server_config", "server_config.jsonl")
	mapper.Add("user_data", "user_data.jsonl")
	mapper.Add("cache:v1", "cache_v1.jsonl")

	// Test Count
	if mapper.Count() != 3 {
		t.Errorf("Count() = %d, want 3", mapper.Count())
	}

	// Test Find
	files := mapper.Find("server_config")
	if len(files) != 1 || files[0].FileName != "server_config.jsonl" {
		t.Errorf("Find failed: got %v", files)
	}

	// Test ListAll
	allKeys := mapper.ListAll()
	if len(allKeys) != 3 {
		t.Errorf("ListAll() = %d keys, want 3", len(allKeys))
	}

	// Test Remove
	mapper.Remove("user_data")
	if mapper.Count() != 2 {
		t.Errorf("After Remove, Count() = %d, want 2", mapper.Count())
	}

	// Verify removed
	files = mapper.Find("user_data")
	if len(files) != 0 {
		t.Error("Key should be removed")
	}
}

func TestKeyMapperCollisionHandling(t *testing.T) {
	mapper := NewKeyMapper()

	// Add keys that sanitize to same name
	mapper.Add("user/data", "user_data_abc123.jsonl")
	mapper.Add("user_data", "user_data.jsonl")

	// Both should be findable
	files1 := mapper.Find("user/data")
	if len(files1) == 0 {
		t.Error("Should find user/data")
	}

	files2 := mapper.Find("user_data")
	if len(files2) == 0 {
		t.Error("Should find user_data")
	}

	// Should have 2 total entries
	if mapper.Count() != 2 {
		t.Errorf("Count() = %d, want 2", mapper.Count())
	}
}

func TestKeyMapperClear(t *testing.T) {
	mapper := NewKeyMapper()
	mapper.Add("key1", "key1.jsonl")
	mapper.Add("key2", "key2.jsonl")

	mapper.Clear()

	if mapper.Count() != 0 {
		t.Error("Clear should remove all entries")
	}
}

// ========== Scanner Tests ==========

func TestScanner(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test JSONL files with valid records
	files := map[string]string{
		"config.jsonl": `{"_meta":{"k":"config","v":1,"op":"put","ts":"2024-01-01T00:00:00Z"},"data":{"value":"test"}}` + "\n",
		"cache.jsonl":  `{"_meta":{"k":"cache:v1","v":1,"op":"put","ts":"2024-01-01T00:00:00Z"},"data":{"value":"cache"}}` + "\n",
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Scan directory
	scanner := NewScanner()
	mapper, err := scanner.ScanNamespace(tmpDir)
	if err != nil {
		t.Fatalf("ScanNamespace failed: %v", err)
	}

	// Verify results
	if mapper.Count() != 2 {
		t.Errorf("Scanner found %d keys, want 2", mapper.Count())
	}

	// Check specific keys
	files1 := mapper.Find("config")
	if len(files1) == 0 {
		t.Error("Should find 'config' key")
	}

	files2 := mapper.Find("cache:v1")
	if len(files2) == 0 {
		t.Error("Should find 'cache:v1' key")
	}
}

func TestScannerSkipsInvalidFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create invalid file
	invalidPath := filepath.Join(tmpDir, "invalid.jsonl")
	os.WriteFile(invalidPath, []byte("not valid json\n"), 0644)

	// Create valid file
	validPath := filepath.Join(tmpDir, "valid.jsonl")
	validContent := `{"_meta":{"k":"valid","v":1,"op":"put","ts":"2024-01-01T00:00:00Z"},"data":{}}` + "\n"
	os.WriteFile(validPath, []byte(validContent), 0644)

	// Scan should skip invalid and process valid
	scanner := NewScanner()
	mapper, err := scanner.ScanNamespace(tmpDir)
	if err != nil {
		t.Fatalf("ScanNamespace failed: %v", err)
	}

	if mapper.Count() != 1 {
		t.Errorf("Should find 1 valid key, got %d", mapper.Count())
	}
}

func TestScannerEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	scanner := NewScanner()
	mapper, err := scanner.ScanNamespace(tmpDir)
	if err != nil {
		t.Fatalf("ScanNamespace failed: %v", err)
	}

	if mapper.Count() != 0 {
		t.Error("Empty directory should result in empty mapper")
	}
}

// ========== Cache Tests ==========

func TestCache(t *testing.T) {
	cache := NewCache(100*time.Millisecond, 0.0)

	// Set value
	cache.Set("key1", map[string]interface{}{"value": "test"})

	// Get value
	val, ok := cache.Get("key1")
	if !ok {
		t.Fatal("Get should find value")
	}

	data, ok := val.(map[string]interface{})
	if !ok || data["value"] != "test" {
		t.Error("Get returned wrong value")
	}

	// Get non-existent
	_, ok = cache.Get("nonexistent")
	if ok {
		t.Error("Get should return false for non-existent key")
	}
}

func TestCacheTTL(t *testing.T) {
	cache := NewCache(50*time.Millisecond, 0.0)

	cache.Set("key1", "value1")

	// Should exist immediately
	_, ok := cache.Get("key1")
	if !ok {
		t.Error("Value should exist immediately after Set")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	_, ok = cache.Get("key1")
	if ok {
		t.Error("Value should be expired")
	}
}

func TestCacheJitter(t *testing.T) {
	cache := NewCache(100*time.Millisecond, 0.2)

	// Set multiple values
	for i := 0; i < 10; i++ {
		cache.Set("key", "value")
	}

	// With jitter, TTLs should vary
	// This is a probabilistic test - we just verify cache works with jitter enabled
	_, ok := cache.Get("key")
	if !ok {
		t.Error("Cache with jitter should store values")
	}
}

func TestCacheDelete(t *testing.T) {
	cache := NewCache(time.Second, 0.0)

	cache.Set("key1", "value1")
	cache.Delete("key1")

	_, ok := cache.Get("key1")
	if ok {
		t.Error("Deleted key should not exist")
	}
}

func TestCacheDeleteMultiple(t *testing.T) {
	cache := NewCache(time.Second, 0.0)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	cache.DeleteMultiple([]string{"key1", "key3"})

	_, ok1 := cache.Get("key1")
	_, ok2 := cache.Get("key2")
	_, ok3 := cache.Get("key3")

	if ok1 {
		t.Error("key1 should be deleted")
	}
	if !ok2 {
		t.Error("key2 should still exist")
	}
	if ok3 {
		t.Error("key3 should be deleted")
	}
}

func TestCacheClear(t *testing.T) {
	cache := NewCache(time.Second, 0.0)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	cache.Clear()

	_, ok1 := cache.Get("key1")
	_, ok2 := cache.Get("key2")

	if ok1 || ok2 {
		t.Error("Clear should remove all entries")
	}
}

func TestCacheOverwrite(t *testing.T) {
	cache := NewCache(time.Second, 0.0)

	cache.Set("key1", "value1")
	cache.Set("key1", "value2")

	val, ok := cache.Get("key1")
	if !ok || val != "value2" {
		t.Error("Set should overwrite existing value")
	}
}

func TestCacheNilValue(t *testing.T) {
	cache := NewCache(time.Second, 0.0)

	cache.Set("key1", nil)

	val, ok := cache.Get("key1")
	if !ok {
		t.Error("Cache should support nil values")
	}
	if val != nil {
		t.Error("Retrieved value should be nil")
	}
}
