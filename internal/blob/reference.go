// Package blob provides blob file management for large binary data.
package blob

// Reference represents a reference to a blob file in JSONL records.
// Instead of storing large binary data inline, we store a reference that points to
// a file in the _blobs/ directory.
//
// Example JSON representation:
//
//	{
//	  "$blob": true,
//	  "loc": "_blobs/avatar_abc123.jpg",
//	  "hash": "abc123def456...",
//	  "size": 102400,
//	  "mime": "image/jpeg",
//	  "name": "avatar.jpg"
//	}
type Reference struct {
	// IsBlob marks this as a blob reference (always true)
	IsBlob bool `json:"$blob"`

	// Location is the relative path to the blob file (e.g., "_blobs/file_abc123.jpg")
	Location string `json:"loc"`

	// Hash is the SHA256 hash of the file content
	Hash string `json:"hash"`

	// Size is the file size in bytes
	Size int64 `json:"size"`

	// MimeType is the MIME type of the file (e.g., "image/jpeg")
	MimeType string `json:"mime,omitempty"`

	// Name is the original file name (e.g., "avatar.jpg")
	Name string `json:"name,omitempty"`
}

// NewReference creates a new blob reference.
func NewReference(location, hash string, size int64, mimeType, name string) *Reference {
	return &Reference{
		IsBlob:   true,
		Location: location,
		Hash:     hash,
		Size:     size,
		MimeType: mimeType,
		Name:     name,
	}
}

// IsValid checks if the reference is valid.
func (r *Reference) IsValid() bool {
	return r.IsBlob && r.Location != "" && r.Hash != "" && r.Size >= 0
}

// IsBlobReference checks if a map represents a blob reference.
// This is used during deserialization to detect blob references.
func IsBlobReference(data map[string]interface{}) bool {
	isBlob, ok := data["$blob"].(bool)
	return ok && isBlob
}

// FromMap creates a Reference from a map (used during deserialization).
func FromMap(data map[string]interface{}) (*Reference, bool) {
	if !IsBlobReference(data) {
		return nil, false
	}

	ref := &Reference{
		IsBlob: true,
	}

	if loc, ok := data["loc"].(string); ok {
		ref.Location = loc
	}

	if hash, ok := data["hash"].(string); ok {
		ref.Hash = hash
	}

	if size, ok := data["size"].(float64); ok {
		ref.Size = int64(size)
	}

	if mime, ok := data["mime"].(string); ok {
		ref.MimeType = mime
	}

	if name, ok := data["name"].(string); ok {
		ref.Name = name
	}

	if !ref.IsValid() {
		return nil, false
	}

	return ref, true
}

// ToMap converts a Reference to a map (used during serialization).
func (r *Reference) ToMap() map[string]interface{} {
	m := map[string]interface{}{
		"$blob": true,
		"loc":   r.Location,
		"hash":  r.Hash,
		"size":  r.Size,
	}

	if r.MimeType != "" {
		m["mime"] = r.MimeType
	}

	if r.Name != "" {
		m["name"] = r.Name
	}

	return m
}
