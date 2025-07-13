package parser

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// SPD byte offsets for DDR4 (JEDEC SPD Rev 1.1)
const (
	// Basic configuration
	SPD_BYTES_USED    = 0x00 // Number of bytes used / total
	SPD_REVISION      = 0x01 // SPD Revision
	SPD_DRAM_TYPE     = 0x02 // DRAM Device Type
	SPD_MODULE_TYPE   = 0x03 // Module Type
	SPD_DENSITY_BANKS = 0x04 // SDRAM Density and Banks
	SPD_ADDRESSING    = 0x05 // SDRAM Addressing
	SPD_PRIMARY_BUS   = 0x0D // Module Memory Bus Width
	SPD_MODULE_ORG    = 0x0C // Module Organization

	// Timing parameters
	SPD_MTB_DIVISOR       = 0x14 // Medium Timebase (MTB) Dividend
	SPD_MTB_DIVIDEND      = 0x15 // Medium Timebase (MTB) Divisor
	SPD_MIN_CYCLE_TIME    = 0x12 // Minimum Cycle Time (tCKAVGmin)
	SPD_CAS_LATENCIES_1   = 0x14 // CAS Latencies Supported, First Byte
	SPD_CAS_LATENCIES_2   = 0x15 // CAS Latencies Supported, Second Byte
	SPD_CAS_LATENCIES_3   = 0x16 // CAS Latencies Supported, Third Byte
	SPD_CAS_LATENCIES_4   = 0x17 // CAS Latencies Supported, Fourth Byte
	SPD_MIN_CAS_LATENCY   = 0x18 // Minimum CAS Latency Time (tAAmin)
	SPD_MIN_RAS_TO_CAS    = 0x19 // Minimum RAS to CAS Delay Time (tRCDmin)
	SPD_MIN_RAS_PRECHARGE = 0x1A // Minimum Row Precharge Delay Time (tRPmin)
	SPD_UPPER_NIBBLES     = 0x1B // Upper nibbles for tRAS and tRC
	SPD_MIN_ACTIVE        = 0x1C // Minimum Active to Precharge Delay Time (tRASmin)
	SPD_MIN_ROW_CYCLE     = 0x1D // Minimum Active to Active/Refresh Delay Time (tRCmin)
	SPD_MIN_RFC1          = 0x1E // Minimum Refresh Recovery Delay Time (tRFC1min) LSB
	SPD_MIN_RFC1_MSB      = 0x1F // Minimum Refresh Recovery Delay Time (tRFC1min) MSB
	SPD_MIN_RFC2          = 0x20 // Minimum Refresh Recovery Delay Time (tRFC2min) LSB
	SPD_MIN_RFC2_MSB      = 0x21 // Minimum Refresh Recovery Delay Time (tRFC2min) MSB
	SPD_MIN_RFC4          = 0x22 // Minimum Refresh Recovery Delay Time (tRFC4min) LSB
	SPD_MIN_RFC4_MSB      = 0x23 // Minimum Refresh Recovery Delay Time (tRFC4min) MSB
	SPD_MIN_FAW           = 0x24 // Minimum Four Activate Window Delay Time (tFAWmin)
	SPD_MIN_RRD_S         = 0x26 // Minimum Row Active to Row Active Delay Time (tRRD_Smin)
	SPD_MIN_RRD_L         = 0x27 // Minimum Row Active to Row Active Delay Time (tRRD_Lmin)

	// Module-specific section (starts at 128)
	SPD_MODULE_MFG_ID_LSB = 0x140 // Module Manufacturer ID Code, LSB
	SPD_MODULE_MFG_ID_MSB = 0x141 // Module Manufacturer ID Code, MSB
	SPD_MODULE_MFG_LOC    = 0x142 // Module Manufacturing Location
	SPD_MODULE_MFG_DATE_Y = 0x143 // Module Manufacturing Date Year
	SPD_MODULE_MFG_DATE_W = 0x144 // Module Manufacturing Date Week
	SPD_MODULE_SERIAL     = 0x145 // Module Serial Number (4 bytes)
	SPD_MODULE_PART_NUM   = 0x149 // Module Part Number (20 bytes)
	SPD_MODULE_REV_CODE   = 0x15D // Module Revision Code

	// DDR5 specific offsets
	SPD5_DENSITY         = 0x04 // Different encoding for DDR5
	SPD5_FIRST_USED_BYTE = 0xC0 // First used byte in DDR5
)

