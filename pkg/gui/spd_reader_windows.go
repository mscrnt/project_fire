//go:build windows
// +build windows

package gui

import (
	"encoding/binary"
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// SPDReader provides direct SPD (Serial Presence Detect) reading capabilities
type SPDReader struct {
	dll                 *syscall.LazyDLL
	procInitialize      *syscall.LazyProc
	procDeinitialize    *syscall.LazyProc
	procGetAdapterCount *syscall.LazyProc
	procGetAdapterInfo  *syscall.LazyProc
	procSmbusReadBlock  *syscall.LazyProc
	initialized         bool
}

// SMBUSAdapterInfo matches the C struct from OlsApi.h
type SMBUSAdapterInfo struct {
	Reserved     byte
	ChannelCount byte
	BasePort     uint16
	VendorID     uint32
	DeviceID     uint32
	Bus          byte
	Device       byte
	Function     byte
	Reserved2    byte
}

// SPDData contains parsed SPD information
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
	ModuleSize        uint64  // in bytes
	CapacityGB        float64 // in GB
	Speed             uint32  // in MHz
	DataRateMTs       int     // MT/s
	PCRate            int     // PC rating
	BaseFreqMHz       float64 // Base frequency in MHz
	Voltage           float32
	Ranks             int
	DataWidth         int

	// DDR5 specific
	BankGroups    byte
	BanksPerGroup byte

	// Timing parameters
	CASLatency    int
	RAStoCASDElay int
	RASPrecharge  int
	tRAS          int
	tRC           int
	tRFC          int
	CommandRate   string

	// Timing struct for compatibility
	Timings struct {
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

	// XMP/EXPO profiles
	HasXMP       bool
	HasEXPO      bool
	ProfileCount int

	// Raw SPD data
	RawSPD []byte
}

// NewSPDReader creates a new SPD reader instance
func NewSPDReader() *SPDReader {
	// Try different possible DLL names
	dll := syscall.NewLazyDLL("OlsApi.dll")

	// Check if DLL can be loaded
	if err := dll.Load(); err != nil {
		// Try alternative names
		dll = syscall.NewLazyDLL("WinRing0x64.dll")
		if err := dll.Load(); err != nil {
			dll = syscall.NewLazyDLL("OlsApi64.dll")
			if err := dll.Load(); err != nil {
				DebugLog("SPD", fmt.Sprintf("WinRing0 DLL not found: %v", err))
			}
		}
	}

	return &SPDReader{
		dll: dll,
	}
}

// Initialize initializes the WinRing0 driver
func (r *SPDReader) Initialize() error {
	if r.initialized {
		return nil
	}

	// Check if DLL is loaded
	if r.dll == nil {
		return fmt.Errorf("WinRing0 DLL not loaded")
	}

	// Try to load the DLL
	if err := r.dll.Load(); err != nil {
		return fmt.Errorf("failed to load WinRing0 DLL: %v", err)
	}

	r.procInitialize = r.dll.NewProc("InitializeOls")
	r.procDeinitialize = r.dll.NewProc("DeinitializeOls")
	r.procGetAdapterCount = r.dll.NewProc("GetSmbusAdapterCount")
	r.procGetAdapterInfo = r.dll.NewProc("GetSmbusAdapterInfo")
	r.procSmbusReadBlock = r.dll.NewProc("SmbusReadBlock")

	// Check if procedures are found
	if err := r.procInitialize.Find(); err != nil {
		return fmt.Errorf("WinRing0 DLL found but InitializeOls not available: %v", err)
	}

	// Check other procedures
	if err := r.procGetAdapterCount.Find(); err != nil {
		DebugLog("SPD", fmt.Sprintf("Warning: GetSmbusAdapterCount not found: %v", err))
	}

	ret, _, err := r.procInitialize.Call()
	if ret == 0 {
		return fmt.Errorf("failed to initialize WinRing0 driver (needs Administrator): %v", err)
	}

	r.initialized = true
	return nil
}

// Close deinitializes the WinRing0 driver
func (r *SPDReader) Close() {
	if r.initialized {
		r.procDeinitialize.Call()
		r.initialized = false
	}
}

// ReadAllSPD reads SPD data from all memory modules
func (r *SPDReader) ReadAllSPD() ([]SPDData, error) {
	DebugLog("SPD", "Entering ReadAllSPD")

	if !r.initialized {
		DebugLog("SPD", "Not initialized, initializing now")
		if err := r.Initialize(); err != nil {
			return nil, err
		}
	}

	var results []SPDData

	DebugLog("SPD", "Getting adapter count...")

	// Get adapter count
	var count uint32
	ret, _, err := r.procGetAdapterCount.Call(uintptr(unsafe.Pointer(&count)))
	DebugLog("SPD", fmt.Sprintf("GetAdapterCount returned: ret=%d, err=%v", ret, err))

	if ret == 0 {
		return nil, fmt.Errorf("failed to get adapter count: %v", err)
	}

	DebugLog("SPD", fmt.Sprintf("Found %d SMBUS adapters", count))

	// For each adapter
	for i := uint32(0); i < count; i++ {
		var info SMBUSAdapterInfo
		ret, _, _ := r.procGetAdapterInfo.Call(
			uintptr(i),
			uintptr(unsafe.Pointer(&info)),
		)
		if ret == 0 {
			DebugLog("SPD", fmt.Sprintf("Failed to get info for adapter %d", i))
			continue
		}

		DebugLog("SPD", fmt.Sprintf("Adapter %d: BasePort=0x%X, VendorID=0x%X, DeviceID=0x%X",
			i, info.BasePort, info.VendorID, info.DeviceID))

		// Try SPD addresses 0x50-0x57 (8 possible DIMM slots)
		for addr := byte(0x50); addr <= 0x57; addr++ {
			spd := make([]byte, 512) // DDR5 uses 512 bytes
			length := r.readSPDBlock(byte(i), addr, spd)

			if length >= 256 { // Valid SPD data
				DebugLog("SPD", fmt.Sprintf("Found SPD data at address 0x%X (length=%d)", addr, length))
				if data, err := r.parseSPD(spd[:length]); err == nil {
					// Set slot number based on address
					data.Slot = int(addr - 0x50)
					DebugLog("SPD", fmt.Sprintf("Parsed SPD: Type=%s, Size=%d MB, Speed=%d MHz, PartNumber=%s",
						data.MemoryType, data.ModuleSize/(1024*1024), data.Speed, data.PartNumber))
					results = append(results, data)
				} else {
					DebugLog("SPD", fmt.Sprintf("Failed to parse SPD at 0x%X: %v", addr, err))
				}
			}
		}
	}

	DebugLog("SPD", fmt.Sprintf("Total SPD entries found: %d", len(results)))

	return results, nil
}

// readSPDBlock reads a block of SPD data
func (r *SPDReader) readSPDBlock(adapter, addr byte, buf []byte) int {
	length := uint32(len(buf))
	ret, _, _ := r.procSmbusReadBlock.Call(
		uintptr(adapter),
		uintptr(addr),
		uintptr(0x00), // command
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&length)),
	)
	if ret == 0 {
		return 0
	}
	return int(length)
}

