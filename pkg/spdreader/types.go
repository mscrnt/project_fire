package spdreader

// SPDModule represents a memory module with parsed SPD data
type SPDModule struct {
	Slot              int     `json:"slot"`
	Type              string  `json:"type"`              // DDR4, DDR5, etc.
	BaseFreqMHz       float64 `json:"baseFreqMHz"`       // Base frequency in MHz
	DataRateMTs       int     `json:"dataRateMTs"`       // Data rate in MT/s (e.g., 3200)
	PCRate            int     `json:"pcRate"`            // PC rating (e.g., 25600 for PC4-25600)
	CapacityGB        float64 `json:"capacityGB"`        // Capacity in GB
	Ranks             int     `json:"ranks"`             // Number of ranks
	DataWidth         int     `json:"dataWidth"`         // Data width (e.g., 64)
	JEDECManufacturer string  `json:"jedecManufacturer"` // JEDEC manufacturer name
	PartNumber        string  `json:"partNumber"`        // Part number
	Serial            string  `json:"serial"`            // Serial number
	ManufacturingDate string  `json:"manufacturingDate"` // Manufacturing date (YYWW)
	Timings           Timings `json:"timings"`           // Memory timings
	RawSPD            []byte  `json:"-"`                 // Raw SPD data
}

// Timings represents memory timing parameters
type Timings struct {
	CL   int `json:"cl"`    // CAS Latency
	RCD  int `json:"rcd"`   // RAS to CAS Delay
	RP   int `json:"rp"`    // RAS Precharge
	RAS  int `json:"ras"`   // Active to Precharge Delay
	RC   int `json:"rc"`    // Row Cycle Time
	RFC  int `json:"rfc"`   // Refresh Cycle Time
	RRDS int `json:"rrd_s"` // Row to Row Delay (Same bank group)
	RRDL int `json:"rrd_l"` // Row to Row Delay (Different bank group)
	FAW  int `json:"faw"`   // Four Activate Window
}
