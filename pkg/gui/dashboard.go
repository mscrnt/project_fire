package gui

import (
	"fmt"
	"image/color"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/net"
)

// Dashboard represents the live monitoring dashboard
type Dashboard struct {
	content   fyne.CanvasObject
	
	// Labels for real-time values
	cpuLabel    *widget.Label
	memLabel    *widget.Label
	diskLabel   *widget.Label
	netLabel    *widget.Label
	
	// Charts
	cpuChart    *LineChart
	memChart    *LineChart
	
	// Control
	running     bool
	mu          sync.Mutex
	stopChan    chan bool
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
	// Create labels
	d.cpuLabel = widget.NewLabel("CPU: --%")
	d.cpuLabel.TextStyle = fyne.TextStyle{Bold: true}
	
	d.memLabel = widget.NewLabel("Memory: --%")
	d.memLabel.TextStyle = fyne.TextStyle{Bold: true}
	
	d.diskLabel = widget.NewLabel("Disk: --")
	d.diskLabel.TextStyle = fyne.TextStyle{Bold: true}
	
	d.netLabel = widget.NewLabel("Network: --")
	d.netLabel.TextStyle = fyne.TextStyle{Bold: true}
	
	// Create charts
	d.cpuChart = NewLineChart("CPU Usage", 60, 100)
	d.memChart = NewLineChart("Memory Usage", 60, 100)
	
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
	
	diskCard := widget.NewCard("Disk I/O", "", container.NewVBox(
		d.diskLabel,
		widget.NewLabel(""),
		widget.NewLabel("Read: 0 MB/s"),
		widget.NewLabel("Write: 0 MB/s"),
	))
	
	netCard := widget.NewCard("Network", "", container.NewVBox(
		d.netLabel,
		widget.NewLabel(""),
		widget.NewLabel("Upload: 0 MB/s"),
		widget.NewLabel("Download: 0 MB/s"),
	))
	
	// Layout
	d.content = container.NewGridWithColumns(2,
		cpuCard,
		memCard,
		diskCard,
		netCard,
	)
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
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			d.updateStats()
		case <-d.stopChan:
			return
		}
	}
}

// updateStats updates all statistics
func (d *Dashboard) updateStats() {
	// CPU
	if cpuPercent, err := cpu.Percent(0, false); err == nil && len(cpuPercent) > 0 {
		d.cpuLabel.SetText(fmt.Sprintf("CPU: %.1f%%", cpuPercent[0]))
		d.cpuChart.AddValue(cpuPercent[0])
	}
	
	// Memory
	if vmStat, err := mem.VirtualMemory(); err == nil {
		d.memLabel.SetText(fmt.Sprintf("Memory: %.1f%% (%s / %s)", 
			vmStat.UsedPercent,
			formatBytes(vmStat.Used),
			formatBytes(vmStat.Total)))
		d.memChart.AddValue(vmStat.UsedPercent)
	}
	
	// Disk
	if partitions, err := disk.Partitions(false); err == nil && len(partitions) > 0 {
		if usage, err := disk.Usage(partitions[0].Mountpoint); err == nil {
			d.diskLabel.SetText(fmt.Sprintf("Disk: %.1f%% (%s / %s)",
				usage.UsedPercent,
				formatBytes(usage.Used),
				formatBytes(usage.Total)))
		}
	}
	
	// Network
	if interfaces, err := net.IOCounters(false); err == nil && len(interfaces) > 0 {
		d.netLabel.SetText(fmt.Sprintf("Network: %s sent, %s recv",
			formatBytes(interfaces[0].BytesSent),
			formatBytes(interfaces[0].BytesRecv)))
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