// Memory types
const (
	DRAM_TYPE_DDR4    = 0x0C
	DRAM_TYPE_DDR4E   = 0x0E
	DRAM_TYPE_LPDDR4  = 0x10
	DRAM_TYPE_LPDDR4X = 0x11
	DRAM_TYPE_DDR5    = 0x12
	DRAM_TYPE_LPDDR5  = 0x13
)

// ParseSPD parses raw SPD data into a structured format
func ParseSPD(data []byte) (*Module, error) {
	if len(data) < 128 {
		return nil, fmt.Errorf("SPD data too short: %d bytes", len(data))
	}

	module := &Module{}

	// Determine memory type
	dramType := data[SPD_DRAM_TYPE]
	switch dramType {
	case DRAM_TYPE_DDR4, DRAM_TYPE_DDR4E:
		return parseDDR4(data)
	case DRAM_TYPE_DDR5:
		return parseDDR5(data)
	case DRAM_TYPE_LPDDR4, DRAM_TYPE_LPDDR4X:
		module.Type = "LPDDR4"
		return parseDDR4(data) // Similar structure to DDR4
	case DRAM_TYPE_LPDDR5:
		module.Type = "LPDDR5"
		return parseDDR5(data)
	default:
		return nil, fmt.Errorf("unsupported memory type: 0x%02X", dramType)
	}
}

// parseDDR4 parses DDR4 SPD data
func parseDDR4(data []byte) (*Module, error) {
	m := &Module{
		Type: "DDR4",
	}

	// Calculate capacity
	densityBanks := data[SPD_DENSITY_BANKS]
	density := densityBanks & 0x0F // bits 0-3
	// bankBits := (densityBanks >> 4) & 0x03      // bits 4-5 - unused
	// bankGroupBits := (densityBanks >> 6) & 0x03 // bits 6-7 - unused

	// Density in Gb
	var densityGb int
	switch density {
	case 0x00:
		densityGb = 0 // 256Mb
	case 0x01:
		densityGb = 0 // 512Mb
	case 0x02:
		densityGb = 1 // 1Gb
	case 0x03:
		densityGb = 2 // 2Gb
	case 0x04:
		densityGb = 4 // 4Gb
	case 0x05:
		densityGb = 8 // 8Gb
	case 0x06:
		densityGb = 16 // 16Gb
	case 0x07:
		densityGb = 32 // 32Gb
	case 0x08:
		densityGb = 12 // 12Gb
	case 0x09:
		densityGb = 24 // 24Gb
	}

	// Module organization
	moduleOrg := data[SPD_MODULE_ORG]
	ranks := int(((moduleOrg >> 3) & 0x07) + 1)
	deviceWidth := int(4 << (moduleOrg & 0x07))

	// Bus width
	busWidth := data[SPD_PRIMARY_BUS]
	primaryBusWidth := int(8 << (busWidth & 0x07))

	// Calculate module capacity
	// Capacity (GB) = (density_per_die * primary_bus_width * ranks) / (8 * device_width)
	m.CapacityGB = float64(densityGb*primaryBusWidth*ranks) / float64(8*deviceWidth)
	m.Ranks = ranks
	m.DataWidth = primaryBusWidth

	// Calculate speed
	// Medium Timebase (MTB) in ps
	mtbDividend := int(data[SPD_MTB_DIVIDEND])
	mtbDivisor := int(data[SPD_MTB_DIVISOR])
	if mtbDivisor == 0 {
		mtbDivisor = 1
	}
	mtb := float64(mtbDividend) / float64(mtbDivisor) * 1000.0 // Convert to ps

	// Minimum cycle time
	tCKmin := float64(data[SPD_MIN_CYCLE_TIME]) * mtb
	if tCKmin > 0 {
		m.BaseFreqMHz = 1000000.0 / tCKmin // ps to MHz
		m.DataRateMTs = int(2 * m.BaseFreqMHz)

		// Calculate PC rating (MT/s * bus_width_bytes / 8)
		m.PCRate = m.DataRateMTs * primaryBusWidth / 8
	}

	// Parse timings
	m.Timings = parseDDR4Timings(data, mtb)

	// Parse manufacturer info (if we have module-specific data)
	if len(data) >= 384 {
		m.JEDECManufacturer = getJEDECManufacturer(data[SPD_MODULE_MFG_ID_LSB], data[SPD_MODULE_MFG_ID_MSB])
		m.PartNumber = strings.TrimSpace(string(data[SPD_MODULE_PART_NUM : SPD_MODULE_PART_NUM+20]))

		// Serial number (4 bytes, little-endian)
		serial := binary.LittleEndian.Uint32(data[SPD_MODULE_SERIAL : SPD_MODULE_SERIAL+4])
		m.Serial = fmt.Sprintf("%08X", serial)

		// Manufacturing date
		year := data[SPD_MODULE_MFG_DATE_Y]
		week := data[SPD_MODULE_MFG_DATE_W]
		if year != 0 && week != 0 {
			m.ManufacturingDate = fmt.Sprintf("20%02d-W%02d", year, week)
		}
	}

	return m, nil
}

