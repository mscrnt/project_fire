package gui

import (
	"image/color"
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// NavigationButton represents a button in the vertical navigation
type NavigationButton struct {
	widget.BaseWidget

	label     string
	icon      fyne.Resource
	onTapped  func()
	selected  bool
	collapsed bool                // Whether to show only icon
	renderer  fyne.WidgetRenderer // Store renderer reference
}

// NewNavigationButton creates a new navigation button
func NewNavigationButton(label string, icon fyne.Resource, onTapped func()) *NavigationButton {
	n := &NavigationButton{
		label:    label,
		icon:     icon,
		onTapped: onTapped,
	}
	n.ExtendBaseWidget(n)
	return n
}

// SetSelected updates the selected state
func (n *NavigationButton) SetSelected(selected bool) {
	n.selected = selected
	if n.renderer != nil {
		n.Refresh()
	}
}

// SetCollapsed updates the collapsed state
func (n *NavigationButton) SetCollapsed(collapsed bool) {
	n.collapsed = collapsed
	n.Refresh()
}

// Tapped handles tap events
func (n *NavigationButton) Tapped(*fyne.PointEvent) {
	if n.onTapped != nil {
		n.onTapped()
	}
}

// MouseIn handles mouse enter events
func (n *NavigationButton) MouseIn(*desktop.MouseEvent) {
	if n.renderer != nil {
		if r, ok := n.renderer.(*navigationButtonRenderer); ok {
			r.hoverBg.Show()
			n.Refresh()
		}
	}
}

// MouseOut handles mouse leave events
func (n *NavigationButton) MouseOut() {
	if n.renderer != nil {
		if r, ok := n.renderer.(*navigationButtonRenderer); ok {
			r.hoverBg.Hide()
			n.Refresh()
		}
	}
}

// MouseMoved handles mouse move events
func (n *NavigationButton) MouseMoved(*desktop.MouseEvent) {
	// Nothing to do on move
}

// CreateRenderer creates the renderer for the navigation button
func (n *NavigationButton) CreateRenderer() fyne.WidgetRenderer {
	var iconObj fyne.CanvasObject
	if n.icon != nil {
		// Use canvas.Image to maintain aspect ratio
		img := canvas.NewImageFromResource(n.icon)
		img.SetMinSize(fyne.NewSize(20, 20))   // Consistent icon size
		img.FillMode = canvas.ImageFillContain // Maintain aspect ratio
		// For SVG icons, ensure they render with proper colors
		img.Translucency = 0 // Fully opaque
		iconObj = img
	} else {
		icon := widget.NewIcon(theme.HomeIcon())
		iconObj = icon
	}

	label := widget.NewLabel(n.label)
	label.Alignment = fyne.TextAlignLeading
	label.TextStyle = fyne.TextStyle{Bold: n.selected}

	// Horizontal layout: icon and label in a single row
	var content fyne.CanvasObject
	if n.collapsed {
		// Only show icon when collapsed
		content = container.NewCenter(iconObj)
	} else {
		// Show icon and label with proper spacing - left aligned, vertically centered
		// Wrap each element to center it vertically
		centeredIcon := container.NewCenter(iconObj)
		centeredLabel := container.NewCenter(label)

		// Add small spacer between icon and label
		spacer := canvas.NewRectangle(color.Transparent)
		spacer.SetMinSize(fyne.NewSize(8, 0))

		// Create horizontal box with vertically centered elements
		hbox := container.NewHBox(centeredIcon, spacer, centeredLabel)

		// Add left padding for proper indentation
		leftPadding := canvas.NewRectangle(color.Transparent)
		leftPadding.SetMinSize(fyne.NewSize(8, 0))
		content = container.NewHBox(leftPadding, hbox)
	}

	// Background - transparent by default
	bg := canvas.NewRectangle(color.Transparent)
	bg.CornerRadius = 6

	// Selection outline - ember color
	selectionOutline := canvas.NewRectangle(color.Transparent)
	selectionOutline.StrokeColor = ColorEmber
	selectionOutline.StrokeWidth = 1.5 // Thinner outline
	selectionOutline.CornerRadius = 4  // Smaller radius

	// Hover effect - very subtle
	hoverBg := canvas.NewRectangle(color.RGBA{0x44, 0x44, 0x44, 0x33}) // Very transparent grey
	hoverBg.CornerRadius = 6
	hoverBg.Hide()

	objects := []fyne.CanvasObject{bg, hoverBg, selectionOutline, content}

	renderer := &navigationButtonRenderer{
		button:           n,
		bg:               bg,
		hoverBg:          hoverBg,
		selectionOutline: selectionOutline,
		content:          content,
		label:            label,
		icon:             iconObj,
		objects:          objects,
	}

	// Store renderer reference
	n.renderer = renderer

	return renderer
}

type navigationButtonRenderer struct {
	button           *NavigationButton
	bg               *canvas.Rectangle
	hoverBg          *canvas.Rectangle
	selectionOutline *canvas.Rectangle
	content          fyne.CanvasObject
	label            *widget.Label
	icon             fyne.CanvasObject
	objects          []fyne.CanvasObject
}

func (r *navigationButtonRenderer) Layout(size fyne.Size) {
	r.bg.Resize(size)
	r.hoverBg.Resize(size)
	r.selectionOutline.Resize(size)
	r.content.Resize(size)
}

func (r *navigationButtonRenderer) MinSize() fyne.Size {
	if r.button.collapsed {
		return fyne.NewSize(50, 40) // Narrower when collapsed
	}
	return fyne.NewSize(180, 40) // Reduced height for tighter layout
}

func (r *navigationButtonRenderer) Refresh() {
	// Update label bold state based on selection
	r.label.TextStyle = fyne.TextStyle{Bold: r.button.selected}
	r.label.Refresh()

	if r.button.selected {
		// Show outline only when selected - more ember/red
		r.selectionOutline.StrokeColor = ColorEmber
		r.bg.FillColor = color.RGBA{ColorEmber.R, ColorEmber.G, ColorEmber.B, 0x20} // Subtle ember fill
	} else {
		r.selectionOutline.StrokeColor = color.Transparent
		r.bg.FillColor = color.Transparent
	}
	r.bg.Refresh()
	r.selectionOutline.Refresh()

	// Update content based on collapsed state
	if r.button.collapsed {
		r.label.Hide()
	} else {
		r.label.Show()
	}
}

func (r *navigationButtonRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *navigationButtonRenderer) Destroy() {}

// NavigationSidebar creates the vertical navigation sidebar
type NavigationSidebar struct {
	container            *fyne.Container
	buttons              []*NavigationButton
	content              *fyne.Container
	currentIndex         int
	collapsed            bool
	collapseBtn          *widget.Button
	collapseBtnContainer *fyne.Container

	// Content pages
	systemInfo fyne.CanvasObject
	tests      fyne.CanvasObject
	history    fyne.CanvasObject
	reports    fyne.CanvasObject
	settings   fyne.CanvasObject
}

// NewNavigationSidebar creates a new navigation sidebar
func NewNavigationSidebar() *NavigationSidebar {
	n := &NavigationSidebar{
		buttons:      make([]*NavigationButton, 0),
		currentIndex: -1,
	}

	// Create navigation buttons with custom icons
	// Use theme icons as fallback if custom icons fail to load
	systemIcon := GetSystemIcon()
	if systemIcon == nil {
		systemIcon = theme.InfoIcon()
	}
	systemInfoBtn := NewNavigationButton("SYSTEM INFO", systemIcon, func() {
		n.ShowPage(0)
	})
	n.buttons = append(n.buttons, systemInfoBtn)

	testIcon := GetTestIcon()
	if testIcon == nil {
		testIcon = theme.ConfirmIcon()
	}
	testsBtn := NewNavigationButton("STABILITY TEST", testIcon, func() {
		n.ShowPage(1)
	})
	n.buttons = append(n.buttons, testsBtn)

	gaugeIcon := GetGaugeIcon()
	if gaugeIcon == nil {
		gaugeIcon = theme.StorageIcon()
	}
	historyBtn := NewNavigationButton("BENCHMARKS", gaugeIcon, func() {
		n.ShowPage(2)
	})
	n.buttons = append(n.buttons, historyBtn)

	cpuIcon := GetCPUIcon()
	if cpuIcon == nil {
		cpuIcon = theme.ViewRefreshIcon()
	}
	reportsBtn := NewNavigationButton("MONITORING", cpuIcon, func() {
		n.ShowPage(3)
	})
	n.buttons = append(n.buttons, reportsBtn)

	settingsIcon := GetSettingsIcon()
	if settingsIcon == nil {
		settingsIcon = theme.SettingsIcon()
	}
	settingsBtn := NewNavigationButton("SETTINGS", settingsIcon, func() {
		n.ShowPage(4)
	})
	n.buttons = append(n.buttons, settingsBtn)

	// Create button container with better spacing
	buttonContainer := container.NewVBox()

	// Add navigation buttons without spacing for tighter layout
	for _, btn := range n.buttons[:5] { // First 5 buttons (main navigation)
		buttonContainer.Add(btn)
	}

	// Add spacer to push bottom buttons down
	buttonContainer.Add(layout.NewSpacer())

	// Add Buy Me Coffee button
	supportIcon := GetSupportIcon()
	if supportIcon == nil {
		supportIcon = theme.HelpIcon()
	}
	supportBtn := NewNavigationButton("BUY ME COFFEE", supportIcon, func() {
		// Open Buy Me a Coffee link
		url := "https://buymeacoffee.com/mscrnt"
		if err := fyne.CurrentApp().OpenURL(parseURL(url)); err != nil {
			DebugLog("ERROR", "Failed to open URL: %v", err)
		}
	})
	n.buttons = append(n.buttons, supportBtn)
	buttonContainer.Add(supportBtn)

	// Create collapse/expand button at the very bottom
	n.collapseBtn = widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		n.ToggleCollapse()
	})
	n.collapseBtn.Importance = widget.LowImportance

	// Store collapse button container for dynamic alignment
	n.collapseBtnContainer = container.NewBorder(nil, nil, nil, n.collapseBtn, nil)

	// Add minimal spacer before collapse button
	collapseSpacer := canvas.NewRectangle(color.Transparent)
	collapseSpacer.SetMinSize(fyne.NewSize(0, 2)) // 75% reduction from 8px
	buttonContainer.Add(collapseSpacer)
	buttonContainer.Add(n.collapseBtnContainer)

	// Navigation background
	navBg := canvas.NewRectangle(color.RGBA{0x2a, 0x2a, 0x2a, 0xff})

	// Navigation container with reduced padding
	// Create custom padding with smaller values
	topPad := canvas.NewRectangle(color.Transparent)
	topPad.SetMinSize(fyne.NewSize(0, 8))
	bottomPad := canvas.NewRectangle(color.Transparent)
	bottomPad.SetMinSize(fyne.NewSize(0, 8))
	leftPad := canvas.NewRectangle(color.Transparent)
	leftPad.SetMinSize(fyne.NewSize(12, 0))
	rightPad := canvas.NewRectangle(color.Transparent)
	rightPad.SetMinSize(fyne.NewSize(12, 0))

	padding := container.NewBorder(topPad, bottomPad, leftPad, rightPad, buttonContainer)
	n.container = container.NewStack(navBg, padding)

	// Content container
	n.content = container.NewStack()

	return n
}

