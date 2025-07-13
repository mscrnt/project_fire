//go:build !windows
// +build !windows

package wmi

// Reader interface for reading memory info via WMI (stub)
type Reader interface {
	ReadMemoryInfo() ([]Module, error)
}

// reader implements Reader interface (stub)
type reader struct{}

// New creates a new WMI reader (stub for non-Windows)
func New() (*reader, error) {
	return &reader{}, nil
}

// ReadMemoryInfo reads memory module information via WMI (stub)
func (r *reader) ReadMemoryInfo() ([]Module, error) {
	return nil, nil
}
