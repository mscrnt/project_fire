package gui

import (
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Update represents a progress update message
type Update struct {
	Step  int
	Total int
	Text  string
}

// StartupTask represents a task to run during startup
type StartupTask struct {
	Name string
	Fn   func() error
}

// StaticCache holds preloaded component data
type StaticCache struct {
	Motherboard    *MotherboardInfo
	MemoryModules  []MemoryModule
	GPUs           []GPUInfo
	StorageDevices []StorageInfo
	Fans           []FanInfo
	SysInfo        *SystemInfo
}

// FireProgressBar is a custom progress bar with gradient from blue to fire red
type FireProgressBar struct {
	widget.ProgressBar
}

// NewFireProgressBar creates a progress bar with fire gradient
func NewFireProgressBar() *FireProgressBar {
	p := &FireProgressBar{}
	p.ExtendBaseWidget(p)
	return p
}

// CreateRenderer creates a custom renderer for the fire progress bar
func (p *FireProgressBar) CreateRenderer() fyne.WidgetRenderer {
	return &fireProgressRenderer{
		progress: p,
		bar:      canvas.NewRectangle(color.RGBA{0, 100, 255, 255}), // Start blue
	}
}

type fireProgressRenderer struct {
	progress *FireProgressBar
	bar      *canvas.Rectangle
}

func (r *fireProgressRenderer) Destroy() {}

func (r *fireProgressRenderer) Layout(size fyne.Size) {
	r.bar.Resize(fyne.NewSize(size.Width*float32(r.progress.Value), size.Height))
}

func (r *fireProgressRenderer) MinSize() fyne.Size {
	return fyne.NewSize(800, 50) // Larger progress bar
}

func (r *fireProgressRenderer) Refresh() {
	// Gradient from blue to fire red based on progress
	progress := float32(r.progress.Value)

	// Blue (0,100,255) to Fire Red (255,69,0)
	red := uint8(progress * 255)
	green := uint8(100 - (progress * 31)) // 100 to 69
	blue := uint8(255 - (progress * 255)) // 255 to 0

	r.bar.FillColor = color.RGBA{red, green, blue, 255}
	r.bar.Refresh()

	r.bar.Resize(fyne.NewSize(r.progress.Size().Width*float32(r.progress.Value), r.progress.Size().Height))
}

func (r *fireProgressRenderer) Objects() []fyne.CanvasObject {
	// Background
	bg := canvas.NewRectangle(theme.DisabledColor())
	bg.Resize(r.progress.Size())

	return []fyne.CanvasObject{bg, r.bar}
}

// CreateLoadingOverlay creates a loading screen overlay
func CreateLoadingOverlay() (fyne.CanvasObject, *widget.RichText, *FireProgressBar) {
	DebugLog("LOADING_UI", "CreateLoadingOverlay called")

	// Load larger logo
	logoResource, err := fyne.LoadResourceFromPath("assets/logos/fire_1024.png")
	var logo *canvas.Image
	if err != nil {
		DebugLog("LOADING_UI", "Failed to load 1024 logo, trying 512: "+err.Error())
		// Try 512 as fallback
		logoResource, err = fyne.LoadResourceFromPath("assets/logos/fire_512.png")
	}

	if err == nil {
		logo = canvas.NewImageFromResource(logoResource)
		logo.FillMode = canvas.ImageFillContain
		logo.SetMinSize(fyne.NewSize(512, 512)) // Even larger logo
		DebugLog("LOADING_UI", "Logo loaded successfully, size: 512x512")
	} else {
		DebugLog("LOADING_UI", "Failed to load any logo: "+err.Error())
	}

	// Create larger title
	titleStyle := &canvas.Text{
		Text:      "F.I.R.E.",
		TextSize:  72, // Much larger
		Color:     theme.ForegroundColor(),
		TextStyle: fyne.TextStyle{Bold: true},
		Alignment: fyne.TextAlignCenter,
	}

	// Create larger subtitle
	subtitleStyle := &canvas.Text{
		Text:      "Full Intensity Rigorous Evaluation",
		TextSize:  32, // Larger
		Color:     theme.ForegroundColor(),
		Alignment: fyne.TextAlignCenter,
	}

	// Create loading label with large text
	loadingLabel := widget.NewRichTextFromMarkdown("### Initializing...")
	for _, seg := range loadingLabel.Segments {
		if textSeg, ok := seg.(*widget.TextSegment); ok {
			textSeg.Style.Alignment = fyne.TextAlignCenter
			textSeg.Style.SizeName = "28" // Larger loading text
			textSeg.Style.TextStyle = fyne.TextStyle{Bold: true}
		}
	}

	// Create custom fire progress bar
	progressBar := NewFireProgressBar()
	progressBar.SetValue(0)

	DebugLog("LOADING_UI", "Creating fire progress bar with gradient")

	// Create a container for the progress bar with fixed size
	progressContainer := container.NewWithoutLayout(progressBar)
	progressContainer.Resize(fyne.NewSize(800, 50))
	progressBar.Resize(fyne.NewSize(800, 50))

	DebugLog("LOADING_UI", "Building content layout")

	// Build the main content
	var mainContent *fyne.Container
	if logo != nil {
		mainContent = container.NewVBox(
			layout.NewSpacer(),
			container.NewCenter(logo),
			container.NewPadded(),
			container.NewCenter(titleStyle),
			container.NewCenter(subtitleStyle),
			container.NewPadded(),
			container.NewPadded(),
			container.NewCenter(loadingLabel),
			container.NewPadded(),
			container.NewCenter(progressContainer),
			layout.NewSpacer(),
		)
		DebugLog("LOADING_UI", "Content created with logo")
	} else {
		mainContent = container.NewVBox(
			layout.NewSpacer(),
			layout.NewSpacer(),
			container.NewCenter(titleStyle),
			container.NewCenter(subtitleStyle),
			container.NewPadded(),
			container.NewPadded(),
			container.NewCenter(loadingLabel),
			container.NewPadded(),
			container.NewCenter(progressContainer),
			layout.NewSpacer(),
		)
		DebugLog("LOADING_UI", "Content created without logo")
	}

	// Center everything
	centeredContent := container.NewCenter(mainContent)

	DebugLog("LOADING_UI", "Loading overlay created successfully")

	return centeredContent, loadingLabel, progressBar
}

// LoadComponentsAsync loads all components in background and sends progress updates
func LoadComponentsAsync(updates chan<- Update) *StaticCache {
	cache := &StaticCache{}

	tasks := []StartupTask{
		{Name: "Loading CPU information...", Fn: func() error {
			DebugLog("STARTUP", "Detecting CPU information...")
			start := time.Now()
			cache.SysInfo, _ = GetSystemInfo()
			DebugLog("TIMING", fmt.Sprintf("GetSystemInfo took %v", time.Since(start)))
			return nil
		}},
		{Name: "Loading motherboard details...", Fn: func() error {
			DebugLog("STARTUP", "Loading motherboard details...")
			start := time.Now()
			cache.Motherboard, _ = GetMotherboardInfo()
			DebugLog("TIMING", fmt.Sprintf("GetMotherboardInfo took %v", time.Since(start)))
			return nil
		}},
		{Name: "Scanning memory modules...", Fn: func() error {
			DebugLog("STARTUP", "Scanning memory modules...")
			start := time.Now()
			cache.MemoryModules, _ = GetMemoryModules()
			DebugLog("TIMING", fmt.Sprintf("GetMemoryModules took %v", time.Since(start)))
			DebugLog("STARTUP", fmt.Sprintf("Loaded %d memory modules", len(cache.MemoryModules)))
			return nil
		}},
		{Name: "Detecting graphics cards...", Fn: func() error {
			DebugLog("STARTUP", "Detecting graphics cards...")
			start := time.Now()
			cache.GPUs, _ = GetGPUInfo()
			DebugLog("TIMING", fmt.Sprintf("GetGPUInfo took %v", time.Since(start)))
			DebugLog("STARTUP", fmt.Sprintf("Loaded %d GPUs", len(cache.GPUs)))
			return nil
		}},
		{Name: "Scanning storage devices...", Fn: func() error {
			DebugLog("STARTUP", "Scanning storage devices...")
			start := time.Now()
			devices, err := quickStorageScan()
			if err == nil {
				cache.StorageDevices = devices
			}
			DebugLog("TIMING", fmt.Sprintf("quickStorageScan took %v", time.Since(start)))
			return nil
		}},
		{Name: "Detecting cooling systems...", Fn: func() error {
			DebugLog("STARTUP", "Detecting cooling systems...")
			start := time.Now()
			cache.Fans, _ = GetFanInfo()
			DebugLog("TIMING", fmt.Sprintf("GetFanInfo took %v", time.Since(start)))
			return nil
		}},
		{Name: "Initializing sensor monitoring...", Fn: func() error {
			DebugLog("STARTUP", "Initializing sensor monitoring...")
			time.Sleep(50 * time.Millisecond)
			return nil
		}},
	}

	// Execute tasks and send updates
	for i, task := range tasks {
		start := time.Now()

		// Send progress update
		updates <- Update{
			Step:  i + 1,
			Total: len(tasks),
			Text:  task.Name,
		}

		// Execute the task
		if err := task.Fn(); err != nil {
			DebugLog("ERROR", fmt.Sprintf("Task '%s' failed: %v", task.Name, err))
		}

		// Ensure minimum visibility time
		if elapsed := time.Since(start); elapsed < 200*time.Millisecond {
			time.Sleep(200*time.Millisecond - elapsed)
		}
	}

	DebugLog("STARTUP", fmt.Sprintf("Component loading complete - %d GPUs, %d memory modules",
		len(cache.GPUs), len(cache.MemoryModules)))

	return cache
}

// quickStorageScan performs a quick scan to get basic storage info
func quickStorageScan() ([]StorageInfo, error) {
	DebugLog("STARTUP", "Performing quick storage scan...")

	devices, err := GetStorageInfo()
	if err != nil {
		return nil, err
	}

	// For the quick scan, clear out slow fields like SMART data
	for i := range devices {
		devices[i].SMART = nil
	}

	return devices, nil
}
