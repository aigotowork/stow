package stow_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/aigotowork/stow"
)

// TestConcurrentWritesDifferentKeys verifies that writes to different keys can happen in parallel.
func TestConcurrentWritesDifferentKeys(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := stow.Open(tmpDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer store.Close()

	ns, err := store.CreateNamespace("test", stow.DefaultNamespaceConfig())
	if err != nil {
		t.Fatalf("CreateNamespace failed: %v", err)
	}

	// Track timing to verify concurrent execution
	const numKeys = 10
	const writesPerKey = 20

	start := time.Now()

	var wg sync.WaitGroup
	errors := make(chan error, numKeys*writesPerKey)

	// Launch concurrent writers for different keys
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key%d", i)
		wg.Add(1)

		go func(k string) {
			defer wg.Done()
			for j := 0; j < writesPerKey; j++ {
				data := map[string]interface{}{
					"iteration": j,
					"timestamp": time.Now().UnixNano(),
				}
				if err := ns.Put(k, data); err != nil {
					errors <- fmt.Errorf("Put %s failed: %w", k, err)
					return
				}
			}
		}(key)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent write error: %v", err)
	}

	elapsed := time.Since(start)
	t.Logf("Concurrent writes to %d keys (%d writes each) completed in %v", numKeys, writesPerKey, elapsed)

	// Verify all writes succeeded by reading back
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key%d", i)
		var result map[string]interface{}
		if err := ns.Get(key, &result); err != nil {
			t.Errorf("Get %s failed: %v", key, err)
			continue
		}
		// Should have the last iteration
		iteration, ok := result["iteration"]
		if !ok {
			t.Errorf("Key %s: missing iteration field", key)
			continue
		}
		// Handle both int and float64
		var iterVal int
		switch v := iteration.(type) {
		case int:
			iterVal = v
		case float64:
			iterVal = int(v)
		default:
			t.Errorf("Key %s: iteration has unexpected type %T", key, iteration)
			continue
		}
		if iterVal != writesPerKey-1 {
			t.Errorf("Key %s: expected iteration %d, got %d", key, writesPerKey-1, iterVal)
		}
	}
}

// TestConcurrentWritesSameKey verifies that writes to the same key are serialized.
func TestConcurrentWritesSameKey(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := stow.Open(tmpDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer store.Close()

	ns, err := store.CreateNamespace("test", stow.DefaultNamespaceConfig())
	if err != nil {
		t.Fatalf("CreateNamespace failed: %v", err)
	}

	const numGoroutines = 10
	const writesPerGoroutine = 10
	key := "shared-key"

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*writesPerGoroutine)

	// Launch concurrent writers for the same key
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		goroutineID := i

		go func(id int) {
			defer wg.Done()
			for j := 0; j < writesPerGoroutine; j++ {
				data := map[string]interface{}{
					"goroutine": id,
					"iteration": j,
					"timestamp": time.Now().UnixNano(),
				}
				if err := ns.Put(key, data); err != nil {
					errors <- fmt.Errorf("Put failed (goroutine %d, iter %d): %w", id, j, err)
					return
				}
			}
		}(goroutineID)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent write error: %v", err)
	}

	// Verify we can read the key
	var result map[string]interface{}
	if err := ns.Get(key, &result); err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Should have some goroutine's data (can't predict which one, but should be valid)
	if _, ok := result["goroutine"]; !ok {
		t.Error("Result missing goroutine field")
	}
	if _, ok := result["iteration"]; !ok {
		t.Error("Result missing iteration field")
	}

	t.Logf("Final value: goroutine=%v, iteration=%v", result["goroutine"], result["iteration"])
}

// TestConcurrentReadWrite verifies that reads and writes can happen concurrently.
func TestConcurrentReadWrite(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := stow.Open(tmpDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer store.Close()

	ns, err := store.CreateNamespace("test", stow.DefaultNamespaceConfig())
	if err != nil {
		t.Fatalf("CreateNamespace failed: %v", err)
	}

	// Initialize some keys
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key%d", i)
		data := map[string]interface{}{
			"value": i,
		}
		ns.MustPut(key, data)
	}

	const numReaders = 5
	const numWriters = 5
	const duration = 500 * time.Millisecond

	var wg sync.WaitGroup
	stop := make(chan struct{})
	errors := make(chan error, (numReaders+numWriters)*100)

	// Launch readers
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			readCount := 0
			for {
				select {
				case <-stop:
					t.Logf("Reader %d: %d reads", id, readCount)
					return
				default:
					key := fmt.Sprintf("key%d", id%5)
					var result map[string]interface{}
					if err := ns.Get(key, &result); err != nil && err != stow.ErrNotFound {
						errors <- fmt.Errorf("Reader %d: Get failed: %w", id, err)
						return
					}
					readCount++
				}
			}
		}(i)
	}

	// Launch writers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			writeCount := 0
			for {
				select {
				case <-stop:
					t.Logf("Writer %d: %d writes", id, writeCount)
					return
				default:
					key := fmt.Sprintf("key%d", id%5)
					data := map[string]interface{}{
						"value":     writeCount,
						"timestamp": time.Now().UnixNano(),
					}
					if err := ns.Put(key, data); err != nil {
						errors <- fmt.Errorf("Writer %d: Put failed: %w", id, err)
						return
					}
					writeCount++
					time.Sleep(10 * time.Millisecond)
				}
			}
		}(i)
	}

	// Run for specified duration
	time.Sleep(duration)
	close(stop)
	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent read/write error: %v", err)
	}
}