// parseSPD parses SPD data based on revision
func (r *SPDReader) parseSPD(spd []byte) (SPDData, error) {
	if len(spd) < 128 {
		return SPDData{}, fmt.Errorf("SPD data too short")
	}

	data := SPDData{
		RawSPD: spd,
	}

	// SPD revision
	data.Revision = spd[2]

	// Memory type detection
	var memTypeCode byte
	if data.Revision >= 5 { // DDR5
		memTypeCode = spd[3] & 0x0F
	} else { // DDR4 and earlier
		memTypeCode = spd[2]
	}

	data.MemoryTypeCode = memTypeCode
	data.MemoryType = r.getMemoryTypeName(memTypeCode)

	// Parse based on memory type
	if data.Revision >= 5 {
		r.parseDDR5SPD(spd, &data)
	} else {
		r.parseDDR4SPD(spd, &data)
	}

	// Calculate additional fields
	data.CapacityGB = float64(data.ModuleSize) / (1024 * 1024 * 1024)
	data.DataRateMTs = int(data.Speed)
	data.PCRate = data.DataRateMTs * 8
	data.BaseFreqMHz = float64(data.Speed) / 2.0

	// Get manufacturer name
	data.JEDECManufacturer = GetManufacturerName(data.ManufacturerID)

	// Default values
	if data.Ranks == 0 {
		data.Ranks = 1
	}
	if data.DataWidth == 0 {
		data.DataWidth = 64
	}

	// Populate timing struct
	data.Timings.CL = data.CASLatency
	data.Timings.RCD = data.RAStoCASDElay
	data.Timings.RP = data.RASPrecharge
	data.Timings.RAS = data.tRAS
	data.Timings.RC = data.tRC
	data.Timings.RFC = data.tRFC
	// Default values for RRDS/RRDL/FAW
	data.Timings.RRDS = 4
	data.Timings.RRDL = 6
	data.Timings.FAW = 16

	return data, nil
}

