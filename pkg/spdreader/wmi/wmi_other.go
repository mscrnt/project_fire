//go:build !windows
// +build !windows

package wmi

// Reader interface for reading memory info via WMI (stub)
type Reader interface {
	ReadMemoryInfo() ([]Module, error)
}

// WMIReader implements Reader interface (stub)
type WMIReader struct{}

// New creates a new WMI reader (stub for non-Windows)
func New() (*WMIReader, error) {
	return &WMIReader{}, nil
}

// ReadMemoryInfo reads memory module information via WMI (stub)
func (r *WMIReader) ReadMemoryInfo() ([]Module, error) {
	return nil, nil
}
