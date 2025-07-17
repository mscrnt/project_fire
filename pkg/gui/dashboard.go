package gui

import (
	"fmt"
	"image/color"
	"sort"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/shirou/gopsutil/v3/disk"
)

// Dashboard represents the F.I.R.E. System Monitor dashboard
type Dashboard struct {
	content      fyne.CanvasObject
	summaryStrip fyne.CanvasObject // Separate summary strip
	window       fyne.Window       // Reference to main window

	// System info
	sysInfo *SystemInfo

	// Update control
	running  bool
	mu       sync.Mutex
	stopChan chan bool

	// Summary cards
	cpuSummary     *SummaryCard
	memorySummary  *SummaryCard
	gpuSummary     *SummaryCard
	gpuSummaries   []*SummaryCard // For multiple GPUs
	storageSummary *SummaryCard
	currentGPU     int                // Currently displayed GPU
	gpuTabs        *container.AppTabs // GPU tabs

	// Component list and details
	componentList  *widget.List
	detailsGrid    *fyne.Container
	components     []Component
	selectedIndex  int
	storageDevices []StorageInfo // Keep storage devices for details dialog

	// Update tickers
	updateTicker *time.Ticker

	// Cached data
	lastGPUInfo       []GPUInfo
	lastGPUUpdate     time.Time
	lastStorageInfo   []StorageInfo
	lastStorageUpdate time.Time

	// Metric history tracking
	cpuDieTempHistory *MetricHistory
	cpuPowerHistory   *MetricHistory
	cpuUsageHistory   *MetricHistory
	cpuClockHistory   *MetricHistory

	// Static component cache - populated once at startup
	staticComponentCache struct {
		motherboard    *MotherboardInfo
		memoryModules  []MemoryModule
		gpus           []GPUInfo
		storageDevices []StorageInfo
		fans           []FanInfo
	}
	cacheInitialized bool
}

// Component represents a hardware component
type Component struct {
	Type    string // CPU, Memory, GPU, Storage, Motherboard
	Name    string
	Icon    string
	Details map[string]string  // Static details (model, serial, etc.)
	Metrics map[string]float64 // Dynamic metrics (temp, usage, etc.) - moved to details dialog
	Index   int                // Component index for lookups
}

// SummaryCard represents a summary metric card
type SummaryCard struct {
	container *fyne.Container
	title     fyne.CanvasObject
	metrics   map[string]*MetricBar
}

// CreateDashboard creates a F.I.R.E. System Monitor dashboard
// Pass cache as nil to have the dashboard load its own data
func CreateDashboard(cache *StaticCache) *Dashboard {
	d := &Dashboard{
		stopChan:          make(chan bool),
		components:        make([]Component, 0),
		selectedIndex:     -1,
		cpuDieTempHistory: NewMetricHistory(),
		cpuPowerHistory:   NewMetricHistory(),
		cpuUsageHistory:   NewMetricHistory(),
		cpuClockHistory:   NewMetricHistory(),
		storageDevices:    make([]StorageInfo, 0),
	}

	// Copy the preloaded cache if provided
	if cache != nil {
		DebugLog("DEBUG", fmt.Sprintf("CreateDashboard - Using provided cache: %d GPUs, %d memory modules", len(cache.GPUs), len(cache.MemoryModules)))
		d.staticComponentCache.motherboard = cache.Motherboard
		d.staticComponentCache.memoryModules = cache.MemoryModules
		d.staticComponentCache.gpus = cache.GPUs
		d.staticComponentCache.storageDevices = cache.StorageDevices
		d.staticComponentCache.fans = cache.Fans
		d.cacheInitialized = true

		// Also set storage devices and system info
		d.storageDevices = cache.StorageDevices
		if cache.SysInfo != nil {
			d.sysInfo = cache.SysInfo
		}
	} else {
		DebugLog("DEBUG", "CreateDashboard - No cache provided, will load data on demand")
	}

	// Initialize with some default values so tooltips show data immediately
	d.cpuDieTempHistory.Add(45.0)
	d.cpuPowerHistory.Add(35.0)
	d.cpuUsageHistory.Add(20.0)
	d.cpuClockHistory.Add(3.5)

	d.build()
	return d
}

// SetWindow sets the window reference for dialog display
func (d *Dashboard) SetWindow(w fyne.Window) {
	d.window = w
}

// build creates the dashboard UI
func (d *Dashboard) build() {
	DebugLog("DEBUG", "Dashboard.build() - Checking system info...")
	// Get initial system info if not already loaded from cache
	if d.sysInfo == nil {
		DebugLog("DEBUG", "Dashboard.build() - Getting system info...")
		d.sysInfo, _ = GetSystemInfo()
	} else {
		DebugLog("DEBUG", "Dashboard.build() - Using cached system info")
	}

	// Initialize static component cache if not already done
	if !d.cacheInitialized {
		DebugLog("DEBUG", "Dashboard.build() - Initializing static cache...")
		d.initializeStaticCache()
	}

	DebugLog("DEBUG", "Dashboard.build() - Populating components...")
	d.populateComponents()

	// Create summary strip
	DebugLog("DEBUG", "Dashboard.build() - Creating summary strip...")
	summaryStrip := d.createSummaryStrip()

	// Create main content area
	DebugLog("DEBUG", "Dashboard.build() - Creating main content...")
	mainContent := d.createMainContent()

	// Store summary strip separately
	d.summaryStrip = summaryStrip

	// Main content is just the components and details
	d.content = mainContent

	DebugLog("DEBUG", "Dashboard.build() - Complete")
}

