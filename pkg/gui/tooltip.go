package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// TooltipWidget wraps a widget with tooltip functionality
type TooltipWidget struct {
	widget.BaseWidget

	child   fyne.CanvasObject
	tooltip string
	popup   *widget.PopUp
	window  fyne.Window
}

// NewTooltipWidget creates a widget with a tooltip
func NewTooltipWidget(child fyne.CanvasObject, tooltip string, window fyne.Window) *TooltipWidget {
	t := &TooltipWidget{
		child:   child,
		tooltip: tooltip,
		window:  window,
	}
	t.ExtendBaseWidget(t)
	return t
}

// CreateRenderer creates the renderer for the tooltip widget
func (t *TooltipWidget) CreateRenderer() fyne.WidgetRenderer {
	return &tooltipRenderer{
		tooltip: t,
		objects: []fyne.CanvasObject{t.child},
	}
}

// MouseIn shows the tooltip
func (t *TooltipWidget) MouseIn(*desktop.MouseEvent) {
	if t.tooltip == "" {
		return
	}

	label := widget.NewLabel(t.tooltip)
	label.Wrapping = fyne.TextWrapWord

	content := container.NewPadded(label)
	t.popup = widget.NewPopUp(content, t.window.Canvas())

	// Position near the widget
	pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(t)
	size := t.Size()
	t.popup.Move(fyne.NewPos(pos.X, pos.Y+size.Height+5))
	t.popup.Show()
}

// MouseOut hides the tooltip
func (t *TooltipWidget) MouseOut() {
	if t.popup != nil {
		t.popup.Hide()
		t.popup = nil
	}
}

// MouseMoved is required by the interface
func (t *TooltipWidget) MouseMoved(*desktop.MouseEvent) {}

type tooltipRenderer struct {
	tooltip *TooltipWidget
	objects []fyne.CanvasObject
}

func (r *tooltipRenderer) Layout(size fyne.Size) {
	r.tooltip.child.Resize(size)
	r.tooltip.child.Move(fyne.NewPos(0, 0))
}

func (r *tooltipRenderer) MinSize() fyne.Size {
	return r.tooltip.child.MinSize()
}

func (r *tooltipRenderer) Refresh() {
	r.tooltip.child.Refresh()
}

func (r *tooltipRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *tooltipRenderer) Destroy() {
	if r.tooltip.popup != nil {
		r.tooltip.popup.Hide()
	}
}

// AddTooltip wraps a widget with a tooltip
func AddTooltip(w fyne.CanvasObject, tooltip string, window fyne.Window) fyne.CanvasObject {
	if tooltip == "" {
		return w
	}
	return NewTooltipWidget(w, tooltip, window)
}