// SetSystemInfo sets the system info page
func (n *NavigationSidebar) SetSystemInfo(content fyne.CanvasObject) {
	DebugLog("DEBUG", "NavigationSidebar.SetSystemInfo called")
	n.systemInfo = content
	if n.currentIndex == -1 {
		DebugLog("DEBUG", "Showing page 0 (system info) by default")
		n.ShowPage(0) // Show system info by default
	}
	DebugLog("DEBUG", "NavigationSidebar.SetSystemInfo completed")
}

// SetTests sets the tests page
func (n *NavigationSidebar) SetTests(content fyne.CanvasObject) {
	n.tests = content
}

// SetHistory sets the history page
func (n *NavigationSidebar) SetHistory(content fyne.CanvasObject) {
	n.history = content
}

// SetReports sets the reports page
func (n *NavigationSidebar) SetReports(content fyne.CanvasObject) {
	n.reports = content
}

// SetSettings sets the settings page
func (n *NavigationSidebar) SetSettings(content fyne.CanvasObject) {
	n.settings = content
}

// ShowPage shows the specified page
func (n *NavigationSidebar) ShowPage(index int) {
	DebugLog("DEBUG", "ShowPage called with index %d", index)
	if index < 0 || index >= len(n.buttons) {
		DebugLog("DEBUG", "Invalid index %d (buttons: %d)", index, len(n.buttons))
		return
	}

	// Update button selection
	DebugLog("DEBUG", "Updating button selection...")
	for i, btn := range n.buttons {
		if btn == nil {
			DebugLog("ERROR", "Button %d is nil!", i)
			continue
		}
		btn.SetSelected(i == index)
	}

	// Update content
	DebugLog("DEBUG", "Clearing content objects...")
	if n.content == nil {
		DebugLog("ERROR", "Content container is nil!")
		return
	}
	n.content.Objects = nil

	DebugLog("DEBUG", "Setting content for index %d", index)
	switch index {
	case 0:
		if n.systemInfo != nil {
			DebugLog("DEBUG", "Setting system info content")
			n.content.Objects = []fyne.CanvasObject{n.systemInfo}
		} else {
			DebugLog("DEBUG", "System info is nil")
		}
	case 1:
		if n.tests != nil {
			DebugLog("DEBUG", "Setting tests content")
			n.content.Objects = []fyne.CanvasObject{n.tests}
		} else {
			DebugLog("DEBUG", "Tests content is nil")
		}
	case 2:
		if n.history != nil {
			n.content.Objects = []fyne.CanvasObject{n.history}
		}
	case 3:
		if n.reports != nil {
			n.content.Objects = []fyne.CanvasObject{n.reports}
		}
	case 4:
		if n.settings != nil {
			n.content.Objects = []fyne.CanvasObject{n.settings}
		}
	}

	DebugLog("DEBUG", "Refreshing content...")
	n.content.Refresh()
	n.currentIndex = index
	DebugLog("DEBUG", "ShowPage completed")
}