// createSummaryStrip creates the top summary cards
func (d *Dashboard) createSummaryStrip() *fyne.Container {
	// Get CPU name
	cpuName := "CPU"
	if d.sysInfo != nil && d.sysInfo.CPU.Model != "" {
		cpuName = d.sysInfo.CPU.Model
	}

	// CPU Summary with actual CPU name - metrics in specific order
	d.cpuSummary = d.createCompactSummaryCard("CPU", cpuName, []string{"Temp", "Voltage", "Power", "Usage", "Speed"}, map[string]color.Color{
		"Temp":    ColorTemperature,
		"Voltage": ColorVoltage,
		"Power":   ColorPower,
		"Usage":   ColorCPUUsage,
		"Speed":   ColorFrequency,
	})

	// Memory Summary - metrics in specific order
	d.memorySummary = d.createCompactSummaryCard("Memory", "Memory", []string{"Temp", "Used", "Total"}, map[string]color.Color{
		"Temp":  ColorTemperature,
		"Used":  ColorMemoryUsage,
		"Total": ColorFrequency,
	})

	// GPU Summaries - create one for each GPU from cache
	gpus := d.staticComponentCache.gpus
	d.gpuSummaries = make([]*SummaryCard, 0)

	if len(gpus) > 0 {
		// Create tabs for multiple GPUs
		d.gpuTabs = container.NewAppTabs()

		// Create compact tabs
		for i := range gpus {
			tabLabel := fmt.Sprintf("%d", i+1)
			d.gpuTabs.Append(container.NewTabItem(tabLabel, widget.NewLabel(""))) // Empty content
		}

		for _, gpu := range gpus {
			// Use GPU name
			gpuName := fmt.Sprintf("%s %s", gpu.Vendor, gpu.Name)
			gpuCard := d.createCompactSummaryCard("GPU", gpuName, []string{"Temp", "Voltage", "Power", "Usage", "Speed", "VRAM"}, map[string]color.Color{
				"Temp":    ColorTemperature,
				"Voltage": ColorVoltage,
				"Power":   ColorPower,
				"Usage":   ColorGPUUsage,
				"Speed":   ColorFrequency,
				"VRAM":    ColorMemoryUsage,
			})
			d.gpuSummaries = append(d.gpuSummaries, gpuCard)
		}

		// Set the first GPU as current
		d.currentGPU = 0
		d.gpuSummary = d.gpuSummaries[0]
	} else {
		// No GPU detected
		d.gpuSummary = d.createCompactSummaryCard("GPU", "No GPU Detected", []string{"Temp", "Voltage", "Power", "Usage", "Speed", "VRAM"}, map[string]color.Color{
			"Temp":    ColorTemperature,
			"Voltage": ColorVoltage,
			"Power":   ColorPower,
			"Usage":   ColorGPUUsage,
			"Speed":   ColorFrequency,
			"VRAM":    ColorMemoryUsage,
		})
		d.gpuTabs = container.NewAppTabs(
			container.NewTabItem("N/A", d.gpuSummary.container),
		)
	}

	// For GPU, we'll use the first card if available, or the no-GPU card
	var gpuContainer fyne.CanvasObject
	if len(d.gpuSummaries) > 0 {
		// Update tab selection handler to update GPU name in the card
		d.gpuTabs.OnSelected = func(tab *container.TabItem) {
			// Get current tab index
			for i, t := range d.gpuTabs.Items {
				if t == tab {
					d.currentGPU = i
					// Update the GPU name in the first card (display card)
					if i < len(gpus) && len(d.gpuSummaries) > 0 {
						gpu := gpus[i]
						gpuName := fmt.Sprintf("%s %s", gpu.Vendor, gpu.Name)
						d.updateGPUCardTitle(d.gpuSummaries[0], gpuName)
					}
					break
				}
			}
		}
		gpuContainer = d.gpuSummaries[0].container
	} else {
		gpuContainer = d.gpuSummary.container
	}

	// Storage Summary - show primary storage device from cache
	storageDevices := d.staticComponentCache.storageDevices
	storageName := "Storage"
	if len(storageDevices) > 0 {
		// Use the first storage device (usually the boot drive)
		storage := storageDevices[0]
		if storage.Model != "" {
			storageName = storage.Model
		} else {
			storageName = fmt.Sprintf("%s Drive", storage.Mountpoint)
		}
	}

	d.storageSummary = d.createCompactSummaryCard("Storage", storageName, []string{"Temp", "Health", "Used", "Read", "Write"}, map[string]color.Color{
		"Temp":   ColorTemperature,
		"Health": ColorGood,
		"Used":   ColorMemoryUsage,
		"Read":   ColorCPUUsage,
		"Write":  ColorGPUUsage,
	})

	// Create a full-width header with dark background
	headerBg := canvas.NewRectangle(color.RGBA{0x1a, 0x1a, 0x1a, 0xff})

	// Create proportional layout: CPU 25%, Memory 20%, GPU 30%, Storage 25%
	proportionalLayout := container.New(&proportionalSplitLayout{
		ratios: []float32{0.25, 0.20, 0.30, 0.25},
	},
		d.cpuSummary.container,
		d.memorySummary.container,
		gpuContainer,
		d.storageSummary.container,
	)

	// Wrap in horizontal scroll container
	scrollableContent := container.NewHScroll(proportionalLayout)
	scrollableContent.SetMinSize(fyne.NewSize(0, 90)) // Maintain header height for 900p

	// Stack the background and scrollable content
	fullHeader := container.NewStack(
		headerBg,
		scrollableContent,
	)

	// Return the full-width header
	return fullHeader
}

