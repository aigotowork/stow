package fsutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ========== FindFiles Tests ==========

func TestFindFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("1"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.log"), []byte("2"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "subdir", "file3.txt"), []byte("3"), 0644)

	// Find all .txt files
	matches, err := FindFiles(tmpDir, "*.txt")
	if err != nil {
		t.Fatalf("FindFiles failed: %v", err)
	}

	if len(matches) != 2 {
		t.Fatalf("Expected 2 .txt files, got %d", len(matches))
	}

	// Find .log files
	matches, err = FindFiles(tmpDir, "*.log")
	if err != nil {
		t.Fatalf("FindFiles failed: %v", err)
	}

	if len(matches) != 1 {
		t.Fatalf("Expected 1 .log file, got %d", len(matches))
	}
}

func TestFindFilesNoMatches(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files that don't match pattern
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("1"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.log"), []byte("2"), 0644)

	// Search for non-existent pattern
	matches, err := FindFiles(tmpDir, "*.pdf")
	if err != nil {
		t.Fatalf("FindFiles failed: %v", err)
	}

	if len(matches) != 0 {
		t.Errorf("Expected 0 matches, got %d", len(matches))
	}
}

func TestFindFilesInvalidPattern(t *testing.T) {
	tmpDir := t.TempDir()

	// Some patterns that might be invalid depending on the platform
	// Just verify it doesn't panic
	_, err := FindFiles(tmpDir, "[")
	// Error is expected, just don't panic
	if err == nil {
		t.Log("FindFiles with '[' pattern did not error (platform-specific behavior)")
	}
}

// ========== WalkFilesWithExt Tests ==========

func TestWalkFilesWithExt(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files with different extensions
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("1"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("2"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file3.log"), []byte("3"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "subdir", "file4.txt"), []byte("4"), 0644)

	// Walk .txt files
	var txtFiles []string
	err := WalkFilesWithExt(tmpDir, ".txt", func(path string) error {
		txtFiles = append(txtFiles, path)
		return nil
	})

	if err != nil {
		t.Fatalf("WalkFilesWithExt failed: %v", err)
	}

	if len(txtFiles) != 3 {
		t.Errorf("Expected 3 .txt files, got %d", len(txtFiles))
	}
}

func TestWalkFilesWithExtNoMatches(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files without target extension
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("1"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.log"), []byte("2"), 0644)

	// Walk .pdf files (none exist)
	var pdfFiles []string
	err := WalkFilesWithExt(tmpDir, ".pdf", func(path string) error {
		pdfFiles = append(pdfFiles, path)
		return nil
	})

	if err != nil {
		t.Fatalf("WalkFilesWithExt failed: %v", err)
	}

	if len(pdfFiles) != 0 {
		t.Errorf("Expected 0 .pdf files, got %d", len(pdfFiles))
	}
}

func TestWalkFilesWithExtFuncError(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("1"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("2"), 0644)

	// Function that returns error on first file
	callCount := 0
	err := WalkFilesWithExt(tmpDir, ".txt", func(path string) error {
		callCount++
		if callCount == 1 {
			return os.ErrInvalid
		}
		return nil
	})

	if err == nil {
		t.Error("WalkFilesWithExt should propagate function error")
	}
}

func TestWalkFilesWithExtDeepNesting(t *testing.T) {
	tmpDir := t.TempDir()

	// Create deeply nested structure
	deep := filepath.Join(tmpDir, "a", "b", "c", "d", "e")
	os.MkdirAll(deep, 0755)
	os.WriteFile(filepath.Join(deep, "deep.txt"), []byte("deep"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "shallow.txt"), []byte("shallow"), 0644)

	// Walk should find both
	var txtFiles []string
	err := WalkFilesWithExt(tmpDir, ".txt", func(path string) error {
		txtFiles = append(txtFiles, path)
		return nil
	})

	if err != nil {
		t.Fatalf("WalkFilesWithExt failed: %v", err)
	}

	if len(txtFiles) != 2 {
		t.Errorf("Expected 2 .txt files, got %d", len(txtFiles))
	}
}

// ========== DirSize Tests ==========

func TestDirSize(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files with known sizes
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("12345"), 0644)       // 5 bytes
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("1234567890"), 0644) // 10 bytes

	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "subdir", "file3.txt"), []byte("123"), 0644) // 3 bytes

	// Calculate size
	size, err := DirSize(tmpDir)
	if err != nil {
		t.Fatalf("DirSize failed: %v", err)
	}

	expected := int64(5 + 10 + 3)
	if size != expected {
		t.Fatalf("Expected size %d, got %d", expected, size)
	}
}

func TestDirSizeEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	size, err := DirSize(tmpDir)
	if err != nil {
		t.Fatalf("DirSize failed: %v", err)
	}

	if size != 0 {
		t.Errorf("Empty directory size should be 0, got %d", size)
	}
}

func TestDirSizeNonExistent(t *testing.T) {
	_, err := DirSize("/nonexistent/path")
	if err == nil {
		t.Error("DirSize should fail for non-existent directory")
	}
}

func TestDirSizeNestedStructure(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested structure
	os.MkdirAll(filepath.Join(tmpDir, "a", "b"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "f1.txt"), []byte("12345"), 0644) // 5 bytes
	os.WriteFile(filepath.Join(tmpDir, "a", "f2.txt"), []byte("123456789"), 0644) // 9 bytes
	os.WriteFile(filepath.Join(tmpDir, "a", "b", "f3.txt"), []byte("12"), 0644) // 2 bytes

	size, err := DirSize(tmpDir)
	if err != nil {
		t.Fatalf("DirSize failed: %v", err)
	}

	expected := int64(5 + 9 + 2)
	if size != expected {
		t.Errorf("DirSize = %d, want %d", size, expected)
	}
}

// ========== CountFiles Tests ==========

func TestCountFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files and directories
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("1"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("2"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "subdir", "file3.txt"), []byte("3"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "subdir", "file4.txt"), []byte("4"), 0644)

	count, err := CountFiles(tmpDir)
	if err != nil {
		t.Fatalf("CountFiles failed: %v", err)
	}

	expected := 4
	if count != expected {
		t.Errorf("CountFiles = %d, want %d", count, expected)
	}
}

func TestCountFilesEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	count, err := CountFiles(tmpDir)
	if err != nil {
		t.Fatalf("CountFiles failed: %v", err)
	}

	if count != 0 {
		t.Errorf("Empty directory count should be 0, got %d", count)
	}
}

func TestCountFilesOnlyDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create only directories, no files
	os.Mkdir(filepath.Join(tmpDir, "dir1"), 0755)
	os.Mkdir(filepath.Join(tmpDir, "dir2"), 0755)
	os.Mkdir(filepath.Join(tmpDir, "dir1", "subdir"), 0755)

	count, err := CountFiles(tmpDir)
	if err != nil {
		t.Fatalf("CountFiles failed: %v", err)
	}

	if count != 0 {
		t.Errorf("Directory-only tree count should be 0, got %d", count)
	}
}

func TestCountFilesNonExistent(t *testing.T) {
	_, err := CountFiles("/nonexistent/path")
	if err == nil {
		t.Error("CountFiles should fail for non-existent directory")
	}
}

func TestCountFilesDeepNesting(t *testing.T) {
	tmpDir := t.TempDir()

	// Create deeply nested files
	for i := 0; i < 5; i++ {
		path := tmpDir
		for j := 0; j <= i; j++ {
			path = filepath.Join(path, "level")
			os.MkdirAll(path, 0755)
		}
		os.WriteFile(filepath.Join(path, "file.txt"), []byte("test"), 0644)
	}

	count, err := CountFiles(tmpDir)
	if err != nil {
		t.Fatalf("CountFiles failed: %v", err)
	}

	if count != 5 {
		t.Errorf("CountFiles = %d, want 5", count)
	}
}

// ========== IsHidden Tests ==========

func TestIsHidden(t *testing.T) {
	tests := []struct {
		path   string
		hidden bool
	}{
		{"/path/to/.hidden", true},
		{"/path/to/normal.txt", false},
		{".dotfile", true},
		{"regular", false},
		{"/path/.hidden/file.txt", false}, // Only checks basename
		{"..config", true},
		{"..", true}, // Parent directory reference
		{".", true},  // Current directory reference
	}

	for _, tt := range tests {
		result := IsHidden(tt.path)
		if result != tt.hidden {
			t.Errorf("IsHidden(%q) = %v, want %v", tt.path, result, tt.hidden)
		}
	}
}

// ========== FilterHidden Tests ==========

func TestFilterHidden(t *testing.T) {
	paths := []string{
		"/path/to/file.txt",
		"/path/to/.hidden",
		"/path/normal.log",
		".dotfile",
		"regular.md",
	}

	filtered := FilterHidden(paths)

	expected := []string{
		"/path/to/file.txt",
		"/path/normal.log",
		"regular.md",
	}

	if len(filtered) != len(expected) {
		t.Errorf("FilterHidden returned %d paths, want %d", len(filtered), len(expected))
	}

	// Verify each expected path is in filtered
	for _, exp := range expected {
		found := false
		for _, f := range filtered {
			if f == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("FilterHidden missing expected path: %q", exp)
		}
	}

	// Verify no hidden paths in filtered
	for _, f := range filtered {
		if IsHidden(f) {
			t.Errorf("FilterHidden included hidden path: %q", f)
		}
	}
}

