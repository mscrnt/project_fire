package gui

import (
	"fmt"
	"log"
	"runtime"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/mscrnt/project_fire/pkg/spdreader"
)

// MemoryDetailsPage shows detailed memory information including SPD data
type MemoryDetailsPage struct {
	window       fyne.Window
	container    *fyne.Container
	modules      []MemoryModule
	spdModules   []spdreader.SPDModule
	selectedSlot int
}

// NewMemoryDetailsPage creates a new memory details page
func NewMemoryDetailsPage(window fyne.Window) *MemoryDetailsPage {
	return &MemoryDetailsPage{
		window:       window,
		selectedSlot: 0,
	}
}

// CreateContent creates the memory details page content
func (p *MemoryDetailsPage) CreateContent() fyne.CanvasObject {
	// Header
	header := widget.NewLabelWithStyle("Memory Details", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	// Get memory modules
	modules, err := GetMemoryModules()
	if err != nil {
		log.Printf("Error getting memory modules: %v", err)
	}
	p.modules = modules

	// Create module selector
	moduleOptions := make([]string, 0, len(modules))
	for _, module := range modules {
		moduleOptions = append(moduleOptions, fmt.Sprintf("Slot %d: %s", module.Row, module.PartNumber))
	}

	if len(moduleOptions) == 0 {
		moduleOptions = append(moduleOptions, "No memory modules detected")
	}

	// SPD data button (Windows only with admin)
	var spdButton *widget.Button
	if runtime.GOOS == "windows" && IsRunningAsAdmin() {
		spdButton = widget.NewButtonWithIcon("Read SPD Data", theme.InfoIcon(), func() {
			p.readSPDData()
		})
		spdButton.Importance = widget.HighImportance
	}

	// Module selector
	moduleSelect := widget.NewSelect(moduleOptions, func(selected string) {
		// Update selected slot
		for i, opt := range moduleOptions {
			if opt == selected {
				p.selectedSlot = i
				p.updateDetailsDisplay()
				break
			}
		}
	})

	if len(moduleOptions) > 0 {
		moduleSelect.SetSelectedIndex(0)
	}

	// Create details container
	detailsContainer := container.NewVBox()
	p.container = detailsContainer

	// Initial display
	p.updateDetailsDisplay()

	// Layout
	content := container.NewBorder(
		container.NewVBox(
			header,
			widget.NewSeparator(),
			container.NewBorder(nil, nil, widget.NewLabel("Select Module:"), spdButton, moduleSelect),
			widget.NewSeparator(),
		),
		nil, nil, nil,
		container.NewVScroll(detailsContainer),
	)

	return content
}

// updateDetailsDisplay updates the details display for the selected module
func (p *MemoryDetailsPage) updateDetailsDisplay() {
	p.container.RemoveAll()

	if p.selectedSlot >= len(p.modules) {
		p.container.Add(widget.NewLabel("No module selected"))
		return
	}

	module := p.modules[p.selectedSlot]

	// Basic Information
	basicInfo := widget.NewCard("Basic Information", "", container.NewVBox(
		p.createInfoRow("Slot:", module.Slot),
		p.createInfoRow("Bank Label:", module.BankLabel),
		p.createInfoRow("Size:", fmt.Sprintf("%.1f GB", module.SizeGB)),
		p.createInfoRow("Type:", module.Type),
		p.createInfoRow("Form Factor:", module.FormFactor),
		p.createInfoRow("Speed:", fmt.Sprintf("%d MHz", module.Speed)),
		p.createInfoRow("Data Rate:", fmt.Sprintf("%d MT/s", module.DataRate)),
		p.createInfoRow("PC Rating:", fmt.Sprintf("PC%d-%d", getPCGeneration(module.Type), module.PCRating)),
	))

	// Manufacturer Information
	mfgInfo := widget.NewCard("Manufacturer Information", "", container.NewVBox(
		p.createInfoRow("Module Manufacturer:", module.Manufacturer),
		p.createInfoRow("Chip Manufacturer:", module.ChipManufacturer),
		p.createInfoRow("Part Number:", module.PartNumber),
		p.createInfoRow("Serial Number:", module.SerialNumber),
	))

	// Add to container
	p.container.Add(basicInfo)
	p.container.Add(mfgInfo)

	// If we have SPD data for this slot, show additional details
	if p.selectedSlot < len(p.spdModules) {
		spdModule := p.spdModules[p.selectedSlot]

		// Timing Information
		timingInfo := widget.NewCard("Timing Information", "", container.NewVBox(
			p.createInfoRow("CAS Latency (CL):", fmt.Sprintf("%d", spdModule.Timings.CL)),
			p.createInfoRow("RAS to CAS Delay (tRCD):", fmt.Sprintf("%d", spdModule.Timings.RCD)),
			p.createInfoRow("RAS Precharge (tRP):", fmt.Sprintf("%d", spdModule.Timings.RP)),
			p.createInfoRow("Active to Precharge (tRAS):", fmt.Sprintf("%d", spdModule.Timings.RAS)),
			p.createInfoRow("Row Cycle Time (tRC):", fmt.Sprintf("%d", spdModule.Timings.RC)),
			p.createInfoRow("Refresh Cycle Time (tRFC):", fmt.Sprintf("%d", spdModule.Timings.RFC)),
			p.createInfoRow("RRD Same Bank Group (tRRD_S):", fmt.Sprintf("%d", spdModule.Timings.RRD_S)),
			p.createInfoRow("RRD Different Bank Group (tRRD_L):", fmt.Sprintf("%d", spdModule.Timings.RRD_L)),
			p.createInfoRow("Four Activate Window (tFAW):", fmt.Sprintf("%d", spdModule.Timings.FAW)),
		))

		// Advanced Information from SPD
		advancedInfo := widget.NewCard("Advanced SPD Information", "", container.NewVBox(
			p.createInfoRow("JEDEC Manufacturer:", spdModule.JEDECManufacturer),
			p.createInfoRow("Manufacturing Date:", spdModule.ManufacturingDate),
			p.createInfoRow("Ranks:", fmt.Sprintf("%d", spdModule.Ranks)),
			p.createInfoRow("Data Width:", fmt.Sprintf("x%d", spdModule.DataWidth)),
			p.createInfoRow("Base Frequency:", fmt.Sprintf("%.1f MHz", spdModule.BaseFreqMHz)),
		))

		p.container.Add(timingInfo)
		p.container.Add(advancedInfo)

		// Raw SPD data viewer
		if len(spdModule.RawSPD) > 0 {
			spdDataButton := widget.NewButton("View Raw SPD Data", func() {
				p.showRawSPDData(spdModule.RawSPD)
			})
			p.container.Add(container.NewCenter(spdDataButton))
		}
	}

	p.container.Refresh()
}

// createInfoRow creates a formatted info row
func (p *MemoryDetailsPage) createInfoRow(label, value string) *fyne.Container {
	labelWidget := widget.NewLabelWithStyle(label, fyne.TextAlignLeading, fyne.TextStyle{})
	valueWidget := widget.NewLabelWithStyle(value, fyne.TextAlignTrailing, fyne.TextStyle{Bold: true})

	return container.NewBorder(nil, nil, labelWidget, valueWidget)
}

// readSPDData attempts to read SPD data using the integrated reader
func (p *MemoryDetailsPage) readSPDData() {
	progressDialog := dialog.NewProgressInfinite("Reading SPD Data", "Accessing memory modules...", p.window)
	progressDialog.Show()

	go func() {
		defer progressDialog.Hide()

		// Create SPD reader
		reader, err := spdreader.New()
		if err != nil {
			// This is expected on non-Windows platforms
			if runtime.GOOS != "windows" {
				dialog.ShowInformation("Platform Not Supported", 
					"SPD reading is only available on Windows.", p.window)
			} else {
				dialog.ShowError(fmt.Errorf("Failed to initialize SPD reader: %v", err), p.window)
			}
			return
		}
		defer reader.Close()

		// Read all modules
		spdModules, err := reader.ReadAllModules()
		if err != nil {
			dialog.ShowError(fmt.Errorf("Failed to read SPD data: %v", err), p.window)
			return
		}

		// Update on UI thread
		p.window.Canvas().Content().Refresh()

		// Store SPD modules
		p.spdModules = spdModules

		// Update display
		p.updateDetailsDisplay()

		// Show success message
		msg := fmt.Sprintf("Successfully read SPD data from %d module(s)", len(spdModules))
		dialog.ShowInformation("SPD Read Complete", msg, p.window)
	}()
}

// showRawSPDData shows raw SPD data in a hex viewer
func (p *MemoryDetailsPage) showRawSPDData(data []byte) {
	// Create hex view
	hexView := ""
	for i := 0; i < len(data); i += 16 {
		// Address
		hexView += fmt.Sprintf("%04X: ", i)

		// Hex bytes
		for j := 0; j < 16 && i+j < len(data); j++ {
			hexView += fmt.Sprintf("%02X ", data[i+j])
			if j == 7 {
				hexView += " "
			}
		}

		// Padding if needed
		if i+16 > len(data) {
			remaining := 16 - (len(data) - i)
			for j := 0; j < remaining; j++ {
				hexView += "   "
				if j == 7-(len(data)-i) {
					hexView += " "
				}
			}
		}

		hexView += " |"

		// ASCII representation
		for j := 0; j < 16 && i+j < len(data); j++ {
			b := data[i+j]
			if b >= 32 && b < 127 {
				hexView += string(b)
			} else {
				hexView += "."
			}
		}

		hexView += "|\n"
	}

	// Create scrollable text entry
	entry := widget.NewMultiLineEntry()
	entry.SetText(hexView)
	entry.TextStyle = fyne.TextStyle{Monospace: true}
	entry.Disable()

	// Create dialog
	dialog := dialog.NewCustom("Raw SPD Data", "Close",
		container.NewScroll(entry),
		p.window)
	dialog.Resize(fyne.NewSize(700, 500))
	dialog.Show()
}

// getPCGeneration returns the PC generation number based on memory type
func getPCGeneration(memType string) int {
	memType = strings.ToUpper(memType)
	switch {
	case strings.Contains(memType, "DDR5"):
		return 5
	case strings.Contains(memType, "DDR4"):
		return 4
	case strings.Contains(memType, "DDR3"):
		return 3
	case strings.Contains(memType, "DDR2"):
		return 2
	case strings.Contains(memType, "DDR"):
		return 1
	default:
		return 0
	}
}