// createCompactSummaryCard creates a compact summary card with metrics in specific order
func (d *Dashboard) createCompactSummaryCard(title, deviceName string, metricOrder []string, metrics map[string]color.Color) *SummaryCard {
	card := &SummaryCard{
		metrics: make(map[string]*MetricBar),
	}

	// Title with icon
	var iconResource fyne.Resource
	switch title {
	case "CPU":
		iconResource = GetCPUIcon()
	case "Memory":
		iconResource = GetMemoryIcon()
	case "GPU":
		iconResource = GetGPUIcon()
	case "Storage":
		iconResource = GetStorageIcon()
	}

	// Use device name if provided, otherwise use title
	displayName := deviceName
	if displayName == "" {
		displayName = title
	}

	var titleContent fyne.CanvasObject
	if iconResource != nil {
		icon := canvas.NewImageFromResource(iconResource)
		icon.SetMinSize(fyne.NewSize(16, 16)) // Even smaller icon for compact height
		icon.FillMode = canvas.ImageFillContain
		titleLabel := widget.NewLabelWithStyle(displayName, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		titleLabel.TextStyle.Monospace = false
		titleLabel.TextStyle.TabWidth = 0
		// Truncate long names for compact display
		if len(displayName) > 25 {
			displayName = displayName[:22] + "..."
			titleLabel.SetText(displayName)
		}

		// For GPU, add tabs to the title row
		if title == "GPU" && d.gpuTabs != nil && len(d.gpuSummaries) > 0 {
			titleContent = container.NewBorder(
				nil, nil,
				container.NewHBox(icon, titleLabel), // Left: icon and name
				d.gpuTabs,                           // Right: tabs
				nil,
			)
		} else {
			titleContent = container.NewHBox(icon, titleLabel)
		}
	} else {
		titleContent = widget.NewLabelWithStyle(displayName, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	}

	card.title = titleContent

	// Create metric bars in specified order
	metricContainers := make([]fyne.CanvasObject, 0)
	for _, name := range metricOrder {
		if barColor, ok := metrics[name]; ok {
			// Create metric bar - show bar for all except Voltage
			showBar := name != "Voltage"
			bar := NewMetricBar(name, barColor, showBar)
			card.metrics[name] = bar
			metricContainers = append(metricContainers, bar)
		}
	}

	// Build card with spacing between metrics
	spacedMetrics := make([]fyne.CanvasObject, 0)
	for i, metric := range metricContainers {
		spacedMetrics = append(spacedMetrics, metric)
		// Add spacer between metrics (but not after the last one)
		if i < len(metricContainers)-1 {
			spacedMetrics = append(spacedMetrics, widget.NewLabel(" ")) // Small spacer
		}
	}
	metricsRow := container.NewHBox(spacedMetrics...)

	// Card content - title above, metrics below
	content := container.NewVBox(
		card.title,
		metricsRow,
	)

	// Card background - match the header background
	bg := canvas.NewRectangle(color.RGBA{0x2a, 0x2a, 0x2a, 0xff})
	bg.StrokeColor = color.RGBA{0x33, 0x33, 0x33, 0xff}
	bg.StrokeWidth = 1

	// Add internal padding
	paddedContent := container.NewBorder(
		nil, nil,
		widget.NewLabel("  "), // Left padding
		widget.NewLabel("  "), // Right padding
		content,
	)

	// Center the content vertically
	centeredContent := container.NewCenter(paddedContent)

	card.container = container.NewStack(bg, centeredContent)
	return card
}

// createSummaryCard creates a summary card with metrics
func (d *Dashboard) createSummaryCard(title, deviceName string, metrics map[string]color.Color) *SummaryCard {
	card := &SummaryCard{
		metrics: make(map[string]*MetricBar),
	}

	// Title with icon
	var iconResource fyne.Resource
	switch title {
	case "CPU":
		iconResource = GetCPUIcon()
	case "Memory":
		iconResource = GetMemoryIcon()
	case "GPU":
		iconResource = GetGPUIcon()
	case "Storage":
		iconResource = GetStorageIcon()
	}

	// Use device name if provided, otherwise use title
	displayName := deviceName
	if displayName == "" {
		displayName = title
	}

	var titleContent fyne.CanvasObject
	if iconResource != nil {
		icon := canvas.NewImageFromResource(iconResource)
		icon.SetMinSize(fyne.NewSize(20, 20))   // Smaller icon for compact height
		icon.FillMode = canvas.ImageFillContain // Maintain aspect ratio
		titleLabel := widget.NewLabelWithStyle(displayName, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		titleContent = container.NewHBox(icon, titleLabel)
	} else {
		titleContent = widget.NewLabelWithStyle(displayName, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	}

	card.title = titleContent

	// Create metric bars
	metricContainers := make([]fyne.CanvasObject, 0)
	for name, barColor := range metrics {
		// Create metric bar - show bar for all except Voltage
		showBar := name != "Voltage"
		bar := NewMetricBar(name, barColor, showBar)
		card.metrics[name] = bar
		metricContainers = append(metricContainers, bar)
	}

	// Card background
	bg := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))
	bg.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	bg.StrokeWidth = 1

	// Build card with two rows - title on top, metrics below
	// First row: device name with icon
	// Second row: metrics arranged horizontally
	content := container.NewVBox(
		card.title,
		container.NewHBox(metricContainers...),
	)

	// Use minimal padding for compact height
	card.container = container.NewStack(bg, container.NewBorder(
		nil, nil,
		widget.NewLabel(" "), // Small left padding
		widget.NewLabel(" "), // Small right padding
		content,
	))
	return card
}

// createWelcomePane creates a welcoming interface for the details panel
func (d *Dashboard) createWelcomePane() fyne.CanvasObject {
	// Main title with larger font and color
	titleLabel := widget.NewRichTextFromMarkdown("# Welcome to F.I.R.E. System Monitor")

	// Subtitle with ember color
	subtitleText := canvas.NewText("Full Intensity Rigorous Evaluation", ColorEmber)
	subtitleText.Alignment = fyne.TextAlignCenter
	subtitleText.TextStyle = fyne.TextStyle{Italic: true}

	// Add colored separator
	separator1 := canvas.NewRectangle(ColorEmber)
	separator1.SetMinSize(fyne.NewSize(200, 2))

	// Add spacing
	spacer1 := canvas.NewRectangle(color.Transparent)
	spacer1.SetMinSize(fyne.NewSize(0, 20))

	// Create sections with colored headers
	gettingStartedBg := canvas.NewRectangle(color.RGBA{0x2a, 0x2a, 0x2a, 0xff})
	gettingStartedBg.CornerRadius = 8
	gettingStartedTitle := canvas.NewText("Getting Started", ColorGood)
	gettingStartedTitle.TextStyle = fyne.TextStyle{Bold: true}
	gettingStartedTitle.TextSize = 16

	// Instructions with icons using colored bullets
	bullet1 := canvas.NewText("▸", ColorGood)
	instruction1 := widget.NewLabel("Click on any hardware component in the list to view detailed information")
	instruction1.Wrapping = fyne.TextWrapWord
	row1 := container.NewBorder(nil, nil, bullet1, nil, instruction1)

	bullet2 := canvas.NewText("▸", ColorWarning)
	instruction2 := widget.NewLabel("Monitor real-time performance metrics in the header")
	instruction2.Wrapping = fyne.TextWrapWord
	row2 := container.NewBorder(nil, nil, bullet2, nil, instruction2)

	bullet3 := canvas.NewText("▸", ColorCPUUsage)
	instruction3 := widget.NewLabel("Navigate between different monitoring modes using the sidebar")
	instruction3.Wrapping = fyne.TextWrapWord
	row3 := container.NewBorder(nil, nil, bullet3, nil, instruction3)

	bullet4 := canvas.NewText("▸", ColorEmber)
	instruction4 := widget.NewLabel("View historical data and trends for each component")
	instruction4.Wrapping = fyne.TextWrapWord
	row4 := container.NewBorder(nil, nil, bullet4, nil, instruction4)

	instructionsBox := container.NewVBox(
		row1,
		widget.NewSeparator(),
		row2,
		widget.NewSeparator(),
		row3,
		widget.NewSeparator(),
		row4,
	)

	// Add spacing
	spacer2 := canvas.NewRectangle(color.Transparent)
	spacer2.SetMinSize(fyne.NewSize(0, 30))

	// System overview section with background
	overviewBg := canvas.NewRectangle(color.RGBA{0x1a, 0x1a, 0x1a, 0xff})
	overviewBg.CornerRadius = 8
	overviewTitle := canvas.NewText("System Overview", ColorCPUUsage)
	overviewTitle.TextStyle = fyne.TextStyle{Bold: true}
	overviewTitle.TextSize = 16
	overviewTitle.Alignment = fyne.TextAlignCenter

	// Create colored system info cards
	cpuCard := canvas.NewRectangle(color.RGBA{ColorCPUUsage.R, ColorCPUUsage.G, ColorCPUUsage.B, 0x20})
	cpuCard.CornerRadius = 4
	cpuInfo := widget.NewLabel(fmt.Sprintf("CPU\n%s\n%d cores", d.sysInfo.CPU.Model, d.sysInfo.CPU.LogicalCores))
	cpuInfo.Alignment = fyne.TextAlignCenter
	cpuContainer := container.NewStack(cpuCard, container.NewPadded(cpuInfo))

	memCard := canvas.NewRectangle(color.RGBA{ColorMemoryUsage.R, ColorMemoryUsage.G, ColorMemoryUsage.B, 0x20})
	memCard.CornerRadius = 4
	memInfo := widget.NewLabel(fmt.Sprintf("MEMORY\n%.1f GB\nTotal", d.sysInfo.Memory.TotalGB))
	memInfo.Alignment = fyne.TextAlignCenter
	memContainer := container.NewStack(memCard, container.NewPadded(memInfo))

	storageCount := len(d.components) - 2
	for _, comp := range d.components {
		if comp.Type == "GPU" {
			storageCount--
		}
	}
	storageCard := canvas.NewRectangle(color.RGBA{ColorGPUUsage.R, ColorGPUUsage.G, ColorGPUUsage.B, 0x20})
	storageCard.CornerRadius = 4
	storageInfo := widget.NewLabel(fmt.Sprintf("STORAGE\n%d devices\ndetected", storageCount))
	storageInfo.Alignment = fyne.TextAlignCenter
	storageContainer := container.NewStack(storageCard, container.NewPadded(storageInfo))

	systemCards := container.NewGridWithColumns(3, cpuContainer, memContainer, storageContainer)

	// Add spacing
	spacer3 := canvas.NewRectangle(color.Transparent)
	spacer3.SetMinSize(fyne.NewSize(0, 30))

	// Tips section with accent background
	tipsBg := canvas.NewRectangle(color.RGBA{ColorEmber.R, ColorEmber.G, ColorEmber.B, 0x10})
	tipsBg.CornerRadius = 8
	tipsTitle := canvas.NewText("Pro Tips", ColorEmber)
	tipsTitle.TextStyle = fyne.TextStyle{Bold: true}
	tipsTitle.TextSize = 16
	tipsTitle.Alignment = fyne.TextAlignCenter

	tipBullet1 := canvas.NewText("★", ColorEmber)
	tip1 := widget.NewLabel("Use STABILITY TEST mode to stress test your system")
	tip1.Wrapping = fyne.TextWrapWord
	tipRow1 := container.NewBorder(nil, nil, tipBullet1, nil, tip1)

	tipBullet2 := canvas.NewText("★", ColorEmber)
	tip2 := widget.NewLabel("BENCHMARKS provide performance comparisons")
	tip2.Wrapping = fyne.TextWrapWord
	tipRow2 := container.NewBorder(nil, nil, tipBullet2, nil, tip2)

	tipBullet3 := canvas.NewText("★", ColorEmber)
	tip3 := widget.NewLabel("MONITORING shows real-time system metrics")
	tip3.Wrapping = fyne.TextWrapWord
	tipRow3 := container.NewBorder(nil, nil, tipBullet3, nil, tip3)

	tipBullet4 := canvas.NewText("★", ColorEmber)
	tip4 := widget.NewLabel("Customize alerts and thresholds in SETTINGS")
	tip4.Wrapping = fyne.TextWrapWord
	tipRow4 := container.NewBorder(nil, nil, tipBullet4, nil, tip4)

	tipsContent := container.NewVBox(tipRow1, tipRow2, tipRow3, tipRow4)
	tipsBox := container.NewStack(tipsBg, container.NewPadded(tipsContent))

	// Build the complete welcome pane with colored sections
	content := container.NewVBox(
		container.NewCenter(titleLabel),
		container.NewCenter(subtitleText),
		container.NewCenter(separator1),
		spacer1,
		container.NewPadded(gettingStartedTitle),
		container.NewPadded(instructionsBox),
		spacer2,
		overviewTitle,
		container.NewPadded(systemCards),
		spacer3,
		tipsTitle,
		container.NewPadded(tipsBox),
	)

	// Use NewMax to fill the entire space
	return container.NewStack(content)
}

// createMainContent creates the two-column main area
func (d *Dashboard) createMainContent() *fyne.Container {
	// Component list (left) with custom selection
	d.componentList = widget.NewList(
		func() int { return len(d.components) },
		func() fyne.CanvasObject {
			// Create background to override default selection
			bg := canvas.NewRectangle(color.Transparent)

			name := widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{})
			// Create outline for selection
			outline := canvas.NewRectangle(color.Transparent)
			outline.StrokeColor = color.Transparent
			outline.StrokeWidth = 2  // Slightly thicker for visibility
			outline.CornerRadius = 6 // Match navbar radius

			// Stack: background, outline, padded label
			content := container.NewStack(
				bg, // This will block the default selection background
				outline,
				container.NewPadded(name),
			)
			return content
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if i >= len(d.components) {
				return
			}
			comp := d.components[i]
			content := o.(*fyne.Container)
			bg := content.Objects[0].(*canvas.Rectangle)
			outline := content.Objects[1].(*canvas.Rectangle)
			padded := content.Objects[2].(*fyne.Container)
			name := padded.Objects[0].(*widget.Label)

			// Always keep background matching the list background
			bg.FillColor = color.RGBA{0x19, 0x19, 0x19, 0xff} // Match the panel background
			bg.Refresh()

			// Truncate long component names (no icons) - increased limit
			displayName := truncateText(comp.Name, 50)
			name.SetText(displayName)

			// Highlight selected with outline only
			if i == d.selectedIndex {
				name.TextStyle = fyne.TextStyle{Bold: true}
				outline.StrokeColor = ColorEmber
				outline.FillColor = color.RGBA{ColorEmber.R, ColorEmber.G, ColorEmber.B, 0x20}
			} else {
				name.TextStyle = fyne.TextStyle{}
				outline.StrokeColor = color.Transparent
				outline.FillColor = color.Transparent
			}
			name.Refresh()
			outline.Refresh()
		},
	)

	d.componentList.OnSelected = func(id widget.ListItemID) {
		d.selectedIndex = id
		d.updateDetails()
		d.componentList.Refresh() // Force immediate visual update
	}

	// Details grid (right) - Initialize as VBox
	d.detailsGrid = container.NewVBox()

	// Create welcome pane
	welcomeContainer := d.createWelcomePane()
	d.detailsGrid.Add(welcomeContainer)

	// Create fixed layout with components list and details panel
	// Using a custom layout to maintain fixed 30/70 split
	// Create centered Hardware header with double font size
	hardwareHeader := widget.NewLabelWithStyle("HARDWARE", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	componentsPanel := container.NewBorder(
		container.NewPadded(hardwareHeader),
		nil, nil, nil,
		d.componentList,
	)

	// Create scrollable details panel
	detailsScroll := container.NewVScroll(d.detailsGrid)
	detailsScroll.SetMinSize(fyne.NewSize(0, 400)) // Ensure minimum height

	detailsPanel := container.NewBorder(
		container.NewPadded(widget.NewLabelWithStyle("INFORMATION", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})),
		nil, nil, nil,
		detailsScroll,
	)

	// Create a fixed layout container
	content := container.New(&fixedSplitLayout{leftRatio: 0.3},
		componentsPanel,
		detailsPanel,
	)

	return container.NewPadded(content)
}

// initializeStaticCache populates the static component cache once at startup
func (d *Dashboard) initializeStaticCache() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.cacheInitialized {
		return
	}

	// Cache all static component information
	// This runs once at startup to avoid repeated queries

	// Get all static info upfront
	DebugLog("DEBUG", "initializeStaticCache - Getting motherboard info...")
	d.staticComponentCache.motherboard, _ = GetMotherboardInfo()

	DebugLog("DEBUG", "initializeStaticCache - Getting memory modules...")
	d.staticComponentCache.memoryModules, _ = GetMemoryModules()

	DebugLog("DEBUG", "initializeStaticCache - Getting GPU info...")
	d.staticComponentCache.gpus, _ = GetGPUInfo()

	DebugLog("DEBUG", "initializeStaticCache - Getting storage info...")
	// Skip storage info during initial load as it's slow and blocks UI
	// We'll load it asynchronously later
	d.staticComponentCache.storageDevices = []StorageInfo{}
	DebugLog("DEBUG", "initializeStaticCache - Skipping storage info (will load async)")

	DebugLog("DEBUG", "initializeStaticCache - Getting fan info...")
	d.staticComponentCache.fans, _ = GetFanInfo()

	// Also cache storage devices for later use
	d.storageDevices = d.staticComponentCache.storageDevices

	d.cacheInitialized = true
	DebugLog("DEBUG", "initializeStaticCache - Complete")
}

