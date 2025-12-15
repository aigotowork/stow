package blob

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
)

// ComputeSHA256 computes the SHA256 hash of data from a reader.
// It reads the data in chunks and computes the hash incrementally,
// so it doesn't load the entire file into memory.
//
// Returns the full hex-encoded hash string.
func ComputeSHA256(r io.Reader) (string, error) {
	h := sha256.New()

	if _, err := io.Copy(h, r); err != nil {
		return "", fmt.Errorf("failed to compute hash: %w", err)
	}

	hashBytes := h.Sum(nil)
	return hex.EncodeToString(hashBytes), nil
}

// ComputeSHA256FromBytes computes SHA256 hash from byte slice.
func ComputeSHA256FromBytes(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// HashPrefix returns the first n characters of a hash.
// This is used to generate shorter file names while maintaining uniqueness.
//
// Example: HashPrefix("abc123def456...", 8) returns "abc123de"
func HashPrefix(hash string, n int) string {
	if n <= 0 || n > len(hash) {
		return hash
	}
	return hash[:n]
}

// DefaultHashPrefixLength is the default length for hash prefixes in file names.
// Using 16 characters gives us 64 bits of entropy, which is sufficient for
// avoiding collisions in typical use cases.
const DefaultHashPrefixLength = 16

// ShortHash returns a shortened version of the hash for file names.
func ShortHash(hash string) string {
	return HashPrefix(hash, DefaultHashPrefixLength)
}
