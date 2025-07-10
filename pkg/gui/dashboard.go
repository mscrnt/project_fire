package gui

import (
	"fmt"
	"image/color"
	"runtime"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

// Dashboard represents the live monitoring dashboard
type Dashboard struct {
	content fyne.CanvasObject

	// System info (static)
	sysInfoLabel *widget.Label
	cpuInfoLabel *widget.Label
	memInfoLabel *widget.Label

	// Labels for real-time values
	cpuLabel  *widget.Label
	memLabel  *widget.Label
	diskLabel *widget.Label
	netLabel  *widget.Label

	// Disk list
	diskList *widget.List
	disks    []DiskInfo

	// Charts
	cpuChart *LineChart
	memChart *LineChart

	// Control
	running  bool
	mu       sync.Mutex
	stopChan chan bool

	// Cached system info
	sysInfo *SystemInfo
}

// NewDashboard creates a new dashboard
func NewDashboard() *Dashboard {
	d := &Dashboard{
		stopChan: make(chan bool),
	}
	d.build()
	return d
}

// build creates the dashboard UI
func (d *Dashboard) build() {
	// Get initial system info
	d.sysInfo, _ = GetSystemInfo()

	// Create system info labels
	d.sysInfoLabel = widget.NewLabel("System: Loading...")
	d.cpuInfoLabel = widget.NewLabel("CPU: Loading...")
	d.memInfoLabel = widget.NewLabel("Memory: Loading...")

	// Update system info labels
	if d.sysInfo != nil {
		d.sysInfoLabel.SetText(fmt.Sprintf("System: %s (%s %s)",
			d.sysInfo.Host.Hostname, d.sysInfo.Host.Platform, d.sysInfo.Host.PlatformVersion))
		d.cpuInfoLabel.SetText(fmt.Sprintf("CPU: %s", d.sysInfo.CPU.FormatCPUInfo()))
		d.memInfoLabel.SetText(fmt.Sprintf("Memory: %.1f GB installed", d.sysInfo.Memory.HostTotalGB))
	}

	// Create labels for real-time values
	d.cpuLabel = widget.NewLabel("Usage: --%")
	d.cpuLabel.TextStyle = fyne.TextStyle{Bold: true}

	d.memLabel = widget.NewLabel("Used: --%")
	d.memLabel.TextStyle = fyne.TextStyle{Bold: true}

	d.diskLabel = widget.NewLabel("Disks: --")
	d.diskLabel.TextStyle = fyne.TextStyle{Bold: true}

	d.netLabel = widget.NewLabel("Network: --")
	d.netLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Create disk list
	if d.sysInfo != nil {
		d.disks = d.sysInfo.Disks
	}
	d.diskList = widget.NewList(
		func() int { return len(d.disks) },
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			label := o.(*widget.Label)
			if i < len(d.disks) {
				disk := d.disks[i]
				label.SetText(fmt.Sprintf("%s: %.1f/%.1f GB (%.1f%%)",
					disk.MountPoint, disk.UsedGB, disk.TotalGB, disk.UsedPercent))
			}
		},
	)

	// Create charts
	d.cpuChart = NewLineChart("CPU Usage", 60, 100)
	d.memChart = NewLineChart("Memory Usage", 60, 100)

	// Create system info card
	sysCard := widget.NewCard("System Information", "", container.NewVBox(
		d.sysInfoLabel,
		d.cpuInfoLabel,
		d.memInfoLabel,
	))

	// Create info cards
	cpuCard := widget.NewCard("CPU", "", container.NewVBox(
		d.cpuLabel,
		widget.NewLabel(""),
		d.cpuChart,
	))

	memCard := widget.NewCard("Memory", "", container.NewVBox(
		d.memLabel,
		widget.NewLabel(""),
		d.memChart,
	))

	diskCard := widget.NewCard("Storage", "", container.NewVBox(
		d.diskLabel,
		container.NewScroll(d.diskList),
	))
	diskCard.Resize(fyne.NewSize(400, 300))

	netCard := widget.NewCard("Network", "", container.NewVBox(
		d.netLabel,
		widget.NewLabel(""),
		widget.NewLabel("Upload: 0 MB/s"),
		widget.NewLabel("Download: 0 MB/s"),
	))

	// Layout with system info at top
	topSection := container.NewVBox(
		sysCard,
		container.NewGridWithColumns(2,
			cpuCard,
			memCard,
		),
	)

	bottomSection := container.NewGridWithColumns(2,
		diskCard,
		netCard,
	)

	d.content = container.NewVBox(
		topSection,
		bottomSection,
	)

	// Update initial values
	d.updateStats()
}

