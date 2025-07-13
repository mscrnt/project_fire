package parser

// Module represents parsed SPD data (internal to parser package)
type Module struct {
	Type              string
	BaseFreqMHz       float64
	DataRateMTs       int
	PCRate            int
	CapacityGB        float64
	Ranks             int
	DataWidth         int
	JEDECManufacturer string
	PartNumber        string
	Serial            string
	ManufacturingDate string
	Timings           Timings
}

// Timings represents memory timing parameters
type Timings struct {
	CL   int // CAS Latency
	RCD  int // RAS to CAS Delay
	RP   int // RAS Precharge
	RAS  int // Active to Precharge Delay
	RC   int // Row Cycle Time
	RFC  int // Refresh Cycle Time
	RRDS int // Row to Row Delay (Same bank group)
	RRDL int // Row to Row Delay (Different bank group)
	FAW  int // Four Activate Window
}
