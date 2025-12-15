package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

// ========== EnsureDir Tests ==========

func TestEnsureDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Test creating single directory
	dir1 := filepath.Join(tmpDir, "dir1")
	err := EnsureDir(dir1, 0755)
	if err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	if !DirExists(dir1) {
		t.Fatal("Directory was not created")
	}

	// Test creating nested directories
	dir2 := filepath.Join(tmpDir, "parent", "child", "grandchild")
	err = EnsureDir(dir2, 0755)
	if err != nil {
		t.Fatalf("EnsureDir (nested) failed: %v", err)
	}

	if !DirExists(dir2) {
		t.Fatal("Nested directory was not created")
	}

	// Test idempotence (calling again should not error)
	err = EnsureDir(dir1, 0755)
	if err != nil {
		t.Fatalf("EnsureDir (idempotent) failed: %v", err)
	}
}

func TestEnsureDirFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file
	filePath := filepath.Join(tmpDir, "file.txt")
	os.WriteFile(filePath, []byte("test"), 0644)

	// Try to create directory with same name
	err := EnsureDir(filePath, 0755)
	if err == nil {
		t.Error("EnsureDir should fail when path exists as a file")
	}
}

func TestEnsureDirPermissions(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name string
		perm os.FileMode
	}{
		{"read-write", 0755},
		{"restricted", 0700},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dirPath := filepath.Join(tmpDir, "perm_"+tt.name)
			err := EnsureDir(dirPath, tt.perm)
			if err != nil {
				t.Fatalf("EnsureDir failed: %v", err)
			}

			info, err := os.Stat(dirPath)
			if err != nil {
				t.Fatalf("Failed to stat directory: %v", err)
			}

			gotPerm := info.Mode().Perm()
			if gotPerm != tt.perm {
				t.Errorf("Permissions = %o, want %o", gotPerm, tt.perm)
			}
		})
	}
}

// ========== FileExists Tests ==========

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Test non-existent file
	if FileExists(filepath.Join(tmpDir, "noexist.txt")) {
		t.Fatal("FileExists returned true for non-existent file")
	}

	// Test existing file
	testFile := filepath.Join(tmpDir, "exists.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	if !FileExists(testFile) {
		t.Fatal("FileExists returned false for existing file")
	}

	// Test directory (should return false)
	if FileExists(tmpDir) {
		t.Fatal("FileExists returned true for directory")
	}
}

// ========== DirExists Tests ==========

func TestDirExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Test existing directory
	if !DirExists(tmpDir) {
		t.Fatal("DirExists returned false for existing directory")
	}

	// Test non-existent directory
	if DirExists(filepath.Join(tmpDir, "noexist")) {
		t.Fatal("DirExists returned true for non-existent directory")
	}

	// Test file (should return false)
	testFile := filepath.Join(tmpDir, "file.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	if DirExists(testFile) {
		t.Fatal("DirExists returned true for file")
	}
}

// ========== FileSize Tests ==========

func TestFileSize(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		content  []byte
		expected int64
	}{
		{"empty", []byte{}, 0},
		{"small", []byte("hello"), 5},
		{"medium", []byte("hello world! this is a test file."), 33},
		{"large", make([]byte, 10*1024), 10 * 1024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, tt.name+".txt")
			os.WriteFile(testFile, tt.content, 0644)

			size := FileSize(testFile)
			if size != tt.expected {
				t.Errorf("FileSize() = %d, want %d", size, tt.expected)
			}
		})
	}
}

func TestFileSizeNonExistent(t *testing.T) {
	size := FileSize("/nonexistent/file.txt")
	if size != 0 {
		t.Errorf("FileSize(nonexistent) = %d, want 0", size)
	}
}

func TestFileSizeDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	size := FileSize(tmpDir)
	if size != 0 {
		t.Errorf("FileSize(directory) = %d, want 0", size)
	}
}

func TestFileSizeSymlink(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file
	targetFile := filepath.Join(tmpDir, "target.txt")
	content := []byte("test content")
	os.WriteFile(targetFile, content, 0644)

	// Create a symlink
	linkFile := filepath.Join(tmpDir, "link.txt")
	err := os.Symlink(targetFile, linkFile)
	if err != nil {
		t.Skipf("Cannot create symlink: %v", err)
	}

	// FileSize should follow the symlink
	size := FileSize(linkFile)
	if size != int64(len(content)) {
		t.Errorf("FileSize(symlink) = %d, want %d", size, len(content))
	}
}

// ========== CleanPath Tests ==========

func TestCleanPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"./path/to/file", "path/to/file"},
		{"/path//to///file", "/path/to/file"},
		{"/path/./to/file", "/path/to/file"},
		{"/path/to/../file", "/path/file"},
		{"path/to/file/", "path/to/file"},
		{"", "."},
		{".", "."},
		{"..", ".."},
		{"/", "/"},
		{"//", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := CleanPath(tt.input)
			if result != tt.expected {
				t.Errorf("CleanPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCleanPathSpecialCases(t *testing.T) {
	// Test with actual file paths
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	cleaned := CleanPath(testFile)
	if !filepath.IsAbs(cleaned) && filepath.IsAbs(testFile) {
		t.Error("CleanPath changed absolute path to relative")
	}
}

// ========== AbsPath Tests ==========

func TestAbsPath(t *testing.T) {
	// Test relative path
	absPath, err := AbsPath(".")
	if err != nil {
		t.Fatalf("AbsPath failed: %v", err)
	}

	if !filepath.IsAbs(absPath) {
		t.Errorf("AbsPath(\".\") = %q, should be absolute", absPath)
	}

	// Test already absolute path
	tmpDir := t.TempDir()
	absPath2, err := AbsPath(tmpDir)
	if err != nil {
		t.Fatalf("AbsPath failed: %v", err)
	}

	if absPath2 != tmpDir {
		t.Errorf("AbsPath(%q) = %q, want %q", tmpDir, absPath2, tmpDir)
	}
}

func TestAbsPathNonExistent(t *testing.T) {
	// AbsPath should work even for non-existent paths
	absPath, err := AbsPath("nonexistent/path/file.txt")
	if err != nil {
		t.Fatalf("AbsPath failed: %v", err)
	}

	if !filepath.IsAbs(absPath) {
		t.Errorf("AbsPath(nonexistent) = %q, should be absolute", absPath)
	}
}

func TestAbsPathEmptyString(t *testing.T) {
	absPath, err := AbsPath("")
	if err != nil {
		t.Fatalf("AbsPath(\"\") failed: %v", err)
	}

	// Should return current directory
	if !filepath.IsAbs(absPath) {
		t.Errorf("AbsPath(\"\") = %q, should be absolute", absPath)
	}
}

// ========== RemoveAll Tests ==========

func TestRemoveAll(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "toremove")

	// Create directory with files
	os.Mkdir(testDir, 0755)
	os.WriteFile(filepath.Join(testDir, "file1.txt"), []byte("1"), 0644)
	os.Mkdir(filepath.Join(testDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(testDir, "subdir", "file2.txt"), []byte("2"), 0644)

	// Remove all
	err := RemoveAll(testDir)
	if err != nil {
		t.Fatalf("RemoveAll failed: %v", err)
	}

	// Verify removed
	if DirExists(testDir) {
		t.Fatal("Directory still exists after RemoveAll")
	}

	// Test removing non-existent (should not error)
	err = RemoveAll(filepath.Join(tmpDir, "noexist"))
	if err != nil {
		t.Fatalf("RemoveAll failed for non-existent path: %v", err)
	}
}

func TestRemoveAllFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "file.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	// RemoveAll should work on files too
	err := RemoveAll(testFile)
	if err != nil {
		t.Fatalf("RemoveAll failed for file: %v", err)
	}

	if FileExists(testFile) {
		t.Error("File still exists after RemoveAll")
	}
}

func TestRemoveAllNestedStructure(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "complex")

	// Create complex nested structure
	os.MkdirAll(filepath.Join(testDir, "a", "b", "c"), 0755)
	os.MkdirAll(filepath.Join(testDir, "x", "y", "z"), 0755)
	os.WriteFile(filepath.Join(testDir, "file1.txt"), []byte("1"), 0644)
	os.WriteFile(filepath.Join(testDir, "a", "file2.txt"), []byte("2"), 0644)
	os.WriteFile(filepath.Join(testDir, "a", "b", "file3.txt"), []byte("3"), 0644)
	os.WriteFile(filepath.Join(testDir, "a", "b", "c", "file4.txt"), []byte("4"), 0644)
	os.WriteFile(filepath.Join(testDir, "x", "y", "z", "file5.txt"), []byte("5"), 0644)

	// Remove all
	err := RemoveAll(testDir)
	if err != nil {
		t.Fatalf("RemoveAll failed: %v", err)
	}

	if DirExists(testDir) {
		t.Error("Complex directory structure still exists after RemoveAll")
	}
}

// ========== ListFiles Tests ==========

func TestListFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some files and directories
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("1"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("2"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "subdir", "file3.txt"), []byte("3"), 0644)

	// List files (should only include files in top level)
	files, err := ListFiles(tmpDir)
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(files))
	}
}

func TestListFilesEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	files, err := ListFiles(tmpDir)
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("Expected 0 files in empty dir, got %d", len(files))
	}
}

func TestListFilesNonExistent(t *testing.T) {
	_, err := ListFiles("/nonexistent/path")
	if err == nil {
		t.Error("ListFiles should fail for non-existent directory")
	}
}

// ========== ListDirs Tests ==========

func TestListDirs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some directories and files
	os.Mkdir(filepath.Join(tmpDir, "dir1"), 0755)
	os.Mkdir(filepath.Join(tmpDir, "dir2"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("test"), 0644)

	// List directories
	dirs, err := ListDirs(tmpDir)
	if err != nil {
		t.Fatalf("ListDirs failed: %v", err)
	}

	if len(dirs) != 2 {
		t.Fatalf("Expected 2 directories, got %d", len(dirs))
	}
}

func TestListDirsEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	dirs, err := ListDirs(tmpDir)
	if err != nil {
		t.Fatalf("ListDirs failed: %v", err)
	}

	if len(dirs) != 0 {
		t.Errorf("Expected 0 dirs in empty dir, got %d", len(dirs))
	}
}

func TestListDirsNonExistent(t *testing.T) {
	_, err := ListDirs("/nonexistent/path")
	if err == nil {
		t.Error("ListDirs should fail for non-existent directory")
	}
}
