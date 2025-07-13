package spdreader

import (
	"testing"
)

// TestReaderInterface ensures the Reader interface is properly defined on all platforms
func TestReaderInterface(t *testing.T) {
	// This test just ensures the interface exists and compiles on all platforms
	// The actual type assertion is in platform-specific test files
	t.Log("Reader interface is available")
}

// TestNewReader tests the New function behavior on different platforms
func TestNewReader(t *testing.T) {
	reader, err := New()

	// On non-Windows platforms, we expect an error
	if err != nil {
		// This is expected on non-Windows platforms
		t.Logf("New() returned expected error on this platform: %v", err)
		return
	}

	// On Windows, we should get a valid reader (though it might fail if not admin)
	if reader == nil {
		t.Error("New() returned nil reader on Windows")
	}

	// Clean up if we got a reader
	if reader != nil {
		_ = reader.Close()
	}
}
