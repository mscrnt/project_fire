package gui

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// MetricBar displays a metric with both bar and text
type MetricBar struct {
	widget.BaseWidget

	label    string
	value    float64
	unit     string
	altValue float64
	altUnit  string
	max      float64
	barColor color.Color
	showBar  bool

	// Tooltip data
	minValue     float64
	maxValue     float64
	avgValue     float64
	hasHistory   bool
	tooltip      *widget.PopUp
	tooltipLabel *widget.Label
	tooltipTimer *time.Timer
	updateTicker *time.Ticker

	// Change detection
	prevValue    float64
	prevAltValue float64
}

// NewMetricBar creates a new metric bar display
func NewMetricBar(label string, barColor color.Color, showBar bool) *MetricBar {
	m := &MetricBar{
		label:    label,
		barColor: barColor,
		max:      100, // Default max for percentages
		showBar:  showBar,
	}
	m.ExtendBaseWidget(m)
	return m
}

// SetValue updates the metric value
func (m *MetricBar) SetValue(value float64, unit string, altValue float64, altUnit string) {
	// Only update and refresh if value has changed
	if m.value == value && m.altValue == altValue && m.unit == unit && m.altUnit == altUnit {
		return
	}

	m.prevValue = m.value
	m.prevAltValue = m.altValue

	m.value = value
	m.unit = unit
	m.altValue = altValue
	m.altUnit = altUnit

	// Update bar color based on value and metric type
	m.updateBarColor()

	m.Refresh()
}

// updateBarColor updates the bar color based on the metric type and value
func (m *MetricBar) updateBarColor() {
	if !m.showBar {
		return
	}

	switch m.label {
	case "Temp":
		// Temperature thresholds (Celsius)
		// CPU/GPU: <60°C green, 60-75°C yellow, 75-85°C orange, >85°C red
		// Memory: <50°C green, 50-65°C yellow, 65-75°C orange, >75°C red
		switch {
		case m.value < 50:
			m.barColor = ColorGood // Green
		case m.value < 65:
			m.barColor = ColorWarning // Yellow
		case m.value < 80:
			m.barColor = ColorCaution // Orange
		default:
			m.barColor = ColorCritical // Red
		}

	case "Usage", "Used", "VRAM":
		// Usage percentages: <60% green, 60-80% yellow, 80-90% orange, >90% red
		switch {
		case m.value < 60:
			m.barColor = ColorGood
		case m.value < 80:
			m.barColor = ColorWarning
		case m.value < 90:
			m.barColor = ColorCaution
		default:
			m.barColor = ColorCritical
		}

	case "Power":
		// Power as percentage of TDP/limit
		// This would need max power info - for now use fixed thresholds
		// <100W green, 100-200W yellow, 200-300W orange, >300W red
		switch {
		case m.value < 100:
			m.barColor = ColorGood
		case m.value < 200:
			m.barColor = ColorWarning
		case m.value < 300:
			m.barColor = ColorCaution
		default:
			m.barColor = ColorCritical
		}

	case "Speed":
		// Speed is good when high, so inverse colors
		// For CPU GHz: >4.0 green, 3.0-4.0 yellow, 2.0-3.0 orange, <2.0 red
		switch m.unit {
		case "GHz":
			switch {
			case m.value > 4.0:
				m.barColor = ColorGood
			case m.value > 3.0:
				m.barColor = ColorWarning
			case m.value > 2.0:
				m.barColor = ColorCaution
			default:
				m.barColor = ColorCritical
			}
		case "MHz":
			// GPU MHz: >1500 green, 1000-1500 yellow, 500-1000 orange, <500 red
			switch {
			case m.value > 1500:
				m.barColor = ColorGood
			case m.value > 1000:
				m.barColor = ColorWarning
			case m.value > 500:
				m.barColor = ColorCaution
			default:
				m.barColor = ColorCritical
			}
		}

	case "Total":
		// Memory total - just use a neutral color
		m.barColor = ColorFrequency
	}
}

// SetMax sets the maximum value for the bar
func (m *MetricBar) SetMax(maxValue float64) {
	m.max = maxValue
	m.Refresh()
}

// SetHistory updates the historical data for tooltips
func (m *MetricBar) SetHistory(minVal, maxVal, avg float64) {
	m.minValue = minVal
	m.maxValue = maxVal
	m.avgValue = avg
	m.hasHistory = true
	m.Refresh()
}

// MouseIn is called when the mouse enters the widget
func (m *MetricBar) MouseIn(event *desktop.MouseEvent) {
	// Cancel any existing timer
	if m.tooltipTimer != nil {
		m.tooltipTimer.Stop()
	}

	// Start a timer to show tooltip after a short delay
	m.tooltipTimer = time.AfterFunc(500*time.Millisecond, func() {
		if m.tooltip == nil {
			m.showTooltip(event)
		}
	})
}

// MouseOut is called when the mouse leaves the widget
func (m *MetricBar) MouseOut() {
	// Cancel timer if tooltip hasn't shown yet
	if m.tooltipTimer != nil {
		m.tooltipTimer.Stop()
		m.tooltipTimer = nil
	}
	m.hideTooltip()
}

