//go:build !windows
// +build !windows

package spdreader

import (
	"fmt"
)

// Reader interface for SPD reading implementations
type Reader interface {
	ReadAllModules() ([]SPDModule, error)
	Close() error
}

// SPDReader is the main SPD reader implementation (stub for non-Windows)
type SPDReader struct{}

// New creates a new SPD reader instance (stub for non-Windows)
func New() (Reader, error) {
	return nil, fmt.Errorf("SPD reading is not supported on this platform")
}

// ReadAllModules reads SPD data from all memory modules (stub)
func (r *SPDReader) ReadAllModules() ([]SPDModule, error) {
	return nil, fmt.Errorf("SPD reading is not supported on this platform")
}

// Close cleans up resources (stub)
func (r *SPDReader) Close() error {
	return nil
}