// parseDDR5 parses DDR5 SPD data
func parseDDR5(data []byte) (*Module, error) {
	m := &Module{
		Type: "DDR5",
	}

	// DDR5 has different SPD layout
	// First used byte at 0xC0
	if len(data) < 0x200 {
		return nil, fmt.Errorf("DDR5 SPD data too short")
	}

	// Density calculation for DDR5
	densityByte := data[SPD5_DENSITY]
	density := densityByte & 0x1F // bits 0-4

	var densityGb int
	switch density {
	case 0x05:
		densityGb = 8
	case 0x06:
		densityGb = 16
	case 0x07:
		densityGb = 24
	case 0x08:
		densityGb = 32
	case 0x09:
		densityGb = 48
	case 0x0A:
		densityGb = 64
	}

	// Module organization
	moduleOrg := data[0x06]
	ranks := int(((moduleOrg >> 3) & 0x07) + 1)

	// For DDR5, calculate differently
	m.CapacityGB = float64(densityGb*ranks) / 8
	m.Ranks = ranks
	m.DataWidth = 64 // DDR5 is always 64-bit

	// DDR5 speeds
	speedByte := data[0xC0]
	switch speedByte {
	case 0x00:
		m.DataRateMTs = 3200
	case 0x01:
		m.DataRateMTs = 3600
	case 0x02:
		m.DataRateMTs = 4000
	case 0x03:
		m.DataRateMTs = 4400
	case 0x04:
		m.DataRateMTs = 4800
	case 0x05:
		m.DataRateMTs = 5200
	case 0x06:
		m.DataRateMTs = 5600
	case 0x07:
		m.DataRateMTs = 6000
	case 0x08:
		m.DataRateMTs = 6400
	case 0x09:
		m.DataRateMTs = 6800
	case 0x0A:
		m.DataRateMTs = 7200
	}

	m.BaseFreqMHz = float64(m.DataRateMTs) / 2
	m.PCRate = m.DataRateMTs * 8 // 64-bit / 8

	// Parse manufacturer info
	if len(data) >= 0x200 {
		mfgOffset := 0x200
		m.JEDECManufacturer = getJEDECManufacturer(data[mfgOffset], data[mfgOffset+1])
		m.PartNumber = strings.TrimSpace(string(data[mfgOffset+4 : mfgOffset+24]))

		serial := binary.LittleEndian.Uint32(data[mfgOffset+25 : mfgOffset+29])
		m.Serial = fmt.Sprintf("%08X", serial)
	}

	return m, nil
}

