package stow_test

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/aigotowork/stow"
)

// TestReaderField_BasicString tests storing a struct with io.Reader field containing string data
func TestReaderField_BasicString(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	type Document struct {
		Title   string
		Content io.Reader
	}

	content := "Hello, this is test content stored via io.Reader field!"
	doc := Document{
		Title:   "Test Document",
		Content: strings.NewReader(content),
	}

	// Store the document - Content field should be stored as blob
	err := ns.Put("doc1", doc)
	if err != nil {
		t.Fatalf("Put with io.Reader field failed: %v", err)
	}

	// Retrieve metadata (io.Reader fields can't be deserialized)
	var retrieved struct {
		Title string
	}
	err = ns.Get("doc1", &retrieved)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.Title != doc.Title {
		t.Errorf("Title mismatch: got %q, want %q", retrieved.Title, doc.Title)
	}
}

// TestReaderField_LargeData tests storing large data through io.Reader field
func TestReaderField_LargeData(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	type Blob struct {
		Name   string
		Size   int64
		Hash   string
		Data   io.Reader
	}

	// Create 1MB of data
	dataSize := 1024 * 1024
	data := make([]byte, dataSize)
	for i := range data {
		data[i] = byte(i % 256)
	}

	// Calculate hash
	hash := sha256.Sum256(data)
	hashStr := fmt.Sprintf("%x", hash)

	blob := Blob{
		Name: "large-blob",
		Size: int64(dataSize),
		Hash: hashStr,
		Data: bytes.NewReader(data),
	}

	// Store with io.Reader field
	err := ns.Put("blob1", blob)
	if err != nil {
		t.Fatalf("Put with large io.Reader field failed: %v", err)
	}

	// Retrieve metadata
	var retrieved struct {
		Name string
		Size int64
		Hash string
	}
	err = ns.Get("blob1", &retrieved)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.Name != blob.Name {
		t.Errorf("Name mismatch: got %q, want %q", retrieved.Name, blob.Name)
	}

	if retrieved.Size != blob.Size {
		t.Errorf("Size mismatch: got %d, want %d", retrieved.Size, blob.Size)
	}

	if retrieved.Hash != blob.Hash {
		t.Errorf("Hash mismatch: got %q, want %q", retrieved.Hash, blob.Hash)
	}
}

// TestReaderField_MultipleReaders tests struct with multiple io.Reader fields
func TestReaderField_MultipleReaders(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	type MultiContent struct {
		Title    string
		Content1 io.Reader
		Content2 io.Reader
		Summary  string
	}

	content1 := "First content stream"
	content2 := "Second content stream"

	multi := MultiContent{
		Title:    "Multi-content Document",
		Content1: strings.NewReader(content1),
		Content2: strings.NewReader(content2),
		Summary:  "Document with multiple content streams",
	}

	// Store
	err := ns.Put("multi1", multi)
	if err != nil {
		t.Fatalf("Put with multiple io.Reader fields failed: %v", err)
	}

	// Retrieve metadata
	var retrieved struct {
		Title   string
		Summary string
	}
	err = ns.Get("multi1", &retrieved)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.Title != multi.Title {
		t.Errorf("Title mismatch: got %q, want %q", retrieved.Title, multi.Title)
	}

	if retrieved.Summary != multi.Summary {
		t.Errorf("Summary mismatch: got %q, want %q", retrieved.Summary, multi.Summary)
	}
}

// TestReaderField_EmptyReader tests storing empty io.Reader
func TestReaderField_EmptyReader(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	type Document struct {
		Title   string
		Content io.Reader
	}

	doc := Document{
		Title:   "Empty Document",
		Content: strings.NewReader(""),
	}

	// Store with empty reader
	err := ns.Put("empty1", doc)
	if err != nil {
		t.Fatalf("Put with empty io.Reader field failed: %v", err)
	}

	// Retrieve
	var retrieved struct {
		Title string
	}
	err = ns.Get("empty1", &retrieved)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.Title != doc.Title {
		t.Errorf("Title mismatch: got %q, want %q", retrieved.Title, doc.Title)
	}
}

