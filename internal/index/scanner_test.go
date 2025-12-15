package index

import (
	"os"
	"path/filepath"
	"testing"
)

// ========== Basic Scanner Tests ==========

func TestScanner(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test JSONL files with valid records
	files := map[string]string{
		"config.jsonl": `{"_meta":{"k":"config","v":1,"op":"put","ts":"2024-01-01T00:00:00Z"},"data":{"value":"test"}}` + "\n",
		"cache.jsonl":  `{"_meta":{"k":"cache:v1","v":1,"op":"put","ts":"2024-01-01T00:00:00Z"},"data":{"value":"cache"}}` + "\n",
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Scan directory
	scanner := NewScanner()
	mapper, err := scanner.ScanNamespace(tmpDir)
	if err != nil {
		t.Fatalf("ScanNamespace failed: %v", err)
	}

	// Verify results
	if mapper.Count() != 2 {
		t.Errorf("Scanner found %d keys, want 2", mapper.Count())
	}

	// Check specific keys
	files1 := mapper.Find("config")
	if len(files1) == 0 {
		t.Error("Should find 'config' key")
	}

	files2 := mapper.Find("cache:v1")
	if len(files2) == 0 {
		t.Error("Should find 'cache:v1' key")
	}
}

func TestScannerSkipsInvalidFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create invalid file
	invalidPath := filepath.Join(tmpDir, "invalid.jsonl")
	os.WriteFile(invalidPath, []byte("not valid json\n"), 0644)

	// Create valid file
	validPath := filepath.Join(tmpDir, "valid.jsonl")
	validContent := `{"_meta":{"k":"valid","v":1,"op":"put","ts":"2024-01-01T00:00:00Z"},"data":{}}` + "\n"
	os.WriteFile(validPath, []byte(validContent), 0644)

	// Scan should skip invalid and process valid
	scanner := NewScanner()
	mapper, err := scanner.ScanNamespace(tmpDir)
	if err != nil {
		t.Fatalf("ScanNamespace failed: %v", err)
	}

	if mapper.Count() != 1 {
		t.Errorf("Should find 1 valid key, got %d", mapper.Count())
	}
}

func TestScannerEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	scanner := NewScanner()
	mapper, err := scanner.ScanNamespace(tmpDir)
	if err != nil {
		t.Fatalf("ScanNamespace failed: %v", err)
	}

	if mapper.Count() != 0 {
		t.Error("Empty directory should result in empty mapper")
	}
}

// ========== ScanAndValidate Tests ==========

func TestScannerScanAndValidate(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files without conflicts
	files := map[string]string{
		"key1.jsonl": `{"_meta":{"k":"key1","v":1,"op":"put","ts":"2024-01-01T00:00:00Z"},"data":{}}` + "\n",
		"key2.jsonl": `{"_meta":{"k":"key2","v":1,"op":"put","ts":"2024-01-01T00:00:00Z"},"data":{}}` + "\n",
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		os.WriteFile(path, []byte(content), 0644)
	}

	scanner := NewScanner()
	mapper, issues, err := scanner.ScanAndValidate(tmpDir)
	if err != nil {
		t.Fatalf("ScanAndValidate failed: %v", err)
	}

	if mapper.Count() != 2 {
		t.Errorf("Count = %d, want 2", mapper.Count())
	}

	if len(issues) != 0 {
		t.Errorf("Should have no issues, got %d", len(issues))
	}
}

func TestScannerScanAndValidateWithConflicts(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files with conflicts (sanitize to same name)
	files := map[string]string{
		"user_data.jsonl":        `{"_meta":{"k":"user/data","v":1,"op":"put","ts":"2024-01-01T00:00:00Z"},"data":{}}` + "\n",
		"user_data_abc123.jsonl": `{"_meta":{"k":"user_data","v":1,"op":"put","ts":"2024-01-01T00:00:00Z"},"data":{}}` + "\n",
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		os.WriteFile(path, []byte(content), 0644)
	}

	scanner := NewScanner()
	mapper, issues, err := scanner.ScanAndValidate(tmpDir)
	if err != nil {
		t.Fatalf("ScanAndValidate failed: %v", err)
	}

	if mapper.Count() != 2 {
		t.Errorf("Count = %d, want 2", mapper.Count())
	}

	// Should detect conflicts
	if len(issues) == 0 {
		t.Error("Should detect key collisions")
	}
}

func TestScannerScanAndValidateEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	scanner := NewScanner()
	mapper, issues, err := scanner.ScanAndValidate(tmpDir)
	if err != nil {
		t.Fatalf("ScanAndValidate failed: %v", err)
	}

	if mapper.Count() != 0 {
		t.Error("Empty dir should have 0 keys")
	}

	if len(issues) != 0 {
		t.Error("Empty dir should have no issues")
	}
}

// ========== CountFiles Tests ==========

func TestScannerCountFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	for i := 0; i < 5; i++ {
		name := "file" + string(rune('0'+i)) + ".jsonl"
		path := filepath.Join(tmpDir, name)
		content := `{"_meta":{"k":"key","v":1,"op":"put","ts":"2024-01-01T00:00:00Z"},"data":{}}` + "\n"
		os.WriteFile(path, []byte(content), 0644)
	}

	count, err := CountFiles(tmpDir)
	if err != nil {
		t.Fatalf("CountFiles failed: %v", err)
	}

	if count != 5 {
		t.Errorf("CountFiles = %d, want 5", count)
	}
}

func TestScannerCountFilesEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	count, err := CountFiles(tmpDir)
	if err != nil {
		t.Fatalf("CountFiles failed: %v", err)
	}

	if count != 0 {
		t.Errorf("CountFiles = %d, want 0", count)
	}
}

func TestScannerCountFilesSkipsBlobDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create file in main dir
	mainFile := filepath.Join(tmpDir, "main.jsonl")
	os.WriteFile(mainFile, []byte("content\n"), 0644)

	// Create _blobs directory with file
	blobDir := filepath.Join(tmpDir, "_blobs")
	os.MkdirAll(blobDir, 0755)
	blobFile := filepath.Join(blobDir, "blob.jsonl")
	os.WriteFile(blobFile, []byte("blob\n"), 0644)

	count, err := CountFiles(tmpDir)
	if err != nil {
		t.Fatalf("CountFiles failed: %v", err)
	}

	// Should only count main file, not blob file
	if count != 1 {
		t.Errorf("CountFiles = %d, want 1 (should skip _blobs dir)", count)
	}
}

// ========== ListKeys Tests ==========

func TestScannerListKeys(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	expectedKeys := []string{"key1", "key2", "key3"}
	for _, key := range expectedKeys {
		name := key + ".jsonl"
		path := filepath.Join(tmpDir, name)
		content := `{"_meta":{"k":"` + key + `","v":1,"op":"put","ts":"2024-01-01T00:00:00Z"},"data":{}}` + "\n"
		os.WriteFile(path, []byte(content), 0644)
	}

	keys, err := ListKeys(tmpDir)
	if err != nil {
		t.Fatalf("ListKeys failed: %v", err)
	}

	if len(keys) != len(expectedKeys) {
		t.Errorf("ListKeys returned %d keys, want %d", len(keys), len(expectedKeys))
	}

	// Verify all keys are present
	keyMap := make(map[string]bool)
	for _, k := range keys {
		keyMap[k] = true
	}

	for _, expected := range expectedKeys {
		if !keyMap[expected] {
			t.Errorf("Missing key %q in result", expected)
		}
	}
}

func TestScannerListKeysEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	keys, err := ListKeys(tmpDir)
	if err != nil {
		t.Fatalf("ListKeys failed: %v", err)
	}

	if len(keys) != 0 {
		t.Errorf("ListKeys = %d keys, want 0", len(keys))
	}
}

func TestScannerListKeysSkipsInvalid(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid file
	validPath := filepath.Join(tmpDir, "valid.jsonl")
	validContent := `{"_meta":{"k":"valid","v":1,"op":"put","ts":"2024-01-01T00:00:00Z"},"data":{}}` + "\n"
	os.WriteFile(validPath, []byte(validContent), 0644)

	// Create invalid file
	invalidPath := filepath.Join(tmpDir, "invalid.jsonl")
	os.WriteFile(invalidPath, []byte("not json\n"), 0644)

	keys, err := ListKeys(tmpDir)
	if err != nil {
		t.Fatalf("ListKeys failed: %v", err)
	}

	// Should only return valid key
	if len(keys) != 1 || keys[0] != "valid" {
		t.Errorf("ListKeys = %v, want [valid]", keys)
	}
}

func TestScannerListKeysSkipsBlobDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create file in main dir
	mainFile := filepath.Join(tmpDir, "main.jsonl")
	mainContent := `{"_meta":{"k":"main","v":1,"op":"put","ts":"2024-01-01T00:00:00Z"},"data":{}}` + "\n"
	os.WriteFile(mainFile, []byte(mainContent), 0644)

	// Create _blobs directory with file
	blobDir := filepath.Join(tmpDir, "_blobs")
	os.MkdirAll(blobDir, 0755)
	blobFile := filepath.Join(blobDir, "blob.jsonl")
	blobContent := `{"_meta":{"k":"blob","v":1,"op":"put","ts":"2024-01-01T00:00:00Z"},"data":{}}` + "\n"
	os.WriteFile(blobFile, []byte(blobContent), 0644)

	keys, err := ListKeys(tmpDir)
	if err != nil {
		t.Fatalf("ListKeys failed: %v", err)
	}

	// Should only return main key, not blob key
	if len(keys) != 1 || keys[0] != "main" {
		t.Errorf("ListKeys = %v, want [main] (should skip _blobs dir)", keys)
	}
}

// ========== Error Handling Tests ==========

func TestScannerNonExistentDirectory(t *testing.T) {
	scanner := NewScanner()
	_, err := scanner.ScanNamespace("/nonexistent/path")
	if err == nil {
		t.Error("ScanNamespace should fail for non-existent directory")
	}
}

func TestScannerFileInsteadOfDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "notadir.txt")
	os.WriteFile(filePath, []byte("content"), 0644)

	scanner := NewScanner()
	mapper, err := scanner.ScanNamespace(filePath)
	// Scanner will not fail but will return empty mapper for a file
	if err != nil {
		t.Fatalf("ScanNamespace failed: %v", err)
	}
	if mapper.Count() != 0 {
		t.Error("Should return empty mapper for a file path")
	}
}

// ========== Performance Tests ==========

func TestScannerLargeNamespace(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large namespace test in short mode")
	}

	tmpDir := t.TempDir()

	// Create many files
	const numFiles = 1000
	for i := 0; i < numFiles; i++ {
		name := "key_" + string(rune(i%100)) + ".jsonl"
		path := filepath.Join(tmpDir, name)
		content := `{"_meta":{"k":"key","v":1,"op":"put","ts":"2024-01-01T00:00:00Z"},"data":{}}` + "\n"
		os.WriteFile(path, []byte(content), 0644)
	}

	scanner := NewScanner()
	mapper, err := scanner.ScanNamespace(tmpDir)
	if err != nil {
		t.Fatalf("ScanNamespace failed: %v", err)
	}

	// Should scan all files
	if mapper.Count() == 0 {
		t.Error("Should scan files in large namespace")
	}
}
