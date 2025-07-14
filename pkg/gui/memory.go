package gui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/mscrnt/project_fire/pkg/telemetry"
)

// MemoryModule represents a single RAM module with CPU-Z style details
type MemoryModule struct {
	// Basic identification
	Row       int    // Row number (1, 2, ...)
	Slot      string // e.g. "P0 CHANNEL A/DIMM 1"
	BankLabel string // e.g. "P0 CHANNEL A"
	Number    string // Same as Row as string
	Name      string // Full descriptive name

	// Memory specifications
	Size       uint64  // Size in bytes
	SizeGB     float64 // Size in GB
	Speed      uint32  // Configured speed in MHz
	Type       string  // e.g. "DDR5 SDRAM"
	FormFactor string  // e.g. "DIMM"

	// Frequency and timing
	BaseFrequency float64 // Base frequency in MHz (half of data rate)
	DataRate      int     // Data rate in MT/s (e.g. 6000)
	PCRating      int     // PC rating (e.g. 48000 for PC5-48000)

	// Manufacturer information
	Manufacturer     string // Module vendor (e.g. "G.Skill")
	ChipManufacturer string // Die vendor (e.g. "SK Hynix")
	PartNumber       string // Part number
	SerialNumber     string // Serial number (hex)

	// Raw data for future use
	SMBIOSType int // Raw SMBIOS memory type code
}