// TestReaderField_RandomBinaryData tests streaming random binary data
func TestReaderField_RandomBinaryData(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	type Binary struct {
		Type   string
		Size   int64
		Stream io.Reader
	}

	// Generate random data
	dataSize := 50 * 1024 // 50KB
	data := make([]byte, dataSize)
	_, err := rand.Read(data)
	if err != nil {
		t.Fatalf("Failed to generate random data: %v", err)
	}

	binary := Binary{
		Type:   "random",
		Size:   int64(dataSize),
		Stream: bytes.NewReader(data),
	}

	// Store
	err = ns.Put("random1", binary)
	if err != nil {
		t.Fatalf("Put with random binary io.Reader failed: %v", err)
	}

	// Retrieve
	var retrieved struct {
		Type string
		Size int64
	}
	err = ns.Get("random1", &retrieved)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.Type != binary.Type {
		t.Errorf("Type mismatch: got %q, want %q", retrieved.Type, binary.Type)
	}

	if retrieved.Size != binary.Size {
		t.Errorf("Size mismatch: got %d, want %d", retrieved.Size, binary.Size)
	}
}

// TestReaderField_WithUpdate tests updating a value with io.Reader field
func TestReaderField_WithUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	type File struct {
		Name    string
		Version int
		Content io.Reader
	}

	// First version
	content1 := "Version 1 content"
	file1 := File{
		Name:    "document.txt",
		Version: 1,
		Content: strings.NewReader(content1),
	}

	err := ns.Put("file1", file1)
	if err != nil {
		t.Fatalf("First Put failed: %v", err)
	}

	// Second version (update)
	content2 := "Version 2 content - updated with more information"
	file2 := File{
		Name:    "document.txt",
		Version: 2,
		Content: strings.NewReader(content2),
	}

	err = ns.Put("file1", file2)
	if err != nil {
		t.Fatalf("Second Put failed: %v", err)
	}

	// Retrieve and verify latest version
	var retrieved struct {
		Name    string
		Version int
	}
	err = ns.Get("file1", &retrieved)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.Version != 2 {
		t.Errorf("Version mismatch: got %d, want 2", retrieved.Version)
	}

	// Check history
	history, err := ns.GetHistory("file1")
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}

	if len(history) != 2 {
		t.Errorf("Expected 2 versions in history, got %d", len(history))
	}
}

// TestReaderField_ConcurrentWrites tests concurrent writes with io.Reader fields
func TestReaderField_ConcurrentWrites(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns, _ := store.CreateNamespace("test", stow.DefaultNamespaceConfig())

	type Upload struct {
		ID   int
		Data io.Reader
	}

	const numUploads = 10
	const dataSize = 10 * 1024 // 10KB each

	errChan := make(chan error, numUploads)
	doneChan := make(chan bool, numUploads)

	// Start concurrent uploads
	for i := 0; i < numUploads; i++ {
		go func(id int) {
			data := make([]byte, dataSize)
			for j := range data {
				data[j] = byte((id + j) % 256)
			}

			upload := Upload{
				ID:   id,
				Data: bytes.NewReader(data),
			}

			key := fmt.Sprintf("upload%d", id)
			err := ns.Put(key, upload)
			if err != nil {
				errChan <- fmt.Errorf("upload %d failed: %w", id, err)
			} else {
				doneChan <- true
			}
		}(i)
	}

	// Wait for all uploads to complete
	completed := 0
	for completed < numUploads {
		select {
		case err := <-errChan:
			t.Errorf("Concurrent upload error: %v", err)
			completed++
		case <-doneChan:
			completed++
		}
	}

	// Verify all uploads
	for i := 0; i < numUploads; i++ {
		key := fmt.Sprintf("upload%d", i)
		var upload struct {
			ID int
		}
		err := ns.Get(key, &upload)
		if err != nil {
			t.Errorf("Failed to retrieve upload %d: %v", i, err)
			continue
		}

		if upload.ID != i {
			t.Errorf("Upload %d: ID mismatch, got %d", i, upload.ID)
		}
	}
}

// TestReaderField_WithOtherFields tests io.Reader field mixed with other types
func TestReaderField_WithOtherFields(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	type ComplexDoc struct {
		Title      string
		Author     string
		Tags       []string
		Metadata   map[string]interface{}
		Content    io.Reader
		SmallData  []byte
		LargeData  []byte `stow:"file"`
	}

	content := "Main content via io.Reader"
	smallData := []byte("small inline data")
	largeData := make([]byte, 10*1024) // 10KB

	doc := ComplexDoc{
		Title:     "Complex Document",
		Author:    "Test Author",
		Tags:      []string{"test", "complex", "reader"},
		Metadata:  map[string]interface{}{"version": 1, "draft": false},
		Content:   strings.NewReader(content),
		SmallData: smallData,
		LargeData: largeData,
	}

	// Store
	err := ns.Put("complex1", doc)
	if err != nil {
		t.Fatalf("Put with complex struct failed: %v", err)
	}

	// Retrieve
	var retrieved struct {
		Title     string
		Author    string
		Tags      []string
		Metadata  map[string]interface{}
		SmallData []byte
		LargeData []byte
	}
	err = ns.Get("complex1", &retrieved)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.Title != doc.Title {
		t.Errorf("Title mismatch: got %q, want %q", retrieved.Title, doc.Title)
	}

	if len(retrieved.Tags) != len(doc.Tags) {
		t.Errorf("Tags count mismatch: got %d, want %d", len(retrieved.Tags), len(doc.Tags))
	}

	if !bytes.Equal(retrieved.SmallData, doc.SmallData) {
		t.Error("SmallData mismatch")
	}

	if !bytes.Equal(retrieved.LargeData, doc.LargeData) {
		t.Error("LargeData mismatch")
	}
}