// parseDDR5SPD parses DDR5 specific SPD data
func (r *SPDReader) parseDDR5SPD(spd []byte, data *SPDData) {
	// Module organization
	// Byte 6: SDRAM density and banks
	density := (spd[6] & 0x0F)       // bits 0-3
	bankBits := (spd[6] >> 4) & 0x03 // bits 4-5
	data.BankGroups = 1 << bankBits

	// Byte 7: SDRAM Addressing (for future use)
	// rowBits := (spd[7] & 0x1F) + 12
	// colBits := ((spd[7] >> 5) & 0x07) + 9

	// Calculate module size
	// Size = density * 8 * (bus width / 8) * ranks
	densityMB := 1 << (density + 8) // Convert to MB
	busWidth := 64                  // Standard for DDR5
	ranks := (spd[234] & 0x07) + 1
	data.ModuleSize = uint64(densityMB) * uint64(busWidth/8) * uint64(ranks) * 1024 * 1024

	// Speed - MTB (Medium Timebase)
	mtb := 0.125 // 125ps for DDR5
	// ftb := 1.0   // 1ps for DDR5 (for future fine timing)

	// tCKavg min (bytes 18-19)
	tCKmin := int(spd[18]) | (int(spd[19]) << 8)
	if tCKmin > 0 {
		freqMHz := 1000000.0 / (float64(tCKmin) * mtb)
		data.Speed = uint32(freqMHz * 2) // DDR = Double Data Rate
	}

	// Voltage (byte 14)
	vdd := spd[14]
	if vdd&0x01 != 0 {
		data.Voltage = 1.1
	}

	// Part number (bytes 521-550 for DDR5)
	if len(spd) >= 551 {
		partBytes := spd[521:551]
		data.PartNumber = strings.TrimSpace(string(partBytes))
	}

	// Serial number (bytes 517-520)
	if len(spd) >= 521 {
		data.SerialNumber = binary.LittleEndian.Uint32(spd[517:521])
	}

	// Manufacturer ID (bytes 512-513)
	if len(spd) >= 514 {
		data.ManufacturerID = binary.LittleEndian.Uint16(spd[512:514])
	}

	// Manufacturing date (bytes 515-516)
	if len(spd) >= 517 {
		year := spd[515]
		week := spd[516]
		data.ManufacturingDate = fmt.Sprintf("Week %d, 20%02d", week, year)
	}

	// CAS Latency
	// DDR5 uses different encoding
	cl := int(spd[20]) | (int(spd[21]) << 8) | (int(spd[22]) << 16)
	for i := 0; i < 24; i++ {
		if cl&(1<<i) != 0 {
			data.CASLatency = i + 20 // DDR5 starts at CL20
			break
		}
	}

	// Additional timing parameters for DDR5
	data.RAStoCASDElay = int(spd[23])
	data.RASPrecharge = int(spd[24])
	data.tRAS = int(spd[25]) | (int(spd[26]&0x0F) << 8)
	data.tRC = int(spd[27]) | (int(spd[26]&0xF0) << 4)
	data.tRFC = int(spd[28]) | (int(spd[29]) << 8)

	// Check for XMP/EXPO profiles (byte 640 onwards)
	if len(spd) >= 700 {
		if spd[640] == 0x0C && spd[641] == 0x4A { // XMP 3.0 magic
			data.HasXMP = true
			data.ProfileCount = int(spd[642] & 0x03)
		} else if spd[640] == 0x08 && spd[641] == 0x00 { // AMD EXPO
			data.HasEXPO = true
			data.ProfileCount = int(spd[642] & 0x03)
		}
	}
}

