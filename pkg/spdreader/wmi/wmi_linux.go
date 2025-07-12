//go:build !windows
// +build !windows

package wmi

// Module represents a memory module from WMI (stub for non-Windows)
type Module struct {
	Slot              int
	Type              string
	BaseFreqMHz       float64
	DataRateMTs       int
	PCRate            int
	CapacityGB        float64
	Ranks             int
	DataWidth         int
	JEDECManufacturer string
	PartNumber        string
	Serial            string
	Timings           Timings
}

// Timings represents memory timing parameters
type Timings struct {
	CL    int
	RCD   int
	RP    int
	RAS   int
	RC    int
	RFC   int
	RRD_S int
	RRD_L int
	FAW   int
}

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