// Content returns the dashboard content
func (d *Dashboard) Content() fyne.CanvasObject {
	return d.content
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

	go d.monitor()
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

	d.stopChan <- true
}

// Refresh refreshes the dashboard
func (d *Dashboard) Refresh() {
	// Update current values
	d.updateStats()
}

// monitor runs the monitoring loop
func (d *Dashboard) monitor() {
	// Use a timer that can be refreshed on the main thread
	timer := time.NewTimer(1 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			d.updateStats()
			timer.Reset(1 * time.Second)
		case <-d.stopChan:
			return
		}
	}
}

// updateStats updates all statistics
func (d *Dashboard) updateStats() {
	// Gather data off-thread
	cpuPercent, _ := cpu.Percent(0, false)
	vmStat, _ := mem.VirtualMemory()
	netIO, _ := net.IOCounters(false)

	// Update system info periodically (less frequently)
	if d.sysInfo == nil || time.Now().Unix()%60 == 0 {
		if newInfo, err := GetSystemInfo(); err == nil {
			d.sysInfo = newInfo
			d.disks = newInfo.Disks
		}
	}

	// Schedule UI updates on the main thread using Fyne v2.6's DoFromGoroutine
	if app := fyne.CurrentApp(); app != nil && app.Driver() != nil {
		app.Driver().DoFromGoroutine(func() {
			// CPU
			if len(cpuPercent) > 0 {
				d.cpuLabel.SetText(fmt.Sprintf("Usage: %.1f%%", cpuPercent[0]))
				d.cpuChart.AddValue(cpuPercent[0])
			}

			// Memory
			if vmStat != nil && d.sysInfo != nil {
				usedGB := float64(vmStat.Used) / 1024 / 1024 / 1024
				totalGB := float64(vmStat.Total) / 1024 / 1024 / 1024

				// Show WSL memory vs host memory if different
				if d.sysInfo.Memory.HostTotalGB > totalGB {
					d.memLabel.SetText(fmt.Sprintf("Used: %.1f / %.1f GB (%.1f%%) of WSL allocation",
						usedGB, totalGB, vmStat.UsedPercent))
				} else {
					d.memLabel.SetText(fmt.Sprintf("Used: %.1f / %.1f GB (%.1f%%)",
						usedGB, totalGB, vmStat.UsedPercent))
				}
				d.memChart.AddValue(vmStat.UsedPercent)
			}

			// Disk summary
			if len(d.disks) > 0 {
				d.diskLabel.SetText(fmt.Sprintf("Disks: %d mounted", len(d.disks)))
				// Refresh disk list
				d.diskList.Refresh()
			}

			// Network
			if len(netIO) > 0 {
				d.netLabel.SetText(fmt.Sprintf("Network: %s sent, %s recv",
					formatBytes(netIO[0].BytesSent),
					formatBytes(netIO[0].BytesRecv)))
			}
		}, false) // false means don't wait for completion
	}
}

// LineChart is a simple line chart widget
type LineChart struct {
	widget.BaseWidget
	title    string
	values   []float64
	maxValue float64
	capacity int
	mu       sync.Mutex
}

// NewLineChart creates a new line chart
func NewLineChart(title string, capacity int, maxValue float64) *LineChart {
	c := &LineChart{
		title:    title,
		values:   make([]float64, 0, capacity),
		maxValue: maxValue,
		capacity: capacity,
	}
	c.ExtendBaseWidget(c)
	return c
}

// AddValue adds a value to the chart
func (c *LineChart) AddValue(value float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.values = append(c.values, value)
	if len(c.values) > c.capacity {
		c.values = c.values[1:]
	}
	c.Refresh()
}

// CreateRenderer creates the chart renderer
func (c *LineChart) CreateRenderer() fyne.WidgetRenderer {
	return &lineChartRenderer{
		chart: c,
	}
}

// MinSize returns the minimum size
func (c *LineChart) MinSize() fyne.Size {
	return fyne.NewSize(300, 150)
}

// lineChartRenderer renders the line chart
type lineChartRenderer struct {
	chart *LineChart
}

func (r *lineChartRenderer) MinSize() fyne.Size {
	return r.chart.MinSize()
}