// populateComponents populates the component list from cached static data
func (d *Dashboard) populateComponents() {
	d.components = []Component{}

	// CPU - from system info (always available)
	if d.sysInfo != nil && d.sysInfo.CPU.Model != "" {
		d.components = append(d.components, Component{
			Type:  "CPU",
			Icon:  "🔥",
			Name:  d.sysInfo.CPU.Model,
			Index: len(d.components),
			Details: map[string]string{
				"Model":          d.sysInfo.CPU.Model,
				"Vendor":         d.sysInfo.CPU.Vendor,
				"Physical Cores": fmt.Sprintf("%d", d.sysInfo.CPU.PhysicalCores),
				"Logical Cores":  fmt.Sprintf("%d", d.sysInfo.CPU.LogicalCores),
			},
		})
	}

	// Motherboard - from cache
	motherboard := d.staticComponentCache.motherboard
	if motherboard != nil && motherboard.Model != "" {
		mbDetails := map[string]string{
			"Manufacturer": motherboard.Manufacturer,
			"Model":        motherboard.Model,
		}
		if motherboard.Version != "" && motherboard.Version != "Not Available" {
			mbDetails["Version"] = motherboard.Version
		}
		if motherboard.BIOS.Vendor != "" {
			mbDetails["BIOS Vendor"] = motherboard.BIOS.Vendor
		}
		if motherboard.BIOS.Version != "" {
			mbDetails["BIOS Version"] = motherboard.BIOS.Version
		}
		if motherboard.BIOS.ReleaseDate != "" {
			mbDetails["BIOS Date"] = FormatBIOSDate(motherboard.BIOS.ReleaseDate)
		}

		// Add chipset info if available
		if motherboard.ChipsetInfo.Model != "" {
			chipset := motherboard.ChipsetInfo.Model
			if motherboard.ChipsetInfo.Vendor != "" {
				chipset = fmt.Sprintf("%s %s", motherboard.ChipsetInfo.Vendor, motherboard.ChipsetInfo.Model)
			}
			mbDetails["Chipset"] = chipset
		}

		// Add memory slot info
		if motherboard.Features.MemorySlots > 0 {
			mbDetails["Memory Slots"] = fmt.Sprintf("%d", motherboard.Features.MemorySlots)
		}
		if motherboard.Features.MaxMemory > 0 {
			maxMemGB := float64(motherboard.Features.MaxMemory) / (1024 * 1024 * 1024)
			mbDetails["Max Memory"] = fmt.Sprintf("%.0f GB", maxMemGB)
		}

		mbName := motherboard.Model
		if motherboard.Manufacturer != "" && motherboard.Manufacturer != "Not Available" {
			mbName = fmt.Sprintf("%s %s", motherboard.Manufacturer, motherboard.Model)
		}

		d.components = append(d.components, Component{
			Type:    "Motherboard",
			Icon:    "🔧",
			Name:    mbName,
			Index:   len(d.components),
			Details: mbDetails,
		})
	}

	// Memory - show individual modules from cache
	memoryModules := d.staticComponentCache.memoryModules
	DebugLog("DEBUG", fmt.Sprintf("populateComponents - Found %d memory modules in cache", len(memoryModules)))
	if len(memoryModules) > 0 {
		// Add individual memory modules
		for i := range memoryModules {
			// Update module with proper row number if not set
			if memoryModules[i].Row == 0 {
				memoryModules[i].Row = i + 1
				memoryModules[i].Number = fmt.Sprintf("%d", i+1)
			}

			// Ensure all calculated fields are populated
			module := &memoryModules[i]
			if module.SizeGB == 0 && module.Size > 0 {
				module.SizeGB = float64(module.Size) / (1024 * 1024 * 1024)
			}
			if module.BaseFrequency == 0 && module.Speed > 0 {
				module.BaseFrequency = float64(module.Speed) / 2.0
			}
			if module.DataRate == 0 && module.Speed > 0 {
				module.DataRate = int(module.Speed)
			}
			if module.PCRating == 0 && module.DataRate > 0 {
				module.PCRating = module.DataRate * 8
			}
			if module.ChipManufacturer == "" {
				module.ChipManufacturer = getChipManufacturer(module.Manufacturer, module.PartNumber)
			}
			// Build comprehensive details with all CPU-Z style fields
			memDetails := map[string]string{}

			// Add Name field if available
			if module.Name != "" {
				memDetails["Name"] = module.Name
			} else {
				// Build name if not set
				module.Name = fmt.Sprintf("Row %d [%s/%s] – %.0f GB %s %s %s",
					i+1, module.BankLabel, module.Slot, module.SizeGB, module.Type,
					module.Manufacturer, module.PartNumber)
			}

			memDetails["Number"] = fmt.Sprintf("%d", i+1)
			memDetails["Type"] = module.Type

			if module.Manufacturer != "" && module.Manufacturer != "Unknown" &&
				module.Manufacturer != "Not Specified" && module.Manufacturer != "NO DIMM" {
				memDetails["Manufacturer"] = module.Manufacturer
			}

			// Add chip manufacturer if available
			if module.ChipManufacturer != "" && module.ChipManufacturer != "Unknown" {
				memDetails["Chip manufacturer"] = module.ChipManufacturer
			}

			// Add base frequency with full format
			if module.BaseFrequency > 0 && module.DataRate > 0 && module.PCRating > 0 {
				memDetails["Base frequency"] = fmt.Sprintf("%.1f MHz (DDR5-%d / PC5-%d)",
					module.BaseFrequency, module.DataRate, module.PCRating)
			} else if module.Speed > 0 {
				// Calculate if not already set
				baseFreq := float64(module.Speed) / 2.0
				dataRate := int(module.Speed)
				pcRating := dataRate * 8
				pcPrefix := "PC5"
				switch module.Type {
				case "DDR4":
					pcPrefix = "PC4"
				case "DDR3":
					pcPrefix = "PC3"
				}
				memDetails["Base frequency"] = fmt.Sprintf("%.1f MHz (%s-%d / %s-%d)",
					baseFreq, module.Type, dataRate, pcPrefix, pcRating)
			}

			// Size in GBytes format
			if module.SizeGB > 0 {
				memDetails["Size"] = fmt.Sprintf("%.0f GBytes", module.SizeGB)
			} else {
				memDetails["Size"] = FormatMemorySize(module.Size)
			}

			if module.PartNumber != "" && module.PartNumber != "Unknown" &&
				module.PartNumber != "Not Specified" {
				memDetails["Part number"] = module.PartNumber
			}

			if module.SerialNumber != "" && module.SerialNumber != "Unknown" {
				memDetails["Serial number"] = module.SerialNumber
			}

			// Additional details that might be useful
			memDetails["Slot"] = module.Slot
			if module.FormFactor != "" && module.FormFactor != "Unknown" {
				memDetails["Form Factor"] = module.FormFactor
			}

			// Build display name
			memName := fmt.Sprintf("%s %s", FormatMemorySize(module.Size), module.Type)
			if module.Speed > 0 {
				memName = fmt.Sprintf("%s %s @ %d MHz", FormatMemorySize(module.Size), module.Type, module.Speed)
			}
			if module.Manufacturer != "" && module.Manufacturer != "Unknown" &&
				module.Manufacturer != "Not Specified" && module.Manufacturer != "NO DIMM" {
				memName = fmt.Sprintf("%s %s", module.Manufacturer, memName)
			}

			// Add slot info to name if available
			if module.Slot != "" && module.Slot != "Unknown" {
				memName = fmt.Sprintf("%s (Slot: %s)", memName, module.Slot)
			}

			d.components = append(d.components, Component{
				Type:    "Memory",
				Icon:    "💾",
				Name:    memName,
				Index:   len(d.components),
				Details: memDetails,
			})
		}
	} else if d.sysInfo != nil {
		// Fallback to system memory if no modules detected
		memDetails := map[string]string{
			"Total":     fmt.Sprintf("%.1f GB", d.sysInfo.Memory.TotalGB),
			"Available": fmt.Sprintf("%.1f GB", d.sysInfo.Memory.AvailableGB),
			"Used":      fmt.Sprintf("%.1f GB", d.sysInfo.Memory.UsedGB),
		}

		// Show host memory if in WSL
		memName := fmt.Sprintf("System Memory (%.1f GB)", d.sysInfo.Memory.TotalGB)
		if d.sysInfo.Host.IsWSL && d.sysInfo.Memory.HostTotalGB > 0 {
			memName = fmt.Sprintf("System Memory (%.1f GB WSL / %.1f GB Host)",
				d.sysInfo.Memory.TotalGB, d.sysInfo.Memory.HostTotalGB)
			memDetails["Host Total"] = fmt.Sprintf("%.1f GB", d.sysInfo.Memory.HostTotalGB)
			memDetails["Environment"] = "WSL2"
		}

		d.components = append(d.components, Component{
			Type:    "Memory",
			Icon:    "💾",
			Name:    memName,
			Index:   len(d.components),
			Details: memDetails,
		})
	}

	// GPU - from cache
	gpus := d.staticComponentCache.gpus
	DebugLog("DEBUG", fmt.Sprintf("populateComponents - Found %d GPUs in cache", len(gpus)))
	for i, gpu := range gpus {
		// Clean up GPU name - remove vendor from name if it's already included
		gpuName := gpu.Name
		if strings.HasPrefix(strings.ToUpper(gpu.Name), strings.ToUpper(gpu.Vendor)) {
			gpuName = strings.TrimPrefix(gpu.Name, gpu.Vendor)
			gpuName = strings.TrimPrefix(gpuName, " ")
		}

		displayName := gpuName
		if gpu.Vendor != "" && !strings.Contains(strings.ToUpper(gpuName), strings.ToUpper(gpu.Vendor)) {
			displayName = fmt.Sprintf("%s %s", gpu.Vendor, gpuName)
		}

		d.components = append(d.components, Component{
			Type:  "GPU",
			Icon:  "🎮",
			Name:  displayName,
			Index: len(d.components),
			Details: map[string]string{
				"Name":         gpu.Name,
				"Vendor":       gpu.Vendor,
				"Memory Total": fmt.Sprintf("%d MB", gpu.MemoryTotal/(1024*1024)),
				"GPU Index":    fmt.Sprintf("%d", i),
			},
		})
	}

	// Storage devices - from cache
	storageDevices := d.staticComponentCache.storageDevices
	for i := range storageDevices {
		storage := &storageDevices[i]
		icon := "💾"
		switch storage.Type {
		case "NVME":
			icon = "⚡"
		case "SSD":
			icon = "💿"
		case "USB":
			icon = "🔌"
		case "Windows Drive":
			icon = "🪟"
		}

		// Build display name based on available information
		displayName := ""
		if storage.Model != "" {
			displayName = storage.Model
			if storage.Vendor != "" && !strings.Contains(strings.ToLower(storage.Model), strings.ToLower(storage.Vendor)) {
				displayName = fmt.Sprintf("%s %s", storage.Vendor, storage.Model)
			}
			// Add mount point/drive letter
			displayName = fmt.Sprintf("%s (%s)", displayName, storage.Mountpoint)
		} else {
			// Fallback to mount point if no model info
			displayName = fmt.Sprintf("%s Drive", storage.Mountpoint)
		}

		// Add size to display name
		sizeGB := float64(storage.Size) / (1024 * 1024 * 1024)
		if sizeGB >= 1000 {
			displayName = fmt.Sprintf("%s - %.1f TB", displayName, sizeGB/1024)
		} else {
			displayName = fmt.Sprintf("%s - %.1f GB", displayName, sizeGB)
		}

		// Build details map with ONLY static info
		details := map[string]string{
			"Technology":  storage.Type, // NVMe, SSD, HDD
			"Capacity":    fmt.Sprintf("%.1f GB", float64(storage.Size)/(1024*1024*1024)),
			"Mount Point": storage.Mountpoint,
			"File System": storage.Filesystem,
		}

		// Add model and identification info
		if storage.Model != "" {
			details["Model"] = storage.Model
		}
		if storage.Vendor != "" {
			details["Vendor"] = storage.Vendor
		}
		if storage.Controller != "" {
			details["Controller"] = storage.Controller
		}
		if storage.Firmware != "" {
			details["Firmware"] = storage.Firmware
		}
		if storage.Serial != "" {
			details["Serial"] = storage.Serial
		}
		if storage.Interface != "" {
			details["Interface"] = storage.Interface
		}

		d.components = append(d.components, Component{
			Type:    "Storage",
			Icon:    icon,
			Name:    displayName,
			Index:   len(d.components),
			Details: details,
			Metrics: map[string]float64{"storageIndex": float64(i)}, // Keep index for details lookup
		})
	}

	// Fans - from cache
	fans := d.staticComponentCache.fans
	for _, fan := range fans {
		icon := "🌀"
		switch fan.Type {
		case "CPU":
			icon = "❄️"
		case "GPU":
			icon = "🔥"
		}

		d.components = append(d.components, Component{
			Type:  "Fan",
			Icon:  icon,
			Name:  fan.Name,
			Index: len(d.components),
			Details: map[string]string{
				"Name": fan.Name,
				"Type": fan.Type,
			},
		})
	}

	// System
	if d.sysInfo != nil {
		d.components = append(d.components, Component{
			Type:  "System",
			Icon:  "🖥️",
			Name:  fmt.Sprintf("%s - %s", d.sysInfo.Host.Hostname, d.sysInfo.Host.Platform),
			Index: len(d.components),
			Details: map[string]string{
				"Hostname":     d.sysInfo.Host.Hostname,
				"Platform":     d.sysInfo.Host.Platform,
				"Version":      d.sysInfo.Host.PlatformVersion,
				"Kernel":       d.sysInfo.Host.KernelVersion,
				"Architecture": d.sysInfo.Host.Architecture,
			},
		})
	}
}

