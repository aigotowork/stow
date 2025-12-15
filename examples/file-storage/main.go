package main

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/aigotowork/stow"
)

// Document represents a text document with metadata
type Document struct {
	Title       string    `json:"title"`
	Author      string    `json:"author"`
	Content     []byte    `json:"content" stow:"file"` // Always store as blob
	Tags        []string  `json:"tags"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	WordCount   int       `json:"word_count"`
	IsPublished bool      `json:"is_published"`
}

// TextFile represents a simple text file
type TextFile struct {
	Filename string    `json:"filename"`
	Content  []byte    `json:"content" stow:"file"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
}

// Image represents an image file with metadata
type Image struct {
	Filename string `json:"filename"`
	Data     []byte `json:"data" stow:"file"` // Image data stored as blob
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Format   string `json:"format"`
	Size     int64  `json:"size"`
}

func main() {
	storePath := "./data/file_storage_example"

	store, err := stow.Open(storePath)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	fmt.Println("=== File Storage Examples ===")
	fmt.Println()

	// Example 1: Store text documents
	demonstrateTextDocuments(store)

	// Example 2: Store text files
	demonstrateTextFiles(store)

	// Example 3: Store images (simulated)
	demonstrateImages(store)

	// Example 4: Force inline storage
	demonstrateForceInline(store)

	fmt.Println("\n=== All examples completed! ===")
	fmt.Printf("\nCheck the data at: %s\n", storePath)
	fmt.Println("- Documents are stored in 'documents/' namespace")
	fmt.Println("- Text files are stored in 'files/' namespace")
	fmt.Println("- Images are stored in 'images/' namespace")
	fmt.Println("- Large binary files are automatically stored in _blobs/")}