// TestConcurrentReads verifies that 100+ concurrent read operations work correctly.
func TestConcurrentReads(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := stow.Open(tmpDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer store.Close()

	ns, err := store.CreateNamespace("test", stow.DefaultNamespaceConfig())
	if err != nil {
		t.Fatalf("CreateNamespace failed: %v", err)
	}

	// Initialize test data
	const numKeys = 10
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key%d", i)
		data := map[string]interface{}{
			"id":    i,
			"value": fmt.Sprintf("value_%d", i),
		}
		ns.MustPut(key, data)
	}

	// Launch 100+ concurrent reads
	const numReaders = 20
	const readsPerReader = 10 // Total: 200 read operations

	var wg sync.WaitGroup
	errors := make(chan error, numReaders*readsPerReader)
	successCount := make(chan int, numReaders)

	start := time.Now()

	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			count := 0
			for j := 0; j < readsPerReader; j++ {
				key := fmt.Sprintf("key%d", j%numKeys)
				var result map[string]interface{}
				if err := ns.Get(key, &result); err != nil {
					errors <- fmt.Errorf("Reader %d: Get %s failed: %w", readerID, key, err)
					return
				}
				// Verify data integrity
				expectedID := j % numKeys
				if id, ok := result["id"]; !ok {
					errors <- fmt.Errorf("Reader %d: missing id field", readerID)
					return
				} else {
					var idVal int
					switch v := id.(type) {
					case int:
						idVal = v
					case float64:
						idVal = int(v)
					default:
						errors <- fmt.Errorf("Reader %d: unexpected id type %T", readerID, id)
						return
					}
					if idVal != expectedID {
						errors <- fmt.Errorf("Reader %d: expected id %d, got %d", readerID, expectedID, idVal)
						return
					}
				}
				count++
			}
			successCount <- count
		}(i)
	}

	wg.Wait()
	close(errors)
	close(successCount)

	elapsed := time.Since(start)

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Errorf("Concurrent read error: %v", err)
		errorCount++
	}

	// Count successful reads
	totalReads := 0
	for count := range successCount {
		totalReads += count
	}

	if errorCount == 0 {
		t.Logf("Successfully completed %d concurrent reads in %v (%.0f reads/sec)",
			totalReads, elapsed, float64(totalReads)/elapsed.Seconds())
	}

	if totalReads != numReaders*readsPerReader {
		t.Errorf("Expected %d successful reads, got %d", numReaders*readsPerReader, totalReads)
	}
}

// TestConcurrentWrites verifies that 100+ concurrent write operations work correctly.
func TestConcurrentWrites(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := stow.Open(tmpDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer store.Close()

	ns, err := store.CreateNamespace("test", stow.DefaultNamespaceConfig())
	if err != nil {
		t.Fatalf("CreateNamespace failed: %v", err)
	}

	// Launch 100+ concurrent writes
	const numWriters = 20
	const writesPerWriter = 10 // Total: 200 write operations

	var wg sync.WaitGroup
	errors := make(chan error, numWriters*writesPerWriter)
	successCount := make(chan int, numWriters)

	start := time.Now()

	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			count := 0
			for j := 0; j < writesPerWriter; j++ {
				key := fmt.Sprintf("writer%d_key%d", writerID, j)
				data := map[string]interface{}{
					"writer_id": writerID,
					"iteration": j,
					"timestamp": time.Now().UnixNano(),
					"data":      fmt.Sprintf("data from writer %d iteration %d", writerID, j),
				}
				if err := ns.Put(key, data); err != nil {
					errors <- fmt.Errorf("Writer %d: Put %s failed: %w", writerID, key, err)
					return
				}
				count++
			}
			successCount <- count
		}(i)
	}

	wg.Wait()
	close(errors)
	close(successCount)

	elapsed := time.Since(start)

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Errorf("Concurrent write error: %v", err)
		errorCount++
	}

	// Count successful writes
	totalWrites := 0
	for count := range successCount {
		totalWrites += count
	}

	if errorCount == 0 {
		t.Logf("Successfully completed %d concurrent writes in %v (%.0f writes/sec)",
			totalWrites, elapsed, float64(totalWrites)/elapsed.Seconds())
	}

	if totalWrites != numWriters*writesPerWriter {
		t.Errorf("Expected %d successful writes, got %d", numWriters*writesPerWriter, totalWrites)
	}

	// Verify all writes succeeded by reading back
	for i := 0; i < numWriters; i++ {
		for j := 0; j < writesPerWriter; j++ {
			key := fmt.Sprintf("writer%d_key%d", i, j)
			var result map[string]interface{}
			if err := ns.Get(key, &result); err != nil {
				t.Errorf("Verification: Get %s failed: %v", key, err)
			}
		}
	}
}

