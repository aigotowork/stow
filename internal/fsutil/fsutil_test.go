package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Test basic write
	data := []byte("Hello, Stow!")
	err := AtomicWriteFile(testFile, data, 0644)
	if err != nil {
		t.Fatalf("AtomicWriteFile failed: %v", err)
	}

	// Verify file exists
	if !FileExists(testFile) {
		t.Fatal("File was not created")
	}

	// Verify content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != string(data) {
		t.Fatalf("Content mismatch: got %q, want %q", string(content), string(data))
	}

	// Test overwrite
	newData := []byte("Updated content")
	err = AtomicWriteFile(testFile, newData, 0644)
	if err != nil {
		t.Fatalf("AtomicWriteFile (overwrite) failed: %v", err)
	}

	content, _ = os.ReadFile(testFile)
	if string(content) != string(newData) {
		t.Fatalf("Overwrite failed: got %q, want %q", string(content), string(newData))
	}
}

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
	}

	for _, tt := range tests {
		result := IsHidden(tt.path)
		if result != tt.hidden {
			t.Errorf("IsHidden(%q) = %v, want %v", tt.path, result, tt.hidden)
		}
	}
}