// updateDetails updates the details panel with static info only
func (d *Dashboard) updateDetails() {
	if d.selectedIndex < 0 || d.selectedIndex >= len(d.components) {
		return
	}

	comp := &d.components[d.selectedIndex]

	// Create new VBox for details
	newDetailsContent := container.NewVBox()

	// Add component name as header
	header := widget.NewLabelWithStyle(
		comp.Name,
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)
	headerContainer := container.NewBorder(
		nil, nil, nil, nil,
		container.NewPadded(header),
	)
	newDetailsContent.Add(headerContainer)
	newDetailsContent.Add(widget.NewSeparator())

	// Sort keys for consistent display order
	keys := make([]string, 0, len(comp.Details))

	// For memory components, use a specific order
	if comp.Type == "Memory" {
		// Define the preferred order for memory fields
		preferredOrder := []string{
			"Name",
			"Number",
			"Type",
			"Manufacturer",
			"Chip manufacturer",
			"Base frequency",
			"Size",
			"Part number",
			"Serial number",
			"Slot",
			"Form Factor",
		}

		// Add keys in preferred order if they exist
		for _, key := range preferredOrder {
			if _, exists := comp.Details[key]; exists {
				keys = append(keys, key)
			}
		}

		// Add any remaining keys that weren't in the preferred order
		for k := range comp.Details {
			found := false
			for _, pk := range preferredOrder {
				if k == pk {
					found = true
					break
				}
			}
			if !found {
				keys = append(keys, k)
			}
		}
	} else {
		// For other components, use alphabetical order
		for k := range comp.Details {
			keys = append(keys, k)
		}
		sort.Strings(keys)
	}

	// Create details in table-like format
	rowIndex := 0
	for _, key := range keys {
		value := comp.Details[key]

		// Create row background (alternating colors)
		var rowBg *canvas.Rectangle
		if rowIndex%2 == 0 {
			rowBg = canvas.NewRectangle(color.RGBA{0x1a, 0x1a, 0x1a, 0xff}) // Slightly lighter
		} else {
			rowBg = canvas.NewRectangle(color.RGBA{0x11, 0x11, 0x11, 0xff}) // Match background
		}
		rowBg.Resize(fyne.NewSize(0, 30)) // Set height

		// Key label - right-aligned
		keyLabel := widget.NewLabelWithStyle(key, fyne.TextAlignTrailing, fyne.TextStyle{})

		// Value label - left-aligned
		valueLabel := widget.NewLabel(value)
		valueLabel.Alignment = fyne.TextAlignLeading
		valueLabel.Wrapping = fyne.TextWrapBreak

		// Create row
		row := container.NewStack(
			rowBg,
			container.New(&tableRowLayout{keyWidth: 160}, // Narrower key column
				keyLabel,
				valueLabel,
			),
		)

		newDetailsContent.Add(row)
		rowIndex++
	}

	// Replace the entire details grid content
	d.detailsGrid.Objects = nil
	for _, obj := range newDetailsContent.Objects {
		d.detailsGrid.Add(obj)
	}
	d.detailsGrid.Refresh()

	// Add "View Details" button for all components (shows dynamic metrics)
	d.detailsGrid.Add(widget.NewSeparator())

	// Create button text based on component type
	buttonText := "View Details"
	switch comp.Type {
	case "CPU":
		buttonText = "View CPU Metrics & Temperatures"
	case "Memory":
		buttonText = "View Memory Usage & Performance"
	case "GPU":
		buttonText = "View GPU Metrics & Performance"
	case "Storage":
		buttonText = "View SMART Data & Performance"
	case "Motherboard":
		buttonText = "View Sensors & Voltages"
	case "Fan":
		buttonText = "View Fan Speeds & Control"
	case "System":
		buttonText = "View System Statistics"
	}

	// Create a centered button container
	viewDetailsBtn := widget.NewButton(buttonText, func() {
		d.ShowComponentDetails(comp)
	})
	viewDetailsBtn.Importance = widget.HighImportance

	buttonContainer := container.NewCenter(viewDetailsBtn)
	d.detailsGrid.Add(buttonContainer)

	safeRefresh(d.detailsGrid)
}