// TestReaderField_NilReader tests behavior with nil io.Reader
func TestReaderField_NilReader(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	type Document struct {
		Title   string
		Content io.Reader
	}

	doc := Document{
		Title:   "Document with nil reader",
		Content: nil,
	}

	// Store with nil reader
	err := ns.Put("nil-reader", doc)
	if err != nil {
		t.Fatalf("Put with nil io.Reader failed: %v", err)
	}

	// Retrieve
	var retrieved struct {
		Title string
	}
	err = ns.Get("nil-reader", &retrieved)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.Title != doc.Title {
		t.Errorf("Title mismatch: got %q, want %q", retrieved.Title, doc.Title)
	}
}

// TestReaderField_ArrayWithFileFields tests array of structs where each element has file fields
func TestReaderField_ArrayWithFileFields(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	// Attachment struct with file content
	type Attachment struct {
		Filename string    `json:"filename"`
		MimeType string    `json:"mime_type"`
		Size     int64     `json:"size"`
		Content  io.Reader `json:"content"` // File content
	}

	type Email struct {
		Subject     string       `json:"subject"`
		From        string       `json:"from"`
		To          string       `json:"to"`
		Attachments []Attachment `json:"attachments"` // Array of attachments with file content
	}

	// Create email with multiple attachments
	attachment1 := Attachment{
		Filename: "document.pdf",
		MimeType: "application/pdf",
		Size:     1024 * 50, // 50KB
		Content:  bytes.NewReader(make([]byte, 1024*50)),
	}

	attachment2 := Attachment{
		Filename: "image.jpg",
		MimeType: "image/jpeg",
		Size:     1024 * 100, // 100KB
		Content:  bytes.NewReader(make([]byte, 1024*100)),
	}

	attachment3 := Attachment{
		Filename: "spreadsheet.xlsx",
		MimeType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		Size:     1024 * 200, // 200KB
		Content:  bytes.NewReader(make([]byte, 1024*200)),
	}

	email := Email{
		Subject:     "Test Email with Attachments",
		From:        "sender@example.com",
		To:          "receiver@example.com",
		Attachments: []Attachment{attachment1, attachment2, attachment3},
	}

	// Store email with array of attachments
	err := ns.Put("email1", email)
	if err != nil {
		t.Fatalf("Put with array of file fields failed: %v", err)
	}

	// Retrieve metadata using map to avoid type matching issues
	var retrieved map[string]interface{}
	err = ns.Get("email1", &retrieved)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if subject, ok := retrieved["subject"].(string); !ok || subject != email.Subject {
		t.Errorf("Subject mismatch: got %v, want %q", retrieved["subject"], email.Subject)
	}

	// The attachments field will be retrieved as the typed slice []Attachment
	// We need to use reflection to iterate over it
	attachmentsField := retrieved["attachments"]
	if attachmentsField == nil {
		t.Fatalf("Attachments field is nil")
	}

	attachmentsValue := reflect.ValueOf(attachmentsField)
	if attachmentsValue.Kind() != reflect.Slice {
		t.Fatalf("Attachments field is not a slice, got kind %v", attachmentsValue.Kind())
	}

	attachmentsLen := attachmentsValue.Len()
	if attachmentsLen != 3 {
		t.Errorf("Expected 3 attachments, got %d", attachmentsLen)
	}

	// Verify each attachment metadata
	for i := 0; i < attachmentsLen; i++ {
		att := attachmentsValue.Index(i)

		// Get filename field
		filenameField := att.FieldByName("Filename")
		if !filenameField.IsValid() {
			t.Errorf("Attachment %d: Filename field not found", i)
			continue
		}
		expectedFilename := email.Attachments[i].Filename
		if filenameField.String() != expectedFilename {
			t.Errorf("Attachment %d: filename mismatch, got %q, want %q", i, filenameField.String(), expectedFilename)
		}

		// Get size field
		sizeField := att.FieldByName("Size")
		if !sizeField.IsValid() {
			t.Errorf("Attachment %d: Size field not found", i)
			continue
		}
		expectedSize := email.Attachments[i].Size
		if sizeField.Int() != expectedSize {
			t.Errorf("Attachment %d: size mismatch, got %d, want %d", i, sizeField.Int(), expectedSize)
		}
	}

	t.Logf("Successfully stored and retrieved email with %d attachments (each with io.Reader content)", attachmentsLen)
}