func (r *lineChartRenderer) Layout(size fyne.Size) {
	// No layout needed
}

func (r *lineChartRenderer) Refresh() {
	// Refresh handled by Objects()
}

func (r *lineChartRenderer) Objects() []fyne.CanvasObject {
	r.chart.mu.Lock()
	defer r.chart.mu.Unlock()

	objects := []fyne.CanvasObject{}

	// Background
	bg := canvas.NewRectangle(color.RGBA{240, 240, 240, 255})
	bg.Resize(r.chart.MinSize())
	objects = append(objects, bg)

	// Border
	border := canvas.NewRectangle(color.Transparent)
	border.StrokeColor = color.RGBA{200, 200, 200, 255}
	border.StrokeWidth = 1
	border.Resize(r.chart.MinSize())
	objects = append(objects, border)

	// Draw lines
	if len(r.chart.values) > 1 {
		width := r.chart.MinSize().Width
		height := r.chart.MinSize().Height

		for i := 1; i < len(r.chart.values); i++ {
			x1 := width * float32(i-1) / float32(r.chart.capacity)
			y1 := height - (height * float32(r.chart.values[i-1]) / float32(r.chart.maxValue))
			x2 := width * float32(i) / float32(r.chart.capacity)
			y2 := height - (height * float32(r.chart.values[i]) / float32(r.chart.maxValue))

			line := canvas.NewLine(color.RGBA{66, 165, 245, 255})
			line.StrokeWidth = 2
			line.Position1 = fyne.NewPos(x1, y1)
			line.Position2 = fyne.NewPos(x2, y2)
			objects = append(objects, line)
		}
	}

	return objects
}

func (r *lineChartRenderer) Destroy() {
	// Nothing to destroy
}

// SystemInfo holds detailed system information
type SystemInfo struct {
	CPU    CPUInfo
	Memory MemoryInfo
	Disks  []DiskInfo
	Host   HostInfo
}

// CPUInfo holds CPU details
type CPUInfo struct {
	Brand         string
	Model         string
	PhysicalCores int
	LogicalCores  int
	MaxSpeed      float64 // MHz
	CurrentSpeed  float64 // MHz
}

// MemoryInfo holds memory details
type MemoryInfo struct {
	TotalGB     float64
	AvailableGB float64
	UsedGB      float64
	UsedPercent float64

	// WSL specific - try to detect host memory
	HostTotalGB float64
}

// DiskInfo holds disk details
type DiskInfo struct {
	Device      string
	MountPoint  string
	FileSystem  string
	TotalGB     float64
	UsedGB      float64
	FreeGB      float64
	UsedPercent float64
}

// HostInfo holds host system details
type HostInfo struct {
	Hostname        string
	OS              string
	Platform        string
	PlatformVersion string
	KernelVersion   string
	Architecture    string
	IsWSL           bool
}

// GetSystemInfo gathers comprehensive system information
func GetSystemInfo() (*SystemInfo, error) {
	info := &SystemInfo{}

	// Get CPU info
	if cpuInfos, err := cpu.Info(); err == nil && len(cpuInfos) > 0 {
		// Use first CPU info for brand/model
		info.CPU.Brand = cpuInfos[0].VendorID
		info.CPU.Model = cpuInfos[0].ModelName
		info.CPU.MaxSpeed = cpuInfos[0].Mhz
	}

	// Get CPU counts
	if logical, err := cpu.Counts(true); err == nil {
		info.CPU.LogicalCores = logical
	}
	if physical, err := cpu.Counts(false); err == nil {
		info.CPU.PhysicalCores = physical
	}

	// Get current CPU speed
	if percents, err := cpu.Percent(0, true); err == nil && len(percents) > 0 {
		// Estimate current speed based on usage (simplified)
		info.CPU.CurrentSpeed = info.CPU.MaxSpeed
	}

	// Get memory info
	if vmStat, err := mem.VirtualMemory(); err == nil {
		info.Memory.TotalGB = float64(vmStat.Total) / 1024 / 1024 / 1024
		info.Memory.AvailableGB = float64(vmStat.Available) / 1024 / 1024 / 1024
		info.Memory.UsedGB = float64(vmStat.Used) / 1024 / 1024 / 1024
		info.Memory.UsedPercent = vmStat.UsedPercent

		// Try to detect host memory in WSL
		info.Memory.HostTotalGB = detectHostMemory(info.Memory.TotalGB)
	}

	// Get host info
	if hostInfo, err := host.Info(); err == nil {
		info.Host.Hostname = hostInfo.Hostname
		info.Host.OS = hostInfo.OS
		info.Host.Platform = hostInfo.Platform
		info.Host.PlatformVersion = hostInfo.PlatformVersion
		info.Host.KernelVersion = hostInfo.KernelVersion
		info.Host.Architecture = runtime.GOARCH
		info.Host.IsWSL = strings.Contains(strings.ToLower(hostInfo.KernelVersion), "microsoft")
	}

	// Get disk info - filter to relevant partitions only
	if partitions, err := disk.Partitions(true); err == nil {
		for _, partition := range partitions {
			// Skip irrelevant partitions
			if shouldSkipPartition(partition) {
				continue
			}

			usage, err := disk.Usage(partition.Mountpoint)
			if err != nil {
				continue
			}

			// Skip tiny partitions (< 1GB)
			if usage.Total < 1024*1024*1024 {
				continue
			}

			diskInfo := DiskInfo{
				Device:      partition.Device,
				MountPoint:  partition.Mountpoint,
				FileSystem:  partition.Fstype,
				TotalGB:     float64(usage.Total) / 1024 / 1024 / 1024,
				UsedGB:      float64(usage.Used) / 1024 / 1024 / 1024,
				FreeGB:      float64(usage.Free) / 1024 / 1024 / 1024,
				UsedPercent: usage.UsedPercent,
			}

			info.Disks = append(info.Disks, diskInfo)
		}
	}

	return info, nil
}