// ToggleCollapse toggles the collapsed state of the sidebar
func (n *NavigationSidebar) ToggleCollapse() {
	n.collapsed = !n.collapsed

	// Update all buttons
	for _, btn := range n.buttons {
		btn.SetCollapsed(n.collapsed)
	}

	// Update collapse button icon and alignment
	if n.collapsed {
		n.collapseBtn.SetIcon(theme.NavigateNextIcon())
		// Center when collapsed
		n.collapseBtnContainer.Objects = []fyne.CanvasObject{container.NewCenter(n.collapseBtn)}
	} else {
		n.collapseBtn.SetIcon(theme.NavigateBackIcon())
		// Right align when open
		n.collapseBtnContainer.Objects = []fyne.CanvasObject{
			container.NewBorder(nil, nil, nil, n.collapseBtn, nil),
		}
	}

	// Refresh containers
	n.collapseBtnContainer.Refresh()
	n.container.Refresh()
}

// CreateLayout creates the main layout with sidebar and content
func (n *NavigationSidebar) CreateLayout() fyne.CanvasObject {
	// Create border layout with fixed width sidebar
	// This is more efficient than split container and not adjustable
	return container.NewBorder(nil, nil, n.container, nil, n.content)
}

// parseURL safely parses a URL string
func parseURL(urlStr string) *url.URL {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil
	}
	return u
}