func demonstrateTextDocuments(store stow.Store) {
	fmt.Println("1. Text Documents Example")
	fmt.Println("   Storing documents with text content...")

	ns, err := store.GetNamespace("documents")
	if err != nil {
		log.Fatal(err)
	}

	// Create a sample document
	doc := Document{
		Title:  "Introduction to Stow",
		Author: "Example Author",
		Content: []byte(`
Stow is a simple embedded key-value storage engine designed for Go applications.
It uses human-readable JSONL format for data storage and supports automatic
blob routing for large binary data.

Key Features:
- Transparent JSONL storage format
- Automatic blob management for large files
- Version history tracking
- Zero dependencies
- Simple API

This makes it perfect for configuration management, document storage, and
lightweight application data persistence.
`),
		Tags:        []string{"tutorial", "introduction", "storage"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		WordCount:   58,
		IsPublished: true,
	}

	// Store the document
	if err := ns.Put("intro-to-stow", doc); err != nil {
		log.Fatal(err)
	}
	fmt.Println("   ✓ Stored document: 'intro-to-stow'")

	// Store another document
	doc2 := Document{
		Title:  "Advanced Features",
		Author: "Example Author",
		Content: []byte(`
This document covers advanced features of Stow including:
1. Nested struct support
2. Concurrent operations
3. Async compaction
4. Custom struct tags
5. Blob deduplication
`),
		Tags:        []string{"advanced", "features"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		WordCount:   23,
		IsPublished: false,
	}

	if err := ns.Put("advanced-features", doc2); err != nil {
		log.Fatal(err)
	}
	fmt.Println("   ✓ Stored document: 'advanced-features'")

	// Retrieve and display a document
	var retrieved Document
	if err := ns.Get("intro-to-stow", &retrieved); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   ✓ Retrieved document:\n")
	fmt.Printf("     - Title: %s\n", retrieved.Title)
	fmt.Printf("     - Author: %s\n", retrieved.Author)
	fmt.Printf("     - Word Count: %d\n", retrieved.WordCount)
	fmt.Printf("     - Content size: %d bytes\n", len(retrieved.Content))
	fmt.Printf("     - Tags: %v\n", retrieved.Tags)

	// List all documents
	keys, err := ns.List()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   ✓ Total documents: %d\n", len(keys))
	fmt.Println()
}

func demonstrateTextFiles(store stow.Store) {
	fmt.Println("2. Text Files Example")
	fmt.Println("   Storing plain text files...")

	ns, err := store.GetNamespace("files")
	if err != nil {
		log.Fatal(err)
	}

	// Simulate storing multiple text files
	files := []struct {
		name    string
		content string
	}{
		{"README.md", "# My Project\n\nThis is a sample README file.\n\n## Installation\n\n```bash\ngo get example.com/myproject\n```"},
		{"config.yaml", "server:\n  host: localhost\n  port: 8080\n  ssl: true\n\nlogging:\n  level: info\n  file: /var/log/app.log"},
		{"notes.txt", "Meeting notes from 2024-12-14:\n- Completed async compact implementation\n- Added nested struct support\n- Updated documentation"},
	}

	for _, f := range files {
		file := TextFile{
			Filename: f.name,
			Content:  []byte(f.content),
			Size:     int64(len(f.content)),
			Modified: time.Now(),
		}

		if err := ns.Put(f.name, file); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("   ✓ Stored file: '%s' (%d bytes)\n", f.name, len(f.content))
	}

	// Retrieve a file
	var readme TextFile
	if err := ns.Get("README.md", &readme); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   ✓ Retrieved: %s\n", readme.Filename)
	fmt.Printf("     Content preview: %s...\n", string(readme.Content[:50]))
	fmt.Println()
}

func demonstrateImages(store stow.Store) {
	fmt.Println("3. Images Example")
	fmt.Println("   Storing image files (simulated)...")

	ns, err := store.GetNamespace("images")
	if err != nil {
		log.Fatal(err)
	}

	// Simulate storing images
	// In real usage, you would read actual image files
	images := []struct {
		filename string
		width    int
		height   int
		format   string
		size     int
	}{
		{"profile-photo.jpg", 800, 600, "JPEG", 150000},
		{"banner.png", 1920, 1080, "PNG", 500000},
		{"thumbnail.jpg", 200, 200, "JPEG", 15000},
	}

	for _, img := range images {
		// Simulate image data (in real usage, read from file)
		imageData := make([]byte, img.size)
		// Fill with some pattern data to simulate image
		for i := range imageData {
			imageData[i] = byte(i % 256)
		}

		image := Image{
			Filename: img.filename,
			Data:     imageData,
			Width:    img.width,
			Height:   img.height,
			Format:   img.format,
			Size:     int64(img.size),
		}

		if err := ns.Put(img.filename, image); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("   ✓ Stored image: '%s' (%dx%d, %.1f KB)\n",
			img.filename, img.width, img.height, float64(img.size)/1024)
	}

	// Retrieve an image
	var banner Image
	if err := ns.Get("banner.png", &banner); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   ✓ Retrieved: %s\n", banner.Filename)
	fmt.Printf("     - Dimensions: %dx%d\n", banner.Width, banner.Height)
	fmt.Printf("     - Format: %s\n", banner.Format)
	fmt.Printf("     - Data size: %.1f KB\n", float64(len(banner.Data))/1024)
	fmt.Println()
}

func demonstrateForceInline(store stow.Store) {
	fmt.Println("4. Force Inline Storage Example")
	fmt.Println("   Forcing small files to be stored inline...")

	ns, err := store.GetNamespace("inline")
	if err != nil {
		log.Fatal(err)
	}

	// Small text that would normally be inlined
	smallText := []byte("This is a small text file.")

	// Store with default behavior (will be inlined due to size)
	if err := ns.Put("small-default", map[string]interface{}{
		"content": smallText,
	}); err != nil {
		log.Fatal(err)
	}
	fmt.Println("   ✓ Stored with default behavior (inlined)")

	// Store with ForceFile (will be stored as blob despite small size)
	if err := ns.Put("small-forced-file", map[string]interface{}{
		"content": smallText,
	}, stow.WithForceFile()); err != nil {
		log.Fatal(err)
	}
	fmt.Println("   ✓ Stored with ForceFile option (stored as blob)")

	// Large data forced to be inline (not recommended, for demo only)
	largeText := bytes.Repeat([]byte("A"), 2000)
	if err := ns.Put("large-forced-inline", map[string]interface{}{
		"content": largeText,
	}, stow.WithForceInline()); err != nil {
		log.Fatal(err)
	}
	fmt.Println("   ✓ Stored with ForceInline option (inlined despite size)")

	fmt.Printf("   ℹ Note: Check the JSONL files to see the difference\n")
	fmt.Println()
}
