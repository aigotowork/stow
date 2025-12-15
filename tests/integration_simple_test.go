package stow_test

import (
	"testing"
	"github.com/aigotowork/stow"
)

// Helper type for simple string values
type StringValue struct {
	Value string
}

// Test with simple wrapper
func TestSimpleValueWrapper(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	// Put
	ns.MustPut("key1", StringValue{Value: "hello"})

	// Get
	var result StringValue
	ns.MustGet("key1", &result)

	if result.Value != "hello" {
		t.Errorf("Value mismatch: got %q", result.Value)
	}
}
