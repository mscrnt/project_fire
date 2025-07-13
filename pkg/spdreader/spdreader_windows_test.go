//go:build windows
// +build windows

package spdreader

import (
	"testing"
)

// TestSPDReaderInterface ensures SPDReader implements Reader interface on Windows
func TestSPDReaderInterface(t *testing.T) {
	var _ Reader = (*SPDReader)(nil)
}