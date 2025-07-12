//go:build windows
// +build windows

package driver

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc/mgr"
)

// Embed the driver files
// Note: You'll need to place the actual driver files in this directory
// For now, we'll create placeholder embed directives

//go:embed RWEverything.sys
var driverSys []byte

//go:embed EWD.dll
var driverDll []byte

// Driver interface for SMBus operations
type Driver interface {
	GetAdapterCount() (uint8, error)
	GetAdapterInfo(index uint8) (*AdapterInfo, error)
	ReadSPD(adapter, addr uint8) ([]byte, error)
	Close() error
}

// AdapterInfo contains SMBus adapter information
type AdapterInfo struct {
	Name        string
	Description string
	VendorID    uint16
	DeviceID    uint16
}

// SMBusDriver implements the Driver interface using RWEverything
type SMBusDriver struct {
	dllHandle     windows.Handle
	tempDir       string
	serviceName   string
	serviceHandle *mgr.Service
	mgr           *mgr.Mgr

	// Function pointers
	initializeDriver *windows.LazyProc
	shutdownDriver   *windows.LazyProc
	getAdapterCount  *windows.LazyProc
	getAdapterInfo   *windows.LazyProc
	readSpdBlock     *windows.LazyProc
}

const (
	serviceName = "RWEverythingDriver"
	driverName  = "RWEverything.sys"
	dllName     = "EWD.dll"
)

// New creates a new SMBus driver instance
func New() (*SMBusDriver, error) {
	d := &SMBusDriver{
		serviceName: serviceName,
	}

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "spdreader_")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %v", err)
	}
	d.tempDir = tempDir

	// Extract driver files
	if err := d.extractFiles(); err != nil {
		d.cleanup()
		return nil, fmt.Errorf("failed to extract driver files: %v", err)
	}

	// Install driver service
	if err := d.installService(); err != nil {
		d.cleanup()
		return nil, fmt.Errorf("failed to install driver service: %v", err)
	}

	// Load DLL and get function pointers
	if err := d.loadDLL(); err != nil {
		d.cleanup()
		return nil, fmt.Errorf("failed to load driver DLL: %v", err)
	}

	// Initialize driver
	if err := d.initialize(); err != nil {
		d.cleanup()
		return nil, fmt.Errorf("failed to initialize driver: %v", err)
	}

	return d, nil
}

// extractFiles extracts the embedded driver files to temp directory
func (d *SMBusDriver) extractFiles() error {
	// Write driver sys file
	sysPath := filepath.Join(d.tempDir, driverName)
	if err := os.WriteFile(sysPath, driverSys, 0644); err != nil {
		return fmt.Errorf("failed to write driver sys file: %v", err)
	}

	// Write driver DLL file
	dllPath := filepath.Join(d.tempDir, dllName)
	if err := os.WriteFile(dllPath, driverDll, 0644); err != nil {
		return fmt.Errorf("failed to write driver DLL file: %v", err)
	}

	return nil
}

// installService installs the driver as a Windows service
func (d *SMBusDriver) installService() error {
	// Connect to service manager
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %v", err)
	}
	d.mgr = m

	// Check if service already exists
	service, err := m.OpenService(d.serviceName)
	if err == nil {
		// Service exists, try to stop and delete it
		service.Control(windows.SERVICE_CONTROL_STOP)
		service.Delete()
		service.Close()
	}

	// Create new service
	sysPath := filepath.Join(d.tempDir, driverName)
	service, err = m.CreateService(d.serviceName,
		sysPath,
		mgr.Config{
			ServiceType:  windows.SERVICE_KERNEL_DRIVER,
			StartType:    mgr.StartManual,
			ErrorControl: mgr.ErrorNormal,
			DisplayName:  "RWEverything Driver for SPD Reader",
		})
	if err != nil {
		return fmt.Errorf("failed to create service: %v", err)
	}
	d.serviceHandle = service

	// Start the service
	if err := service.Start(); err != nil {
		return fmt.Errorf("failed to start service: %v", err)
	}

	return nil
}

