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
	dashboard  *Dashboard
	testWizard *TestWizard
	history    *History
	compare    *Compare
	aiInsights *AIInsights
	certs      *Certificates

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
	// Set window size
	g.window.Resize(fyne.NewSize(1200, 800))
	g.window.CenterOnScreen()

	// Create menu
	g.createMenu()

	// Initialize components
	g.dashboard = NewDashboard()
	g.testWizard = NewTestWizard(g.dbPath)
	g.history = NewHistory(g.dbPath)
	g.compare = NewCompare(g.dbPath)
	g.aiInsights = NewAIInsights()
	g.certs = NewCertificates(g.dbPath)

	// Create tabs
	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon("Dashboard", theme.HomeIcon(), g.dashboard.Content()),
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
		g.dashboard.Stop()
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
	// Start dashboard monitoring
	g.dashboard.Start()

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
	if g.app.Settings().Theme() == theme.DarkTheme() {
		g.app.Settings().SetTheme(theme.LightTheme())
	} else {
		g.app.Settings().SetTheme(theme.DarkTheme())
	}
}

func (g *FireGUI) refresh() {
	// Refresh all components
	g.dashboard.Refresh()
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
