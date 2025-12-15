package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/aigotowork/stow"
)

// Document represents a document with io.Reader content field
type Document struct {
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	Content   io.Reader `json:"content"` // io.Reader field will be stored as blob
	CreatedAt time.Time `json:"created_at"`
}

// VideoFile with streaming data
type VideoFile struct {
	Filename   string    `json:"filename"`
	Duration   int       `json:"duration_seconds"`
	Resolution string    `json:"resolution"`
	VideoData  io.Reader `json:"video_data"` // Large video stream
	UploadedAt time.Time `json:"uploaded_at"`
}

// LogFile with streaming log content
type LogFile struct {
	Service    string    `json:"service"`
	Date       string    `json:"date"`
	Level      string    `json:"level"`
	LogContent io.Reader `json:"log_content"` // Log stream
	LineCount  int       `json:"line_count"`
}

// ImageMetadata with image reader
type ImageMetadata struct {
	Filename string    `json:"filename"`
	Width    int       `json:"width"`
	Height   int       `json:"height"`
	Format   string    `json:"format"`
	ImageData io.Reader `json:"image_data"` // Image binary data
	Size     int64     `json:"size"`
}

func main() {
	fmt.Println("=== io.Reader Field Examples ===")
	fmt.Println()

	// Initialize store
	store, err := stow.Open("./data/reader_field_example")
	if err != nil {
		panic(err)
	}
	defer store.Close()

	// Example 1: Document with string reader content
	example1DocumentWithStringReader(store)

	// Example 2: Video file with large data stream
	example2VideoFileStream(store)

	// Example 3: Log file with multi-reader content
	example3LogFileWithMultiReader(store)

	// Example 4: Image with binary data stream
	example4ImageWithBinaryStream(store)

	// Example 5: Reading from actual file
	example5ReadingFromActualFile(store)

	fmt.Println()
	fmt.Println("=== All examples completed! ===")
	fmt.Println()
	fmt.Println("Check the data at: ./data/reader_field_example")
	fmt.Println("- io.Reader fields are automatically stored as blobs in _blobs/")
	fmt.Println("- Metadata is stored in JSONL files")
}

// Example 1: Document with string reader content
func example1DocumentWithStringReader(store stow.Store) {
	fmt.Println("1. Document with io.Reader Field (String Content)")

	ns, _ := store.GetNamespace("documents")

	content := `# Introduction to Stow io.Reader Fields

Stow automatically recognizes io.Reader fields in structs and stores
them as blob files. This allows you to stream large content efficiently
without loading everything into memory.

## Benefits

- Memory efficient streaming
- Automatic blob storage
- Type-safe struct fields
- Clean API

This document demonstrates storing text content through an io.Reader field.`

	doc := Document{
		Title:     "Streaming Guide",
		Author:    "Stow Team",
		Content:   strings.NewReader(content),
		CreatedAt: time.Now(),
	}

	// Store - the Content field will be automatically stored as a blob
	err := ns.Put("streaming-guide", doc)
	if err != nil {
		fmt.Printf("   ✗ Failed: %v\n", err)
		return
	}

	fmt.Printf("   ✓ Stored document: '%s' by %s\n", doc.Title, doc.Author)
	fmt.Printf("   ✓ Content field (io.Reader) stored as blob\n")
	fmt.Println()

	// Scenario 1: Retrieve with metadata only (no content)
	fmt.Println("   Scenario 1: Retrieve metadata only")
	var retrieved1 struct {
		Title     string    `json:"title"`
		Author    string    `json:"author"`
		CreatedAt time.Time `json:"created_at"`
	}
	ns.MustGet("streaming-guide", &retrieved1)
	fmt.Printf("   ✓ Retrieved: %s by %s\n", retrieved1.Title, retrieved1.Author)
	fmt.Println()

	// Scenario 2: Retrieve content as []byte
	fmt.Println("   Scenario 2: Retrieve content as []byte")
	var retrieved2 struct {
		Title     string    `json:"title"`
		Author    string    `json:"author"`
		Content   []byte    `json:"content"` // Retrieve as []byte
		CreatedAt time.Time `json:"created_at"`
	}
	ns.MustGet("streaming-guide", &retrieved2)
	fmt.Printf("   ✓ Content as []byte: %d bytes\n", len(retrieved2.Content))
	fmt.Printf("   ✓ Preview: %.50s...\n", string(retrieved2.Content))
	fmt.Println()

	// Scenario 3: Retrieve content as io.Reader
	fmt.Println("   Scenario 3: Retrieve content as io.Reader")
	var retrieved3 struct {
		Title     string    `json:"title"`
		Author    string    `json:"author"`
		Content   io.Reader `json:"content"` // Retrieve as io.Reader
		CreatedAt time.Time `json:"created_at"`
	}
	ns.MustGet("streaming-guide", &retrieved3)
	if retrieved3.Content != nil {
		// Read from the reader
		contentBytes, _ := io.ReadAll(retrieved3.Content)
		fmt.Printf("   ✓ Content as io.Reader: %d bytes\n", len(contentBytes))
		fmt.Printf("   ✓ Preview: %.50s...\n", string(contentBytes))
	}
	fmt.Println()
}

