package index

import (
	"sync"
	"testing"
)

// ========== Basic KeyMapper Tests ==========

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

// ========== FindExact Tests ==========

func TestMapperFindExact(t *testing.T) {
	mapper := NewKeyMapper()

	mapper.Add("user/data", "user_data_abc123.jsonl")
	mapper.Add("user_data", "user_data.jsonl")

	// FindExact should return exact match
	fileName1 := mapper.FindExact("user/data")
	if fileName1 != "user_data_abc123.jsonl" {
		t.Errorf("FindExact('user/data') = %q, want 'user_data_abc123.jsonl'", fileName1)
	}

	fileName2 := mapper.FindExact("user_data")
	if fileName2 != "user_data.jsonl" {
		t.Errorf("FindExact('user_data') = %q, want 'user_data.jsonl'", fileName2)
	}

	// Non-existent key should return empty string
	fileName3 := mapper.FindExact("nonexistent")
	if fileName3 != "" {
		t.Errorf("FindExact('nonexistent') = %q, want ''", fileName3)
	}
}

func TestMapperFindExactAfterUpdate(t *testing.T) {
	mapper := NewKeyMapper()

	// Add initial mapping
	mapper.Add("key1", "file1.jsonl")

	// Update mapping
	mapper.Add("key1", "file2.jsonl")

	// Should return updated file name
	fileName := mapper.FindExact("key1")
	if fileName != "file2.jsonl" {
		t.Errorf("FindExact should return updated file name, got %q", fileName)
	}
}

// ========== RemoveByFileName Tests ==========

func TestMapperRemoveByFileName(t *testing.T) {
	mapper := NewKeyMapper()

	mapper.Add("key1", "file1.jsonl")
	mapper.Add("key2", "file2.jsonl")
	mapper.Add("key3", "file3.jsonl")

	// Remove by file name
	mapper.RemoveByFileName("file2.jsonl")

	if mapper.Count() != 2 {
		t.Errorf("After RemoveByFileName, Count() = %d, want 2", mapper.Count())
	}

	// Verify key2 is removed
	fileName := mapper.FindExact("key2")
	if fileName != "" {
		t.Error("key2 should be removed")
	}

	// Verify other keys still exist
	if mapper.FindExact("key1") == "" {
		t.Error("key1 should still exist")
	}
	if mapper.FindExact("key3") == "" {
		t.Error("key3 should still exist")
	}
}

func TestMapperRemoveByFileNameWithCollisions(t *testing.T) {
	mapper := NewKeyMapper()

	// Add keys with same clean key
	mapper.Add("user/data", "user_data_abc123.jsonl")
	mapper.Add("user_data", "user_data.jsonl")

	// Remove one file
	mapper.RemoveByFileName("user_data.jsonl")

	// Count should be 1
	if mapper.Count() != 1 {
		t.Errorf("Count() = %d, want 1", mapper.Count())
	}

	// user/data should still exist
	if mapper.FindExact("user/data") == "" {
		t.Error("user/data should still exist")
	}

	// user_data should be removed
	if mapper.FindExact("user_data") != "" {
		t.Error("user_data should be removed")
	}
}

func TestMapperRemoveByFileNameNonExistent(t *testing.T) {
	mapper := NewKeyMapper()

	mapper.Add("key1", "file1.jsonl")

	// Remove non-existent file should not panic
	mapper.RemoveByFileName("nonexistent.jsonl")

	// Count should remain 1
	if mapper.Count() != 1 {
		t.Errorf("Count() = %d, want 1", mapper.Count())
	}
}

// ========== GetConflicts Tests ==========

func TestMapperGetConflicts(t *testing.T) {
	mapper := NewKeyMapper()

	// Add conflicting keys
	mapper.Add("user/data", "user_data_abc123.jsonl")
	mapper.Add("user_data", "user_data.jsonl")
	mapper.Add("user:data", "user_data_def456.jsonl")

	// Get conflicts for "user/data"
	conflicts := mapper.GetConflicts("user/data")
	if len(conflicts) != 2 {
		t.Errorf("GetConflicts() returned %d conflicts, want 2", len(conflicts))
	}

	// Verify conflicts contain the other keys
	conflictMap := make(map[string]bool)
	for _, k := range conflicts {
		conflictMap[k] = true
	}

	if !conflictMap["user_data"] || !conflictMap["user:data"] {
		t.Errorf("GetConflicts() = %v, want [user_data, user:data]", conflicts)
	}
}