// parseDDR4SPD parses DDR4 specific SPD data
func (r *SPDReader) parseDDR4SPD(spd []byte, data *SPDData) {
	// Module organization
	// Byte 4: SDRAM density and banks
	density := (spd[4] & 0x0F)

	// Byte 6: Module organization
	busWidth := 8 << (spd[13] & 0x07)
	ranks := (spd[12] & 0x07) + 1

	// Calculate module size
	densityMB := 256 << density // DDR4 density encoding
	data.ModuleSize = uint64(densityMB) * uint64(busWidth/8) * uint64(ranks) * 1024 * 1024

	// Speed
	mtb := 0.125 // 125ps for DDR4
	tCKmin := int(spd[18])
	if tCKmin > 0 {
		freqMHz := 1000000.0 / (float64(tCKmin) * mtb)
		data.Speed = uint32(freqMHz * 2)
	}

	// Part number (bytes 329-348)
	if len(spd) >= 349 {
		partBytes := spd[329:349]
		data.PartNumber = strings.TrimSpace(string(partBytes))
	}

	// Serial number (bytes 325-328)
	if len(spd) >= 329 {
		data.SerialNumber = binary.LittleEndian.Uint32(spd[325:329])
	}

	// Manufacturer ID (bytes 320-321)
	if len(spd) >= 322 {
		data.ManufacturerID = binary.LittleEndian.Uint16(spd[320:322])
	}

	// CAS Latency
	cl := uint32(spd[14]) | (uint32(spd[15]) << 8) | (uint32(spd[16]) << 16) | (uint32(spd[17]) << 24)
	for i := 0; i < 32; i++ {
		if cl&(1<<i) != 0 {
			data.CASLatency = i + 7 // DDR4 starts at CL7
			break
		}
	}

	// Additional timing parameters for DDR4
	data.RAStoCASDElay = int(spd[25])
	data.RASPrecharge = int(spd[26])
	data.tRAS = int(spd[28]) | (int(spd[27]&0x0F) << 8)
	data.tRC = int(spd[29]) | (int(spd[27]&0xF0) << 4)
	data.tRFC = int(spd[30]) | (int(spd[31]) << 8)

	// Check for XMP profiles
	if len(spd) >= 400 {
		if spd[384] == 0x0C && spd[385] == 0x4A { // XMP 2.0 magic
			data.HasXMP = true
			data.ProfileCount = 2 // XMP 2.0 supports up to 2 profiles
		}
	}
}

// getMemoryTypeName converts memory type code to string
func (r *SPDReader) getMemoryTypeName(code byte) string {
	switch code {
	case 0x0B:
		return "DDR3 SDRAM"
	case 0x0C:
		return "DDR4 SDRAM"
	case 0x0D:
		return "DDR5 SDRAM"
	case 0x0E:
		return "LPDDR4 SDRAM"
	case 0x0F:
		return "LPDDR4X SDRAM"
	case 0x10:
		return "LPDDR5 SDRAM"
	case 0x1B:
		return "HBM2"
	default:
		return fmt.Sprintf("Unknown (0x%02X)", code)
	}
}

