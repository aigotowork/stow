package index

import (
	"sync"
	"testing"
	"time"
)

// ========== Basic Cache Tests ==========

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

// ========== New Tests: Exists, Count, Keys ==========

func TestCacheExists(t *testing.T) {
	cache := NewCache(time.Second, 0.0)

	// Non-existent key
	if cache.Exists("nonexistent") {
		t.Error("Exists should return false for non-existent key")
	}

	// Set a key
	cache.Set("key1", "value1")
	if !cache.Exists("key1") {
		t.Error("Exists should return true for existing key")
	}

	// Delete the key
	cache.Delete("key1")
	if cache.Exists("key1") {
		t.Error("Exists should return false after deletion")
	}
}

func TestCacheExistsExpired(t *testing.T) {
	cache := NewCache(50*time.Millisecond, 0.0)

	cache.Set("key1", "value1")

	// Should exist initially
	if !cache.Exists("key1") {
		t.Error("Key should exist immediately after Set")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should not exist after expiration
	if cache.Exists("key1") {
		t.Error("Exists should return false for expired key")
	}
}

func TestCacheCount(t *testing.T) {
	cache := NewCache(time.Second, 0.0)

	// Empty cache
	if cache.Count() != 0 {
		t.Errorf("Count() = %d, want 0", cache.Count())
	}

	// Add entries
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	if cache.Count() != 3 {
		t.Errorf("Count() = %d, want 3", cache.Count())
	}

	// Delete one
	cache.Delete("key2")
	if cache.Count() != 2 {
		t.Errorf("After delete, Count() = %d, want 2", cache.Count())
	}

	// Clear all
	cache.Clear()
	if cache.Count() != 0 {
		t.Errorf("After clear, Count() = %d, want 0", cache.Count())
	}
}

func TestCacheCountExcludesExpired(t *testing.T) {
	cache := NewCache(50*time.Millisecond, 0.0)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	// Should count all entries initially
	if cache.Count() != 2 {
		t.Errorf("Count() = %d, want 2", cache.Count())
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Count should exclude expired entries
	if cache.Count() != 0 {
		t.Errorf("Count() should exclude expired entries, got %d", cache.Count())
	}
}

func TestCacheKeys(t *testing.T) {
	cache := NewCache(time.Second, 0.0)

	// Empty cache
	keys := cache.Keys()
	if len(keys) != 0 {
		t.Errorf("Keys() returned %d keys, want 0", len(keys))
	}

	// Add entries
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	keys = cache.Keys()
	if len(keys) != 3 {
		t.Errorf("Keys() returned %d keys, want 3", len(keys))
	}

	// Verify all keys are present
	keyMap := make(map[string]bool)
	for _, k := range keys {
		keyMap[k] = true
	}

	if !keyMap["key1"] || !keyMap["key2"] || !keyMap["key3"] {
		t.Error("Keys() should return all keys")
	}
}

func TestCacheKeysExcludesExpired(t *testing.T) {
	cache := NewCache(50*time.Millisecond, 0.0)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	keys := cache.Keys()
	if len(keys) != 0 {
		t.Errorf("Keys() should exclude expired entries, got %d keys", len(keys))
	}
}

// ========== Cleanup Tests ==========

func TestCacheCleanupExpired(t *testing.T) {
	cache := NewCache(50*time.Millisecond, 0.0)

	// Add entries
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Cleanup expired
	removed := cache.CleanupExpired()
	if removed != 3 {
		t.Errorf("CleanupExpired() removed %d entries, want 3", removed)
	}

	// Verify count is now 0
	if cache.Count() != 0 {
		t.Errorf("After cleanup, Count() = %d, want 0", cache.Count())
	}
}

func TestCacheCleanupExpiredPartial(t *testing.T) {
	cache := NewCache(100*time.Millisecond, 0.0)

	// Add entries with different TTLs
	cache.Set("key1", "value1")
	time.Sleep(60 * time.Millisecond)
	cache.SetWithTTL("key2", "value2", time.Second) // Long TTL

	// Wait for first key to expire but not second
	time.Sleep(60 * time.Millisecond)

	// Cleanup should only remove expired entries
	removed := cache.CleanupExpired()
	if removed != 1 {
		t.Errorf("CleanupExpired() removed %d entries, want 1", removed)
	}

	// key2 should still exist
	if !cache.Exists("key2") {
		t.Error("key2 should still exist after cleanup")
	}
}

func TestCacheStartCleanupWorker(t *testing.T) {
	cache := NewCache(50*time.Millisecond, 0.0)

	// Add entries
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	// Start cleanup worker
	stop := cache.StartCleanupWorker(30 * time.Millisecond)
	defer close(stop)

	// Wait for expiration and cleanup
	time.Sleep(150 * time.Millisecond)

	// Entries should be cleaned up
	if cache.Count() != 0 {
		t.Error("Cleanup worker should have removed expired entries")
	}
}

func TestCacheStartCleanupWorkerStop(t *testing.T) {
	cache := NewCache(50*time.Millisecond, 0.0)

	cache.Set("key1", "value1")

	// Start and immediately stop cleanup worker
	stop := cache.StartCleanupWorker(30 * time.Millisecond)
	close(stop)

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// This test just verifies the worker can be stopped without panicking
}

// ========== Stats Tests ==========

func TestCacheStats(t *testing.T) {
	cache := NewCache(100*time.Millisecond, 0.2)

	// Empty cache
	stats := cache.Stats()
	if stats.TotalEntries != 0 {
		t.Errorf("Empty cache: TotalEntries = %d, want 0", stats.TotalEntries)
	}
	if stats.TTL != 100*time.Millisecond {
		t.Errorf("Stats.TTL = %v, want %v", stats.TTL, 100*time.Millisecond)
	}
	if stats.Jitter != 0.2 {
		t.Errorf("Stats.Jitter = %f, want 0.2", stats.Jitter)
	}

	// Add entries
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	stats = cache.Stats()
	if stats.TotalEntries != 3 {
		t.Errorf("TotalEntries = %d, want 3", stats.TotalEntries)
	}
	if stats.ValidEntries != 3 {
		t.Errorf("ValidEntries = %d, want 3", stats.ValidEntries)
	}
	if stats.ExpiredEntries != 0 {
		t.Errorf("ExpiredEntries = %d, want 0", stats.ExpiredEntries)
	}
}

func TestCacheStatsWithExpired(t *testing.T) {
	cache := NewCache(50*time.Millisecond, 0.0)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Add a new valid entry
	cache.Set("key3", "value3")

	stats := cache.Stats()
	if stats.TotalEntries != 3 {
		t.Errorf("TotalEntries = %d, want 3", stats.TotalEntries)
	}
	if stats.ValidEntries != 1 {
		t.Errorf("ValidEntries = %d, want 1", stats.ValidEntries)
	}
	if stats.ExpiredEntries != 2 {
		t.Errorf("ExpiredEntries = %d, want 2", stats.ExpiredEntries)
	}
}

func TestCacheStatsHitRate(t *testing.T) {
	cache := NewCache(100*time.Millisecond, 0.0)

	// Add entries
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	stats := cache.Stats()
	hitRate := stats.HitRate()

	// All valid, hit rate should be 1.0
	if hitRate != 1.0 {
		t.Errorf("HitRate() = %f, want 1.0", hitRate)
	}

	// Wait for partial expiration
	time.Sleep(60 * time.Millisecond)
	cache.SetWithTTL("key4", "value4", time.Second) // Add a fresh one
	time.Sleep(60 * time.Millisecond)

	stats = cache.Stats()
	hitRate = stats.HitRate()

	// Some expired, hit rate should be < 1.0
	if hitRate >= 1.0 {
		t.Errorf("HitRate() = %f, should be < 1.0 with expired entries", hitRate)
	}
}

func TestCacheStatsHitRateEmptyCache(t *testing.T) {
	cache := NewCache(time.Second, 0.0)

	stats := cache.Stats()
	hitRate := stats.HitRate()

	// Empty cache should return 0
	if hitRate != 0 {
		t.Errorf("Empty cache HitRate() = %f, want 0", hitRate)
	}
}

// ========== Edge Cases ==========

func TestCacheTTLEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		ttl  time.Duration
	}{
		{"zero TTL", 0},
		{"negative TTL", -1 * time.Second},
		{"very short TTL", 1 * time.Millisecond},
		{"very long TTL", 24 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewCache(tt.ttl, 0.0)
			cache.Set("key", "value")

			// Should not panic
			_, _ = cache.Get("key")
		})
	}
}

func TestCacheJitterEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		jitter float64
	}{
		{"negative jitter", -0.5},
		{"zero jitter", 0.0},
		{"max jitter", 1.0},
		{"excessive jitter", 2.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewCache(100*time.Millisecond, tt.jitter)
			cache.Set("key", "value")

			// Should not panic
			_, _ = cache.Get("key")
		})
	}
}

// ========== Concurrent Tests ==========

func TestCacheConcurrentReadWrite(t *testing.T) {
	cache := NewCache(time.Second, 0.0)

	const goroutines = 50
	const operations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines * 2) // Half readers, half writers

	// Writers
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operations; j++ {
				key := "key" + string(rune(id%10))
				cache.Set(key, id)
			}
		}(i)
	}

	// Readers
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operations; j++ {
				key := "key" + string(rune(id%10))
				cache.Get(key)
			}
		}(i)
	}

	wg.Wait()

	// Should not panic or deadlock
}

func TestCacheConcurrentDeletes(t *testing.T) {
	cache := NewCache(time.Second, 0.0)

	// Pre-populate cache
	for i := 0; i < 100; i++ {
		cache.Set("key"+string(rune(i)), i)
	}

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				cache.Delete("key" + string(rune(id*10+j)))
			}
		}(i)
	}

	wg.Wait()

	// All keys should be deleted
	if cache.Count() != 0 {
		t.Errorf("After concurrent deletes, Count() = %d, want 0", cache.Count())
	}
}

func TestCacheConcurrentClear(t *testing.T) {
	cache := NewCache(time.Second, 0.0)

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			if id%2 == 0 {
				// Half clear the cache
				cache.Clear()
			} else {
				// Half add entries
				cache.Set("key", id)
			}
		}(i)
	}

	wg.Wait()

	// Should not panic or deadlock
}

// ========== Large Dataset Tests ==========

func TestCacheLargeDataset(t *testing.T) {
	cache := NewCache(time.Second, 0.0)

	// Add many entries
	const numEntries = 10000
	for i := 0; i < numEntries; i++ {
		cache.Set("key"+string(rune(i)), i)
	}

	if cache.Count() != numEntries {
		t.Errorf("Count() = %d, want %d", cache.Count(), numEntries)
	}

	// Verify some entries
	val, ok := cache.Get("key" + string(rune(5000)))
	if !ok || val != 5000 {
		t.Error("Should find entry in large dataset")
	}
}
