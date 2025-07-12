//go:build !windows
// +build !windows

package driver

// Driver interface for SPD reading drivers (stub for non-Windows)
type Driver interface {
	GetAdapterCount() (uint8, error)
	ReadSPD(adapter, addr uint8) ([]byte, error)
	Close() error
}

// New creates a new driver instance (stub for non-Windows)
func New() (Driver, error) {
	return nil, nil
}