// GetMemoryModules returns individual memory modules
func GetMemoryModules() ([]MemoryModule, error) {
	switch runtime.GOOS {
	case "windows":
		// Check if running as admin
		if IsRunningAsAdmin() {
			DebugLog("MEMORY", "Running as Administrator - enhanced memory detection available")
			// For now, skip SPD reader as WinRing0 doesn't support SMBUS
			// We'll use enhanced WMI detection instead
			DebugLog("MEMORY", "Using enhanced WMI detection (SPD reading requires specialized hardware access)")
		} else {
			DebugLog("MEMORY", "Not running as Administrator - using basic WMI detection")
		}
		// Fall back to WMI
		DebugLog("MEMORY", "Using WMI for memory detection")
		return getMemoryModulesWindows()
	case "linux":
		return getMemoryModulesLinux()
	case "darwin":
		return getMemoryModulesDarwin()
	default:
		return nil, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// getMemoryModulesWindows uses WMI to get memory module information
func getMemoryModulesWindows() ([]MemoryModule, error) {
	var modules []MemoryModule

	// Use wmic to get memory information including SMBIOSMemoryType
	cmd := exec.Command("cmd", "/c", "wmic memorychip get Capacity,Speed,SMBIOSMemoryType,Manufacturer,PartNumber,SerialNumber,DeviceLocator,FormFactor,ConfiguredClockSpeed,BankLabel /format:csv")

	output, err := cmd.Output()
	if err != nil {
		return modules, err
	}

	lines := strings.Split(string(output), "\r\n")
	var headers []string
	moduleIndex := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Split(line, ",")

		// First line with multiple fields is headers
		if len(headers) == 0 && len(fields) > 1 {
			headers = fields
			continue
		}

		// Skip if not enough fields
		if len(fields) < 5 {
			continue
		}

		// Create a map for easier field access
		fieldMap := make(map[string]string)
		for j, header := range headers {
			if j < len(fields) {
				fieldMap[strings.TrimSpace(header)] = strings.TrimSpace(fields[j])
			}
		}

		// Parse capacity
		capacity, _ := strconv.ParseUint(fieldMap["Capacity"], 10, 64)
		if capacity == 0 {
			continue // Skip empty slots
		}

		// Parse speed - prefer ConfiguredClockSpeed over Speed
		speedStr := fieldMap["ConfiguredClockSpeed"]
		if speedStr == "" || speedStr == "0" {
			speedStr = fieldMap["Speed"]
		}
		speed, _ := strconv.ParseUint(speedStr, 10, 32)

		// Get memory type using SMBIOSMemoryType
		smbiosType := fieldMap["SMBIOSMemoryType"]
		smbiosTypeInt, _ := strconv.Atoi(smbiosType)
		memType := getSMBIOSMemoryTypeName(smbiosType)
		DebugLog("MEMORY", fmt.Sprintf("SMBIOSMemoryType: %s -> %s for %s", smbiosType, memType, fieldMap["DeviceLocator"]))

		// Get form factor
		formFactor := getFormFactorName(fieldMap["FormFactor"])

		// Calculate derived values
		sizeGB := float64(capacity) / (1024 * 1024 * 1024)
		baseFreq := float64(speed) / 2.0 // DDR = Double Data Rate
		dataRate := int(speed)

		// Calculate PC rating based on memory type
		var pcRating int
		var pcPrefix string
		switch {
		case strings.Contains(memType, "DDR5"):
			pcPrefix = "PC5"
			pcRating = dataRate * 8 // DDR5: MT/s * 8
		case strings.Contains(memType, "DDR4"):
			pcPrefix = "PC4"
			pcRating = dataRate * 8 // DDR4: MT/s * 8
		case strings.Contains(memType, "DDR3"):
			pcPrefix = "PC3"
			pcRating = dataRate * 8 // DDR3: MT/s * 8
		}

		// Clean up manufacturer and part number
		manufacturer := cleanManufacturerName(fieldMap["Manufacturer"])
		partNumber := strings.TrimSpace(fieldMap["PartNumber"])
		serialNumber := strings.TrimSpace(fieldMap["SerialNumber"])
		slot := strings.TrimSpace(fieldMap["DeviceLocator"])
		bankLabel := strings.TrimSpace(fieldMap["BankLabel"])

		moduleIndex++

		module := MemoryModule{
			Row:              moduleIndex,
			Slot:             slot,
			BankLabel:        bankLabel,
			Number:           fmt.Sprintf("%d", moduleIndex),
			Size:             capacity,
			SizeGB:           sizeGB,
			Speed:            uint32(speed),
			Type:             memType,
			FormFactor:       formFactor,
			BaseFrequency:    baseFreq,
			DataRate:         dataRate,
			PCRating:         pcRating,
			Manufacturer:     manufacturer,
			ChipManufacturer: getChipManufacturer(manufacturer, partNumber),
			PartNumber:       partNumber,
			SerialNumber:     serialNumber,
			SMBIOSType:       smbiosTypeInt,
		}

		// Build the full name string CPU-Z style
		if pcRating > 0 {
			module.Name = fmt.Sprintf("Row %d [%s/%s] – %.0f GB %s-%d %s %s %s",
				module.Row, bankLabel, slot, sizeGB, pcPrefix, pcRating, memType, manufacturer, partNumber)
		} else {
			module.Name = fmt.Sprintf("Row %d [%s/%s] – %.0f GB %s %s %s",
				module.Row, bankLabel, slot, sizeGB, memType, manufacturer, partNumber)
		}

		modules = append(modules, module)
	}

	return modules, nil
}

// getMemoryModulesLinux uses dmidecode or /sys to get memory information
func getMemoryModulesLinux() ([]MemoryModule, error) {
	// For WSL, try to get Windows memory info
	if isWSL() {
		return getMemoryModulesWSL()
	}

	// Regular Linux - would need sudo for dmidecode
	return []MemoryModule{}, nil
}

// getMemoryModulesWSL gets memory info from Windows host
func getMemoryModulesWSL() ([]MemoryModule, error) {
	// Try to run Windows wmic command from WSL
	cmd := exec.Command("cmd.exe", "/c", "wmic memorychip get Capacity,Speed,SMBIOSMemoryType,Manufacturer,PartNumber,SerialNumber,DeviceLocator,FormFactor,ConfiguredClockSpeed,BankLabel /format:csv")

	output, err := cmd.Output()
	if err != nil {
		return []MemoryModule{}, err
	}

	// Parse the same way as Windows
	return parseWMICMemoryOutput(string(output))
}

// parseWMICMemoryOutput parses WMIC CSV output
func parseWMICMemoryOutput(output string) ([]MemoryModule, error) {
	var modules []MemoryModule

	lines := strings.Split(output, "\n")
	var headers []string
	moduleIndex := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.Trim(line, "\r")
		if line == "" {
			continue
		}

		fields := strings.Split(line, ",")

		// First line with multiple fields is headers
		if len(headers) == 0 && len(fields) > 1 && strings.Contains(line, "Capacity") {
			headers = fields
			continue
		}

		// Skip if not a data line
		if len(fields) < 5 || strings.Contains(line, "Node") {
			continue
		}

		// Create a map for easier field access
		fieldMap := make(map[string]string)
		for j, header := range headers {
			if j < len(fields) {
				fieldMap[strings.TrimSpace(header)] = strings.TrimSpace(fields[j])
			}
		}

		// Parse capacity
		capacity, _ := strconv.ParseUint(fieldMap["Capacity"], 10, 64)
		if capacity == 0 {
			continue // Skip empty slots
		}

		// Parse speed
		speedStr := fieldMap["ConfiguredClockSpeed"]
		if speedStr == "" || speedStr == "0" {
			speedStr = fieldMap["Speed"]
		}
		speed, _ := strconv.ParseUint(speedStr, 10, 32)

		// Get memory type using SMBIOSMemoryType
		smbiosType := fieldMap["SMBIOSMemoryType"]
		smbiosTypeInt, _ := strconv.Atoi(smbiosType)
		memType := getSMBIOSMemoryTypeName(smbiosType)
		DebugLog("MEMORY", fmt.Sprintf("SMBIOSMemoryType: %s -> %s for %s", smbiosType, memType, fieldMap["DeviceLocator"]))

		// Get form factor
		formFactor := getFormFactorName(fieldMap["FormFactor"])

		// Calculate derived values
		sizeGB := float64(capacity) / (1024 * 1024 * 1024)
		baseFreq := float64(speed) / 2.0 // DDR = Double Data Rate
		dataRate := int(speed)

		// Calculate PC rating based on memory type
		var pcRating int
		var pcPrefix string
		switch {
		case strings.Contains(memType, "DDR5"):
			pcPrefix = "PC5"
			pcRating = dataRate * 8 // DDR5: MT/s * 8
		case strings.Contains(memType, "DDR4"):
			pcPrefix = "PC4"
			pcRating = dataRate * 8 // DDR4: MT/s * 8
		case strings.Contains(memType, "DDR3"):
			pcPrefix = "PC3"
			pcRating = dataRate * 8 // DDR3: MT/s * 8
		}

		// Clean up manufacturer and part number
		manufacturer := cleanManufacturerName(fieldMap["Manufacturer"])
		partNumber := strings.TrimSpace(fieldMap["PartNumber"])
		serialNumber := strings.TrimSpace(fieldMap["SerialNumber"])
		slot := strings.TrimSpace(fieldMap["DeviceLocator"])
		bankLabel := strings.TrimSpace(fieldMap["BankLabel"])

		moduleIndex++

		module := MemoryModule{
			Row:              moduleIndex,
			Slot:             slot,
			BankLabel:        bankLabel,
			Number:           fmt.Sprintf("%d", moduleIndex),
			Size:             capacity,
			SizeGB:           sizeGB,
			Speed:            uint32(speed),
			Type:             memType,
			FormFactor:       formFactor,
			BaseFrequency:    baseFreq,
			DataRate:         dataRate,
			PCRating:         pcRating,
			Manufacturer:     manufacturer,
			ChipManufacturer: getChipManufacturer(manufacturer, partNumber),
			PartNumber:       partNumber,
			SerialNumber:     serialNumber,
			SMBIOSType:       smbiosTypeInt,
		}

		// Build the full name string CPU-Z style
		if pcRating > 0 {
			module.Name = fmt.Sprintf("Row %d [%s/%s] – %.0f GB %s-%d %s %s %s",
				module.Row, bankLabel, slot, sizeGB, pcPrefix, pcRating, memType, manufacturer, partNumber)
		} else {
			module.Name = fmt.Sprintf("Row %d [%s/%s] – %.0f GB %s %s %s",
				module.Row, bankLabel, slot, sizeGB, memType, manufacturer, partNumber)
		}

		modules = append(modules, module)
	}

	return modules, nil
}