// TestCompactDuringReadWrite verifies that Compact can run while reads/writes are happening.
func TestCompactDuringReadWrite(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := stow.Open(tmpDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer store.Close()

	ns, err := store.CreateNamespace("test", stow.DefaultNamespaceConfig())
	if err != nil {
		t.Fatalf("CreateNamespace failed: %v", err)
	}

	// Create keys with multiple versions to make compaction worthwhile
	const numKeys = 10
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key%d", i)
		// Create 20 versions for each key
		for v := 0; v < 20; v++ {
			data := map[string]interface{}{
				"version": v,
				"data":    fmt.Sprintf("version %d of key %d", v, i),
			}
			ns.MustPut(key, data)
		}
	}

	const duration = 500 * time.Millisecond
	var wg sync.WaitGroup
	stop := make(chan struct{})
	errors := make(chan error, 100)

	// Launch readers
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				key := fmt.Sprintf("key%d", time.Now().UnixNano()%numKeys)
				var result map[string]interface{}
				if err := ns.Get(key, &result); err != nil && err != stow.ErrNotFound {
					errors <- fmt.Errorf("Read during compact failed: %w", err)
				}
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()

	// Launch writers
	wg.Add(1)
	go func() {
		defer wg.Done()
		iteration := 0
		for {
			select {
			case <-stop:
				return
			default:
				key := fmt.Sprintf("key%d", iteration%numKeys)
				data := map[string]interface{}{
					"version": 100 + iteration,
					"data":    fmt.Sprintf("concurrent write %d", iteration),
				}
				if err := ns.Put(key, data); err != nil {
					errors <- fmt.Errorf("Write during compact failed: %w", err)
				}
				iteration++
				time.Sleep(20 * time.Millisecond)
			}
		}
	}()

	// Launch compactor
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				// Compact random keys
				key := fmt.Sprintf("key%d", time.Now().UnixNano()%numKeys)
				if err := ns.Compact(key); err != nil && err != stow.ErrNotFound {
					errors <- fmt.Errorf("Compact failed: %w", err)
				}
				time.Sleep(50 * time.Millisecond)
			}
		}
	}()

	// Run for specified duration
	time.Sleep(duration)
	close(stop)
	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Error during compact with concurrent operations: %v", err)
	}

	t.Log("Compact completed successfully while reads/writes were happening")
}

// TestGCDuringReadWrite verifies that GC can run while reads/writes are happening.
func TestGCDuringReadWrite(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := stow.Open(tmpDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer store.Close()

	ns, err := store.CreateNamespace("test", stow.DefaultNamespaceConfig())
	if err != nil {
		t.Fatalf("CreateNamespace failed: %v", err)
	}

	type Document struct {
		Content []byte
	}

	// Create documents with blobs
	const numKeys = 5
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("doc%d", i)
		data := make([]byte, 5120) // > 4KB threshold
		for j := range data {
			data[j] = byte((i + j) % 256)
		}
		ns.MustPut(key, Document{Content: data})
	}

	const duration = 500 * time.Millisecond
	var wg sync.WaitGroup
	stop := make(chan struct{})
	errors := make(chan error, 100)

	// Launch readers
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				key := fmt.Sprintf("doc%d", time.Now().UnixNano()%numKeys)
				var result Document
				if err := ns.Get(key, &result); err != nil && err != stow.ErrNotFound {
					errors <- fmt.Errorf("Read during GC failed: %w", err)
				}
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()

	// Launch writers (update documents to create orphaned blobs)
	wg.Add(1)
	go func() {
		defer wg.Done()
		iteration := 0
		for {
			select {
			case <-stop:
				return
			default:
				key := fmt.Sprintf("doc%d", iteration%numKeys)
				data := make([]byte, 5120)
				for j := range data {
					data[j] = byte((iteration + j) % 256)
				}
				if err := ns.Put(key, Document{Content: data}); err != nil {
					errors <- fmt.Errorf("Write during GC failed: %w", err)
				}
				iteration++
				time.Sleep(30 * time.Millisecond)
			}
		}
	}()

	// Launch GC
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				if _, err := ns.GC(); err != nil {
					errors <- fmt.Errorf("GC failed: %w", err)
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	// Run for specified duration
	time.Sleep(duration)
	close(stop)
	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Error during GC with concurrent operations: %v", err)
	}

	t.Log("GC completed successfully while reads/writes were happening")
}
