package codec

import (
	"testing"
)

// ========== ParseStowTag Tests ==========

func TestParseStowTagBasic(t *testing.T) {
	tests := []struct {
		tag      string
		expected TagInfo
	}{
		{
			tag: "",
			expected: TagInfo{
				IsFile:    false,
				Name:      "",
				NameField: "",
				MimeType:  "",
			},
		},
		{
			tag: "file",
			expected: TagInfo{
				IsFile:    true,
				Name:      "",
				NameField: "",
				MimeType:  "",
			},
		},
		{
			tag: "file,name:avatar.jpg",
			expected: TagInfo{
				IsFile:    true,
				Name:      "avatar.jpg",
				NameField: "",
				MimeType:  "",
			},
		},
		{
			tag: "file,name_field:FileName",
			expected: TagInfo{
				IsFile:    true,
				Name:      "",
				NameField: "FileName",
				MimeType:  "",
			},
		},
		{
			tag: "file,mime:image/jpeg",
			expected: TagInfo{
				IsFile:    true,
				Name:      "",
				NameField: "",
				MimeType:  "image/jpeg",
			},
		},
		{
			tag: "file,name:avatar.jpg,mime:image/jpeg",
			expected: TagInfo{
				IsFile:    true,
				Name:      "avatar.jpg",
				NameField: "",
				MimeType:  "image/jpeg",
			},
		},
		{
			tag: "file,name_field:FileName,mime:image/jpeg",
			expected: TagInfo{
				IsFile:    true,
				Name:      "",
				NameField: "FileName",
				MimeType:  "image/jpeg",
			},
		},
	}

	for _, tt := range tests {
		result := ParseStowTag(tt.tag)
		if result.IsFile != tt.expected.IsFile {
			t.Errorf("ParseStowTag(%q).IsFile = %v, want %v", tt.tag, result.IsFile, tt.expected.IsFile)
		}
		if result.Name != tt.expected.Name {
			t.Errorf("ParseStowTag(%q).Name = %q, want %q", tt.tag, result.Name, tt.expected.Name)
		}
		if result.NameField != tt.expected.NameField {
			t.Errorf("ParseStowTag(%q).NameField = %q, want %q", tt.tag, result.NameField, tt.expected.NameField)
		}
		if result.MimeType != tt.expected.MimeType {
			t.Errorf("ParseStowTag(%q).MimeType = %q, want %q", tt.tag, result.MimeType, tt.expected.MimeType)
		}
	}
}

func TestParseStowTagEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected TagInfo
	}{
		{
			name: "extra whitespace",
			tag:  "file , name: avatar.jpg , mime: image/jpeg",
			expected: TagInfo{
				IsFile:    true,
				Name:      "avatar.jpg",
				NameField: "",
				MimeType:  "image/jpeg",
			},
		},
		{
			name: "invalid key-value format",
			tag:  "file,invalid:key:value",
			expected: TagInfo{
				IsFile:    true,
				Name:      "",
				NameField: "",
				MimeType:  "",
			},
		},
		{
			name: "empty value",
			tag:  "file,name:",
			expected: TagInfo{
				IsFile:    true,
				Name:      "",
				NameField: "",
				MimeType:  "",
			},
		},
		{
			name: "unknown option",
			tag:  "file,unknown:value",
			expected: TagInfo{
				IsFile:    true,
				Name:      "",
				NameField: "",
				MimeType:  "",
			},
		},
		{
			name: "unicode characters",
			tag:  "file,name:文件名.txt",
			expected: TagInfo{
				IsFile:    true,
				Name:      "文件名.txt",
				NameField: "",
				MimeType:  "",
			},
		},
		{
			name: "special characters in value",
			tag:  "file,name:my-file_v2.0.txt",
			expected: TagInfo{
				IsFile:    true,
				Name:      "my-file_v2.0.txt",
				NameField: "",
				MimeType:  "",
			},
		},
		{
			name: "duplicate options",
			tag:  "file,name:first.txt,name:second.txt",
			expected: TagInfo{
				IsFile:    true,
				Name:      "second.txt", // Last one wins
				NameField: "",
				MimeType:  "",
			},
		},
		{
			name: "only options without file",
			tag:  "name:test.txt",
			expected: TagInfo{
				IsFile:    false,
				Name:      "test.txt",
				NameField: "",
				MimeType:  "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseStowTag(tt.tag)
			if result.IsFile != tt.expected.IsFile {
				t.Errorf("IsFile = %v, want %v", result.IsFile, tt.expected.IsFile)
			}
			if result.Name != tt.expected.Name {
				t.Errorf("Name = %q, want %q", result.Name, tt.expected.Name)
			}
			if result.NameField != tt.expected.NameField {
				t.Errorf("NameField = %q, want %q", result.NameField, tt.expected.NameField)
			}
			if result.MimeType != tt.expected.MimeType {
				t.Errorf("MimeType = %q, want %q", result.MimeType, tt.expected.MimeType)
			}
		})
	}
}

// ========== HasStowTag Tests ==========

