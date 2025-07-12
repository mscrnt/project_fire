package gui

import (
	"time"
	
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// FireGUI represents the main GUI application
type FireGUI struct {
	app    fyne.App
	window fyne.Window

	// Navigation
	navigation *NavigationSidebar

	// Main content containers
	dashboard  *Dashboard
	testsPage  *TestsPage
	testWizard *TestWizard
	history    *History
	compare    *Compare
	aiInsights *AIInsights
	certs      *Certificates

	// Current database path
	dbPath string
	
	// Admin status
	isAdmin bool
	adminWarningShown bool
}

// NewFireGUI creates a new F.I.R.E. GUI instance
func NewFireGUI(app fyne.App) *FireGUI {
	DebugLog("DEBUG", "NewFireGUI - Creating GUI instance...")
	gui := &FireGUI{
		app:    app,
		window: app.NewWindow("F.I.R.E. System Monitor"),
		dbPath: getDefaultDBPath(),
	}

	DebugLog("DEBUG", "NewFireGUI - Calling setup()...")
	gui.setup()
	DebugLog("DEBUG", "NewFireGUI - Setup complete")
	return gui
}

// GetDashboard returns the dashboard instance
func (g *FireGUI) GetDashboard() *Dashboard {
	return g.dashboard
}

// setup initializes the GUI layout
func (g *FireGUI) setup() {
	DebugCheckpoint("setup-start")
	DebugLog("DEBUG", "setup() - Applying theme...")
	// Apply FIRE theme
	g.app.Settings().SetTheme(FireDarkTheme{})

	DebugLog("DEBUG", "setup() - Setting window size...")
	// Set window size to 1600x900 (16:9 aspect ratio, HD+)
	g.window.Resize(fyne.NewSize(1600, 900))
	g.window.CenterOnScreen()

	// Check for administrator privileges - defer the warning until window is shown
	g.isAdmin = IsRunningAsAdmin()
	if !g.isAdmin {
		DebugLog("WARNING", "Not running as Administrator - some features will be limited")
	} else {
		DebugLog("INFO", "Running with Administrator privileges")
	}

	// Remove traditional menu bar - we'll integrate actions into navigation

	DebugLog("DEBUG", "setup() - Creating Navigation...")
	g.navigation = NewNavigationSidebar()

	DebugLog("DEBUG", "setup() - Creating Dashboard...")
	g.dashboard = NewDashboard()    // FIRE System Monitor
	g.dashboard.SetWindow(g.window) // Set window reference for dialogs

	DebugLog("DEBUG", "setup() - Creating Tests Page...")
	g.testsPage = NewTestsPage()

	// Delay navigation setup to avoid UI thread deadlock
	DebugLog("DEBUG", "setup() - Deferring navigation page setup...")
	
	// Store references for later setup
	g.navigation.systemInfo = g.dashboard.Content()
	g.navigation.tests = g.testsPage.Content()
	g.navigation.history = widget.NewLabel("History page coming soon...")
	g.navigation.reports = widget.NewLabel("Reports page coming soon...")
	g.navigation.settings = widget.NewLabel("Settings page coming soon...")

	DebugLog("DEBUG", "setup() - Creating other components (commented out for debugging)...")
	// Temporarily comment out other components to isolate the issue
	// g.testWizard = NewTestWizard(g.dbPath)
	// g.history = NewHistory(g.dbPath)
	// g.compare = NewCompare(g.dbPath)
	// g.aiInsights = NewAIInsights()
	// g.certs = NewCertificates(g.dbPath)

	// Get the summary strip from dashboard
	DebugLog("DEBUG", "setup() - Getting summary strip...")
	summaryStrip := g.dashboard.SummaryStrip()
	if summaryStrip == nil {
		DebugLog("ERROR", "Summary strip is nil!")
		summaryStrip = container.NewHBox() // Empty container as fallback
	}

	// Create a container that limits the height of the summary strip
	// to approximately 10% of the window height (90 pixels for 900p)
	// Using a custom layout to enforce the height
	summaryContainer := container.New(&fixedHeightLayout{height: 90}, summaryStrip)

	DebugLog("DEBUG", "setup() - Setting window content...")
	// Set content with summary strip at top (no red header)
	content := container.NewBorder(
		summaryContainer,
		nil, nil, nil,
		g.navigation.CreateLayout(),
	)
	g.window.SetContent(content)

	DebugLog("DEBUG", "setup() - Setting close handler...")
	// Set close handler
	g.window.SetCloseIntercept(func() {
		g.dashboard.Stop()
		g.window.Close()
	})

	DebugLog("DEBUG", "setup() - Complete!")
}

// createMenu creates the application menu
func (g *FireGUI) createMenu() {
	fileMenu := fyne.NewMenu("File",
		fyne.NewMenuItem("Open Database...", g.openDatabase),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Export Report...", g.exportReport),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Quit", func() {
			g.app.Quit()
		}),
	)

	editMenu := fyne.NewMenu("Edit",
		fyne.NewMenuItem("Preferences", g.showPreferences),
	)

	viewMenu := fyne.NewMenu("View",
		fyne.NewMenuItem("Toggle Theme", g.toggleTheme),
		fyne.NewMenuItem("Refresh", g.refresh),
	)

	helpMenu := fyne.NewMenu("Help",
		fyne.NewMenuItem("Documentation", g.showDocumentation),
		fyne.NewMenuItem("About", g.showAbout),
	)

	mainMenu := fyne.NewMainMenu(fileMenu, editMenu, viewMenu, helpMenu)
	g.window.SetMainMenu(mainMenu)
}