func TestFilterHiddenEmpty(t *testing.T) {
	filtered := FilterHidden([]string{})

	if len(filtered) != 0 {
		t.Errorf("FilterHidden(empty) returned %d paths, want 0", len(filtered))
	}
}

func TestFilterHiddenAllHidden(t *testing.T) {
	paths := []string{
		".hidden1",
		".hidden2",
		".config",
	}

	filtered := FilterHidden(paths)

	if len(filtered) != 0 {
		t.Errorf("FilterHidden(all hidden) returned %d paths, want 0", len(filtered))
	}
}

func TestFilterHiddenNoneHidden(t *testing.T) {
	paths := []string{
		"file1.txt",
		"file2.log",
		"document.pdf",
	}

	filtered := FilterHidden(paths)

	if len(filtered) != len(paths) {
		t.Errorf("FilterHidden returned %d paths, want %d", len(filtered), len(paths))
	}
}

func TestFilterHiddenWithRealPaths(t *testing.T) {
	tmpDir := t.TempDir()

	// Create real files (some hidden)
	os.WriteFile(filepath.Join(tmpDir, "visible.txt"), []byte("1"), 0644)
	os.WriteFile(filepath.Join(tmpDir, ".hidden"), []byte("2"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "dir"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "dir", ".secret"), []byte("3"), 0644)

	// List all files recursively
	var allPaths []string
	Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			allPaths = append(allPaths, path)
		}
		return nil
	})

	// Filter hidden
	filtered := FilterHidden(allPaths)

	// Should only have visible.txt
	if len(filtered) != 1 {
		t.Errorf("FilterHidden returned %d files, want 1", len(filtered))
	}

	if len(filtered) > 0 && !strings.HasSuffix(filtered[0], "visible.txt") {
		t.Errorf("FilterHidden returned wrong file: %q", filtered[0])
	}
}

// ========== Walk Tests ==========

func TestWalk(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test structure
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("1"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "subdir", "file2.txt"), []byte("2"), 0644)

	var paths []string
	err := Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		paths = append(paths, path)
		return nil
	})

	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	// Should visit: tmpDir, file1.txt, subdir, subdir/file2.txt
	if len(paths) != 4 {
		t.Errorf("Walk visited %d paths, want 4", len(paths))
	}
}

func TestWalkSkipDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create structure
	os.Mkdir(filepath.Join(tmpDir, "skip"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "skip", "file1.txt"), []byte("1"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "visit"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "visit", "file2.txt"), []byte("2"), 0644)

	var visited []string
	err := Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		visited = append(visited, path)

		// Skip the "skip" directory
		if info.IsDir() && filepath.Base(path) == "skip" {
			return filepath.SkipDir
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	// Should not have visited skip/file1.txt
	for _, p := range visited {
		if strings.Contains(p, "skip"+string(filepath.Separator)+"file1.txt") {
			t.Error("Walk visited file in skipped directory")
		}
	}
}

// ========== Integration Tests ==========

func TestWalkAndFilter(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mixed structure
	os.WriteFile(filepath.Join(tmpDir, "visible.txt"), []byte("1"), 0644)
	os.WriteFile(filepath.Join(tmpDir, ".hidden"), []byte("2"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "dir"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "dir", "file.txt"), []byte("3"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "dir", ".secret"), []byte("4"), 0644)

	// Walk and collect all file paths
	var allFiles []string
	Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			allFiles = append(allFiles, path)
		}
		return nil
	})

	// Filter hidden files
	visible := FilterHidden(allFiles)

	// Should have 2 visible files
	if len(visible) != 2 {
		t.Errorf("Found %d visible files, want 2", len(visible))
	}
}

func TestCountAndSize(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files
	files := []struct {
		name string
		size int
	}{
		{"file1.txt", 100},
		{"file2.txt", 200},
		{"file3.txt", 300},
	}

	for _, f := range files {
		data := make([]byte, f.size)
		os.WriteFile(filepath.Join(tmpDir, f.name), data, 0644)
	}

	// Count files
	count, err := CountFiles(tmpDir)
	if err != nil {
		t.Fatalf("CountFiles failed: %v", err)
	}

	if count != len(files) {
		t.Errorf("CountFiles = %d, want %d", count, len(files))
	}

	// Calculate total size
	size, err := DirSize(tmpDir)
	if err != nil {
		t.Fatalf("DirSize failed: %v", err)
	}

	expectedSize := int64(100 + 200 + 300)
	if size != expectedSize {
		t.Errorf("DirSize = %d, want %d", size, expectedSize)
	}
}