// GetManufacturerName converts JEDEC manufacturer ID to name
func GetManufacturerName(id uint16) string {
	// JEDEC manufacturer IDs (continuation code in high byte, ID in low byte)
	manufacturers := map[uint16]string{
		0x0198: "Kingston",
		0x029E: "Corsair",
		0x04CB: "A-DATA",
		0x04CD: "G.Skill",
		0x059B: "Crucial/Micron",
		0x00CE: "Samsung",
		0x00AD: "SK Hynix",
		0x802C: "Micron",
		0x0F98: "Apacer",
		0x7F7F: "Unknown",
	}

	if name, ok := manufacturers[id]; ok {
		return name
	}

	// Check without continuation code
	lowByte := id & 0xFF
	if name, ok := manufacturers[lowByte]; ok {
		return name
	}

	return fmt.Sprintf("Unknown (0x%04X)", id)
}

// ReadMemoryModulesWithSPD enhances memory module information with SPD data
func ReadMemoryModulesWithSPD() ([]MemoryModule, error) {
	DebugLog("SPD", "Starting ReadMemoryModulesWithSPD")

	// First get basic info from WMI
	modules, err := getMemoryModulesWindows()
	if err != nil {
		DebugLog("SPD", fmt.Sprintf("Failed to get WMI modules: %v", err))
		return nil, err
	}

	DebugLog("SPD", fmt.Sprintf("Got %d modules from WMI", len(modules)))

	// Try to read SPD data
	reader := NewSPDReader()
	defer reader.Close()

	if err := reader.Initialize(); err != nil {
		// If we can't initialize WinRing0, just return WMI data
		DebugLog("SPD", fmt.Sprintf("Failed to initialize SPD reader: %v", err))
		return modules, nil
	}

	DebugLog("SPD", "SPD reader initialized successfully")

	// Add timeout protection for SPD reading
	done := make(chan bool)
	var spdData []SPDData
	var spdErr error

	go func() {
		spdData, spdErr = reader.ReadAllSPD()
		done <- true
	}()

	select {
	case <-done:
		if spdErr != nil {
			DebugLog("SPD", fmt.Sprintf("Failed to read SPD data: %v", spdErr))
			return modules, nil
		}
	case <-time.After(2 * time.Second):
		DebugLog("SPD", "SPD reading timed out after 2 seconds, using WMI data")
		return modules, nil
	}

	DebugLog("SPD", fmt.Sprintf("Read %d SPD entries", len(spdData)))

	// Match SPD data to modules
	matchCount := 0
	for i := range modules {
		for _, spd := range spdData {
			// Match by serial number or part number
			if (modules[i].SerialNumber != "" && fmt.Sprintf("%X", spd.SerialNumber) == modules[i].SerialNumber) ||
				(modules[i].PartNumber != "" && strings.Contains(spd.PartNumber, modules[i].PartNumber)) {
				// Enhance module with SPD data
				modules[i].Type = spd.MemoryType
				modules[i].Speed = spd.Speed
				modules[i].Size = spd.ModuleSize
				modules[i].SizeGB = float64(spd.ModuleSize) / (1024 * 1024 * 1024)

				// Update manufacturer from JEDEC ID
				if spd.ManufacturerID != 0 {
					modules[i].Manufacturer = GetManufacturerName(spd.ManufacturerID)
					modules[i].ChipManufacturer = GetManufacturerName(spd.ManufacturerID)
				}

				// Update part number if SPD has it
				if spd.PartNumber != "" {
					modules[i].PartNumber = spd.PartNumber
				}

				// Add timing info to a new field (would need to extend MemoryModule struct)
				// modules[i].CASLatency = spd.CASLatency
				// modules[i].HasXMP = spd.HasXMP
				// modules[i].HasEXPO = spd.HasEXPO

				DebugLog("SPD", fmt.Sprintf("Enhanced module %d with SPD data", i))
				matchCount++
				break
			}
		}
	}

	DebugLog("SPD", fmt.Sprintf("Enhanced %d modules with SPD data", matchCount))

	return modules, nil
}
