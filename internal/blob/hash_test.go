package blob

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestComputeSHA256Errors tests error handling in ComputeSHA256
func TestComputeSHA256Errors(t *testing.T) {
	t.Run("reader error", func(t *testing.T) {
		// Create a reader that always returns an error
		errReader := &errorReader{err: errors.New("read error")}
		_, err := ComputeSHA256(errReader)
		if err == nil {
			t.Error("Expected error from failing reader")
		}
	})

	t.Run("empty reader", func(t *testing.T) {
		emptyReader := bytes.NewReader([]byte{})
		hash, err := ComputeSHA256(emptyReader)
		if err != nil {
			t.Errorf("ComputeSHA256 with empty reader failed: %v", err)
		}
		// Empty data should produce specific SHA256 hash
		expectedEmptyHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		if hash != expectedEmptyHash {
			t.Errorf("Empty hash = %q, want %q", hash, expectedEmptyHash)
		}
	})

	t.Run("large file", func(t *testing.T) {
		// Create large data (10MB)
		largeData := make([]byte, 10*1024*1024)
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}

		reader := bytes.NewReader(largeData)
		hash, err := ComputeSHA256(reader)
		if err != nil {
			t.Errorf("ComputeSHA256 with large data failed: %v", err)
		}
		if hash == "" {
			t.Error("Hash should not be empty")
		}
		if len(hash) != 64 {
			t.Errorf("Hash length = %d, want 64", len(hash))
		}
	})

	t.Run("file not found", func(t *testing.T) {
		// Try to open and hash non-existent file
		file, err := os.Open("/nonexistent/file.bin")
		if err == nil {
			defer file.Close()
			t.Fatal("Expected error opening non-existent file")
		}
		// This test just verifies the error path exists
	})
}

// errorReader is a helper that always returns an error
type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}

// TestComputeSHA256Consistency tests hash consistency
func TestComputeSHA256Consistency(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "same content produces same hash",
			data: []byte("consistent data"),
		},
		{
			name: "empty data",
			data: []byte{},
		},
		{
			name: "single byte",
			data: []byte{0xFF},
		},
		{
			name: "binary data",
			data: []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Compute hash twice
			hash1, err1 := ComputeSHA256(bytes.NewReader(tt.data))
			hash2, err2 := ComputeSHA256(bytes.NewReader(tt.data))

			if err1 != nil || err2 != nil {
				t.Fatalf("Hash computation failed: %v, %v", err1, err2)
			}

			if hash1 != hash2 {
				t.Errorf("Inconsistent hashes: %q != %q", hash1, hash2)
			}

			// Verify hash using FromBytes
			hash3 := ComputeSHA256FromBytes(tt.data)
			if hash1 != hash3 {
				t.Errorf("ComputeSHA256 and ComputeSHA256FromBytes produce different hashes: %q != %q", hash1, hash3)
			}
		})
	}
}

// TestComputeSHA256DifferentContent tests that different content produces different hashes
func TestComputeSHA256DifferentContent(t *testing.T) {
	testData := [][]byte{
		[]byte("data1"),
		[]byte("data2"),
		[]byte("data1 "), // Note the trailing space
		[]byte("DATA1"),  // Different case
		[]byte{0x01},
		[]byte{0x02},
	}

	hashes := make(map[string]bool)

	for i, data := range testData {
		hash, err := ComputeSHA256(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("Hash computation failed for data[%d]: %v", i, err)
		}

		if hashes[hash] {
			t.Errorf("Collision detected! Data[%d] produces same hash as previous data: %q", i, hash)
		}
		hashes[hash] = true
	}
}

// TestComputeSHA256FromFile tests hashing actual files
func TestComputeSHA256FromFile(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("regular file", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "test.txt")
		testData := []byte("file content for hashing")
		if err := os.WriteFile(testFile, testData, 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		file, err := os.Open(testFile)
		if err != nil {
			t.Fatalf("Failed to open test file: %v", err)
		}
		defer file.Close()

		hash, err := ComputeSHA256(file)
		if err != nil {
			t.Errorf("ComputeSHA256 failed: %v", err)
		}

		// Compare with byte hash
		expectedHash := ComputeSHA256FromBytes(testData)
		if hash != expectedHash {
			t.Errorf("File hash = %q, want %q", hash, expectedHash)
		}
	})

	t.Run("empty file", func(t *testing.T) {
		emptyFile := filepath.Join(tmpDir, "empty.txt")
		if err := os.WriteFile(emptyFile, []byte{}, 0644); err != nil {
			t.Fatalf("Failed to create empty file: %v", err)
		}

		file, err := os.Open(emptyFile)
		if err != nil {
			t.Fatalf("Failed to open empty file: %v", err)
		}
		defer file.Close()

		hash, err := ComputeSHA256(file)
		if err != nil {
			t.Errorf("ComputeSHA256 of empty file failed: %v", err)
		}

		expectedEmptyHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		if hash != expectedEmptyHash {
			t.Errorf("Empty file hash = %q, want %q", hash, expectedEmptyHash)
		}
	})
}

