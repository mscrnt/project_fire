package parser

import (
	"encoding/hex"
	"testing"
)

// Sample DDR4 SPD data (first 384 bytes)
const ddr4SPDHex = `
23100c0245850021000800520020f00a3424280078803c803e00006e037808007d
2b0c1006360000486e05000000000000000000000000000000000000000000000016
361636163616000000002b0c80ad00000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000000000
80ce01214a303038453236363141000000393839382d4b30464e41474320202020202004
0000000000000000000000000000000000000000000000000000000000000000000000
00000000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000
`

// Sample DDR5 SPD data (first 512 bytes)
const ddr5SPDHex = `
5108120a860021000210000000000000fc0000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000000000
0507000000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000000000
002c8002154b5a3455333241363447422d505644355332300000000000000012340000
0000000000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000000000
`

func parseHexString(hexStr string) []byte {
	// Remove whitespace and newlines
	cleaned := ""
	for _, ch := range hexStr {
		if ch != ' ' && ch != '\n' && ch != '\r' && ch != '\t' {
			cleaned += string(ch)
		}
	}

	data, _ := hex.DecodeString(cleaned)
	return data
}

func TestParseDDR4(t *testing.T) {
	spdData := parseHexString(ddr4SPDHex)

	module, err := ParseSPD(spdData)
	if err != nil {
		t.Fatalf("Failed to parse DDR4 SPD: %v", err)
	}

	// Verify basic properties
	if module.Type != "DDR4" {
		t.Errorf("Expected type DDR4, got %s", module.Type)
	}

	// Check capacity (expecting 8GB module)
	if module.CapacityGB < 7.5 || module.CapacityGB > 8.5 {
		t.Errorf("Expected capacity ~8GB, got %.1fGB", module.CapacityGB)
	}

	// Check speed (expecting DDR4-2666)
	if module.DataRateMTs < 2600 || module.DataRateMTs > 2700 {
		t.Errorf("Expected speed ~2666 MT/s, got %d MT/s", module.DataRateMTs)
	}

	// Check manufacturer (0x80CE = Samsung)
	if module.JEDECManufacturer != "Samsung" {
		t.Errorf("Expected manufacturer Samsung, got %s", module.JEDECManufacturer)
	}

	// Check part number
	expectedPartNumber := "9898-K0FNAGC"
	if module.PartNumber != expectedPartNumber {
		t.Errorf("Expected part number %s, got %s", expectedPartNumber, module.PartNumber)
	}

	// Check timings
	if module.Timings.CL == 0 {
		t.Error("CAS Latency should not be 0")
	}
}

func TestParseDDR5(t *testing.T) {
	spdData := parseHexString(ddr5SPDHex)

	module, err := ParseSPD(spdData)
	if err != nil {
		t.Fatalf("Failed to parse DDR5 SPD: %v", err)
	}

	// Verify basic properties
	if module.Type != "DDR5" {
		t.Errorf("Expected type DDR5, got %s", module.Type)
	}

	// Check manufacturer (0x2C80 = Micron)
	if module.JEDECManufacturer != "Micron" {
		t.Errorf("Expected manufacturer Micron, got %s", module.JEDECManufacturer)
	}
}

func TestInvalidSPD(t *testing.T) {
	testCases := []struct {
		name string
		data []byte
	}{
		{
			name: "Too short",
			data: make([]byte, 64),
		},
		{
			name: "Invalid memory type",
			data: func() []byte {
				d := make([]byte, 128)
				d[SPD_REVISION] = 0x11  // Valid revision
				d[SPD_DRAM_TYPE] = 0xFF // Invalid type
				return d
			}(),
		},
		{
			name: "All zeros",
			data: make([]byte, 384),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseSPD(tc.data)
			if err == nil {
				t.Error("Expected error for invalid SPD data")
			}
		})
	}
}

func TestTimingCalculations(t *testing.T) {
	// Create test SPD data with known timing values
	spdData := make([]byte, 384)

	// Set up as DDR4
	spdData[SPD_DRAM_TYPE] = DRAM_TYPE_DDR4
	spdData[SPD_REVISION] = 0x11

	// Set MTB (125ps)
	spdData[SPD_MTB_DIVIDEND] = 8
	spdData[SPD_MTB_DIVISOR] = 64

	// Set minimum cycle time for DDR4-3200 (625ps = 5 * 125ps)
	spdData[SPD_MIN_CYCLE_TIME] = 5

	// Set timing parameters
	spdData[SPD_MIN_CAS_LATENCY] = 22 * 8 / 5   // CL22
	spdData[SPD_MIN_RAS_TO_CAS] = 22 * 8 / 5    // tRCD 22
	spdData[SPD_MIN_RAS_PRECHARGE] = 22 * 8 / 5 // tRP 22
	spdData[SPD_MIN_ACTIVE] = 52                // tRAS 52
	spdData[SPD_MIN_ROW_CYCLE] = 74             // tRC 74

	// Set other required fields
	spdData[SPD_DENSITY_BANKS] = 0x04 // 4Gb density
	spdData[SPD_MODULE_ORG] = 0x01    // x8, 1 rank
	spdData[SPD_PRIMARY_BUS] = 0x03   // 64-bit

	module, err := ParseSPD(spdData)
	if err != nil {
		t.Fatalf("Failed to parse test SPD: %v", err)
	}

	// Verify timing calculations
	if module.Timings.CL != 22 {
		t.Errorf("Expected CL22, got CL%d", module.Timings.CL)
	}

	if module.Timings.RCD != 22 {
		t.Errorf("Expected tRCD 22, got %d", module.Timings.RCD)
	}

	if module.Timings.RP != 22 {
		t.Errorf("Expected tRP 22, got %d", module.Timings.RP)
	}
}

func TestJEDECManufacturer(t *testing.T) {
	testCases := []struct {
		lsb      uint8
		msb      uint8
		expected string
	}{
		{0x80, 0x2C, "Micron"},
		{0x80, 0xCE, "Samsung"},
		{0x80, 0xAD, "SK Hynix"},
		{0x01, 0x98, "Kingston"},
		{0x9E, 0x02, "Corsair"},
		{0xFF, 0xFF, "Bank 128, 0x7F"}, // Unknown manufacturer
	}

	for _, tc := range testCases {
		result := getJEDECManufacturer(tc.lsb, tc.msb)
		if result != tc.expected {
			t.Errorf("JEDEC ID 0x%02X%02X: expected %s, got %s",
				tc.msb, tc.lsb, tc.expected, result)
		}
	}
}

func BenchmarkParseSPD(b *testing.B) {
	spdData := parseHexString(ddr4SPDHex)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseSPD(spdData)
		if err != nil {
			b.Fatal(err)
		}
	}
}
