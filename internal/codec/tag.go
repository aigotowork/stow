// Package codec provides serialization and deserialization for Stow.
package codec

import (
	"strings"
)

// TagInfo contains parsed information from a stow struct tag.
//
// Supported tag format:
//   stow:"file,name:avatar.jpg,name_field:FileName,mime:image/jpeg"
//
// Options:
//   - file: mark this field as a blob file
//   - name:xxx: specify custom file name
//   - name_field:FieldName: use another field's value as file name
//   - mime:xxx: specify MIME type
type TagInfo struct {
	// IsFile indicates if this field should be stored as a blob file
	IsFile bool

	// Name is the custom file name (e.g., "avatar.jpg")
	Name string

	// NameField is the name of another field to use as the file name
	NameField string

	// MimeType is the MIME type (e.g., "image/jpeg")
	MimeType string
}

// ParseStowTag parses a stow struct tag.
//
// Example tags:
//   - `stow:"file"` -> IsFile=true
//   - `stow:"file,name:avatar.jpg"` -> IsFile=true, Name="avatar.jpg"
//   - `stow:"file,name_field:FileName"` -> IsFile=true, NameField="FileName"
//   - `stow:"file,mime:image/jpeg"` -> IsFile=true, MimeType="image/jpeg"
func ParseStowTag(tag string) TagInfo {
	info := TagInfo{}

	if tag == "" {
		return info
	}

	// Split by comma
	parts := strings.Split(tag, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)

		if part == "file" {
			info.IsFile = true
			continue
		}

		// Check for key:value pairs
		if strings.Contains(part, ":") {
			kv := strings.SplitN(part, ":", 2)
			if len(kv) != 2 {
				continue
			}

			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])

			switch key {
			case "name":
				info.Name = value
			case "name_field":
				info.NameField = value
			case "mime":
				info.MimeType = value
			}
		}
	}

	return info
}

// HasStowTag checks if a struct tag contains a stow tag.
func HasStowTag(tag string) bool {
	return tag != ""
}

// IsEmpty checks if the tag info is empty (no options set).
func (t *TagInfo) IsEmpty() bool {
	return !t.IsFile && t.Name == "" && t.NameField == "" && t.MimeType == ""
}

// ShouldStoreAsBlob determines if a field should be stored as a blob based on tag info.
// This only checks the tag; actual decision also depends on data type and size.
func (t *TagInfo) ShouldStoreAsBlob() bool {
	return t.IsFile
}