// parseDDR4Timings parses timing parameters from DDR4 SPD
func parseDDR4Timings(data []byte, mtb float64) Timings {
	t := Timings{}

	// Fine Timebase (FTB) in ps
	// ftb := 1.0 // Default 1ps - unused for now

	// Calculate timings
	tAAmin := float64(data[SPD_MIN_CAS_LATENCY]) * mtb
	tRCDmin := float64(data[SPD_MIN_RAS_TO_CAS]) * mtb
	tRPmin := float64(data[SPD_MIN_RAS_PRECHARGE]) * mtb

	// Upper nibbles
	upperNibbles := data[SPD_UPPER_NIBBLES]
	tRASminUpper := (upperNibbles & 0x0F)
	tRCminUpper := (upperNibbles >> 4) & 0x0F

	tRASmin := float64(uint16(data[SPD_MIN_ACTIVE])|(uint16(tRASminUpper)<<8)) * mtb
	tRCmin := float64(uint16(data[SPD_MIN_ROW_CYCLE])|(uint16(tRCminUpper)<<8)) * mtb

	// Calculate minimum cycle time for conversion
	tCKmin := float64(data[SPD_MIN_CYCLE_TIME]) * mtb
	if tCKmin == 0 {
		tCKmin = 625 // Default to DDR4-3200 (625ps)
	}

	// Convert to clock cycles
	t.CL = int((tAAmin + tCKmin - 1) / tCKmin)
	t.RCD = int((tRCDmin + tCKmin - 1) / tCKmin)
	t.RP = int((tRPmin + tCKmin - 1) / tCKmin)
	t.RAS = int((tRASmin + tCKmin - 1) / tCKmin)
	t.RC = int((tRCmin + tCKmin - 1) / tCKmin)

	// tRFC (Refresh Cycle Time)
	tRFC1 := float64(uint16(data[SPD_MIN_RFC1])|(uint16(data[SPD_MIN_RFC1_MSB])<<8)) * mtb
	t.RFC = int((tRFC1 + tCKmin - 1) / tCKmin)

	// tRRD_S and tRRD_L
	tRRD_S := float64(data[SPD_MIN_RRD_S]) * mtb
	tRRD_L := float64(data[SPD_MIN_RRD_L]) * mtb
	t.RRD_S = int((tRRD_S + tCKmin - 1) / tCKmin)
	t.RRD_L = int((tRRD_L + tCKmin - 1) / tCKmin)

	// tFAW
	tFAW := float64(binary.LittleEndian.Uint16(data[SPD_MIN_FAW:SPD_MIN_FAW+2])) * mtb
	t.FAW = int((tFAW + tCKmin - 1) / tCKmin)

	return t
}

// getJEDECManufacturer returns manufacturer name from JEDEC ID
func getJEDECManufacturer(lsb, msb uint8) string {
	// Common JEDEC manufacturer IDs
	manufacturers := map[uint16]string{
		0x2C80: "Micron",
		0xCE80: "Samsung",
		0xAD80: "SK Hynix",
		0x4F01: "Transcend",
		0x9801: "Kingston",
		0x0B83: "A-DATA",
		0xCD04: "G.Skill",
		0x5105: "Qimonda",
		0x2503: "Kingmax",
		0x029E: "Corsair",
		0xC102: "Infineon",
		0x7F7F: "Unknown",
	}

	id := uint16(msb)<<8 | uint16(lsb)
	if name, ok := manufacturers[id]; ok {
		return name
	}

	// Check continuation codes
	bank := (msb & 0x7F) + 1
	index := lsb & 0x7F

	return fmt.Sprintf("Bank %d, 0x%02X", bank, index)
}