// Example 2: Video file with large data stream
func example2VideoFileStream(store stow.Store) {
	fmt.Println("2. Video File with Large Stream")

	ns, _ := store.GetNamespace("media")

	// Simulate a 5MB video file
	videoSize := 5 * 1024 * 1024
	videoData := make([]byte, videoSize)
	for i := range videoData {
		videoData[i] = byte((i / 1024) % 256)
	}

	video := VideoFile{
		Filename:   "introduction.mp4",
		Duration:   300, // 5 minutes
		Resolution: "1920x1080",
		VideoData:  bytes.NewReader(videoData),
		UploadedAt: time.Now(),
	}

	start := time.Now()
	err := ns.Put("video-001", video)
	if err != nil {
		fmt.Printf("   ✗ Failed: %v\n", err)
		return
	}
	elapsed := time.Since(start)

	fmt.Printf("   ✓ Stored video: %s (%s)\n", video.Filename, video.Resolution)
	fmt.Printf("   ✓ Video stream: %.2f MB in %v\n",
		float64(videoSize)/(1024*1024), elapsed)
	fmt.Printf("   ✓ Speed: %.2f MB/s\n", float64(videoSize)/(1024*1024)/elapsed.Seconds())
	fmt.Println()
}

// Example 3: Log file with multi-reader content
func example3LogFileWithMultiReader(store stow.Store) {
	fmt.Println("3. Log File with Multi-Reader (Chunked Stream)")

	ns, _ := store.GetNamespace("logs")

	// Simulate log file from multiple sources
	chunks := []string{
		"[INFO] 2025-12-14 10:00:00 Service started\n",
		"[INFO] 2025-12-14 10:00:01 Connected to database\n",
		"[INFO] 2025-12-14 10:00:02 Server listening on :8080\n",
		"[WARN] 2025-12-14 10:00:03 High memory usage: 85%\n",
		"[INFO] 2025-12-14 10:00:04 Request processed: GET /api/users\n",
		"[ERROR] 2025-12-14 10:00:05 Database connection lost\n",
		"[INFO] 2025-12-14 10:00:06 Reconnecting to database...\n",
		"[INFO] 2025-12-14 10:00:07 Database connection restored\n",
	}

	var readers []io.Reader
	for _, chunk := range chunks {
		readers = append(readers, strings.NewReader(chunk))
	}

	// Combine all chunks into a single reader
	combinedReader := io.MultiReader(readers...)

	logFile := LogFile{
		Service:    "api-server",
		Date:       "2025-12-14",
		Level:      "mixed",
		LogContent: combinedReader,
		LineCount:  len(chunks),
	}

	err := ns.Put("log-2025-12-14", logFile)
	if err != nil {
		fmt.Printf("   ✗ Failed: %v\n", err)
		return
	}

	fmt.Printf("   ✓ Stored log: %s (%d lines)\n", logFile.Service, logFile.LineCount)
	fmt.Printf("   ✓ Log content streamed from %d chunks\n", len(chunks))
	fmt.Println()
}

// Example 4: Image with binary data stream
func example4ImageWithBinaryStream(store stow.Store) {
	fmt.Println("4. Image with Binary Data Stream")

	ns, _ := store.GetNamespace("images")

	// Simulate a 500KB image
	imageSize := 500 * 1024
	imageData := make([]byte, imageSize)
	// Simulate image pattern (e.g., JPEG header)
	imageData[0], imageData[1] = 0xFF, 0xD8 // JPEG marker
	for i := 2; i < len(imageData); i++ {
		imageData[i] = byte(i % 256)
	}

	image := ImageMetadata{
		Filename:  "profile-photo.jpg",
		Width:     800,
		Height:    600,
		Format:    "JPEG",
		ImageData: bytes.NewReader(imageData),
		Size:      int64(imageSize),
	}

	err := ns.Put("profile-001", image)
	if err != nil {
		fmt.Printf("   ✗ Failed: %v\n", err)
		return
	}

	fmt.Printf("   ✓ Stored image: %s (%dx%d)\n", image.Filename, image.Width, image.Height)
	fmt.Printf("   ✓ Image data: %.1f KB (%s format)\n",
		float64(image.Size)/1024, image.Format)
	fmt.Println()
}

// Example 5: Reading from actual file
func example5ReadingFromActualFile(store stow.Store) {
	fmt.Println("5. Reading from Actual File (if available)")

	ns, _ := store.GetNamespace("files")

	// Create a temporary file for demonstration
	tmpFile, err := os.CreateTemp("", "stow-example-*.txt")
	if err != nil {
		fmt.Printf("   ⚠ Skipped: could not create temp file\n\n")
		return
	}
	defer os.Remove(tmpFile.Name())

	// Write some content
	content := []byte("This is content read from an actual file.\nLine 2\nLine 3\n")
	tmpFile.Write(content)
	tmpFile.Close()

	// Open file for reading
	file, err := os.Open(tmpFile.Name())
	if err != nil {
		fmt.Printf("   ✗ Failed to open file: %v\n", err)
		return
	}
	defer file.Close()

	// Get file info
	info, _ := file.Stat()

	doc := Document{
		Title:     "File Upload Example",
		Author:    "System",
		Content:   file, // Pass the file handle as io.Reader
		CreatedAt: info.ModTime(),
	}

	err = ns.Put("uploaded-file", doc)
	if err != nil {
		fmt.Printf("   ✗ Failed: %v\n", err)
		return
	}

	fmt.Printf("   ✓ Stored document from file: %s\n", doc.Title)
	fmt.Printf("   ✓ File size: %d bytes\n", info.Size())
	fmt.Printf("   ✓ Content streamed directly from file handle\n")
	fmt.Println()
}
