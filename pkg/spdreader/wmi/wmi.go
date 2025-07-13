//go:build windows
// +build windows

package wmi

import (
	"fmt"
	"strings"

	"github.com/StackExchange/wmi"
)

// Reader interface for WMI-based memory reading
type Reader interface {
	ReadMemoryInfo() ([]Module, error)
}

// WMIReader implements memory reading via Windows WMI
type WMIReader struct{}

// Win32_PhysicalMemory WMI class
type Win32_PhysicalMemory struct {
	BankLabel            string
	Capacity             uint64
	ConfiguredClockSpeed uint32
	ConfiguredVoltage    uint32
	DataWidth            uint16
	DeviceLocator        string
	FormFactor           uint16
	HotSwappable         bool
	InterleaveDataDepth  uint16
	InterleavePosition   uint32
	Manufacturer         string
	MaxVoltage           uint32
	MemoryType           uint16
	MinVoltage           uint32
	Model                string
	Name                 string
	OperatingVoltage     uint32
	PartNumber           string
	PositionInRow        uint32
	PoweredOn            bool
	Removable            bool
	Replaceable          bool
	SerialNumber         string
	SKU                  string
	SMBIOSMemoryType     uint32
	Speed                uint32
	Status               string
	Tag                  string
	TotalWidth           uint16
	TypeDetail           uint16
	Version              string
}

// Memory type constants from SMBIOS
const (
	SMBIOSMemoryTypeDDR2   = 19
	SMBIOSMemoryTypeDDR3   = 24
	SMBIOSMemoryTypeDDR4   = 26
	SMBIOSMemoryTypeDDR5   = 34
	SMBIOSMemoryTypeLPDDR  = 28
	SMBIOSMemoryTypeLPDDR2 = 29
	SMBIOSMemoryTypeLPDDR3 = 30
	SMBIOSMemoryTypeLPDDR4 = 31
	SMBIOSMemoryTypeLPDDR5 = 35
)

// New creates a new WMI reader
func New() (*WMIReader, error) {
	return &WMIReader{}, nil
}

// ReadMemoryInfo reads memory information via WMI
func (r *WMIReader) ReadMemoryInfo() ([]Module, error) {
	var results []Win32_PhysicalMemory

	// Query WMI for physical memory
	err := wmi.Query("SELECT * FROM Win32_PhysicalMemory", &results)
	if err != nil {
		return nil, fmt.Errorf("WMI query failed: %v", err)
	}

	modules := make([]Module, 0, len(results))

	for i, mem := range results {
		// Skip empty slots
		if mem.Capacity == 0 {
			continue
		}

		module := Module{
			Slot:       i,
			CapacityGB: float64(mem.Capacity) / (1024 * 1024 * 1024),
			DataWidth:  int(mem.DataWidth),
		}

		// Determine memory type
		module.Type = getMemoryType(mem.SMBIOSMemoryType)

		// Speed information
		if mem.Speed > 0 {
			module.DataRateMTs = int(mem.Speed)
			module.BaseFreqMHz = float64(mem.Speed) / 2

			// Calculate PC rating
			busWidthBytes := 8 // Assume 64-bit
			if mem.DataWidth > 0 {
				busWidthBytes = int(mem.DataWidth) / 8
			}
			module.PCRate = module.DataRateMTs * busWidthBytes
		} else if mem.ConfiguredClockSpeed > 0 {
			// Use configured speed as fallback
			module.DataRateMTs = int(mem.ConfiguredClockSpeed)
			module.BaseFreqMHz = float64(mem.ConfiguredClockSpeed) / 2
		}

		// Manufacturer info
		module.JEDECManufacturer = cleanString(mem.Manufacturer)
		if module.JEDECManufacturer == "" {
			module.JEDECManufacturer = "Unknown"
		}

		module.PartNumber = cleanString(mem.PartNumber)
		module.Serial = cleanString(mem.SerialNumber)

		// Calculate ranks based on form factor and capacity
		// This is an approximation since WMI doesn't provide rank info
		module.Ranks = estimateRanks(module.CapacityGB, module.Type)

		// Add basic timing info if available
		if module.Type == "DDR4" || module.Type == "DDR5" {
			module.Timings = estimateTimings(module.DataRateMTs, module.Type)
		}

		modules = append(modules, module)
	}

	if len(modules) == 0 {
		return nil, fmt.Errorf("no memory modules found via WMI")
	}

	return modules, nil
}

