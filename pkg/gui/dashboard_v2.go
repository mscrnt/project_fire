package gui

import (
	"fmt"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/net"
)

// DashboardV2 represents the enhanced monitoring dashboard
type DashboardV2 struct {
	content fyne.CanvasObject

	// System info (static)
	sysInfo *SystemInfo

	// Update control
	running  bool
	mu       sync.Mutex
	stopChan chan bool

	// Update intervals
	cpuTicker     *time.Ticker
	memoryTicker  *time.Ticker
	gpuTicker     *time.Ticker
	diskTicker    *time.Ticker
	networkTicker *time.Ticker

	// Panels
	cpuPanel     *CPUPanel
	memoryPanel  *MemoryPanel
	gpuPanel     *GPUPanel
	storagePanel *StoragePanel
	networkPanel *NetworkPanel
	systemPanel  *SystemPanel

	// Last update timestamp
	lastUpdateLabel *widget.Label
}

// CPUPanel holds CPU monitoring widgets
type CPUPanel struct {
	card        *widget.Card
	usageLabel  *widget.Label
	speedLabel  *widget.Label
	coresLabel  *widget.Label
	tempLabel   *widget.Label
	chart       *EnhancedLineChart
	minMaxLabel *widget.Label
}

// MemoryPanel holds memory monitoring widgets
type MemoryPanel struct {
	card        *widget.Card
	usageLabel  *widget.Label
	totalLabel  *widget.Label
	availLabel  *widget.Label
	swapLabel   *widget.Label
	chart       *EnhancedLineChart
	minMaxLabel *widget.Label
}

// GPUPanel holds GPU monitoring widgets
type GPUPanel struct {
	card        *widget.Card
	nameLabel   *widget.Label
	usageLabel  *widget.Label
	memoryLabel *widget.Label
	tempLabel   *widget.Label
	powerLabel  *widget.Label
	chart       *EnhancedLineChart
	minMaxLabel *widget.Label
	gpuIndex    int
}

// StoragePanel holds storage monitoring widgets
type StoragePanel struct {
	card       *widget.Card
	volumeList *widget.List
	volumes    []VolumeInfo
	ioLabel    *widget.Label
	readChart  *EnhancedLineChart
	writeChart *EnhancedLineChart
}

// NetworkPanel holds network monitoring widgets
type NetworkPanel struct {
	card           *widget.Card
	interfaceLabel *widget.Label
	ipLabel        *widget.Label
	uploadLabel    *widget.Label
	downloadLabel  *widget.Label
	totalLabel     *widget.Label
	uploadChart    *EnhancedLineChart
	downloadChart  *EnhancedLineChart
	lastBytes      net.IOCountersStat
}

// SystemPanel holds system information
type SystemPanel struct {
	card         *widget.Card
	osLabel      *widget.Label
	kernelLabel  *widget.Label
	uptimeLabel  *widget.Label
	processLabel *widget.Label
}

// VolumeInfo holds information about a storage volume
type VolumeInfo struct {
	MountPoint  string
	Device      string
	FileSystem  string
	TotalGB     float64
	UsedGB      float64
	FreeGB      float64
	UsedPercent float64
}

// NewDashboardV2 creates a new enhanced dashboard
func NewDashboardV2() *DashboardV2 {
	d := &DashboardV2{
		stopChan: make(chan bool),
	}
	d.build()
	return d
}