// Content returns the dashboard content
func (d *Dashboard) Content() fyne.CanvasObject {
	return d.content
}

// SummaryStrip returns the summary strip
func (d *Dashboard) SummaryStrip() fyne.CanvasObject {
	return d.summaryStrip
}

// Start begins monitoring
func (d *Dashboard) Start() {
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return
	}
	d.running = true
	d.mu.Unlock()

	// Load storage info asynchronously after UI is shown
	go func() {
		time.Sleep(500 * time.Millisecond) // Let UI initialize first
		DebugLog("DEBUG", "Loading storage info asynchronously...")
		storageDevices, err := GetStorageInfo()
		if err == nil {
			// Update cache under lock
			d.mu.Lock()
			d.staticComponentCache.storageDevices = storageDevices
			d.storageDevices = storageDevices
			d.populateComponents()
			d.mu.Unlock()

			// Now schedule UI work on the Fyne thread
			fyne.Do(func() {
				d.RefreshComponentList()
			})

			DebugLog("DEBUG", "Storage info loaded successfully and UI refreshed")
		} else {
			DebugLog("ERROR", "Failed to load storage info: %v", err)
		}
	}()

	// Start update timer with responsive interval
	// 1 second provides good responsiveness
	d.updateTicker = time.NewTicker(1 * time.Second)

	// Start CPU metrics updater goroutine
	go d.updateCPUMetricsLoop()

	go d.monitorLoop()
}

// Stop stops monitoring
func (d *Dashboard) Stop() {
	d.mu.Lock()
	if !d.running {
		d.mu.Unlock()
		return
	}
	d.running = false
	d.mu.Unlock()

	if d.updateTicker != nil {
		d.updateTicker.Stop()
	}

	close(d.stopChan)
}

// monitorLoop is the main update loop
func (d *Dashboard) monitorLoop() {
	for {
		select {
		case <-d.updateTicker.C:
			// Check if still running before update
			d.mu.Lock()
			if !d.running {
				d.mu.Unlock()
				return
			}
			d.mu.Unlock()

			// Update metrics directly - we're already in a background goroutine
			d.updateMetrics()

			// Also update the selected component's details if any
			if d.selectedIndex >= 0 {
				fyne.Do(func() {
					d.updateDetails()
				})
			}
		case <-d.stopChan:
			return
		}
	}
}

