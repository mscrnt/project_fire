//go:build !windows
// +build !windows

package wmi

// WMIReader interface for reading memory info via WMI (stub)
type WMIReader interface {
	ReadMemoryInfo() ([]Module, error)
}

// Reader implements WMIReader interface (stub)
type Reader struct{}

// New creates a new WMI reader (stub for non-Windows)
func New() (*Reader, error) {
	return &Reader{}, nil
}

// ReadMemoryInfo reads memory module information via WMI (stub)
func (r *Reader) ReadMemoryInfo() ([]Module, error) {
	return nil, nil
}