func TestMapperGetConflictsNoConflict(t *testing.T) {
	mapper := NewKeyMapper()

	mapper.Add("key1", "file1.jsonl")
	mapper.Add("key2", "file2.jsonl")

	// No conflicts
	conflicts := mapper.GetConflicts("key1")
	if len(conflicts) != 0 {
		t.Errorf("GetConflicts() returned %d conflicts, want 0", len(conflicts))
	}
}

func TestMapperGetConflictsNonExistent(t *testing.T) {
	mapper := NewKeyMapper()

	mapper.Add("key1", "file1.jsonl")

	// Non-existent key
	conflicts := mapper.GetConflicts("nonexistent")
	if len(conflicts) != 0 {
		t.Error("GetConflicts() should return empty for non-existent key")
	}
}

// ========== Stats Tests ==========

func TestMapperStats(t *testing.T) {
	mapper := NewKeyMapper()

	// Empty mapper
	stats := mapper.Stats()
	if stats["total_keys"] != 0 {
		t.Errorf("Empty mapper: total_keys = %v, want 0", stats["total_keys"])
	}
	if stats["unique_clean_keys"] != 0 {
		t.Errorf("Empty mapper: unique_clean_keys = %v, want 0", stats["unique_clean_keys"])
	}
	if stats["conflicts"] != 0 {
		t.Errorf("Empty mapper: conflicts = %v, want 0", stats["conflicts"])
	}

	// Add keys without conflicts
	mapper.Add("key1", "file1.jsonl")
	mapper.Add("key2", "file2.jsonl")
	mapper.Add("key3", "file3.jsonl")

	stats = mapper.Stats()
	if stats["total_keys"] != 3 {
		t.Errorf("total_keys = %v, want 3", stats["total_keys"])
	}
	if stats["unique_clean_keys"] != 3 {
		t.Errorf("unique_clean_keys = %v, want 3", stats["unique_clean_keys"])
	}
	if stats["conflicts"] != 0 {
		t.Errorf("conflicts = %v, want 0", stats["conflicts"])
	}
}

func TestMapperStatsWithConflicts(t *testing.T) {
	mapper := NewKeyMapper()

	// Add conflicting keys
	mapper.Add("user/data", "user_data_abc123.jsonl")
	mapper.Add("user_data", "user_data.jsonl")
	mapper.Add("cache:v1", "cache_v1_abc123.jsonl")
	mapper.Add("cache_v1", "cache_v1.jsonl")

	stats := mapper.Stats()
	if stats["total_keys"] != 4 {
		t.Errorf("total_keys = %v, want 4", stats["total_keys"])
	}
	if stats["unique_clean_keys"] != 2 {
		t.Errorf("unique_clean_keys = %v, want 2", stats["unique_clean_keys"])
	}
	if stats["conflicts"] != 2 {
		t.Errorf("conflicts = %v, want 2 (two groups of conflicts)", stats["conflicts"])
	}
}

// ========== String Tests ==========

func TestMapperString(t *testing.T) {
	mapper := NewKeyMapper()

	mapper.Add("key1", "file1.jsonl")
	mapper.Add("key2", "file2.jsonl")

	str := mapper.String()
	if str == "" {
		t.Error("String() should return non-empty string")
	}

	// Should contain key information
	// Just verify it doesn't panic and returns something
}

// ========== HasConflict Tests (extended) ==========

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

// ========== Edge Cases ==========

func TestMapperEmptyKey(t *testing.T) {
	mapper := NewKeyMapper()

	// Add empty key (should be sanitized to "unnamed")
	mapper.Add("", "file1.jsonl")

	// Should be findable
	files := mapper.Find("")
	if len(files) == 0 {
		t.Error("Should handle empty key")
	}
}