func TestHasStowTag(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected bool
	}{
		{
			name:     "empty tag",
			tag:      "",
			expected: false,
		},
		{
			name:     "file tag",
			tag:      "file",
			expected: true,
		},
		{
			name:     "file with options",
			tag:      "file,name:avatar.jpg",
			expected: true,
		},
		{
			name:     "only options",
			tag:      "name:test.txt",
			expected: true,
		},
		{
			name:     "whitespace only",
			tag:      "   ",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasStowTag(tt.tag)
			if result != tt.expected {
				t.Errorf("HasStowTag(%q) = %v, want %v", tt.tag, result, tt.expected)
			}
		})
	}
}

// ========== TagInfo.IsEmpty Tests ==========

func TestTagInfo_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		tagInfo  TagInfo
		expected bool
	}{
		{
			name:     "empty tag info",
			tagInfo:  TagInfo{},
			expected: true,
		},
		{
			name: "with IsFile",
			tagInfo: TagInfo{
				IsFile: true,
			},
			expected: false,
		},
		{
			name: "with Name",
			tagInfo: TagInfo{
				Name: "test.txt",
			},
			expected: false,
		},
		{
			name: "with NameField",
			tagInfo: TagInfo{
				NameField: "FileName",
			},
			expected: false,
		},
		{
			name: "with MimeType",
			tagInfo: TagInfo{
				MimeType: "image/jpeg",
			},
			expected: false,
		},
		{
			name: "with all fields",
			tagInfo: TagInfo{
				IsFile:    true,
				Name:      "test.txt",
				NameField: "FileName",
				MimeType:  "image/jpeg",
			},
			expected: false,
		},
		{
			name: "with some fields",
			tagInfo: TagInfo{
				IsFile: true,
				Name:   "test.txt",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.tagInfo.IsEmpty()
			if result != tt.expected {
				t.Errorf("TagInfo.IsEmpty() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// ========== TagInfo.ShouldStoreAsBlob Tests ==========

func TestTagInfo_ShouldStoreAsBlob(t *testing.T) {
	tests := []struct {
		name     string
		tagInfo  TagInfo
		expected bool
	}{
		{
			name:     "empty tag info",
			tagInfo:  TagInfo{},
			expected: false,
		},
		{
			name: "with IsFile true",
			tagInfo: TagInfo{
				IsFile: true,
			},
			expected: true,
		},
		{
			name: "with IsFile false",
			tagInfo: TagInfo{
				IsFile: false,
			},
			expected: false,
		},
		{
			name: "with file and all options",
			tagInfo: TagInfo{
				IsFile:    true,
				Name:      "test.txt",
				NameField: "FileName",
				MimeType:  "text/plain",
			},
			expected: true,
		},
		{
			name: "with only Name (no file)",
			tagInfo: TagInfo{
				Name: "test.txt",
			},
			expected: false,
		},
		{
			name: "with only NameField (no file)",
			tagInfo: TagInfo{
				NameField: "FileName",
			},
			expected: false,
		},
		{
			name: "with only MimeType (no file)",
			tagInfo: TagInfo{
				MimeType: "text/plain",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.tagInfo.ShouldStoreAsBlob()
			if result != tt.expected {
				t.Errorf("TagInfo.ShouldStoreAsBlob() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// ========== Integration Tests ==========

func TestParseStowTagAndMethods(t *testing.T) {
	tests := []struct {
		name             string
		tag              string
		shouldHaveTag    bool
		shouldBeEmpty    bool
		shouldStoreBlob  bool
	}{
		{
			name:             "empty string",
			tag:              "",
			shouldHaveTag:    false,
			shouldBeEmpty:    true,
			shouldStoreBlob:  false,
		},
		{
			name:             "file only",
			tag:              "file",
			shouldHaveTag:    true,
			shouldBeEmpty:    false,
			shouldStoreBlob:  true,
		},
		{
			name:             "file with name",
			tag:              "file,name:avatar.jpg",
			shouldHaveTag:    true,
			shouldBeEmpty:    false,
			shouldStoreBlob:  true,
		},
		{
			name:             "name only (no file)",
			tag:              "name:test.txt",
			shouldHaveTag:    true,
			shouldBeEmpty:    false,
			shouldStoreBlob:  false,
		},
		{
			name:             "mime only (no file)",
			tag:              "mime:image/jpeg",
			shouldHaveTag:    true,
			shouldBeEmpty:    false,
			shouldStoreBlob:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test HasStowTag
			hasTag := HasStowTag(tt.tag)
			if hasTag != tt.shouldHaveTag {
				t.Errorf("HasStowTag(%q) = %v, want %v", tt.tag, hasTag, tt.shouldHaveTag)
			}

			// Parse tag
			tagInfo := ParseStowTag(tt.tag)

			// Test IsEmpty
			isEmpty := tagInfo.IsEmpty()
			if isEmpty != tt.shouldBeEmpty {
				t.Errorf("IsEmpty() = %v, want %v", isEmpty, tt.shouldBeEmpty)
			}

			// Test ShouldStoreAsBlob
			shouldStore := tagInfo.ShouldStoreAsBlob()
			if shouldStore != tt.shouldStoreBlob {
				t.Errorf("ShouldStoreAsBlob() = %v, want %v", shouldStore, tt.shouldStoreBlob)
			}
		})
	}
}