// getMemoryModulesDarwin gets memory info on macOS
func getMemoryModulesDarwin() ([]MemoryModule, error) {
	// macOS implementation would go here
	return []MemoryModule{}, nil
}

// getSMBIOSMemoryTypeName converts SMBIOS memory type code to readable name
// SMBIOS Type codes from DMTF specification
// Note: Some BIOS/UEFI implementations may report non-standard codes
//
// SPD Direct Reading Notes:
// For DDR4 and earlier (SPD revision < 5):
//
//	Memory type is in Byte 2
//	0x0B = DDR3
//	0x0C = DDR4
//	0x1B = HBM2
//
// For DDR5 (SPD revision >= 5):
//
//	Byte 2 contains SPD revision
//	Memory type is in Byte 3 (low nibble, bits 0-3)
//	0x0D = DDR5 SDRAM
//	0x0E = LPDDR4 SDRAM
//	0x0F = LPDDR4X SDRAM
func getSMBIOSMemoryTypeName(typeCode string) string {
	// Convert to int for easier comparison
	code, _ := strconv.Atoi(typeCode)

	switch code {
	case 0:
		return "Unknown"
	case 1:
		return "Other"
	case 2:
		return "DRAM"
	case 3:
		return "Synchronous DRAM"
	case 4:
		return "Cache DRAM"
	case 5:
		return "EDO"
	case 6:
		return "EDRAM"
	case 7:
		return "VRAM"
	case 8:
		return "SRAM"
	case 9:
		return "RAM"
	case 10:
		return "ROM"
	case 11:
		return "Flash"
	case 12:
		return "EEPROM"
	case 13:
		return "FEPROM"
	case 14:
		return "EPROM"
	case 15:
		return "CDRAM"
	case 16:
		return "3DRAM"
	case 17:
		return "SDRAM"
	case 18:
		return "SGRAM"
	case 19:
		return "RDRAM"
	case 20:
		return "DDR"
	case 21:
		return "DDR2"
	case 22:
		return "DDR2 FB-DIMM"
	case 24:
		return "DDR3"
	case 25:
		return "FBD2"
	case 26:
		return "DDR4"
	case 27:
		return "DDR5"
	case 28:
		return "LPDDR"
	case 29:
		return "LPDDR2"
	case 30:
		return "LPDDR3"
	case 31:
		return "LPDDR4"
	case 32:
		return "Logical non-volatile device"
	case 33:
		return "LPDDR4" // Some systems report LPDDR4 as 33
	case 34:
		return "DDR5" // Some systems/BIOS report DDR5 as 34
	case 35:
		return "HBM3"
	case 36:
		return "LPDDR5"
	case 42:
		return "DDR5" // Some BIOSes report DDR5 as 42
	default:
		if code > 0 {
			// Record hardware miss for unknown memory type
			telemetry.RecordHardwareMiss("SMBIOSMemoryType", map[string]interface{}{
				"code": code,
				"type": "unknown_memory_type",
			})
			return fmt.Sprintf("Unknown(%d)", code)
		}
		return "RAM"
	}
}

