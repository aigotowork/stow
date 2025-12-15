package index

import (
	"testing"
)

// ========== SanitizeKey Tests ==========

func TestSanitizeKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal_key", "normal_key"},
		{"user/data", "user_data"},
		{"path\\to\\file", "path_to_file"},
		{"file:v1", "file_v1"},
		{"query*", "query"},                   // trailing _ trimmed
		{"what?", "what"},                     // trailing _ trimmed
		{"<tag>", "tag"},                      // leading/trailing _ trimmed
		{"file|pipe", "file_pipe"},
		{`"quoted"`, "quoted"},                // leading/trailing _ trimmed
		{"  spaces  ", "spaces"},
		{"___underscores___", "underscores"},
		{"user/path:v1*", "user_path_v1"},     // trailing _ trimmed
	}

	for _, tt := range tests {
		result := SanitizeKey(tt.input)
		if result != tt.expected {
			t.Errorf("SanitizeKey(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSanitizeKeySpecialChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"forward slash", "user/data", "user_data"},
		{"backslash", "path\\file", "path_file"},
		{"colon", "namespace:key", "namespace_key"},
		{"asterisk", "glob*pattern", "glob_pattern"},
		{"question mark", "what?", "what"},
		{"double quote", `"value"`, "value"},
		{"less than", "<tag>", "tag"},
		{"greater than", "value>", "value"},
		{"pipe", "a|b", "a_b"},
		{"multiple special", "a/b\\c:d*e?f", "a_b_c_d_e_f"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeKey(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeKey(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeKeyUnicode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"用户/数据", "用户_数据"},
		{"ファイル:名前", "ファイル_名前"},
		{"пользователь/данные", "пользователь_данные"},
	}

	for _, tt := range tests {
		result := SanitizeKey(tt.input)
		if result != tt.expected {
			t.Errorf("SanitizeKey(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSanitizeKeyEmoji(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user😀data", "user😀data"}, // Emoji preserved
		{"file/😀/name", "file_😀_name"},
	}

	for _, tt := range tests {
		result := SanitizeKey(tt.input)
		if result != tt.expected {
			t.Errorf("SanitizeKey(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSanitizeKeyWhitespace(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  leading", "leading"},
		{"trailing  ", "trailing"},
		{"  both  ", "both"},
		{"inner  spaces", "inner  spaces"}, // Inner spaces preserved
		{"\ttabs\t", "\ttabs\t"}, // Tabs not trimmed by SanitizeKey (only spaces and underscores)
	}

	for _, tt := range tests {
		result := SanitizeKey(tt.input)
		if result != tt.expected {
			t.Errorf("SanitizeKey(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSanitizeKeyPathTraversal(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"../parent", ".._parent"}, // Dots are not removed, / replaced with _
		{"../../grandparent", ".._.._grandparent"},
		{"./current", "._current"},
		{"path/../other", "path_.._other"},
	}

	for _, tt := range tests {
		result := SanitizeKey(tt.input)
		if result != tt.expected {
			t.Errorf("SanitizeKey(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSanitizeKeyEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", "unnamed"},
		{"only spaces", "   ", "unnamed"},
		{"only underscores", "___", "unnamed"},
		{"only special chars", "/:*?", "unnamed"},
		{"long string", "a/b/c/d/e/f/g/h", "a_b_c_d_e_f_g_h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeKey(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeKey(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// ========== GenerateFileName Tests ==========

func TestGenerateFileName(t *testing.T) {
	// Without hash
	name1 := GenerateFileName("simple_key", false)
	if name1 != "simple_key.jsonl" {
		t.Errorf("GenerateFileName without hash failed: got %q", name1)
	}

	// With hash
	name2 := GenerateFileName("simple_key", true)
	if name2 == "simple_key.jsonl" {
		t.Error("GenerateFileName with hash should add hash suffix")
	}

	// Verify hash is deterministic
	name3 := GenerateFileName("simple_key", true)
	if name2 != name3 {
		t.Error("Hash should be deterministic")
	}

	// Verify sanitization happens
	name4 := GenerateFileName("user/data:v1", false)
	if name4 != "user_data_v1.jsonl" {
		t.Errorf("GenerateFileName should sanitize: got %q", name4)
	}
}

func TestGenerateFileNameWithHash(t *testing.T) {
	tests := []struct {
		key string
	}{
		{"key1"},
		{"user/data"},
		{"namespace:version"},
		{"复杂的键"},
	}

	for _, tt := range tests {
		name1 := GenerateFileName(tt.key, true)
		name2 := GenerateFileName(tt.key, true)

		// Should be deterministic
		if name1 != name2 {
			t.Errorf("GenerateFileName(%q) not deterministic: %q != %q", tt.key, name1, name2)
		}

		// Should have .jsonl extension
		if len(name1) < 6 || name1[len(name1)-6:] != ".jsonl" {
			t.Errorf("GenerateFileName(%q) = %q, should end with .jsonl", tt.key, name1)
		}
	}
}

func TestGenerateFileNameDifferentKeys(t *testing.T) {
	// Different keys should generate different file names (when using hash)
	name1 := GenerateFileName("key1", true)
	name2 := GenerateFileName("key2", true)

	if name1 == name2 {
		t.Error("Different keys should generate different file names")
	}
}

// ========== ExtractKeyFromFileName Tests ==========

func TestExtractKeyFromFileName(t *testing.T) {
	tests := []struct {
		fileName string
		expected string
	}{
		{"user_data_v1.jsonl", "user_data_v1"},
		{"user_data_v1_abc123.jsonl", "user_data_v1"},
		{"simple.jsonl", "simple"},
		{"key_with_hash_abcdef.jsonl", "key_with_hash"},
		{"no_hash.jsonl", "no_hash"},
	}

	for _, tt := range tests {
		result := ExtractKeyFromFileName(tt.fileName)
		if result != tt.expected {
			t.Errorf("ExtractKeyFromFileName(%q) = %q, want %q", tt.fileName, result, tt.expected)
		}
	}
}

func TestExtractKeyFromFileNameWithoutHash(t *testing.T) {
	fileName := "user_data.jsonl"
	expected := "user_data"

	result := ExtractKeyFromFileName(fileName)
	if result != expected {
		t.Errorf("ExtractKeyFromFileName(%q) = %q, want %q", fileName, result, expected)
	}
}

func TestExtractKeyFromFileNameWithHash(t *testing.T) {
	fileName := "user_data_abc123.jsonl"
	expected := "user_data"

	result := ExtractKeyFromFileName(fileName)
	if result != expected {
		t.Errorf("ExtractKeyFromFileName(%q) = %q, want %q", fileName, result, expected)
	}
}

func TestExtractKeyFromFileNameEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		expected string
	}{
		{"no extension", "filename", "filename"},
		{"multiple underscores", "a_b_c_d_e_f.jsonl", "a_b_c_d_e_f"},
		{"hash-like but 7 chars", "key_abcdefg.jsonl", "key_abcdefg"}, // Not a hash
		{"hash-like but not hex", "key_ghijkl.jsonl", "key_ghijkl"},   // Not hex
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractKeyFromFileName(tt.fileName)
			if result != tt.expected {
				t.Errorf("ExtractKeyFromFileName(%q) = %q, want %q", tt.fileName, result, tt.expected)
			}
		})
	}
}

// ========== isHexString Tests ==========

func TestIsHexString(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"abc123", true},
		{"ABCDEF", true},
		{"0123456789", true},
		{"abcdef0123456789", true},
		{"ghijkl", false},
		{"abc xyz", false},
		{"", true}, // Empty string is technically all hex
		{"abc-123", false},
		{"G00000", false},
	}

	for _, tt := range tests {
		result := isHexString(tt.input)
		if result != tt.expected {
			t.Errorf("isHexString(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

// ========== KeyConflict Tests ==========

func TestKeyConflict(t *testing.T) {
	tests := []struct {
		key1     string
		key2     string
		conflict bool
	}{
		{"user/data", "user_data", true},
		{"path:v1", "path_v1", true},
		{"file*", "file", true},
		{"same", "same", false}, // Same key = no conflict
		{"different", "keys", false},
		{"user/data", "user/data:v1", false}, // Different sanitized keys
	}

	for _, tt := range tests {
		result := KeyConflict(tt.key1, tt.key2)
		if result != tt.conflict {
			t.Errorf("KeyConflict(%q, %q) = %v, want %v", tt.key1, tt.key2, result, tt.conflict)
		}
	}
}

func TestKeyConflictSymmetric(t *testing.T) {
	// KeyConflict should be symmetric
	key1, key2 := "user/data", "user_data"

	result1 := KeyConflict(key1, key2)
	result2 := KeyConflict(key2, key1)

	if result1 != result2 {
		t.Error("KeyConflict should be symmetric")
	}
}

// ========== NeedsHashSuffix Tests ==========

func TestNeedsHashSuffix(t *testing.T) {
	tests := []struct {
		key   string
		needs bool
	}{
		{"simple_key", false},
		{"user/data", true},
		{"path:v1", true},
		{"file*", false}, // * gets removed entirely, result differs from cleaned version
		{"normal_underscore", false},
		{"with spaces", false}, // Spaces are trimmed, not replaced
		{"multiple///slashes", true},
	}

	for _, tt := range tests {
		result := NeedsHashSuffix(tt.key)
		if result != tt.needs {
			t.Errorf("NeedsHashSuffix(%q) = %v, want %v (sanitized: %q)", tt.key, result, tt.needs, SanitizeKey(tt.key))
		}
	}
}

func TestNeedsHashSuffixEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		needs bool
	}{
		{"empty string", "", true}, // Empty becomes "unnamed", so it needs hash
		{"only underscores", "___", true}, // Becomes "unnamed"
		{"leading underscore", "_key", false},
		{"trailing underscore", "key_", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NeedsHashSuffix(tt.key)
			if result != tt.needs {
				t.Errorf("NeedsHashSuffix(%q) = %v, want %v (sanitized: %q)", tt.key, result, tt.needs, SanitizeKey(tt.key))
			}
		})
	}
}

// ========== IsValidKey Tests ==========

func TestIsValidKey(t *testing.T) {
	tests := []struct {
		key   string
		valid bool
	}{
		{"normal_key", true},
		{"user/data:v1", true},
		{"", false}, // Empty not valid
		{"a", true}, // Single char valid
	}

	for _, tt := range tests {
		result := IsValidKey(tt.key)
		if result != tt.valid {
			t.Errorf("IsValidKey(%q) = %v, want %v", tt.key, result, tt.valid)
		}
	}
}

func TestIsValidKeyLength(t *testing.T) {
	// Test max length (200 characters)
	validKey := ""
	for i := 0; i < 200; i++ {
		validKey += "a"
	}

	if !IsValidKey(validKey) {
		t.Error("200-character key should be valid")
	}

	// Test over max length (201 characters)
	invalidKey := validKey + "a"

	if IsValidKey(invalidKey) {
		t.Error("201-character key should be invalid")
	}
}

// ========== CleanPath Tests ==========

func TestCleanPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/path/to/file.txt", "file.txt"},
		{"relative/path/file.txt", "file.txt"},
		{"file.txt", "file.txt"},
		{"/", "/"},  // Root returns "/" not "."
		{"", "."},
	}

	for _, tt := range tests {
		result := CleanPath(tt.path)
		if result != tt.expected {
			t.Errorf("CleanPath(%q) = %q, want %q", tt.path, result, tt.expected)
		}
	}
}

// ========== Integration Tests ==========

func TestSanitizeAndGenerateFileName(t *testing.T) {
	// Test complete flow: sanitize -> generate file name
	key := "user/data:v1*"

	// Sanitize
	sanitized := SanitizeKey(key)
	if sanitized != "user_data_v1" {
		t.Errorf("Sanitize failed: got %q", sanitized)
	}

	// Generate file name without hash
	fileName := GenerateFileName(key, false)
	if fileName != "user_data_v1.jsonl" {
		t.Errorf("GenerateFileName failed: got %q", fileName)
	}

	// Extract key back
	extracted := ExtractKeyFromFileName(fileName)
	if extracted != "user_data_v1" {
		t.Errorf("ExtractKeyFromFileName failed: got %q", extracted)
	}
}

func TestCollisionDetection(t *testing.T) {
	// Keys that would collide
	key1 := "user/data"
	key2 := "user_data"

	// Should detect conflict
	if !KeyConflict(key1, key2) {
		t.Error("Should detect conflict between user/data and user_data")
	}

	// Both should need hash suffixes (or at least one does)
	needs1 := NeedsHashSuffix(key1)
	if !needs1 {
		t.Error("user/data should need hash suffix")
	}

	// Generate file names with hash
	file1 := GenerateFileName(key1, true)
	file2 := GenerateFileName(key2, true)

	// Files should be different
	if file1 == file2 {
		t.Error("File names should be different for conflicting keys")
	}
}
