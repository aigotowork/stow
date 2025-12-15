package index

import (
	"math/rand"
	"sync"
	"time"
)

// CacheEntry represents a single cache entry with expiration time.
type CacheEntry struct {
	Value      interface{}
	ExpiresAt  time.Time
	InsertedAt time.Time
}

// IsExpired checks if the cache entry has expired.
func (e *CacheEntry) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

// Cache is a simple in-memory cache with TTL support.
// It uses sync.Map for concurrent access.
type Cache struct {
	data   sync.Map
	ttl    time.Duration
	jitter float64
	mu     sync.RWMutex
}

// NewCache creates a new cache with the given TTL and jitter.
//
// Parameters:
//   - ttl: time-to-live for cache entries
//   - jitter: random jitter factor (0.0-1.0), actual TTL = TTL * (1 + jitter)
//
// Example:
//   - TTL=5min, jitter=0.2 -> actual TTL ranges from 5min to 6min (one-directional jitter)
func NewCache(ttl time.Duration, jitter float64) *Cache {
	if jitter < 0 {
		jitter = 0
	}
	if jitter > 1 {
		jitter = 1
	}

	return &Cache{
		ttl:    ttl,
		jitter: jitter,
	}
}

// Get retrieves a value from the cache.
// Returns (value, true) if found and not expired, (nil, false) otherwise.
func (c *Cache) Get(key string) (interface{}, bool) {
	value, ok := c.data.Load(key)
	if !ok {
		return nil, false
	}

	entry := value.(*CacheEntry)

	// Check if expired
	if entry.IsExpired() {
		c.data.Delete(key)
		return nil, false
	}

	return entry.Value, true
}

// Set stores a value in the cache with TTL.
func (c *Cache) Set(key string, value interface{}) {
	c.SetWithTTL(key, value, c.ttl)
}

// SetWithTTL stores a value with a custom TTL.
func (c *Cache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	// Calculate actual TTL with jitter
	actualTTL := c.calculateTTL(ttl)

	entry := &CacheEntry{
		Value:      value,
		ExpiresAt:  time.Now().Add(actualTTL),
		InsertedAt: time.Now(),
	}

	c.data.Store(key, entry)
}

// Delete removes a key from the cache.
func (c *Cache) Delete(key string) {
	c.data.Delete(key)
}

// DeleteMultiple removes multiple keys from the cache.
func (c *Cache) DeleteMultiple(keys []string) {
	for _, key := range keys {
		c.data.Delete(key)
	}
}

// Clear removes all entries from the cache.
func (c *Cache) Clear() {
	c.data.Range(func(key, value interface{}) bool {
		c.data.Delete(key)
		return true
	})
}

// Exists checks if a key exists in the cache (and is not expired).
func (c *Cache) Exists(key string) bool {
	_, ok := c.Get(key)
	return ok
}

// Count returns the number of non-expired entries in the cache.
func (c *Cache) Count() int {
	count := 0
	c.data.Range(func(key, value interface{}) bool {
		entry := value.(*CacheEntry)
		if !entry.IsExpired() {
			count++
		}
		return true
	})
	return count
}

// Keys returns all non-expired keys in the cache.
func (c *Cache) Keys() []string {
	var keys []string
	c.data.Range(func(key, value interface{}) bool {
		entry := value.(*CacheEntry)
		if !entry.IsExpired() {
			keys = append(keys, key.(string))
		}
		return true
	})
	return keys
}

// CleanupExpired removes all expired entries from the cache.
// Returns the number of entries removed.
func (c *Cache) CleanupExpired() int {
	removed := 0
	c.data.Range(func(key, value interface{}) bool {
		entry := value.(*CacheEntry)
		if entry.IsExpired() {
			c.data.Delete(key)
			removed++
		}
		return true
	})
	return removed
}

// Stats returns cache statistics.
func (c *Cache) Stats() CacheStats {
	stats := CacheStats{
		TTL:    c.ttl,
		Jitter: c.jitter,
	}

	c.data.Range(func(key, value interface{}) bool {
		entry := value.(*CacheEntry)
		stats.TotalEntries++

		if entry.IsExpired() {
			stats.ExpiredEntries++
		} else {
			stats.ValidEntries++
		}

		return true
	})

	return stats
}

// calculateTTL calculates the actual TTL with jitter applied.
// Jitter is applied in one direction only (increasing TTL) to ensure minimum TTL is guaranteed.
func (c *Cache) calculateTTL(baseTTL time.Duration) time.Duration {
	if c.jitter == 0 {
		return baseTTL
	}

	// Generate random jitter factor between 1.0 and (1 + jitter)
	// Example: jitter=0.2 -> factor ranges from 1.0 to 1.2
	// This ensures TTL >= baseTTL (never decreases)
	factor := 1.0 + rand.Float64()*c.jitter

	return time.Duration(float64(baseTTL) * factor)
}

// CacheStats contains cache statistics.
type CacheStats struct {
	TotalEntries   int
	ValidEntries   int
	ExpiredEntries int
	TTL            time.Duration
	Jitter         float64
}

// HitRate returns the cache hit rate (valid / total).
func (s *CacheStats) HitRate() float64 {
	if s.TotalEntries == 0 {
		return 0
	}
	return float64(s.ValidEntries) / float64(s.TotalEntries)
}

// StartCleanupWorker starts a background worker that periodically cleans up expired entries.
// Returns a channel that can be closed to stop the worker.
func (c *Cache) StartCleanupWorker(interval time.Duration) chan struct{} {
	stop := make(chan struct{})

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.CleanupExpired()
			case <-stop:
				return
			}
		}
	}()

	return stop
}