// build creates the enhanced dashboard UI
func (d *DashboardV2) build() {
	// Get initial system info
	d.sysInfo, _ = GetSystemInfo()

	// Create panels
	d.cpuPanel = d.createCPUPanel()
	d.memoryPanel = d.createMemoryPanel()
	d.gpuPanel = d.createGPUPanel()
	d.storagePanel = d.createStoragePanel()
	d.networkPanel = d.createNetworkPanel()
	d.systemPanel = d.createSystemPanel()

	// Create last update label
	d.lastUpdateLabel = widget.NewLabel("Last updated: -")
	d.lastUpdateLabel.TextStyle = fyne.TextStyle{Italic: true}
	d.lastUpdateLabel.Alignment = fyne.TextAlignTrailing

	// Create header
	header := container.NewBorder(
		nil, nil,
		widget.NewLabelWithStyle("F.I.R.E. System Monitor", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		d.lastUpdateLabel,
	)

	// Create main grid with 3 columns
	mainGrid := container.New(layout.NewAdaptiveGridLayout(3),
		container.NewPadded(d.cpuPanel.card),
		container.NewPadded(d.memoryPanel.card),
		container.NewPadded(d.gpuPanel.card),
		container.NewPadded(d.storagePanel.card),
		container.NewPadded(d.networkPanel.card),
		container.NewPadded(d.systemPanel.card),
	)

	// Wrap in scroll container for better handling on small screens
	scrollable := container.NewVScroll(mainGrid)
	scrollable.SetMinSize(fyne.NewSize(1200, 800))

	// Main layout with padding
	d.content = container.NewBorder(
		container.NewPadded(header),
		nil, nil, nil,
		scrollable,
	)

	// Initial update
	d.updateAll()
}

// createCPUPanel creates the CPU monitoring panel
func (d *DashboardV2) createCPUPanel() *CPUPanel {
	p := &CPUPanel{
		usageLabel:  widget.NewLabelWithStyle("0%", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		speedLabel:  widget.NewLabel("Speed: -"),
		coresLabel:  widget.NewLabel("Cores: -"),
		tempLabel:   widget.NewLabel("Temp: -"),
		chart:       NewEnhancedLineChart("CPU Usage", 60, 100),
		minMaxLabel: widget.NewLabel("Min: 0% | Max: 0%"),
	}

	// Update with system info
	if d.sysInfo != nil {
		p.coresLabel.SetText(fmt.Sprintf("Cores: %d / %d threads",
			d.sysInfo.CPU.PhysicalCores, d.sysInfo.CPU.LogicalCores))
	}

	content := container.NewVBox(
		container.NewHBox(
			p.usageLabel,
			layout.NewSpacer(),
			p.tempLabel,
		),
		p.speedLabel,
		p.coresLabel,
		widget.NewSeparator(),
		p.chart,
		container.NewCenter(p.minMaxLabel),
	)

	title := "🔥 CPU"
	if d.sysInfo != nil && d.sysInfo.CPU.Model != "" {
		title = fmt.Sprintf("🔥 CPU - %s", truncateString(d.sysInfo.CPU.Model, 30))
	}

	p.card = widget.NewCard(title, "", content)
	return p
}

// createMemoryPanel creates the memory monitoring panel
func (d *DashboardV2) createMemoryPanel() *MemoryPanel {
	p := &MemoryPanel{
		usageLabel:  widget.NewLabelWithStyle("0%", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		totalLabel:  widget.NewLabel("Total: -"),
		availLabel:  widget.NewLabel("Available: -"),
		swapLabel:   widget.NewLabel("Swap: -"),
		chart:       NewEnhancedLineChart("Memory Usage", 60, 100),
		minMaxLabel: widget.NewLabel("Min: 0% | Max: 0%"),
	}

	// Show host memory if in WSL
	if d.sysInfo != nil && d.sysInfo.Host.IsWSL && d.sysInfo.Memory.HostTotalGB > d.sysInfo.Memory.TotalGB {
		p.totalLabel.SetText(fmt.Sprintf("Total: %.1f GB (Host: %.0f GB)",
			d.sysInfo.Memory.TotalGB, d.sysInfo.Memory.HostTotalGB))
	}

	content := container.NewVBox(
		container.NewHBox(
			p.usageLabel,
			layout.NewSpacer(),
			p.availLabel,
		),
		p.totalLabel,
		p.swapLabel,
		widget.NewSeparator(),
		p.chart,
		container.NewCenter(p.minMaxLabel),
	)

	p.card = widget.NewCard("💾 Memory", "", content)
	return p
}

// createGPUPanel creates the GPU monitoring panel
func (d *DashboardV2) createGPUPanel() *GPUPanel {
	p := &GPUPanel{
		nameLabel:   widget.NewLabel("No GPU detected"),
		usageLabel:  widget.NewLabelWithStyle("0%", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		memoryLabel: widget.NewLabel("Memory: -"),
		tempLabel:   widget.NewLabel("Temp: -"),
		powerLabel:  widget.NewLabel("Power: -"),
		chart:       NewEnhancedLineChart("GPU Usage", 60, 100),
		minMaxLabel: widget.NewLabel("Min: 0% | Max: 0%"),
		gpuIndex:    0,
	}

	content := container.NewVBox(
		p.nameLabel,
		container.NewHBox(
			p.usageLabel,
			layout.NewSpacer(),
			p.tempLabel,
		),
		p.memoryLabel,
		p.powerLabel,
		widget.NewSeparator(),
		p.chart,
		container.NewCenter(p.minMaxLabel),
	)

	p.card = widget.NewCard("🎮 GPU", "", content)
	return p
}

// createStoragePanel creates the storage monitoring panel
func (d *DashboardV2) createStoragePanel() *StoragePanel {
	p := &StoragePanel{
		volumes:    make([]VolumeInfo, 0),
		ioLabel:    widget.NewLabel("I/O: Read 0 MB/s | Write 0 MB/s"),
		readChart:  NewEnhancedLineChart("Read", 30, 100),
		writeChart: NewEnhancedLineChart("Write", 30, 100),
	}

	// Create volume list
	p.volumeList = widget.NewList(
		func() int { return len(p.volumes) },
		func() fyne.CanvasObject {
			return container.NewVBox(
				widget.NewLabel(""),
				widget.NewProgressBar(),
			)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if i >= len(p.volumes) {
				return
			}
			vol := p.volumes[i]
			vbox := o.(*fyne.Container)

			label := vbox.Objects[0].(*widget.Label)
			label.SetText(fmt.Sprintf("%s (%.1f/%.1f GB)",
				vol.MountPoint, vol.UsedGB, vol.TotalGB))

			progress := vbox.Objects[1].(*widget.ProgressBar)
			progress.SetValue(vol.UsedPercent / 100)
		},
	)

	// Set preferred size for volume list
	p.volumeList.Resize(fyne.NewSize(300, 200))

	content := container.NewVBox(
		container.NewMax(p.volumeList),
		widget.NewSeparator(),
		p.ioLabel,
		container.NewGridWithColumns(2,
			p.readChart,
			p.writeChart,
		),
	)

	p.card = widget.NewCard("💿 Storage", "", content)
	return p
}

// createNetworkPanel creates the network monitoring panel
func (d *DashboardV2) createNetworkPanel() *NetworkPanel {
	p := &NetworkPanel{
		interfaceLabel: widget.NewLabel("Interface: -"),
		ipLabel:        widget.NewLabel("IP: -"),
		uploadLabel:    widget.NewLabelWithStyle("↑ 0 MB/s", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		downloadLabel:  widget.NewLabelWithStyle("↓ 0 MB/s", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		totalLabel:     widget.NewLabel("Total: -"),
		uploadChart:    NewEnhancedLineChart("Upload", 30, 10),
		downloadChart:  NewEnhancedLineChart("Download", 30, 10),
	}

	content := container.NewVBox(
		p.interfaceLabel,
		p.ipLabel,
		container.NewHBox(
			p.uploadLabel,
			layout.NewSpacer(),
			p.downloadLabel,
		),
		p.totalLabel,
		widget.NewSeparator(),
		container.NewGridWithColumns(2,
			p.uploadChart,
			p.downloadChart,
		),
	)

	p.card = widget.NewCard("🌐 Network", "", content)
	return p
}

// createSystemPanel creates the system information panel
func (d *DashboardV2) createSystemPanel() *SystemPanel {
	p := &SystemPanel{
		osLabel:      widget.NewLabel("OS: -"),
		kernelLabel:  widget.NewLabel("Kernel: -"),
		uptimeLabel:  widget.NewLabel("Uptime: -"),
		processLabel: widget.NewLabel("Processes: -"),
	}

	// Update with system info
	if d.sysInfo != nil {
		p.osLabel.SetText(fmt.Sprintf("OS: %s %s", d.sysInfo.Host.Platform, d.sysInfo.Host.PlatformVersion))
		p.kernelLabel.SetText(fmt.Sprintf("Kernel: %s", truncateString(d.sysInfo.Host.KernelVersion, 40)))
		if d.sysInfo.Host.IsWSL {
			p.osLabel.SetText(p.osLabel.Text + " (WSL)")
		}
	}

	content := container.NewVBox(
		p.osLabel,
		p.kernelLabel,
		widget.NewSeparator(),
		p.uptimeLabel,
		p.processLabel,
		layout.NewSpacer(),
		widget.NewButton("System Details", func() {
			// TODO: Show detailed system information dialog
		}),
	)

	hostname := "System"
	if d.sysInfo != nil {
		hostname = d.sysInfo.Host.Hostname
	}

	p.card = widget.NewCard(fmt.Sprintf("🖥️ %s", hostname), "", content)
	return p
}

// Content returns the dashboard content
func (d *DashboardV2) Content() fyne.CanvasObject {
	return d.content
}

// Start begins monitoring with staggered update intervals
func (d *DashboardV2) Start() {
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return
	}
	d.running = true
	d.mu.Unlock()

	// Start update timers with different intervals
	d.cpuTicker = time.NewTicker(1 * time.Second)
	d.memoryTicker = time.NewTicker(2 * time.Second)
	d.gpuTicker = time.NewTicker(2 * time.Second)
	d.diskTicker = time.NewTicker(5 * time.Second)
	d.networkTicker = time.NewTicker(1 * time.Second)

	// Start monitoring goroutines
	go d.monitorCPU()
	go d.monitorMemory()
	go d.monitorGPU()
	go d.monitorStorage()
	go d.monitorNetwork()
	go d.monitorSystem()
}

// Stop stops monitoring
func (d *DashboardV2) Stop() {
	d.mu.Lock()
	if !d.running {
		d.mu.Unlock()
		return
	}
	d.running = false
	d.mu.Unlock()

	// Stop all tickers
	if d.cpuTicker != nil {
		d.cpuTicker.Stop()
	}
	if d.memoryTicker != nil {
		d.memoryTicker.Stop()
	}
	if d.gpuTicker != nil {
		d.gpuTicker.Stop()
	}
	if d.diskTicker != nil {
		d.diskTicker.Stop()
	}
	if d.networkTicker != nil {
		d.networkTicker.Stop()
	}

	close(d.stopChan)
}

// updateAll performs initial update of all panels
func (d *DashboardV2) updateAll() {
	d.updateCPU()
	d.updateMemory()
	d.updateGPU()
	var lastDiskIO map[string]disk.IOCountersStat
	d.updateStorage(&lastDiskIO)
	d.updateNetwork()
	d.updateSystem()
	d.updateTimestamp()
}

// updateTimestamp updates the last update timestamp
func (d *DashboardV2) updateTimestamp() {
	timestamp := time.Now().Format("15:04:05")
	if app := fyne.CurrentApp(); app != nil && app.Driver() != nil {
		app.Driver().DoFromGoroutine(func() {
			d.lastUpdateLabel.SetText(fmt.Sprintf("Last updated: %s", timestamp))
		}, false)
	}
}

// Helper function to truncate strings
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
