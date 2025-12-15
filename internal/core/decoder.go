package core

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Decoder decodes JSONL format to Records.
type Decoder struct{}

// NewDecoder creates a new Decoder.
func NewDecoder() *Decoder {
	return &Decoder{}
}

// Decode decodes a single line of JSON to a Record.
// Returns an error if the line is not valid JSON or doesn't match the Record structure.
func (d *Decoder) Decode(line []byte) (*Record, error) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return nil, fmt.Errorf("empty line")
	}

	var record Record
	if err := json.Unmarshal(line, &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal record: %w", err)
	}

	if !record.IsValid() {
		return nil, fmt.Errorf("invalid record structure")
	}

	return &record, nil
}

// DecodeString decodes a JSON string to a Record.
func (d *Decoder) DecodeString(line string) (*Record, error) {
	return d.Decode([]byte(line))
}

// ReadAll reads all records from a file.
// Returns all successfully decoded records.
// Skips lines that can't be decoded (logs them but doesn't fail).
func (d *Decoder) ReadAll(filePath string) ([]*Record, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	var records []*Record
	scanner := bufio.NewScanner(f)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()

		record, err := d.Decode(line)
		if err != nil {
			// Skip invalid lines but continue reading
			// In production, this should log a warning
			continue
		}

		records = append(records, record)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return records, nil
}

// ReadLastValid reads the file from the end and returns the last valid "put" record.
// This is used by Get() to find the most recent value.
// Returns nil if no valid "put" record is found or if the key is deleted.
//
// Algorithm:
// 1. Read file in 4KB chunks from the end
// 2. Search for complete lines (ending with \n)
// 3. Try to decode each line
// 4. Return the first valid "put" record found
// 5. If only "delete" records found, return nil (key is deleted)
func (d *Decoder) ReadLastValid(filePath string) (*Record, error) {
	return d.ReadLastValidReverse(filePath)
}

// ReadLastValidReverse implements efficient reverse file reading using 4KB chunks.
// This minimizes memory usage for large files.
func (d *Decoder) ReadLastValidReverse(filePath string) (*Record, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	// Get file size
	stat, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}
	fileSize := stat.Size()

	if fileSize == 0 {
		return nil, nil
	}

	const chunkSize = 4096 // 4KB chunks
	buffer := make([]byte, chunkSize)
	var remainder []byte // Incomplete line from previous chunk
	pos := fileSize

	for pos > 0 {
		// Determine how much to read
		readSize := chunkSize
		if pos < int64(chunkSize) {
			readSize = int(pos)
		}

		// Move position backwards
		pos -= int64(readSize)

		// Read chunk
		if _, err := f.ReadAt(buffer[:readSize], pos); err != nil && err != io.EOF {
			return nil, fmt.Errorf("failed to read chunk: %w", err)
		}

		chunk := buffer[:readSize]

		// Combine with remainder from previous iteration
		if len(remainder) > 0 {
			chunk = append(chunk, remainder...)
		}

		// Find all newlines in chunk
		lines := bytes.Split(chunk, []byte{'\n'})

		// The last element is either empty (if chunk ended with \n) or incomplete
		if pos > 0 {
			// Save the incomplete first line for next iteration
			remainder = lines[0]
			lines = lines[1:]
		} else {
			// At beginning of file, include the first line if not empty
			if len(lines[0]) == 0 {
				lines = lines[1:]
			}
		}

		// Process lines in reverse order
		for i := len(lines) - 1; i >= 0; i-- {
			line := lines[i]
			if len(bytes.TrimSpace(line)) == 0 {
				continue // Skip empty lines
			}

			record, err := d.Decode(line)
			if err != nil {
				// Skip invalid lines
				continue
			}

			// If it's a delete operation, key is deleted
			if record.Meta.IsDelete() {
				return nil, nil
			}

			// If it's a put operation, return it
			if record.Meta.IsPut() {
				return record, nil
			}
		}
	}

	// No valid record found
	return nil, nil
}

// ReadVersion reads a specific version from a file.
// Returns the record with the specified version number.
func (d *Decoder) ReadVersion(filePath string, version int) (*Record, error) {
	records, err := d.ReadAll(filePath)
	if err != nil {
		return nil, err
	}

	for _, record := range records {
		if record.Meta.Version == version {
			return record, nil
		}
	}

	return nil, fmt.Errorf("version %d not found", version)
}

// CountLines counts the number of lines in a file.
// Used for compaction threshold checking.
func CountLines(filePath string) (int, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		count++
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("error reading file: %w", err)
	}

	return count, nil
}

// GetLatestVersion returns the highest version number in a file.
// Returns 0 if the file is empty or doesn't exist.
func (d *Decoder) GetLatestVersion(filePath string) (int, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return 0, nil
	}

	records, err := d.ReadAll(filePath)
	if err != nil {
		return 0, err
	}

	if len(records) == 0 {
		return 0, nil
	}

	// Find the maximum version
	maxVersion := 0
	for _, record := range records {
		if record.Meta.Version > maxVersion {
			maxVersion = record.Meta.Version
		}
	}

	return maxVersion, nil
}

// AppendRecord appends a record to a file (JSONL append-only mode).
func AppendRecord(filePath string, record *Record) error {
	// Encode the record
	encoder := NewEncoder()
	data, err := encoder.Encode(record)
	if err != nil {
		return fmt.Errorf("failed to encode record: %w", err)
	}

	// Open file in append mode
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	// Write the data
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	// Sync to disk
	if err := f.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	return nil
}

// ReadLastNRecords reads the last N records from a file.
// Used for compaction to keep recent history.
func (d *Decoder) ReadLastNRecords(filePath string, n int) ([]*Record, error) {
	records, err := d.ReadAll(filePath)
	if err != nil {
		return nil, err
	}

	if len(records) <= n {
		return records, nil
	}

	// Return last N records
	return records[len(records)-n:], nil
}

// ReadLines reads all lines from a reader.
func ReadLines(r io.Reader) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}
