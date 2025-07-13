//go:build !windows
// +build !windows

package gui

import "fmt"

// SPDReader provides SPD reading capabilities (stub for non-Windows)
type SPDReader struct{}

// SPDData contains parsed SPD information (stub for non-Windows)
type SPDData struct {
	Slot              int
	Revision          byte
	MemoryType        string
	MemoryTypeCode    byte
	PartNumber        string
	SerialNumber      uint32
	ManufacturerID    uint16
	JEDECManufacturer string
	ManufacturingDate string
	ModuleSize        uint64
	CapacityGB        float64
	Speed             uint32
	DataRateMTs       int
	PCRate            int
	BaseFreqMHz       float64
	Voltage           float32
	Ranks             int
	DataWidth         int
	BankGroups        byte
	BanksPerGroup     byte
	CASLatency        int
	RAStoCASDElay     int
	RASPrecharge      int
	tRAS              int
	tRC               int
	tRFC              int
	CommandRate       string
	Timings           struct {
		CL   int
		RCD  int
		RP   int
		RAS  int
		RC   int
		RFC  int
		RRDS int
		RRDL int
		FAW  int
	}
	HasXMP       bool
	HasEXPO      bool
	ProfileCount int
	RawSPD       []byte
}

// NewSPDReader creates a new SPD reader instance (stub)
func NewSPDReader() *SPDReader {
	return &SPDReader{}
}

// Initialize initializes the SPD reader (stub)
func (r *SPDReader) Initialize() error {
	return fmt.Errorf("SPD reading is not supported on this platform")
}

// Close closes the SPD reader (stub)
func (r *SPDReader) Close() {}

// ReadAllSPD reads SPD data from all memory modules (stub)
func (r *SPDReader) ReadAllSPD() ([]SPDData, error) {
	return nil, fmt.Errorf("SPD reading is not supported on this platform")
}

// GetManufacturerName converts JEDEC manufacturer ID to name (stub)
func GetManufacturerName(id uint16) string {
	return fmt.Sprintf("Unknown (0x%04X)", id)
}

// ReadMemoryModulesWithSPD enhances memory module information with SPD data (stub)
func ReadMemoryModulesWithSPD() ([]MemoryModule, error) {
	return GetMemoryModules()
}