// ShowAndRun displays the window and runs the application
func (g *FireGUI) ShowAndRun() {
	DebugLog("DEBUG", "ShowAndRun() - Starting dashboard monitoring...")
	// Start dashboard monitoring
	g.dashboard.Start()

	// Show the first page before displaying window
	DebugLog("DEBUG", "Showing first navigation page...")
	g.navigation.ShowPage(0)
	
	DebugCheckpoint("window-show")
	DebugLog("DEBUG", "ShowAndRun() - Calling window.ShowAndRun()...")
	
	// Schedule admin notification after window is shown
	go func() {
		// Wait for window to be fully loaded
		time.Sleep(2 * time.Second)
		
		// Show admin notification if needed
		if !g.isAdmin && !g.adminWarningShown {
			fyne.CurrentApp().SendNotification(&fyne.Notification{
				Title:   "Limited Functionality",
				Content: "Running without Administrator privileges. Some features like SPD memory reading will be unavailable.",
			})
			g.adminWarningShown = true
		}
	}()

	// Show window and run
	g.window.ShowAndRun()

	DebugLog("DEBUG", "ShowAndRun() - Window closed")
}

// showAdminWarning displays a warning dialog about limited functionality without admin privileges
func (g *FireGUI) showAdminWarning() {
	features := GetAdminRequiredFeatures()
	content := "F.I.R.E. is running without Administrator privileges.\n\n" +
		"The following features will not be available:\n\n"
	
	for _, feature := range features {
		content += "• " + feature + "\n"
	}
	
	content += "\nTo enable all features, please restart F.I.R.E. as Administrator."
	
	dialog.NewInformation("Administrator Privileges Required", content, g.window).Show()
}

// Menu action handlers
func (g *FireGUI) openDatabase() {
	// TODO: Implement file dialog for database selection
	dialog.ShowInformation("Open Database", "This feature will be implemented soon", g.window)
}

func (g *FireGUI) exportReport() {
	// TODO: Implement report export
	dialog.ShowInformation("Export Report", "This feature will be implemented soon", g.window)
}

func (g *FireGUI) showPreferences() {
	// TODO: Implement preferences dialog
	dialog.ShowInformation("Preferences", "This feature will be implemented soon", g.window)
}

func (g *FireGUI) toggleTheme() {
	// Theme toggling is deprecated in Fyne v2
	// Users should use system theme preferences instead
	dialog.ShowInformation("Theme", "Please use your system theme preferences to change the theme", g.window)
}

func (g *FireGUI) refresh() {
	// Refresh dashboard
	if g.dashboard != nil {
		g.dashboard.updateMetrics()
	}
}

func (g *FireGUI) showDocumentation() {
	// TODO: Open documentation in browser
	dialog.ShowInformation("Documentation", "Visit https://github.com/mscrnt/project_fire", g.window)
}

func (g *FireGUI) showAbout() {
	dialog := widget.NewCard(
		"About F.I.R.E.",
		"Full Intensity Rigorous Evaluation",
		widget.NewLabel("Version: 1.0.0\n\n"+
			"A comprehensive PC test bench for burn-in tests,\n"+
			"endurance stress testing, and benchmark analysis.\n\n"+
			"© 2025 F.I.R.E. Project"),
	)

	popup := widget.NewModalPopUp(dialog, g.window.Canvas())
	dialog.Resize(fyne.NewSize(400, 300))
	popup.Show()
}

// fixedHeightLayout implements a layout that enforces a fixed height
type fixedHeightLayout struct {
	height float32
}

func (f *fixedHeightLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(0, f.height)
	}
	minWidth := float32(0)
	for _, obj := range objects {
		if obj.MinSize().Width > minWidth {
			minWidth = obj.MinSize().Width
		}
	}
	return fyne.NewSize(minWidth, f.height)
}

func (f *fixedHeightLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, obj := range objects {
		obj.Move(fyne.NewPos(0, 0))
		obj.Resize(fyne.NewSize(size.Width, f.height))
	}
}