// MouseMoved is called when the mouse moves within the widget
func (m *MetricBar) MouseMoved(_ *desktop.MouseEvent) {
	// Don't update position on every move to reduce flicker
	// The tooltip will stay visible as long as mouse is over the widget
}

// showTooltip displays the tooltip at the given position
func (m *MetricBar) showTooltip(event *desktop.MouseEvent) {
	m.hideTooltip() // Hide any existing tooltip

	// Get fresh tooltip content with current values
	tooltipContent := m.buildTooltipContent()
	if tooltipContent != "" {
		canvas := fyne.CurrentApp().Driver().CanvasForObject(m)
		if canvas == nil {
			return
		}

		// Create a simple tooltip without card for better performance
		m.tooltipLabel = widget.NewLabel(tooltipContent)
		m.tooltipLabel.TextStyle = fyne.TextStyle{Monospace: true}

		// Get metric name for the header
		metricName := m.label
		switch m.label {
		case "Temp":
			metricName = "CPU Die (average)"
		case "Voltage":
			metricName = "Core 0 VID"
		case "Power":
			metricName = "CPU Package Power"
		case "Usage":
			metricName = "Total CPU Usage"
		case "Speed":
			metricName = "Core 0 T0 Effective Clock"
		}

		// Create container with title
		tooltipCard := widget.NewCard(metricName, "", m.tooltipLabel)

		m.tooltip = widget.NewPopUp(tooltipCard, canvas)

		// Start updating the tooltip content
		m.startTooltipUpdates()

		// Calculate absolute position by walking up the widget tree
		// Currently we just check if the widget belongs to the current canvas
		// Future implementation could walk up the parent tree for better positioning
		// TODO: Implement proper parent tree walking when needed

		// Use the event's absolute position as a more reliable reference
		// Position tooltip near the mouse but offset to avoid interference
		tooltipX := event.AbsolutePosition.X + 20
		tooltipY := event.AbsolutePosition.Y + 20

		// Get canvas size to ensure tooltip stays on screen
		canvasSize := canvas.Size()
		tooltipSize := m.tooltip.MinSize()

		// Adjust if tooltip would go off right edge
		if tooltipX+tooltipSize.Width > canvasSize.Width {
			tooltipX = event.AbsolutePosition.X - tooltipSize.Width - 20
		}

		// Adjust if tooltip would go off bottom
		if tooltipY+tooltipSize.Height > canvasSize.Height {
			tooltipY = event.AbsolutePosition.Y - tooltipSize.Height - 20
		}

		m.tooltip.Move(fyne.NewPos(tooltipX, tooltipY))
		m.tooltip.Show()
	}
}

// hideTooltip hides the current tooltip
func (m *MetricBar) hideTooltip() {
	// Stop update ticker
	if m.updateTicker != nil {
		m.updateTicker.Stop()
		m.updateTicker = nil
	}

	if m.tooltip != nil {
		m.tooltip.Hide()
		m.tooltip = nil
		m.tooltipLabel = nil
	}
}

// startTooltipUpdates starts a ticker to update tooltip content
func (m *MetricBar) startTooltipUpdates() {
	// Update tooltip every 500ms
	m.updateTicker = time.NewTicker(500 * time.Millisecond)

	go func() {
		for range m.updateTicker.C {
			if m.tooltipLabel != nil && m.tooltip != nil {
				// Update content on the UI thread
				newContent := m.buildTooltipContent()
				if m.tooltipLabel != nil { // Double check in case it was cleared
					m.tooltipLabel.SetText(newContent)
				}
			} else {
				// Tooltip was closed
				return
			}
		}
	}()
}

// updateTooltipContent updates the tooltip content with current values
func (m *MetricBar) updateTooltipContent() {
	if m.tooltipLabel != nil {
		newContent := m.buildTooltipContent()
		m.tooltipLabel.SetText(newContent)
	}
}

