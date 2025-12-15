package stow

import (
	"fmt"
	"log"
	"os"
)

// Logger is the interface for logging in Stow.
// Users can provide custom logger implementations.
type Logger interface {
	// Debug logs a debug message with optional fields
	Debug(msg string, fields ...Field)

	// Info logs an info message with optional fields
	Info(msg string, fields ...Field)

	// Warn logs a warning message with optional fields
	Warn(msg string, fields ...Field)

	// Error logs an error message with optional fields
	Error(msg string, fields ...Field)
}

// defaultLogger is the default logger implementation using standard library log.
type defaultLogger struct {
	logger *log.Logger
}

// NewDefaultLogger creates a new default logger that writes to stderr.
func NewDefaultLogger() Logger {
	return &defaultLogger{
		logger: log.New(os.Stderr, "[stow] ", log.LstdFlags),
	}
}

func (l *defaultLogger) Debug(msg string, fields ...Field) {
	l.logger.Printf("[DEBUG] %s %s", msg, formatFields(fields))
}

func (l *defaultLogger) Info(msg string, fields ...Field) {
	l.logger.Printf("[INFO] %s %s", msg, formatFields(fields))
}

func (l *defaultLogger) Warn(msg string, fields ...Field) {
	l.logger.Printf("[WARN] %s %s", msg, formatFields(fields))
}

func (l *defaultLogger) Error(msg string, fields ...Field) {
	l.logger.Printf("[ERROR] %s %s", msg, formatFields(fields))
}

// noopLogger is a logger that does nothing. Useful for testing.
type noopLogger struct{}

// NewNoopLogger creates a logger that discards all log messages.
func NewNoopLogger() Logger {
	return &noopLogger{}
}

func (l *noopLogger) Debug(msg string, fields ...Field) {}
func (l *noopLogger) Info(msg string, fields ...Field)  {}
func (l *noopLogger) Warn(msg string, fields ...Field)  {}
func (l *noopLogger) Error(msg string, fields ...Field) {}

// formatFields formats fields for logging.
func formatFields(fields []Field) string {
	if len(fields) == 0 {
		return ""
	}

	result := "{"
	for i, field := range fields {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf("%s: %v", field.Key, field.Value)
	}
	result += "}"

	return result
}
