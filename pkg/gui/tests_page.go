package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// TestsPage represents the tests selection page
type TestsPage struct {
	content fyne.CanvasObject
}

// TestOption represents a test option
type TestOption struct {
	Name        string
	Description string
	Icon        fyne.Resource
	Category    string
	OnStart     func()
}

// NewTestsPage creates a new tests page
func NewTestsPage() *TestsPage {
	t := &TestsPage{}
	t.build()
	return t
}

// build creates the tests page UI
func (t *TestsPage) build() {
	// Title
	title := widget.NewLabelWithStyle("Performance Tests", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	// Test options
	testOptions := []TestOption{
		// CPU Tests
		{
			Name:        "CPU Stress Test",
			Description: "Test CPU performance under maximum load",
			Icon:        theme.ComputerIcon(),
			Category:    "CPU",
			OnStart:     func() { fmt.Println("Starting CPU stress test...") },
		},
		{
			Name:        "CPU Benchmark",
			Description: "Measure CPU computational performance",
			Icon:        theme.ComputerIcon(),
			Category:    "CPU",
			OnStart:     func() { fmt.Println("Starting CPU benchmark...") },
		},
		// Memory Tests
		{
			Name:        "Memory Test",
			Description: "Test RAM for errors and stability",
			Icon:        theme.StorageIcon(),
			Category:    "Memory",
			OnStart:     func() { fmt.Println("Starting memory test...") },
		},
		{
			Name:        "Memory Bandwidth",
			Description: "Measure memory throughput performance",
			Icon:        theme.StorageIcon(),
			Category:    "Memory",
			OnStart:     func() { fmt.Println("Starting memory bandwidth test...") },
		},
		// GPU Tests
		{
			Name:        "GPU Stress Test",
			Description: "Test GPU stability under load",
			Icon:        theme.ColorPaletteIcon(),
			Category:    "GPU",
			OnStart:     func() { fmt.Println("Starting GPU stress test...") },
		},
		{
			Name:        "3D Graphics Test",
			Description: "Test 3D rendering performance",
			Icon:        theme.ColorPaletteIcon(),
			Category:    "GPU",
			OnStart:     func() { fmt.Println("Starting 3D graphics test...") },
		},
		{
			Name:        "GPU Compute Test",
			Description: "Test GPU compute capabilities",
			Icon:        theme.ColorPaletteIcon(),
			Category:    "GPU",
			OnStart:     func() { fmt.Println("Starting GPU compute test...") },
		},
		// Storage Tests
		{
			Name:        "Disk Speed Test",
			Description: "Measure storage read/write performance",
			Icon:        theme.FolderIcon(),
			Category:    "Storage",
			OnStart:     func() { fmt.Println("Starting disk speed test...") },
		},
		{
			Name:        "SMART Test",
			Description: "Check disk health and SMART data",
			Icon:        theme.FolderIcon(),
			Category:    "Storage",
			OnStart:     func() { fmt.Println("Starting SMART test...") },
		},
		// Combined Tests
		{
			Name:        "Full System Test",
			Description: "Comprehensive test of all components",
			Icon:        theme.ViewFullScreenIcon(),
			Category:    "System",
			OnStart:     func() { fmt.Println("Starting full system test...") },
		},
		{
			Name:        "Stability Test",
			Description: "Long-duration stability testing",
			Icon:        theme.ViewFullScreenIcon(),
			Category:    "System",
			OnStart:     func() { fmt.Println("Starting stability test...") },
		},
		{
			Name:        "Power Test",
			Description: "Test power consumption and efficiency",
			Icon:        theme.ViewFullScreenIcon(),
			Category:    "System",
			OnStart:     func() { fmt.Println("Starting power test...") },
		},
	}

	// Group tests by category
	categories := make(map[string][]TestOption)
	for _, test := range testOptions {
		categories[test.Category] = append(categories[test.Category], test)
	}

	// Create test cards
	content := container.NewVBox(title, widget.NewSeparator())

	// Add category sections
	for _, category := range []string{"CPU", "Memory", "GPU", "Storage", "System"} {
		tests, ok := categories[category]
		if !ok {
			continue
		}

		// Category header
		categoryLabel := widget.NewLabelWithStyle(category+" Tests", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		content.Add(container.NewPadded(categoryLabel))

		// Test cards grid
		grid := container.NewGridWithColumns(3)
		for _, test := range tests {
			card := t.createTestCard(test)
			grid.Add(card)
		}
		content.Add(container.NewPadded(grid))
		content.Add(widget.NewSeparator())
	}

	// Wrap in scroll container
	t.content = container.NewScroll(content)
}

// createTestCard creates a card for a test option
func (t *TestsPage) createTestCard(test TestOption) fyne.CanvasObject {
	// Icon
	icon := widget.NewIcon(test.Icon)

	// Title
	title := widget.NewLabelWithStyle(test.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Description
	description := widget.NewLabel(test.Description)
	description.Wrapping = fyne.TextWrapWord

	// Start button
	startBtn := widget.NewButton("Start", test.OnStart)
	startBtn.Importance = widget.HighImportance

	// Card content
	content := container.NewVBox(
		container.NewHBox(icon, title),
		description,
		widget.NewLabel(""), // Spacer
		startBtn,
	)

	// Card background
	bg := canvas.NewRectangle(ColorCardBackground)
	bg.CornerRadius = 4

	// Card with padding
	card := container.NewStack(
		bg,
		container.NewPadded(content),
	)

	return container.NewPadded(card)
}

// Content returns the tests page content
func (t *TestsPage) Content() fyne.CanvasObject {
	return t.content
}

// createTestsAccordion creates an accordion with test categories
func createTestsAccordion() *widget.Accordion {
	// CPU Tests
	cpuTests := container.NewVBox(
		widget.NewButton("Prime95 Stress Test", func() {}),
		widget.NewButton("Linpack Benchmark", func() {}),
		widget.NewButton("Multi-core Scaling", func() {}),
		widget.NewButton("AVX/AVX2 Test", func() {}),
	)

	// Memory Tests
	memTests := container.NewVBox(
		widget.NewButton("MemTest86", func() {}),
		widget.NewButton("HCI MemTest", func() {}),
		widget.NewButton("Bandwidth Test", func() {}),
		widget.NewButton("Latency Test", func() {}),
	)

	// GPU Tests
	gpuTests := container.NewVBox(
		widget.NewButton("FurMark", func() {}),
		widget.NewButton("Unigine Heaven", func() {}),
		widget.NewButton("3DMark", func() {}),
		widget.NewButton("CUDA/OpenCL Test", func() {}),
	)

	// Storage Tests
	storageTests := container.NewVBox(
		widget.NewButton("CrystalDiskMark", func() {}),
		widget.NewButton("ATTO Benchmark", func() {}),
		widget.NewButton("4K Random Test", func() {}),
		widget.NewButton("Sequential Test", func() {}),
	)

	// Create accordion
	accordion := widget.NewAccordion(
		widget.NewAccordionItem("CPU Tests", cpuTests),
		widget.NewAccordionItem("Memory Tests", memTests),
		widget.NewAccordionItem("GPU Tests", gpuTests),
		widget.NewAccordionItem("Storage Tests", storageTests),
	)

	return accordion
}