// buildTooltipContent creates the tooltip text
func (m *MetricBar) buildTooltipContent() string {
	var content strings.Builder

	// Format value based on unit type for cleaner display
	formatValue := func(val float64, unit string) string {
		switch unit {
		case "V":
			return fmt.Sprintf("%.3f %s", val, unit)
		case "MHz", "MB":
			return fmt.Sprintf("%.0f %s", val, unit)
		case "°C", "°F", "%", "W", "GHz":
			return fmt.Sprintf("%.1f %s", val, unit)
		default:
			return fmt.Sprintf("%.1f %s", val, unit)
		}
	}

	// Current value
	content.WriteString(fmt.Sprintf("Current: %s\n", formatValue(m.value, m.unit)))

	// Add alternative unit if available (e.g., Fahrenheit)
	if m.altValue != 0 && m.altUnit != "" {
		content.WriteString(fmt.Sprintf("         %s\n", formatValue(m.altValue, m.altUnit)))
	}

	// Add historical data if available
	if m.hasHistory && m.maxValue > 0 {
		content.WriteString(fmt.Sprintf("\nMin: %s\n", formatValue(m.minValue, m.unit)))
		content.WriteString(fmt.Sprintf("Avg: %s\n", formatValue(m.avgValue, m.unit)))
		content.WriteString(fmt.Sprintf("Max: %s\n", formatValue(m.maxValue, m.unit)))
	}

	// Add status based on current value
	content.WriteString("\nStatus: ")
	switch m.label {
	case "Temp":
		switch {
		case m.value < 50:
			content.WriteString("Good")
		case m.value < 65:
			content.WriteString("Normal")
		case m.value < 80:
			content.WriteString("High")
		default:
			content.WriteString("Critical")
		}
	case "Usage", "Used", "VRAM":
		switch {
		case m.value < 60:
			content.WriteString("Low")
		case m.value < 80:
			content.WriteString("Moderate")
		case m.value < 90:
			content.WriteString("High")
		default:
			content.WriteString("Very High")
		}
	case "Power":
		if m.max > 0 {
			content.WriteString(fmt.Sprintf("%.0f%% of TDP", (m.value/m.max)*100))
		} else {
			content.WriteString("N/A")
		}
	case "Speed":
		if m.unit == "GHz" {
			switch {
			case m.value > 4.0:
				content.WriteString("High")
			case m.value > 3.0:
				content.WriteString("Normal")
			case m.value > 2.0:
				content.WriteString("Low")
			default:
				content.WriteString("Very Low")
			}
		} else {
			content.WriteString("Active")
		}
	case "Total":
		content.WriteString("System Memory")
	}

	return content.String()
}

// CreateRenderer creates the widget renderer
func (m *MetricBar) CreateRenderer() fyne.WidgetRenderer {
	// Value text only - no label
	valueText := widget.NewLabel("")
	valueText.TextStyle = fyne.TextStyle{Monospace: true}

	// Progress bar (only if showBar is true)
	var bar *canvas.Rectangle
	var barBg *canvas.Rectangle
	if m.showBar {
		barBg = canvas.NewRectangle(color.RGBA{0x33, 0x33, 0x33, 0xff})
		barBg.CornerRadius = 2
		bar = canvas.NewRectangle(m.barColor)
		bar.CornerRadius = 2
	}

	return &metricBarRenderer{
		metric:    m,
		valueText: valueText,
		bar:       bar,
		barBg:     barBg,
	}
}

type metricBarRenderer struct {
	metric    *MetricBar
	valueText *widget.Label
	bar       *canvas.Rectangle
	barBg     *canvas.Rectangle
}

func (r *metricBarRenderer) Layout(size fyne.Size) {
	// Stack layout: value on top, bar underneath
	valueSize := r.valueText.MinSize()

	// Position value text centered
	r.valueText.Resize(fyne.NewSize(size.Width, valueSize.Height))
	r.valueText.Move(fyne.NewPos(0, 0))

	// Position bar underneath if enabled
	if r.metric.showBar && r.barBg != nil && r.bar != nil {
		barY := valueSize.Height + 2
		barHeight := float32(4) // Thinner bar
		barWidth := size.Width

		r.barBg.Resize(fyne.NewSize(barWidth, barHeight))
		r.barBg.Move(fyne.NewPos(0, barY))

		// Calculate bar fill width
		fillRatio := r.metric.value / r.metric.max
		if fillRatio > 1 {
			fillRatio = 1
		}
		if fillRatio < 0 {
			fillRatio = 0
		}
		fillWidth := barWidth * float32(fillRatio)

		r.bar.Resize(fyne.NewSize(fillWidth, barHeight))
		r.bar.Move(fyne.NewPos(0, barY))
	}
}

func (r *metricBarRenderer) MinSize() fyne.Size {
	valueSize := r.valueText.MinSize()
	width := valueSize.Width + 20 // Add horizontal padding for spacing
	height := valueSize.Height

	if r.metric.showBar {
		height += 6  // Add space for bar underneath
		height += 12 // Add gap below bar
	}

	return fyne.NewSize(width, height)
}

func (r *metricBarRenderer) Refresh() {
	// Update value text
	var text string
	if r.metric.value == 0 && r.metric.unit != "°C" && r.metric.unit != "V" {
		text = fmt.Sprintf("-- %s", r.metric.unit)
	} else {
		// Format based on unit type
		switch r.metric.unit {
		case "V":
			text = fmt.Sprintf("%.3f %s", r.metric.value, r.metric.unit)
		case "MHz", "MB":
			text = fmt.Sprintf("%.0f %s", r.metric.value, r.metric.unit)
		default:
			text = fmt.Sprintf("%.1f %s", r.metric.value, r.metric.unit)
		}
	}
	r.valueText.SetText(text)

	// Update bar color if needed
	if r.metric.showBar && r.bar != nil {
		r.bar.FillColor = r.metric.barColor
		r.bar.Refresh()
		r.Layout(r.metric.Size())
	}
}

func (r *metricBarRenderer) Objects() []fyne.CanvasObject {
	objects := []fyne.CanvasObject{r.valueText}
	if r.metric.showBar && r.barBg != nil && r.bar != nil {
		objects = append(objects, r.barBg, r.bar)
	}
	return objects
}

func (r *metricBarRenderer) Destroy() {}
