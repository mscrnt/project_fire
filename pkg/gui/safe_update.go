package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// safeSetText safely updates a label's text from any goroutine
func safeSetText(label *widget.Label, text string) {
	if label == nil {
		return
	}
	fyne.Do(func() {
		label.SetText(text)
	})
}

// safeSetValue safely updates a progress bar's value from any goroutine
func safeSetValue(progress *widget.ProgressBar, value float64) {
	if progress == nil {
		return
	}
	fyne.Do(func() {
		progress.SetValue(value)
	})
}

// safeRefresh safely refreshes a widget from any goroutine
func safeRefresh(obj fyne.CanvasObject) {
	if obj == nil {
		return
	}
	fyne.Do(func() {
		obj.Refresh()
	})
}
