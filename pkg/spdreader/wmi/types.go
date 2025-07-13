// Package wmi provides WMI (Windows Management Instrumentation) interfaces for querying memory module information.
package wmi

// Module represents memory module data from WMI
type Module struct {
	Slot              int
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
	Timings           Timings
}

// Timings represents memory timing parameters
type Timings struct {
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
