//go:build windows
// +build windows

package driver

import (
	"fmt"
	"syscall"
)

// CPUZDriver implements Driver interface using CPU-Z driver
type CPUZDriver struct {
	dll *syscall.LazyDLL

	// Function pointers
	openDriver    *syscall.LazyProc
	closeDriver   *syscall.LazyProc
	readIoPort    *syscall.LazyProc
	writeIoPort   *syscall.LazyProc
	readPciConfig *syscall.LazyProc
	readMsr       *syscall.LazyProc
}

// NewCPUZDriver creates a driver instance using CPU-Z
func NewCPUZDriver() (*CPUZDriver, error) {
	// Try to load CPU-Z DLL (assumes it's installed)
	dll := syscall.NewLazyDLL("cpuz_x64.dll")

	d := &CPUZDriver{
		dll:           dll,
		openDriver:    dll.NewProc("OpenDriver"),
		closeDriver:   dll.NewProc("CloseDriver"),
		readIoPort:    dll.NewProc("ReadIoPortByte"),
		writeIoPort:   dll.NewProc("WriteIoPortByte"),
		readPciConfig: dll.NewProc("ReadPciConfigDword"),
		readMsr:       dll.NewProc("ReadMsr"),
	}

	// Open driver
	ret, _, err := d.openDriver.Call()
	if ret == 0 {
		return nil, fmt.Errorf("failed to open CPU-Z driver: %v", err)
	}

	return d, nil
}

// GetAdapterCount returns 1 for CPU-Z (single adapter)
func (d *CPUZDriver) GetAdapterCount() (uint8, error) {
	// CPU-Z typically only supports the primary SMBus controller
	return 1, nil
}

// GetAdapterInfo returns adapter information
func (d *CPUZDriver) GetAdapterInfo(index uint8) (*AdapterInfo, error) {
	if index > 0 {
		return nil, fmt.Errorf("invalid adapter index")
	}

	return &AdapterInfo{
		Name:        "Primary SMBus Controller",
		Description: "Intel ICH SMBus",
	}, nil
}

// ReadSPD reads SPD data using SMBus bit-banging via I/O ports
func (d *CPUZDriver) ReadSPD(adapter, addr uint8) ([]byte, error) {
	if adapter > 0 {
		return nil, fmt.Errorf("invalid adapter index")
	}

	// Intel ICH SMBus I/O ports
	const (
		SMBHSTSTS  = 0x0000 // Host Status
		SMBHSTCNT  = 0x0002 // Host Control
		SMBHSTCMD  = 0x0003 // Host Command
		SMBHSTADD  = 0x0004 // Host Address
		SMBHSTDAT0 = 0x0005 // Host Data 0
		SMBHSTDAT1 = 0x0006 // Host Data 1
		SMBBLKDAT  = 0x0007 // Block Data
	)

	// Base address for Intel SMBus (typically 0x0400 or 0x0500)
	// This would need to be detected from PCI config
	baseAddr := uint16(0x0400)

	// Read SPD data byte by byte
	data := make([]byte, 512)
	for i := 0; i < 512; i++ {
		// Set up SMBus read byte command
		// Clear status
		d.writeIoPort.Call(uintptr(baseAddr+SMBHSTSTS), 0xFF)

		// Set slave address (addr << 1 | 1 for read)
		d.writeIoPort.Call(uintptr(baseAddr+SMBHSTADD), uintptr((addr<<1)|1))

		// Set command (SPD offset)
		d.writeIoPort.Call(uintptr(baseAddr+SMBHSTCMD), uintptr(i))

		// Start transaction (byte data protocol)
		d.writeIoPort.Call(uintptr(baseAddr+SMBHSTCNT), 0x48)

		// Wait for completion
		timeout := 1000
		for timeout > 0 {
			status, _, _ := d.readIoPort.Call(uintptr(baseAddr + SMBHSTSTS))
			if status&0x02 != 0 { // INTR bit set
				break
			}
			timeout--
		}

		if timeout == 0 {
			// If first byte fails, slot is empty
			if i == 0 {
				return nil, fmt.Errorf("no SPD at address 0x%02X", addr)
			}
			// Otherwise return what we have
			return data[:i], nil
		}

		// Read data byte
		result, _, _ := d.readIoPort.Call(uintptr(baseAddr + SMBHSTDAT0))
		data[i] = byte(result)

		// Clear status
		d.writeIoPort.Call(uintptr(baseAddr+SMBHSTSTS), 0xFF)
	}

	// Validate SPD data
	if !isValidSPD(data) {
		return nil, fmt.Errorf("invalid SPD data")
	}

	return data, nil
}

// Close closes the CPU-Z driver
func (d *CPUZDriver) Close() error {
	if d.closeDriver != nil {
		d.closeDriver.Call()
	}
	return nil
}

// detectSMBusBase detects the SMBus base address from PCI config
func (d *CPUZDriver) detectSMBusBase() (uint16, error) {
	// Intel SMBus is typically at PCI device 31, function 3
	// Base address is at offset 0x20 in PCI config space

	bus, device, function := uint32(0), uint32(31), uint32(3)
	offset := uint32(0x20)

	// Read PCI config dword
	addr := (bus << 16) | (device << 11) | (function << 8) | (offset & 0xFC)
	value, _, err := d.readPciConfig.Call(uintptr(addr))
	if value == 0 || value == 0xFFFFFFFF {
		return 0, fmt.Errorf("failed to read PCI config: %v", err)
	}

	// Extract I/O base address (mask off bit 0)
	baseAddr := uint16(value & 0xFFFE)

	return baseAddr, nil
}