// getFormFactorName converts form factor code to readable name
func getFormFactorName(formFactorCode string) string {
	switch formFactorCode {
	case "8":
		return "DIMM"
	case "12":
		return "SODIMM"
	case "13":
		return "SRIMM"
	default:
		if formFactorCode != "" && formFactorCode != "0" {
			telemetry.RecordHardwareMiss("MemoryFormFactor", map[string]interface{}{
				"code": formFactorCode,
				"type": "unknown_form_factor",
			})
			return fmt.Sprintf("FormFactor %s", formFactorCode)
		}
		return "DIMM"
	}
}

// getChipManufacturer attempts to determine the chip manufacturer from module info
func getChipManufacturer(moduleManufacturer, partNumber string) string {
	// Common chip manufacturers based on part numbers and module vendors
	partLower := strings.ToLower(partNumber)

	// G.Skill often uses SK Hynix or Samsung chips
	if strings.EqualFold(moduleManufacturer, "g.skill") || strings.Contains(moduleManufacturer, "G.SKILL") {
		if strings.Contains(partLower, "h") || strings.Contains(partLower, "3040") {
			return "SK Hynix"
		}
		if strings.Contains(partLower, "s") {
			return "Samsung"
		}
	}

	// Corsair patterns
	if strings.Contains(strings.ToLower(moduleManufacturer), "corsair") {
		if strings.Contains(partLower, "h") {
			return "SK Hynix"
		}
		if strings.Contains(partLower, "s") {
			return "Samsung"
		}
		if strings.Contains(partLower, "m") {
			return "Micron"
		}
	}

	// Kingston often uses their own chips
	if strings.Contains(strings.ToLower(moduleManufacturer), "kingston") {
		return "Kingston"
	}

	// Crucial is Micron's consumer brand
	if strings.Contains(strings.ToLower(moduleManufacturer), "crucial") {
		return "Micron"
	}

	// Samsung modules use Samsung chips
	if strings.Contains(strings.ToLower(moduleManufacturer), "samsung") {
		return "Samsung"
	}

	// Default to unknown or same as module manufacturer
	if moduleManufacturer != "" && !strings.Contains(strings.ToLower(moduleManufacturer), "unknown") {
		return moduleManufacturer
	}

	return "Unknown"
}

// cleanManufacturerName cleans up manufacturer names
func cleanManufacturerName(name string) string {
	name = strings.TrimSpace(name)

	// Common manufacturer ID mappings
	switch strings.ToLower(name) {
	case "04cb", "04cb00000000":
		return "A-DATA"
	case "059b", "059b00000000":
		return "Crucial"
	case "029e", "029e00000000":
		return "Corsair"
	case "04cd", "04cd00000000":
		return "G.Skill"
	case "0198", "019800000000":
		return "Kingston"
	case "80ce", "80ce00000000", "80ce000000000000":
		return "Samsung"
	case "80ad", "80ad00000000", "80ad000000000000":
		return "Hynix"
	case "802c", "802c00000000":
		return "Micron"
	}

	// Remove hex suffixes
	if len(name) > 4 && strings.HasSuffix(strings.ToLower(name), "00000000") {
		name = name[:4]
	}

	return name
}

// FormatMemorySize formats bytes to human readable format
func FormatMemorySize(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		gb := float64(bytes) / float64(GB)
		if gb == float64(int(gb)) {
			return fmt.Sprintf("%d GB", int(gb))
		}
		return fmt.Sprintf("%.1f GB", gb)
	case bytes >= MB:
		return fmt.Sprintf("%d MB", bytes/MB)
	case bytes >= KB:
		return fmt.Sprintf("%d KB", bytes/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
