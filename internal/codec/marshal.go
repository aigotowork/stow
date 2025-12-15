package codec

import (
	"bytes"
	"fmt"
	"io"

	"github.com/aigotowork/stow/internal/blob"
)

// MarshalOptions contains options for marshaling.
type MarshalOptions struct {
	BlobThreshold int64
	ForceFile     bool
	ForceInline   bool
	FileName      string
	MimeType      string
}

// Marshaler handles serialization of values to map[string]interface{}.
// It identifies blob fields and stores them using the blob manager.
type Marshaler struct {
	blobManager *blob.Manager
}

// NewMarshaler creates a new marshaler.
func NewMarshaler(blobManager *blob.Manager) *Marshaler {
	return &Marshaler{
		blobManager: blobManager,
	}
}

// Marshal marshals a value to map[string]interface{}, storing large data as blobs.
//
// Returns:
//   - data: the serialized data (with blob references replacing large fields)
//   - blobRefs: list of blob references created
//   - error: any error that occurred
func (m *Marshaler) Marshal(value interface{}, opts MarshalOptions) (map[string]interface{}, []*blob.Reference, error) {
	// Convert value to map
	data, err := ToMap(value)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to convert to map: %w", err)
	}

	var blobRefs []*blob.Reference

	// Process each field to detect blobs
	for key, fieldValue := range data {
		// Check if this field should be stored as a blob
		shouldStore, blobData := m.shouldStoreAsBlob(fieldValue, opts)
		if !shouldStore {
			continue
		}

		// Store as blob
		ref, err := m.storeBlob(blobData, opts)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to store blob for field %s: %w", key, err)
		}

		// Replace field value with blob reference
		data[key] = ref.ToMap()
		blobRefs = append(blobRefs, ref)
	}

	return data, blobRefs, nil
}

// shouldStoreAsBlob determines if a field value should be stored as a blob.
func (m *Marshaler) shouldStoreAsBlob(value interface{}, opts MarshalOptions) (bool, interface{}) {
	if value == nil {
		return false, nil
	}

	// Check if it's already a blob reference
	if m, ok := value.(map[string]interface{}); ok {
		if blob.IsBlobReference(m) {
			return false, nil
		}
	}

	// If ForceInline is set, never store as blob
	if opts.ForceInline {
		return false, nil
	}

	// Check for io.Reader
	if reader, ok := value.(io.Reader); ok {
		return true, reader
	}

	// Check for []byte
	if bytes, ok := value.([]byte); ok {
		if opts.ForceFile || int64(len(bytes)) > opts.BlobThreshold {
			return true, bytes
		}
	}

	return false, nil
}

// storeBlob stores data as a blob file.
func (m *Marshaler) storeBlob(data interface{}, opts MarshalOptions) (*blob.Reference, error) {
	return m.blobManager.Store(data, opts.FileName, opts.MimeType)
}

// MarshalSimple marshals simple values (non-struct) to interface{}.
// For maps and basic types, returns them as-is.
// For []byte larger than threshold, stores as blob.
func (m *Marshaler) MarshalSimple(value interface{}, opts MarshalOptions) (interface{}, []*blob.Reference, error) {
	if value == nil {
		return nil, nil, nil
	}

	// Check if should be blob
	shouldStore, blobData := m.shouldStoreAsBlob(value, opts)
	if shouldStore {
		ref, err := m.storeBlob(blobData, opts)
		if err != nil {
			return nil, nil, err
		}
		return ref.ToMap(), []*blob.Reference{ref}, nil
	}

	return value, nil, nil
}

// MarshalBytes marshals a []byte value, potentially as a blob.
func (m *Marshaler) MarshalBytes(data []byte, opts MarshalOptions) (interface{}, *blob.Reference, error) {
	if opts.ForceFile || int64(len(data)) > opts.BlobThreshold {
		ref, err := m.blobManager.Store(data, opts.FileName, opts.MimeType)
		if err != nil {
			return nil, nil, err
		}
		return ref.ToMap(), ref, nil
	}

	// Store inline
	return data, nil, nil
}

// MarshalReader marshals an io.Reader as a blob.
func (m *Marshaler) MarshalReader(reader io.Reader, opts MarshalOptions) (interface{}, *blob.Reference, error) {
	ref, err := m.blobManager.Store(reader, opts.FileName, opts.MimeType)
	if err != nil {
		return nil, nil, err
	}
	return ref.ToMap(), ref, nil
}

// Quick helper to store bytes as blob
func (m *Marshaler) StoreBytesAsBlob(data []byte, name, mimeType string) (*blob.Reference, error) {
	return m.blobManager.Store(bytes.NewReader(data), name, mimeType)
}
