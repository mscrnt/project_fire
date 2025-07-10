package gui

import (
	"fmt"
	"image/color"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// EnhancedLineChart is an improved line chart with gridlines, data points, and min/max tracking
type EnhancedLineChart struct {
	widget.BaseWidget
	title    string
	values   []float64
	maxValue float64
	capacity int
	mu       sync.Mutex

	// Track min/max
	minSeen float64
	maxSeen float64

	// Style options
	showGrid       bool
	showDataPoints bool
	lineColor      color.Color
	gridColor      color.Color
	pointColor     color.Color
}

// NewEnhancedLineChart creates a new enhanced line chart
func NewEnhancedLineChart(title string, capacity int, maxValue float64) *EnhancedLineChart {
	c := &EnhancedLineChart{
		title:          title,
		values:         make([]float64, 0, capacity),
		maxValue:       maxValue,
		capacity:       capacity,
		minSeen:        maxValue,
		maxSeen:        0,
		showGrid:       true,
		showDataPoints: true,
		lineColor:      ChartLineColor(),
		gridColor:      ChartGridColor(),
		pointColor:     ChartLineColor(),
	}
	c.ExtendBaseWidget(c)
	return c
}

// AddValue adds a value to the chart
func (c *EnhancedLineChart) AddValue(value float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.values = append(c.values, value)
	if len(c.values) > c.capacity {
		c.values = c.values[1:]
	}

	// Update min/max
	if value < c.minSeen {
		c.minSeen = value
	}
	if value > c.maxSeen {
		c.maxSeen = value
	}

	c.Refresh()
}

// GetMinMax returns the minimum and maximum values seen
func (c *EnhancedLineChart) GetMinMax() (float64, float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.minSeen, c.maxSeen
}

// SetShowGrid enables/disables grid lines
func (c *EnhancedLineChart) SetShowGrid(show bool) {
	c.mu.Lock()
	c.showGrid = show
	c.mu.Unlock()
	c.Refresh()
}

// SetShowDataPoints enables/disables data point markers
func (c *EnhancedLineChart) SetShowDataPoints(show bool) {
	c.mu.Lock()
	c.showDataPoints = show
	c.mu.Unlock()
	c.Refresh()
}

// CreateRenderer creates the chart renderer
func (c *EnhancedLineChart) CreateRenderer() fyne.WidgetRenderer {
	return &enhancedChartRenderer{
		chart: c,
	}
}

// MinSize returns the minimum size
func (c *EnhancedLineChart) MinSize() fyne.Size {
	return fyne.NewSize(300, 120)
}

// enhancedChartRenderer renders the enhanced line chart
type enhancedChartRenderer struct {
	chart   *EnhancedLineChart
	objects []fyne.CanvasObject
}

func (r *enhancedChartRenderer) MinSize() fyne.Size {
	return r.chart.MinSize()
}

func (r *enhancedChartRenderer) Layout(size fyne.Size) {
	// Layout is handled in Objects()
}

func (r *enhancedChartRenderer) Refresh() {
	r.objects = r.render()
}

func (r *enhancedChartRenderer) Objects() []fyne.CanvasObject {
	if r.objects == nil {
		r.objects = r.render()
	}
	return r.objects
}

func (r *enhancedChartRenderer) render() []fyne.CanvasObject {
	r.chart.mu.Lock()
	defer r.chart.mu.Unlock()

	objects := []fyne.CanvasObject{}
	size := r.chart.MinSize()

	// Background with subtle gradient effect
	bg := canvas.NewRectangle(CardBackgroundColor())
	bg.Resize(size)
	objects = append(objects, bg)

	// Border
	border := canvas.NewRectangle(color.Transparent)
	border.StrokeColor = theme.SeparatorColor()
	border.StrokeWidth = 1
	border.Resize(size)
	objects = append(objects, border)

	// Chart area (with padding)
	padding := float32(10)
	chartWidth := size.Width - 2*padding
	chartHeight := size.Height - 2*padding - 20 // Extra space for title

	// Title
	if r.chart.title != "" {
		titleLabel := canvas.NewText(r.chart.title, theme.ForegroundColor())
		titleLabel.TextSize = 10
		titleLabel.Move(fyne.NewPos(padding, 2))
		objects = append(objects, titleLabel)
	}

	// Grid lines
	if r.chart.showGrid {
		// Horizontal grid lines (25%, 50%, 75%)
		for i := 1; i <= 3; i++ {
			y := padding + 20 + chartHeight*float32(i)/4
			line := canvas.NewLine(r.chart.gridColor)
			line.StrokeWidth = 1
			line.Position1 = fyne.NewPos(padding, y)
			line.Position2 = fyne.NewPos(padding+chartWidth, y)
			objects = append(objects, line)

			// Grid labels
			percent := 100 - (i * 25)
			label := canvas.NewText(fmt.Sprintf("%d%%", percent), theme.DisabledColor())
			label.TextSize = 8
			label.Move(fyne.NewPos(2, y-6))
			objects = append(objects, label)
		}

		// Vertical grid lines (every 10 values)
		gridInterval := 10
		if r.chart.capacity > 0 {
			for i := gridInterval; i < r.chart.capacity; i += gridInterval {
				x := padding + chartWidth*float32(i)/float32(r.chart.capacity)
				line := canvas.NewLine(r.chart.gridColor)
				line.StrokeWidth = 1
				line.Position1 = fyne.NewPos(x, padding+20)
				line.Position2 = fyne.NewPos(x, padding+20+chartHeight)
				objects = append(objects, line)
			}
		}
	}

	// Draw the line chart
	if len(r.chart.values) > 1 {
		points := make([]fyne.Position, 0, len(r.chart.values))

		for i, value := range r.chart.values {
			x := padding + chartWidth*float32(i)/float32(r.chart.capacity)
			y := padding + 20 + chartHeight*(1-float32(value)/float32(r.chart.maxValue))

			// Clamp y to chart bounds
			if y < padding+20 {
				y = padding + 20
			} else if y > padding+20+chartHeight {
				y = padding + 20 + chartHeight
			}

			points = append(points, fyne.NewPos(x, y))
		}

		// Draw lines between points
		for i := 1; i < len(points); i++ {
			line := canvas.NewLine(r.chart.lineColor)
			line.StrokeWidth = 2
			line.Position1 = points[i-1]
			line.Position2 = points[i]
			objects = append(objects, line)
		}

		// Draw data points
		if r.chart.showDataPoints && len(points) > 0 {
			// Highlight the last point
			lastPoint := points[len(points)-1]

			// Outer glow effect
			glow := canvas.NewCircle(color.NRGBA{
				R: r.chart.pointColor.(color.NRGBA).R,
				G: r.chart.pointColor.(color.NRGBA).G,
				B: r.chart.pointColor.(color.NRGBA).B,
				A: 0x40,
			})
			glow.Resize(fyne.NewSize(12, 12))
			glow.Move(fyne.NewPos(lastPoint.X-6, lastPoint.Y-6))
			objects = append(objects, glow)

			// Inner point
			point := canvas.NewCircle(r.chart.pointColor)
			point.Resize(fyne.NewSize(6, 6))
			point.Move(fyne.NewPos(lastPoint.X-3, lastPoint.Y-3))
			objects = append(objects, point)

			// Current value label
			if len(r.chart.values) > 0 {
				currentValue := r.chart.values[len(r.chart.values)-1]
				valueLabel := canvas.NewText(fmt.Sprintf("%.1f", currentValue), r.chart.pointColor)
				valueLabel.TextSize = 10
				valueLabel.TextStyle = fyne.TextStyle{Bold: true}

				// Position label above or below point based on position
				labelY := lastPoint.Y - 15
				if labelY < padding+20 {
					labelY = lastPoint.Y + 8
				}
				valueLabel.Move(fyne.NewPos(lastPoint.X-10, labelY))
				objects = append(objects, valueLabel)
			}
		}
	} else if len(r.chart.values) == 1 {
		// Single point
		x := padding
		y := padding + 20 + chartHeight*(1-float32(r.chart.values[0])/float32(r.chart.maxValue))

		point := canvas.NewCircle(r.chart.pointColor)
		point.Resize(fyne.NewSize(6, 6))
		point.Move(fyne.NewPos(x-3, y-3))
		objects = append(objects, point)
	}

	// "No data" message if empty
	if len(r.chart.values) == 0 {
		noDataLabel := canvas.NewText("No data", theme.DisabledColor())
		noDataLabel.TextSize = 12
		noDataLabel.Alignment = fyne.TextAlignCenter
		noDataLabel.Move(fyne.NewPos(size.Width/2-20, size.Height/2-6))
		objects = append(objects, noDataLabel)
	}

	return objects
}

func (r *enhancedChartRenderer) Destroy() {
	// Nothing to destroy
}

