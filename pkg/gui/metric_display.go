package gui

import (
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// MetricDisplay shows a metric with its value in both standard and metric units
type MetricDisplay struct {
	widget.BaseWidget

	label      string
	value      float64
	unit       string
	altValue   float64 // Alternative unit value
	altUnit    string  // Alternative unit
	labelColor color.Color
	valueColor color.Color

	// Animation fields
	displayValue    float64
	displayAltValue float64
	animating       bool
	lastUpdate      time.Time
}

// NewMetricDisplay creates a new metric display
func NewMetricDisplay(label string, labelColor, valueColor color.Color) *MetricDisplay {
	m := &MetricDisplay{
		label:      label,
		labelColor: labelColor,
		valueColor: valueColor,
	}
	m.ExtendBaseWidget(m)
	return m
}

// SetValue updates the metric value with optional alternative unit
func (m *MetricDisplay) SetValue(value float64, unit string, altValue float64, altUnit string) {
	// Initialize display values on first call
	if m.lastUpdate.IsZero() {
		m.displayValue = value
		m.displayAltValue = altValue
		m.lastUpdate = time.Now()
	}

	m.value = value
	m.unit = unit
	m.altValue = altValue
	m.altUnit = altUnit

	// Disable animation for now to avoid thread safety issues
	m.displayValue = value
	m.displayAltValue = altValue
	m.Refresh()
}

// animateTransition smoothly transitions between values
func (m *MetricDisplay) animateTransition() {
	m.animating = true
	startValue := m.displayValue
	startAltValue := m.displayAltValue
	targetValue := m.value
	targetAltValue := m.altValue

	go func() {
		steps := 10
		for i := 0; i <= steps; i++ {
			progress := float64(i) / float64(steps)
			// Use easing function for smooth transition
			eased := easeInOutCubic(progress)

			// Update values in main thread
			fyne.Do(func() {
				m.displayValue = startValue + (targetValue-startValue)*eased
				m.displayAltValue = startAltValue + (targetAltValue-startAltValue)*eased
				m.Refresh()
			})

			time.Sleep(30 * time.Millisecond)
		}
		fyne.Do(func() {
			m.animating = false
		})
	}()
}

// easeInOutCubic provides smooth acceleration and deceleration
func easeInOutCubic(t float64) float64 {
	if t < 0.5 {
		return 4 * t * t * t
	}
	p := 2*t - 2
	return 1 + p*p*p/2
}

// CreateRenderer creates the widget renderer
func (m *MetricDisplay) CreateRenderer() fyne.WidgetRenderer {
	label := widget.NewLabel(m.label + ":")
	label.TextStyle = fyne.TextStyle{Bold: true}

	value := widget.NewLabel("")
	value.TextStyle = fyne.TextStyle{Monospace: true}

	return &metricDisplayRenderer{
		metric: m,
		label:  label,
		value:  value,
	}
}

type metricDisplayRenderer struct {
	metric *MetricDisplay
	label  *widget.Label
	value  *widget.Label
}

func (r *metricDisplayRenderer) Layout(size fyne.Size) {
	labelSize := r.label.MinSize()
	r.label.Resize(fyne.NewSize(labelSize.Width, size.Height))
	r.label.Move(fyne.NewPos(0, 0))

	valueX := labelSize.Width + theme.Padding()
	valueWidth := size.Width - valueX
	r.value.Resize(fyne.NewSize(valueWidth, size.Height))
	r.value.Move(fyne.NewPos(valueX, 0))
}

func (r *metricDisplayRenderer) MinSize() fyne.Size {
	labelSize := r.label.MinSize()
	valueSize := r.value.MinSize()
	return fyne.NewSize(200, fyne.Max(labelSize.Height, valueSize.Height))
}

func (r *metricDisplayRenderer) Refresh() {
	// Format value using display values for smooth animation
	var text string
	if r.metric.altUnit != "" && r.metric.displayAltValue > 0 {
		// Show both units
		text = fmt.Sprintf("%.1f %s (%.1f %s)",
			r.metric.displayValue, r.metric.unit,
			r.metric.displayAltValue, r.metric.altUnit)
	} else {
		// Show single unit
		if r.metric.displayValue == 0 {
			text = fmt.Sprintf("-- %s", r.metric.unit)
		} else {
			text = fmt.Sprintf("%.1f %s", r.metric.displayValue, r.metric.unit)
		}
	}

	r.value.SetText(text)

	// Note: In Fyne v2, we can't directly set label colors
	// The theme controls the colors
	// Removed manual refresh calls to avoid thread safety issues
}

func (r *metricDisplayRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.label, r.value}
}

func (r *metricDisplayRenderer) Destroy() {}

// CreateMetricGrid creates a grid of metrics for a summary card
func CreateMetricGrid(metrics map[string]*MetricDisplay) *fyne.Container {
	objects := make([]fyne.CanvasObject, 0, len(metrics)*2)

	// Add metrics in a specific order for consistency
	order := []string{"Temp", "Usage", "Power", "Freq", "Used", "Available", "Memory"}

	for _, key := range order {
		if metric, ok := metrics[key]; ok {
			objects = append(objects, metric)
		}
	}

	// Add any remaining metrics not in the order
	for key, metric := range metrics {
		found := false
		for _, orderedKey := range order {
			if key == orderedKey {
				found = true
				break
			}
		}
		if !found {
			objects = append(objects, metric)
		}
	}

	// Create vertical box
	return container.NewVBox(objects...)
}