// UpdateMetrics updates all metrics (public method)
func (d *Dashboard) UpdateMetrics() {
	d.updateMetrics()
}

// RefreshComponentList safely refreshes the component list from any goroutine
func (d *Dashboard) RefreshComponentList() {
	if d.componentList != nil {
		// Count storage devices
		storageCount := 0
		d.mu.Lock()
		for _, comp := range d.components {
			if comp.Type == "Storage" {
				storageCount++
			}
		}
		d.mu.Unlock()

		// Send notification if storage devices were loaded
		if storageCount > 0 {
			fyne.CurrentApp().SendNotification(&fyne.Notification{
				Title:   "Storage Devices Loaded",
				Content: fmt.Sprintf("Detected %d storage devices", storageCount),
			})
		}

		// Refresh the list
		d.componentList.Refresh()
	}
}

// getCachedGPUInfo returns cached GPU info if recent, otherwise fetches new data
func (d *Dashboard) getCachedGPUInfo() []GPUInfo {
	d.mu.Lock()
	defer d.mu.Unlock()

	// If data is less than 1 second old, use cached version
	if time.Since(d.lastGPUUpdate) < 1*time.Second && len(d.lastGPUInfo) > 0 {
		return d.lastGPUInfo
	}

	// Fetch new data
	d.lastGPUInfo, _ = GetGPUInfo()
	d.lastGPUUpdate = time.Now()
	return d.lastGPUInfo
}

// updateDynamicStorageMetrics updates only the dynamic metrics of storage devices
// This avoids expensive PowerShell queries for static information
func (d *Dashboard) updateDynamicStorageMetrics(devices []StorageInfo) {
	// Update usage statistics for each device
	for i := range devices {
		// Get usage stats using only the mount point (fast operation)
		usage, err := disk.Usage(devices[i].Mountpoint)
		if err == nil {
			devices[i].Used = usage.Used
			devices[i].Free = usage.Free
			devices[i].UsedPercent = usage.UsedPercent
		}

		// Note: SMART data updates could be added here if needed,
		// but they're also relatively expensive so we skip them
		// for regular updates
	}
}

// getCachedStorageInfo returns cached storage info if recent, otherwise fetches new data
func (d *Dashboard) getCachedStorageInfo() []StorageInfo {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Storage info changes rarely, cache for 30 seconds
	if time.Since(d.lastStorageUpdate) < 30*time.Second && len(d.lastStorageInfo) > 0 {
		return d.lastStorageInfo
	}

	// Use the static cache that was populated at startup
	// We only need to update dynamic metrics (usage, temperature)
	if len(d.staticComponentCache.storageDevices) > 0 {
		// Create a copy of static devices and update only dynamic fields
		d.lastStorageInfo = make([]StorageInfo, len(d.staticComponentCache.storageDevices))
		copy(d.lastStorageInfo, d.staticComponentCache.storageDevices)

		// Update only dynamic metrics (usage percentage) without expensive queries
		d.updateDynamicStorageMetrics(d.lastStorageInfo)
		d.lastStorageUpdate = time.Now()
		return d.lastStorageInfo
	}

	// Fallback: only if no static cache (shouldn't happen)
	d.lastStorageInfo, _ = GetStorageInfo()
	d.lastStorageUpdate = time.Now()
	return d.lastStorageInfo
}

// updateGPUCardTitle updates the GPU name in a GPU card
func (d *Dashboard) updateGPUCardTitle(card *SummaryCard, gpuName string) {
	// Find the title label in the card's title content
	if card != nil && card.title != nil {
		// Try as a border container first (GPU with tabs)
		if border, ok := card.title.(*fyne.Container); ok && len(border.Objects) > 0 {
			// Find the HBox with icon and label
			for _, obj := range border.Objects {
				if hbox, ok := obj.(*fyne.Container); ok && len(hbox.Objects) >= 2 {
					// Second object should be the label
					if label, ok := hbox.Objects[1].(*widget.Label); ok {
						// Truncate if needed
						displayName := gpuName
						if len(displayName) > 25 {
							displayName = displayName[:22] + "..."
						}
						label.SetText(displayName)
						label.Refresh()
						return
					}
				}
			}
		}

		// Try as HBox directly (CPU, Memory)
		if hbox, ok := card.title.(*fyne.Container); ok && len(hbox.Objects) >= 2 {
			if label, ok := hbox.Objects[1].(*widget.Label); ok {
				// Truncate if needed
				displayName := gpuName
				if len(displayName) > 25 {
					displayName = displayName[:22] + "..."
				}
				label.SetText(displayName)
				label.Refresh()
			}
		}
	}
}

// ShowComponentDetails shows a dialog with detailed dynamic metrics for a component
func (d *Dashboard) ShowComponentDetails(comp *Component) {
	// Create content based on component type
	var content fyne.CanvasObject
	title := fmt.Sprintf("%s Details - %s", comp.Type, comp.Name)

	switch comp.Type {
	case "Storage":
		// Special handling for storage - use existing storage details dialog
		if storageIndex, ok := comp.Metrics["storageIndex"]; ok {
			idx := int(storageIndex)
			if idx >= 0 && idx < len(d.staticComponentCache.storageDevices) {
				// Use cached storage info and update only dynamic metrics
				cachedStorage := d.staticComponentCache.storageDevices[idx]

				// Create a copy and update only dynamic fields
				updatedStorage := cachedStorage
				usage, err := disk.Usage(cachedStorage.Mountpoint)
				if err == nil {
					updatedStorage.Used = usage.Used
					updatedStorage.Free = usage.Free
					updatedStorage.UsedPercent = usage.UsedPercent
				}

				d.ShowStorageDetails(&updatedStorage)
				return
			}
		}
		content = d.createGenericDetailsContent(comp)
	case "Memory":
		// Special handling for memory - use CPU-Z style memory details dialog
		// Find the memory module in the cache by matching slot or part number
		for i := range d.staticComponentCache.memoryModules {
			module := &d.staticComponentCache.memoryModules[i]
			// Match by slot name or part number from component details
			if slot, ok := comp.Details["Slot"]; ok && module.Slot == slot {
				d.ShowMemoryDetails(module)
				return
			}
			// Fallback: match by part number
			if partNum, ok := comp.Details["Part Number"]; ok && module.PartNumber == partNum {
				d.ShowMemoryDetails(module)
				return
			}
		}
		// If no match found, use generic details
		content = d.createGenericDetailsContent(comp)
	default:
		content = d.createGenericDetailsContent(comp)
	}

	// Create dialog
	dlg := dialog.NewCustom(title, "Close", content, d.window)
	dlg.Resize(fyne.NewSize(600, 500))
	dlg.Show()
}

// ShowMemoryDetails shows the memory details page for a specific module
func (d *Dashboard) ShowMemoryDetails(_ *MemoryModule) {
	// Create memory details page
	memoryDetailsPage := NewMemoryDetailsPage(d.window)

	// Create dialog with memory details content
	content := memoryDetailsPage.CreateContent()

	dlg := dialog.NewCustom("Memory Module Details", "Close", content, d.window)
	dlg.Resize(fyne.NewSize(800, 600))
	dlg.Show()
}