// TestReaderField_PointerFields tests various pointer field scenarios
func TestReaderField_PointerFields(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	// Test scenario 1: Pointer to []byte (non-nil)
	t.Run("PointerToByteSlice_NonNil", func(t *testing.T) {
		type Document struct {
			Title   string  `json:"title"`
			Content *[]byte `json:"content"` // Pointer to []byte
		}

		content := []byte("Content via pointer to []byte")
		doc := Document{
			Title:   "Pointer Byte Document",
			Content: &content,
		}

		err := ns.Put("doc-ptr-bytes", doc)
		if err != nil {
			t.Fatalf("Put with pointer to []byte failed: %v", err)
		}

		var retrieved Document
		err = ns.Get("doc-ptr-bytes", &retrieved)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if retrieved.Title != doc.Title {
			t.Errorf("Title mismatch: got %q, want %q", retrieved.Title, doc.Title)
		}

		if retrieved.Content == nil {
			t.Error("Content pointer should not be nil")
		} else if !bytes.Equal(*retrieved.Content, content) {
			t.Error("Content mismatch")
		}
	})

	// Test scenario 2: Pointer to []byte (nil)
	t.Run("PointerToByteSlice_Nil", func(t *testing.T) {
		type Document struct {
			Title   string  `json:"title"`
			Content *[]byte `json:"content"` // Nil pointer
		}

		doc := Document{
			Title:   "Nil Pointer Document",
			Content: nil,
		}

		err := ns.Put("doc-ptr-nil", doc)
		if err != nil {
			t.Fatalf("Put with nil pointer to []byte failed: %v", err)
		}

		var retrieved Document
		err = ns.Get("doc-ptr-nil", &retrieved)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if retrieved.Title != doc.Title {
			t.Errorf("Title mismatch: got %q, want %q", retrieved.Title, doc.Title)
		}

		if retrieved.Content != nil {
			t.Error("Content pointer should be nil")
		}
	})

	// Test scenario 3: Pointer to io.Reader (non-nil)
	t.Run("PointerToReader_NonNil", func(t *testing.T) {
		type Document struct {
			Title   string     `json:"title"`
			Content *io.Reader `json:"content"` // Pointer to io.Reader
		}

		content := strings.NewReader("Content via pointer to io.Reader")
		var reader io.Reader = content
		doc := Document{
			Title:   "Pointer Reader Document",
			Content: &reader,
		}

		err := ns.Put("doc-ptr-reader", doc)
		if err != nil {
			t.Fatalf("Put with pointer to io.Reader failed: %v", err)
		}

		// Retrieve metadata only
		var retrieved struct {
			Title string `json:"title"`
		}
		err = ns.Get("doc-ptr-reader", &retrieved)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if retrieved.Title != doc.Title {
			t.Errorf("Title mismatch: got %q, want %q", retrieved.Title, doc.Title)
		}
	})

	// Test scenario 4: Pointer to io.Reader (nil)
	t.Run("PointerToReader_Nil", func(t *testing.T) {
		type Document struct {
			Title   string     `json:"title"`
			Content *io.Reader `json:"content"` // Nil pointer to io.Reader
		}

		doc := Document{
			Title:   "Nil Reader Pointer Document",
			Content: nil,
		}

		err := ns.Put("doc-ptr-reader-nil", doc)
		if err != nil {
			t.Fatalf("Put with nil pointer to io.Reader failed: %v", err)
		}

		var retrieved Document
		err = ns.Get("doc-ptr-reader-nil", &retrieved)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if retrieved.Title != doc.Title {
			t.Errorf("Title mismatch: got %q, want %q", retrieved.Title, doc.Title)
		}

		if retrieved.Content != nil {
			t.Error("Content pointer should be nil")
		}
	})

	// Test scenario 5: Struct with pointer containing []byte
	t.Run("PointerStruct_WithByteSlice", func(t *testing.T) {
		type FileData struct {
			Name    string `json:"name"`
			Content []byte `json:"content"`
		}

		type Document struct {
			Title string    `json:"title"`
			File  *FileData `json:"file"` // Pointer to struct with []byte
		}

		fileData := &FileData{
			Name:    "data.bin",
			Content: make([]byte, 10*1024), // 10KB
		}

		doc := Document{
			Title: "Document with Pointer Struct",
			File:  fileData,
		}

		err := ns.Put("doc-ptr-struct", doc)
		if err != nil {
			t.Fatalf("Put with pointer struct containing []byte failed: %v", err)
		}

		var retrieved Document
		err = ns.Get("doc-ptr-struct", &retrieved)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if retrieved.Title != doc.Title {
			t.Errorf("Title mismatch: got %q, want %q", retrieved.Title, doc.Title)
		}

		if retrieved.File == nil {
			t.Fatal("File pointer should not be nil")
		}

		if retrieved.File.Name != fileData.Name {
			t.Errorf("File name mismatch: got %q, want %q", retrieved.File.Name, fileData.Name)
		}

		if len(retrieved.File.Content) != len(fileData.Content) {
			t.Errorf("File content size mismatch: got %d, want %d", len(retrieved.File.Content), len(fileData.Content))
		}
	})

	// Test scenario 6: Struct with nil pointer to struct containing []byte
	t.Run("PointerStruct_Nil", func(t *testing.T) {
		type FileData struct {
			Name    string `json:"name"`
			Content []byte `json:"content"`
		}

		type Document struct {
			Title string    `json:"title"`
			File  *FileData `json:"file"` // Nil pointer
		}

		doc := Document{
			Title: "Document with Nil Pointer Struct",
			File:  nil,
		}

		err := ns.Put("doc-ptr-struct-nil", doc)
		if err != nil {
			t.Fatalf("Put with nil pointer struct failed: %v", err)
		}

		var retrieved Document
		err = ns.Get("doc-ptr-struct-nil", &retrieved)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if retrieved.Title != doc.Title {
			t.Errorf("Title mismatch: got %q, want %q", retrieved.Title, doc.Title)
		}

		if retrieved.File != nil {
			t.Error("File pointer should be nil")
		}
	})
}

