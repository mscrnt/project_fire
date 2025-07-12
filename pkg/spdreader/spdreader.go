//go:build windows
// +build windows

package spdreader

import (
	"fmt"
	"log"
	"time"

	"github.com/mscrnt/project_fire/pkg/spdreader/driver"
	"github.com/mscrnt/project_fire/pkg/spdreader/parser"
	"github.com/mscrnt/project_fire/pkg/spdreader/wmi"
)

// Reader interface for SPD reading implementations
type Reader interface {
	ReadAllModules() ([]SPDModule, error)
	Close() error
}

// SPDReader is the main SPD reader implementation
type SPDReader struct {
	driver driver.Driver
	wmi    interface {
		ReadMemoryInfo() ([]wmi.Module, error)
	}
}

// New creates a new SPD reader instance
func New() (*SPDReader, error) {
	r := &SPDReader{}

	// Try to initialize driver-based reader
	drv, err := driver.New()
	if err != nil {
		log.Printf("Failed to initialize driver-based reader: %v", err)
		log.Println("Falling back to WMI-based reader")

		// Fall back to WMI
		wmiReader, err := wmi.New()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize any reader: %v", err)
		}
		r.wmi = wmiReader
	} else {
		r.driver = drv
	}

	return r, nil
}

// ReadAllModules reads SPD data from all memory modules
func (r *SPDReader) ReadAllModules() ([]SPDModule, error) {
	// If driver is available, use it
	if r.driver != nil {
		return r.readViaDriver()
	}

	// Otherwise use WMI
	return r.readViaWMI()
}

// readViaDriver reads SPD data using the SMBus driver
func (r *SPDReader) readViaDriver() ([]SPDModule, error) {
	modules := []SPDModule{}

	// Get adapter count
	adapterCount, err := r.driver.GetAdapterCount()
	if err != nil {
		return nil, fmt.Errorf("failed to get adapter count: %v", err)
	}

	// Iterate through all adapters
	for adapter := uint8(0); adapter < adapterCount; adapter++ {
		// Check each possible DIMM address (0x50 to 0x57)
		for addr := uint8(0x50); addr <= 0x57; addr++ {
			// Try to read SPD data
			spdData, err := r.readSPDWithRetry(adapter, addr)
			if err != nil {
				// No module at this address
				continue
			}

			// Parse SPD data
			parsedModule, err := parser.ParseSPD(spdData)
			if err != nil {
				log.Printf("Failed to parse SPD data for adapter %d, address 0x%02X: %v", adapter, addr, err)
				continue
			}

			// Convert parser.Module to SPDModule
			module := SPDModule{
				Slot:              int(adapter)*8 + int(addr-0x50),
				Type:              parsedModule.Type,
				BaseFreqMHz:       parsedModule.BaseFreqMHz,
				DataRateMTs:       parsedModule.DataRateMTs,
				PCRate:            parsedModule.PCRate,
				CapacityGB:        parsedModule.CapacityGB,
				Ranks:             parsedModule.Ranks,
				DataWidth:         parsedModule.DataWidth,
				JEDECManufacturer: parsedModule.JEDECManufacturer,
				PartNumber:        parsedModule.PartNumber,
				Serial:            parsedModule.Serial,
				ManufacturingDate: parsedModule.ManufacturingDate,
				Timings: Timings{
					CL:    parsedModule.Timings.CL,
					RCD:   parsedModule.Timings.RCD,
					RP:    parsedModule.Timings.RP,
					RAS:   parsedModule.Timings.RAS,
					RC:    parsedModule.Timings.RC,
					RFC:   parsedModule.Timings.RFC,
					RRD_S: parsedModule.Timings.RRD_S,
					RRD_L: parsedModule.Timings.RRD_L,
					FAW:   parsedModule.Timings.FAW,
				},
				RawSPD: spdData,
			}

			modules = append(modules, module)
		}
	}

	return modules, nil
}

// readSPDWithRetry reads SPD data with retry logic
func (r *SPDReader) readSPDWithRetry(adapter, addr uint8) ([]byte, error) {
	maxRetries := 3
	retryDelay := 100 * time.Millisecond

	for i := 0; i < maxRetries; i++ {
		data, err := r.driver.ReadSPD(adapter, addr)
		if err == nil {
			return data, nil
		}

		if i < maxRetries-1 {
			time.Sleep(retryDelay)
			retryDelay *= 2 // Exponential backoff
		}
	}

	return nil, fmt.Errorf("failed to read SPD after %d retries", maxRetries)
}

// readViaWMI reads memory information via WMI
func (r *SPDReader) readViaWMI() ([]SPDModule, error) {
	wmiModules, err := r.wmi.ReadMemoryInfo()
	if err != nil {
		return nil, err
	}

	// Convert WMI modules to SPDModule
	modules := make([]SPDModule, len(wmiModules))
	for i, wm := range wmiModules {
		modules[i] = SPDModule{
			Slot:              wm.Slot,
			Type:              wm.Type,
			BaseFreqMHz:       wm.BaseFreqMHz,
			DataRateMTs:       wm.DataRateMTs,
			PCRate:            wm.PCRate,
			CapacityGB:        wm.CapacityGB,
			Ranks:             wm.Ranks,
			DataWidth:         wm.DataWidth,
			JEDECManufacturer: wm.JEDECManufacturer,
			PartNumber:        wm.PartNumber,
			Serial:            wm.Serial,
			Timings: Timings{
				CL:    wm.Timings.CL,
				RCD:   wm.Timings.RCD,
				RP:    wm.Timings.RP,
				RAS:   wm.Timings.RAS,
				RC:    wm.Timings.RC,
				RFC:   wm.Timings.RFC,
				RRD_S: wm.Timings.RRD_S,
				RRD_L: wm.Timings.RRD_L,
				FAW:   wm.Timings.FAW,
			},
		}
	}

	return modules, nil
}

// Close cleans up resources
func (r *SPDReader) Close() error {
	if r.driver != nil {
		return r.driver.Close()
	}
	return nil
}