// shouldSkipPartition determines if a partition should be skipped
func shouldSkipPartition(p disk.PartitionStat) bool {
	// Skip system/special filesystems
	skipFS := []string{"tmpfs", "devtmpfs", "sysfs", "proc", "devpts", "cgroup2",
		"debugfs", "mqueue", "hugetlbfs", "tracefs", "fusectl", "configfs",
		"binfmt_misc", "fuse.snapfuse", "nsfs", "overlay", "rootfs", "iso9660",
		"fuse.gvfsd-fuse", "fuse.portal"}

	for _, fs := range skipFS {
		if p.Fstype == fs {
			return true
		}
	}

	// Skip Docker bind mounts
	if strings.Contains(p.Mountpoint, "docker-desktop-bind-mounts") {
		return true
	}

	// Skip snap mounts
	if strings.HasPrefix(p.Mountpoint, "/snap/") {
		return true
	}

	// Skip WSL system mounts
	skipMounts := []string{"/init", "/sys", "/proc", "/dev", "/run", "/tmp/.X11-unix",
		"/usr/lib/wsl", "/mnt/wslg", "/Docker/host"}
	for _, mount := range skipMounts {
		if strings.HasPrefix(p.Mountpoint, mount) {
			return true
		}
	}

	return false
}

// detectHostMemory tries to detect actual host memory in WSL
func detectHostMemory(wslMemory float64) float64 {
	// WSL2 typically limits memory to 50% or 80% of host by default
	// If we see ~30GB, it's likely a 64GB system
	// If we see ~8GB, it's likely a 16GB system
	// This is a heuristic approach

	if wslMemory > 28 && wslMemory < 34 {
		return 64.0
	} else if wslMemory > 14 && wslMemory < 18 {
		return 32.0
	} else if wslMemory > 6 && wslMemory < 10 {
		return 16.0
	} else if wslMemory > 3 && wslMemory < 5 {
		return 8.0
	}

	// Otherwise, assume WSL has 50% of host memory
	return wslMemory * 2
}

// FormatCPUInfo returns a formatted string of CPU information
func (c *CPUInfo) FormatCPUInfo() string {
	return fmt.Sprintf("%s (%d cores / %d threads) @ %.2f GHz",
		strings.TrimSpace(c.Model),
		c.PhysicalCores,
		c.LogicalCores,
		c.MaxSpeed/1000)
}

// FormatMemoryInfo returns a formatted string of memory information
func (m *MemoryInfo) FormatMemoryInfo() string {
	if m.HostTotalGB > m.TotalGB {
		return fmt.Sprintf("%.1f GB / %.1f GB (%.1f%%) - Host: %.0f GB",
			m.UsedGB, m.TotalGB, m.UsedPercent, m.HostTotalGB)
	}
	return fmt.Sprintf("%.1f GB / %.1f GB (%.1f%%)",
		m.UsedGB, m.TotalGB, m.UsedPercent)
}