// createGenericDetailsContent creates dynamic metrics content for any component
func (d *Dashboard) createGenericDetailsContent(comp *Component) fyne.CanvasObject {
	// Container for dynamic content
	dynamicContent := container.NewVBox()

	// Add a loading indicator
	loadingLabel := widget.NewLabelWithStyle("Loading dynamic metrics...", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
	dynamicContent.Add(loadingLabel)

	// Create scrollable container
	scroll := container.NewVScroll(dynamicContent)

	// Load dynamic metrics in background
	go func() {
		time.Sleep(100 * time.Millisecond) // Small delay for UI responsiveness

		// Get fresh metrics based on component type
		var metrics map[string]float64
		var additionalInfo map[string]string

		switch comp.Type {
		case "CPU":
			metrics, additionalInfo = d.getCPUDynamicMetrics()
		case "Memory":
			metrics, additionalInfo = d.getMemoryDynamicMetrics()
		case "GPU":
			metrics, additionalInfo = d.getGPUDynamicMetrics(comp)
		case "Motherboard":
			metrics, additionalInfo = d.getMotherboardDynamicMetrics()
		case "Fan":
			metrics, additionalInfo = d.getFanDynamicMetrics(comp)
		case "System":
			metrics, additionalInfo = d.getSystemDynamicMetrics()
		default:
			metrics = make(map[string]float64)
			additionalInfo = make(map[string]string)
		}

		// Update UI on main thread
		fyne.Do(func() {
			dynamicContent.Objects = nil // Clear loading indicator

			// Static Information Card
			staticCard := widget.NewCard("Static Information", "Hardware specifications and identifiers",
				d.createStaticInfoGrid(comp.Details),
			)
			dynamicContent.Add(staticCard)

			// Dynamic Metrics Card
			if len(metrics) > 0 {
				metricsCard := widget.NewCard("Live Metrics", "Real-time performance data",
					d.createMetricsGrid(metrics),
				)
				dynamicContent.Add(metricsCard)
			}

			// Additional Information Card
			if len(additionalInfo) > 0 {
				additionalCard := widget.NewCard("Additional Information", "",
					d.createStaticInfoGrid(additionalInfo),
				)
				dynamicContent.Add(additionalCard)
			}

			// Add auto-refresh notice
			refreshLabel := widget.NewLabelWithStyle(
				"Note: Dynamic metrics are updated in real-time in the summary bar.\nThis view shows a snapshot at the time of opening.",
				fyne.TextAlignCenter,
				fyne.TextStyle{Italic: true},
			)
			dynamicContent.Add(refreshLabel)

			dynamicContent.Refresh()
		})
	}()

	return scroll
}

// createStaticInfoGrid creates a grid layout for static information
func (d *Dashboard) createStaticInfoGrid(info map[string]string) fyne.CanvasObject {
	// Sort keys for consistent display
	keys := make([]string, 0, len(info))
	for k := range info {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	grid := container.NewGridWithColumns(2)
	for _, key := range keys {
		keyLabel := widget.NewLabelWithStyle(key+":", fyne.TextAlignTrailing, fyne.TextStyle{})
		valueLabel := widget.NewLabel(info[key])
		valueLabel.Wrapping = fyne.TextWrapBreak
		grid.Add(keyLabel)
		grid.Add(valueLabel)
	}
	return grid
}

// createMetricsGrid creates a grid layout for dynamic metrics
func (d *Dashboard) createMetricsGrid(metrics map[string]float64) fyne.CanvasObject {
	// Sort keys for consistent display
	keys := make([]string, 0, len(metrics))
	for k := range metrics {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	grid := container.NewGridWithColumns(2)
	for _, key := range keys {
		keyLabel := widget.NewLabelWithStyle(key+":", fyne.TextAlignTrailing, fyne.TextStyle{})

		// Format value based on key name
		valueStr := fmt.Sprintf("%.2f", metrics[key])
		switch {
		case strings.Contains(key, "Usage") || strings.Contains(key, "Percent"):
			valueStr = fmt.Sprintf("%.1f%%", metrics[key])
		case strings.Contains(key, "Temperature") || strings.Contains(key, "Temp"):
			valueStr = fmt.Sprintf("%.1f°C", metrics[key])
		case strings.Contains(key, "Power"):
			valueStr = fmt.Sprintf("%.1f W", metrics[key])
		case strings.Contains(key, "Voltage"):
			valueStr = fmt.Sprintf("%.3f V", metrics[key])
		case strings.Contains(key, "Frequency") || strings.Contains(key, "Clock"):
			if metrics[key] > 100 { // MHz
				valueStr = fmt.Sprintf("%.0f MHz", metrics[key])
			} else { // GHz
				valueStr = fmt.Sprintf("%.2f GHz", metrics[key])
			}
		case strings.Contains(key, "GB"):
			valueStr = fmt.Sprintf("%.2f GB", metrics[key])
		case strings.Contains(key, "MB"):
			valueStr = fmt.Sprintf("%.0f MB", metrics[key])
		case strings.Contains(key, "RPM") || strings.Contains(key, "Speed"):
			valueStr = fmt.Sprintf("%.0f RPM", metrics[key])
		}

		valueLabel := widget.NewLabelWithStyle(valueStr, fyne.TextAlignLeading, fyne.TextStyle{Monospace: true})
		grid.Add(keyLabel)
		grid.Add(valueLabel)
	}
	return grid
}

// truncateText truncates text to maxLen and adds ellipsis if needed
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	if maxLen <= 3 {
		return "..."
	}
	return text[:maxLen-3] + "..."
}

// fixedSplitLayout implements a fixed horizontal split layout
type fixedSplitLayout struct {
	leftRatio float32
}

// MinSize returns the minimum size for the fixed split layout
func (f *fixedSplitLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) != 2 {
		return fyne.NewSize(0, 0)
	}

	leftMin := objects[0].MinSize()
	rightMin := objects[1].MinSize()

	minWidth := leftMin.Width + rightMin.Width
	minHeight := fyne.Max(leftMin.Height, rightMin.Height)

	return fyne.NewSize(minWidth, minHeight)
}

// Layout arranges the objects in a fixed horizontal split
func (f *fixedSplitLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) != 2 {
		return
	}

	leftWidth := size.Width * f.leftRatio
	rightWidth := size.Width - leftWidth

	// Position and size the left panel
	objects[0].Move(fyne.NewPos(0, 0))
	objects[0].Resize(fyne.NewSize(leftWidth, size.Height))

	// Position and size the right panel
	objects[1].Move(fyne.NewPos(leftWidth, 0))
	objects[1].Resize(fyne.NewSize(rightWidth, size.Height))
}

// fixedSizeLayout implements a layout with fixed dimensions
type fixedSizeLayout struct {
	width  float32
	height float32
}

// MinSize returns the fixed size
func (f *fixedSizeLayout) MinSize(_ []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(f.width, f.height)
}

// Layout positions objects at the fixed size
func (f *fixedSizeLayout) Layout(objects []fyne.CanvasObject, _ fyne.Size) {
	for _, obj := range objects {
		obj.Move(fyne.NewPos(0, 0))
		obj.Resize(fyne.NewSize(f.width, f.height))
	}
}

// proportionalSplitLayout implements a layout that splits space by ratios
type proportionalSplitLayout struct {
	ratios []float32
}

// MinSize returns the minimum size
func (p *proportionalSplitLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(0, 0)
	}

	// Calculate minimum width based on content
	minWidth := float32(0)
	minHeight := float32(0)

	for _, obj := range objects {
		size := obj.MinSize()
		minWidth += size.Width
		if size.Height > minHeight {
			minHeight = size.Height
		}
	}

	// Ensure minimum width of at least 1400px for proper display at 1600x900
	if minWidth < 1400 {
		minWidth = 1400
	}

	return fyne.NewSize(minWidth, minHeight)
}

// Layout arranges the objects proportionally
func (p *proportionalSplitLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 || len(objects) != len(p.ratios) {
		return
	}

	x := float32(0)
	for i, obj := range objects {
		width := size.Width * p.ratios[i]
		obj.Move(fyne.NewPos(x, 0))
		obj.Resize(fyne.NewSize(width, size.Height))
		x += width
	}
}

// tableRowLayout implements a table-like row layout
type tableRowLayout struct {
	keyWidth float32
}

// MinSize returns the minimum size
func (t *tableRowLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) != 2 {
		return fyne.NewSize(0, 0)
	}
	height := fyne.Max(objects[0].MinSize().Height, objects[1].MinSize().Height)
	// Ensure minimum row height
	if height < 30 {
		height = 30
	}
	return fyne.NewSize(400, height) // Fixed width for consistent layout
}

// Layout arranges the objects in table columns
func (t *tableRowLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) != 2 {
		return
	}

	padding := float32(15) // Padding on left and right

	// Position key column with right padding
	objects[0].Move(fyne.NewPos(padding, 0))
	objects[0].Resize(fyne.NewSize(t.keyWidth-padding*2, size.Height))

	// Position value column with left padding
	objects[1].Move(fyne.NewPos(t.keyWidth+padding, 0))
	valueWidth := size.Width - t.keyWidth - padding*2
	if valueWidth < 0 {
		valueWidth = 0
	}
	objects[1].Resize(fyne.NewSize(valueWidth, size.Height))
}