// getMemoryType converts SMBIOS memory type to string
func getMemoryType(smbiosType uint32) string {
	switch smbiosType {
	case SMBIOSMemoryTypeDDR2:
		return "DDR2"
	case SMBIOSMemoryTypeDDR3:
		return "DDR3"
	case SMBIOSMemoryTypeDDR4:
		return "DDR4"
	case SMBIOSMemoryTypeDDR5:
		return "DDR5"
	case SMBIOSMemoryTypeLPDDR:
		return "LPDDR"
	case SMBIOSMemoryTypeLPDDR2:
		return "LPDDR2"
	case SMBIOSMemoryTypeLPDDR3:
		return "LPDDR3"
	case SMBIOSMemoryTypeLPDDR4:
		return "LPDDR4"
	case SMBIOSMemoryTypeLPDDR5:
		return "LPDDR5"
	default:
		return fmt.Sprintf("Unknown (%d)", smbiosType)
	}
}

// cleanString removes null bytes and trims whitespace
func cleanString(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "\x00")
	s = strings.TrimSpace(s)
	return s
}

// estimateRanks estimates the number of ranks based on capacity and type
func estimateRanks(capacityGB float64, memType string) int {
	// Most consumer modules are single or dual rank
	if capacityGB <= 8 {
		return 1
	} else if capacityGB <= 16 {
		return 2
	} else if capacityGB <= 32 {
		return 2
	} else {
		return 4
	}
}

// estimateTimings provides typical timing values based on speed
func estimateTimings(speedMTs int, memType string) Timings {
	t := Timings{}

	if memType == "DDR4" {
		switch speedMTs {
		case 2133:
			t.CL, t.RCD, t.RP, t.RAS = 15, 15, 15, 36
		case 2400:
			t.CL, t.RCD, t.RP, t.RAS = 16, 16, 16, 39
		case 2666:
			t.CL, t.RCD, t.RP, t.RAS = 18, 18, 18, 42
		case 2933:
			t.CL, t.RCD, t.RP, t.RAS = 19, 19, 19, 43
		case 3200:
			t.CL, t.RCD, t.RP, t.RAS = 22, 22, 22, 52
		case 3600:
			t.CL, t.RCD, t.RP, t.RAS = 18, 22, 22, 42
		default:
			// Generic DDR4 timings
			t.CL, t.RCD, t.RP, t.RAS = 19, 19, 19, 43
		}
	} else if memType == "DDR5" {
		switch speedMTs {
		case 4800:
			t.CL, t.RCD, t.RP, t.RAS = 40, 40, 40, 77
		case 5200:
			t.CL, t.RCD, t.RP, t.RAS = 42, 42, 42, 83
		case 5600:
			t.CL, t.RCD, t.RP, t.RAS = 46, 46, 46, 89
		case 6000:
			t.CL, t.RCD, t.RP, t.RAS = 48, 48, 48, 96
		case 6400:
			t.CL, t.RCD, t.RP, t.RAS = 52, 52, 52, 103
		default:
			// Generic DDR5 timings
			t.CL, t.RCD, t.RP, t.RAS = 46, 46, 46, 89
		}
	}

	// Calculate derived timings
	t.RC = t.RAS + t.RP
	t.RFC = t.RC * 6 // Rough estimate
	t.RRDS = 4
	t.RRDL = 6
	t.FAW = 16

	return t
}