// TestReaderField_ArrayWithPointers tests array of pointers with file fields
func TestReaderField_ArrayWithPointers(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	type FileNode struct {
		Name    string    `json:"name"`
		Content io.Reader `json:"content"`
	}

	type Container struct {
		Name  string      `json:"name"`
		Files []*FileNode `json:"files"` // Array of pointers
	}

	// Create container with mixed nil and non-nil pointers
	file1 := &FileNode{
		Name:    "file1.txt",
		Content: strings.NewReader("Content of file 1"),
	}

	file2 := &FileNode{
		Name:    "file2.txt",
		Content: strings.NewReader("Content of file 2"),
	}

	container := Container{
		Name:  "File Container",
		Files: []*FileNode{file1, nil, file2, nil}, // Mix of nil and non-nil
	}

	err := ns.Put("container1", container)
	if err != nil {
		t.Fatalf("Put with array of pointers failed: %v", err)
	}

	// Retrieve using map to avoid type matching issues
	var retrieved map[string]interface{}
	err = ns.Get("container1", &retrieved)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if name, ok := retrieved["name"].(string); !ok || name != container.Name {
		t.Errorf("Name mismatch: got %v, want %q", retrieved["name"], container.Name)
	}

	// The files field will be retrieved as the typed slice []*FileNode
	// We need to use reflection to iterate over it
	filesField := retrieved["files"]
	if filesField == nil {
		t.Fatalf("Files field is nil")
	}

	filesValue := reflect.ValueOf(filesField)
	if filesValue.Kind() != reflect.Slice {
		t.Fatalf("Files field is not a slice, got kind %v", filesValue.Kind())
	}

	filesLen := filesValue.Len()
	t.Logf("Retrieved %d files (including possible nulls)", filesLen)

	// Verify non-nil entries
	nonNilCount := 0
	for i := 0; i < filesLen; i++ {
		filePtr := filesValue.Index(i)

		// Check if pointer is nil
		if filePtr.IsNil() {
			t.Logf("File %d: nil pointer", i)
			continue
		}

		// Dereference the pointer to get the struct
		fileStruct := filePtr.Elem()

		// Get name field
		nameField := fileStruct.FieldByName("Name")
		if nameField.IsValid() {
			t.Logf("File %d: name = %s", i, nameField.String())
			nonNilCount++
		}
	}

	t.Logf("Found %d non-nil file entries", nonNilCount)
}
