package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// FireGUI represents the main GUI application
type FireGUI struct {
	app    fyne.App
	window fyne.Window

	// Main content containers
	dashboard   *Dashboard
	dashboardV2 *DashboardV2
	testWizard  *TestWizard
	history     *History
	compare     *Compare
	aiInsights  *AIInsights
	certs       *Certificates

	// Current database path
	dbPath string
}

// NewFireGUI creates a new F.I.R.E. GUI instance
func NewFireGUI(app fyne.App) *FireGUI {
	gui := &FireGUI{
		app:    app,
		window: app.NewWindow("F.I.R.E. Test Bench"),
		dbPath: getDefaultDBPath(),
	}

	gui.setup()
	return gui
}

// setup initializes the GUI layout
func (g *FireGUI) setup() {
	// Apply custom F.I.R.E. theme
	g.app.Settings().SetTheme(FireTheme{})

	// Set window size
	g.window.Resize(fyne.NewSize(1400, 900))
	g.window.CenterOnScreen()

	// Create menu
	g.createMenu()

	// Initialize components
	g.dashboard = NewDashboard()     // Keep for compatibility
	g.dashboardV2 = NewDashboardV2() // Enhanced dashboard
	g.testWizard = NewTestWizard(g.dbPath)
	g.history = NewHistory(g.dbPath)
	g.compare = NewCompare(g.dbPath)
	g.aiInsights = NewAIInsights()
	g.certs = NewCertificates(g.dbPath)

	// Create tabs - use enhanced dashboard
	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon("Dashboard", theme.HomeIcon(), g.dashboardV2.Content()),
		container.NewTabItemWithIcon("Test Wizard", theme.DocumentCreateIcon(), g.testWizard.Content()),
		container.NewTabItemWithIcon("History", theme.ListIcon(), g.history.Content()),
		container.NewTabItemWithIcon("Compare", theme.ContentCopyIcon(), g.compare.Content()),
		container.NewTabItemWithIcon("AI Insights", theme.ComputerIcon(), g.aiInsights.Content()),
		container.NewTabItemWithIcon("Certificates", theme.DocumentIcon(), g.certs.Content()),
	)

	// Set content
	g.window.SetContent(tabs)

	// Set close handler
	g.window.SetCloseIntercept(func() {
		g.dashboardV2.Stop()
		g.window.Close()
	})
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
	// Start enhanced dashboard monitoring
	g.dashboardV2.Start()

	// Show window and run
	g.window.ShowAndRun()
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
	// Refresh all components
	g.dashboardV2.updateAll()
	g.history.Refresh()
	g.compare.Refresh()
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