// TestHashPrefixLength tests HashPrefix with various lengths
func TestHashPrefixLength(t *testing.T) {
	fullHash := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"

	tests := []struct {
		name     string
		hash     string
		n        int
		expected string
	}{
		{
			name:     "zero length",
			hash:     fullHash,
			n:        0,
			expected: fullHash, // Returns full hash for n <= 0
		},
		{
			name:     "negative length",
			hash:     fullHash,
			n:        -1,
			expected: fullHash,
		},
		{
			name:     "normal prefix",
			hash:     fullHash,
			n:        8,
			expected: "abcdef01",
		},
		{
			name:     "full length",
			hash:     fullHash,
			n:        len(fullHash),
			expected: fullHash,
		},
		{
			name:     "exceeds length",
			hash:     fullHash,
			n:        1000,
			expected: fullHash,
		},
		{
			name:     "empty hash",
			hash:     "",
			n:        8,
			expected: "",
		},
		{
			name:     "short hash",
			hash:     "abc",
			n:        5,
			expected: "abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HashPrefix(tt.hash, tt.n)
			if result != tt.expected {
				t.Errorf("HashPrefix(%q, %d) = %q, want %q", tt.hash, tt.n, result, tt.expected)
			}
		})
	}
}

// TestShortHashUniqueness tests that short hashes maintain uniqueness
func TestShortHashUniqueness(t *testing.T) {
	// Generate multiple different hashes and verify short versions are unique
	testData := []string{
		"data1",
		"data2",
		"data3",
		"similar1",
		"similar2",
	}

	shortHashes := make(map[string]bool)

	for _, data := range testData {
		fullHash := ComputeSHA256FromBytes([]byte(data))
		shortHash := ShortHash(fullHash)

		if len(shortHash) != DefaultHashPrefixLength {
			t.Errorf("ShortHash length = %d, want %d", len(shortHash), DefaultHashPrefixLength)
		}

		if shortHashes[shortHash] {
			t.Errorf("Short hash collision detected for data %q: %q", data, shortHash)
		}
		shortHashes[shortHash] = true

		// Verify it's actually a prefix
		if !strings.HasPrefix(fullHash, shortHash) {
			t.Errorf("Short hash %q is not a prefix of full hash %q", shortHash, fullHash)
		}
	}
}

// TestHashPrefixWithSpecialCharacters tests hash prefix with various hash formats
func TestHashPrefixWithSpecialCharacters(t *testing.T) {
	tests := []struct {
		name string
		hash string
		n    int
		want string
	}{
		{
			name: "hex lowercase",
			hash: "abcdef0123456789",
			n:    8,
			want: "abcdef01",
		},
		{
			name: "hex uppercase",
			hash: "ABCDEF0123456789",
			n:    8,
			want: "ABCDEF01",
		},
		{
			name: "mixed case",
			hash: "AbCdEf0123456789",
			n:    8,
			want: "AbCdEf01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HashPrefix(tt.hash, tt.n)
			if got != tt.want {
				t.Errorf("HashPrefix() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestComputeSHA256FromBytesKnownValues tests against known SHA256 values
func TestComputeSHA256FromBytesKnownValues(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "",
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			input:    "abc",
			expected: "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			hash := ComputeSHA256FromBytes([]byte(tt.input))
			if hash != tt.expected {
				t.Errorf("ComputeSHA256FromBytes(%q) = %q, want %q", tt.input, hash, tt.expected)
			}
		})
	}
}

// TestHashFunctionConsistency tests consistency between ComputeSHA256 and ComputeSHA256FromBytes
func TestHashFunctionConsistency(t *testing.T) {
	testCases := [][]byte{
		[]byte("test data"),
		[]byte(""),
		[]byte{0x00, 0xFF, 0xAA, 0x55},
		bytes.Repeat([]byte("x"), 10000),
	}

	for i, data := range testCases {
		t.Run(string(rune('A'+i)), func(t *testing.T) {
			hash1, err := ComputeSHA256(bytes.NewReader(data))
			if err != nil {
				t.Fatalf("ComputeSHA256 failed: %v", err)
			}

			hash2 := ComputeSHA256FromBytes(data)

			if hash1 != hash2 {
				t.Errorf("Hash mismatch: ComputeSHA256=%q, ComputeSHA256FromBytes=%q", hash1, hash2)
			}
		})
	}
}

// Benchmark tests
func BenchmarkComputeSHA256Small(b *testing.B) {
	data := bytes.Repeat([]byte("x"), 1024) // 1KB
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(data)
		_, _ = ComputeSHA256(reader)
	}
}

func BenchmarkComputeSHA256Medium(b *testing.B) {
	data := bytes.Repeat([]byte("x"), 1024*1024) // 1MB
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(data)
		_, _ = ComputeSHA256(reader)
	}
}

func BenchmarkComputeSHA256FromBytes(b *testing.B) {
	data := bytes.Repeat([]byte("x"), 1024*1024) // 1MB
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = ComputeSHA256FromBytes(data)
	}
}

func BenchmarkShortHash(b *testing.B) {
	hash := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = ShortHash(hash)
	}
}