// loadDLL loads the driver DLL and gets function pointers
func (d *SMBusDriver) loadDLL() error {
	dllPath := filepath.Join(d.tempDir, dllName)

	// Load DLL
	dll, err := windows.LoadDLL(dllPath)
	if err != nil {
		return fmt.Errorf("failed to load DLL: %v", err)
	}
	d.dllHandle = windows.Handle(dll.Handle)

	// Get function pointers using LazyDLL
	lazyDll := windows.NewLazyDLL(dllPath)
	d.initializeDriver = lazyDll.NewProc("InitializeDriver")
	d.shutdownDriver = lazyDll.NewProc("ShutdownDriver")
	d.getAdapterCount = lazyDll.NewProc("GetSMBusAdapterCount")
	d.getAdapterInfo = lazyDll.NewProc("GetSMBusAdapterInfo")
	d.readSpdBlock = lazyDll.NewProc("ReadSPDBlock")

	return nil
}

// initialize initializes the driver
func (d *SMBusDriver) initialize() error {
	ret, _, err := d.initializeDriver.Call()
	if ret == 0 {
		return fmt.Errorf("failed to initialize driver: %v", err)
	}
	return nil
}

// GetAdapterCount returns the number of SMBus adapters
func (d *SMBusDriver) GetAdapterCount() (uint8, error) {
	count := uint8(0)
	ret, _, err := d.getAdapterCount.Call(uintptr(unsafe.Pointer(&count)))
	if ret == 0 {
		return 0, fmt.Errorf("failed to get adapter count: %v", err)
	}
	return count, nil
}

// GetAdapterInfo returns information about an SMBus adapter
func (d *SMBusDriver) GetAdapterInfo(index uint8) (*AdapterInfo, error) {
	// This would need the actual struct layout from the driver
	// For now, return a placeholder
	return &AdapterInfo{
		Name:        fmt.Sprintf("SMBus Adapter %d", index),
		Description: "Intel SMBus Controller",
	}, nil
}

// ReadSPD reads SPD data from the specified address
func (d *SMBusDriver) ReadSPD(adapter, addr uint8) ([]byte, error) {
	// SPD data is typically 256 or 512 bytes
	buffer := make([]byte, 512)
	length := uint32(len(buffer))

	// Read SPD data in blocks (SMBus typically reads 32 bytes at a time)
	const blockSize = 32
	for offset := uint32(0); offset < 512; offset += blockSize {
		blockLen := uint32(blockSize)
		if offset+blockSize > 512 {
			blockLen = 512 - offset
		}

		ret, _, err := d.readSpdBlock.Call(
			uintptr(adapter),
			uintptr(addr),
			uintptr(offset),
			uintptr(unsafe.Pointer(&buffer[offset])),
			uintptr(blockLen),
		)
		if ret == 0 {
			// If we can't read the first block, the DIMM slot is probably empty
			if offset == 0 {
				return nil, fmt.Errorf("no SPD data at address 0x%02X: %v", addr, err)
			}
			// Otherwise, we've read as much as we can
			length = offset
			break
		}
	}

	// Check if this is valid SPD data
	if !isValidSPD(buffer[:length]) {
		return nil, fmt.Errorf("invalid SPD data at address 0x%02X", addr)
	}

	return buffer[:length], nil
}

// isValidSPD checks if the data looks like valid SPD
func isValidSPD(data []byte) bool {
	if len(data) < 4 {
		return false
	}

	// Check for valid SPD revision byte (byte 1)
	// Valid revisions are typically 0x10-0x13 for DDR4, 0x20+ for DDR5
	revision := data[1]
	if revision < 0x10 || revision > 0x30 {
		return false
	}

	// Check for valid memory type (byte 2)
	memType := data[2]
	// Valid types: 0x0C = DDR4, 0x12 = DDR5
	if memType != 0x0C && memType != 0x12 {
		return false
	}

	// Check CRC bytes aren't all 0xFF
	if bytes.Count(data[0:128], []byte{0xFF}) == 128 {
		return false
	}

	return true
}

// Close cleans up the driver
func (d *SMBusDriver) Close() error {
	// Shutdown driver
	if d.shutdownDriver != nil {
		d.shutdownDriver.Call()
	}

	// Unload DLL
	if d.dllHandle != 0 {
		windows.FreeLibrary(d.dllHandle)
	}

	// Stop and delete service
	if d.serviceHandle != nil {
		d.serviceHandle.Control(windows.SERVICE_CONTROL_STOP)
		d.serviceHandle.Delete()
		d.serviceHandle.Close()
	}

	// Close service manager
	if d.mgr != nil {
		d.mgr.Disconnect()
	}

	// Clean up temp files
	d.cleanup()

	return nil
}

// cleanup removes temporary files
func (d *SMBusDriver) cleanup() {
	if d.tempDir != "" {
		os.RemoveAll(d.tempDir)
	}
}