func TestMapperSpecialCharactersInKey(t *testing.T) {
	mapper := NewKeyMapper()

	// Add keys with special characters
	specialKeys := []string{
		"user/data:v1",
		"file<name>",
		"query*",
		`"quoted"`,
		"path\\to\\file",
	}

	for i, key := range specialKeys {
		fileName := "file" + string(rune(i)) + ".jsonl"
		mapper.Add(key, fileName)
	}

	// All should be added
	if mapper.Count() != len(specialKeys) {
		t.Errorf("Count() = %d, want %d", mapper.Count(), len(specialKeys))
	}

	// All should be findable
	for _, key := range specialKeys {
		files := mapper.Find(key)
		if len(files) == 0 {
			t.Errorf("Should find key %q", key)
		}
	}
}

func TestMapperLongKey(t *testing.T) {
	mapper := NewKeyMapper()

	// Create a very long key
	longKey := ""
	for i := 0; i < 300; i++ {
		longKey += "a"
	}

	mapper.Add(longKey, "file.jsonl")

	// Should be added and findable
	files := mapper.Find(longKey)
	if len(files) == 0 {
		t.Error("Should handle long keys")
	}
}

// ========== Concurrent Tests ==========

func TestMapperConcurrentAdd(t *testing.T) {
	mapper := NewKeyMapper()

	const goroutines = 50
	const keysPerGoroutine = 20

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < keysPerGoroutine; j++ {
				key := "key_" + string(rune(id)) + "_" + string(rune(j))
				fileName := "file_" + string(rune(id)) + "_" + string(rune(j)) + ".jsonl"
				mapper.Add(key, fileName)
			}
		}(i)
	}

	wg.Wait()

	// All keys should be added
	expectedCount := goroutines * keysPerGoroutine
	if mapper.Count() != expectedCount {
		t.Errorf("After concurrent adds, Count() = %d, want %d", mapper.Count(), expectedCount)
	}
}

func TestMapperConcurrentReadWrite(t *testing.T) {
	mapper := NewKeyMapper()

	// Pre-populate
	for i := 0; i < 100; i++ {
		mapper.Add("key"+string(rune(i)), "file"+string(rune(i))+".jsonl")
	}

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines * 2) // Half readers, half writers

	// Writers
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				key := "key" + string(rune(id%10))
				mapper.Add(key, "newfile.jsonl")
			}
		}(i)
	}

	// Readers
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				key := "key" + string(rune(id%10))
				mapper.Find(key)
				mapper.FindExact(key)
				mapper.HasConflict(key)
			}
		}(i)
	}

	wg.Wait()

	// Should not panic or deadlock
}

func TestMapperConcurrentRemove(t *testing.T) {
	mapper := NewKeyMapper()

	// Pre-populate
	for i := 0; i < 100; i++ {
		mapper.Add("key"+string(rune(i)), "file"+string(rune(i))+".jsonl")
	}

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				key := "key" + string(rune(id*10+j))
				mapper.Remove(key)
			}
		}(i)
	}

	wg.Wait()

	// All keys should be removed
	if mapper.Count() != 0 {
		t.Errorf("After concurrent removes, Count() = %d, want 0", mapper.Count())
	}
}

// ========== ListAll Tests (extended) ==========

func TestMapperListAllOrder(t *testing.T) {
	mapper := NewKeyMapper()

	keys := []string{"key3", "key1", "key2"}
	for _, key := range keys {
		mapper.Add(key, key+".jsonl")
	}

	allKeys := mapper.ListAll()
	if len(allKeys) != 3 {
		t.Errorf("ListAll() returned %d keys, want 3", len(allKeys))
	}

	// Verify all keys are present (order doesn't matter)
	keyMap := make(map[string]bool)
	for _, k := range allKeys {
		keyMap[k] = true
	}

	for _, key := range keys {
		if !keyMap[key] {
			t.Errorf("ListAll() missing key %q", key)
		}
	}
}

func TestMapperListAllWithConflicts(t *testing.T) {
	mapper := NewKeyMapper()

	mapper.Add("user/data", "user_data_abc123.jsonl")
	mapper.Add("user_data", "user_data.jsonl")

	allKeys := mapper.ListAll()
	if len(allKeys) != 2 {
		t.Errorf("ListAll() returned %d keys, want 2", len(allKeys))
	}

	// Verify both original keys are present
	keyMap := make(map[string]bool)
	for _, k := range allKeys {
		keyMap[k] = true
	}

	if !keyMap["user/data"] || !keyMap["user_data"] {
		t.Error("ListAll() should return both conflicting keys")
	}
}
