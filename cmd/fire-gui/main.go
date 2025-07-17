// Package main is the entry point for the FIRE GUI application.
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/mscrnt/project_fire/internal/version"
	"github.com/mscrnt/project_fire/pkg/gui"
	"github.com/mscrnt/project_fire/pkg/telemetry"
)

var (
	// Build variables set by ldflags
	buildVersion string
	buildCommit  string
	buildTime    string
)

func main() {
	os.Exit(run())
}

func run() int {
	// Add command-line flags
	debugStorage := flag.Bool("debug-storage", false, "Debug storage detection")
	clearLogs := flag.Bool("clear-logs", true, "Clear logs on startup")
	telemetryEnabled := flag.Bool("telemetry", true, "Enable anonymous telemetry for hardware compatibility")
	telemetryEndpoint := flag.String("telemetry-endpoint", "", "Custom telemetry endpoint")
	noSplash := flag.Bool("no-splash", false, "Skip startup splash screen")
	enableDebugServer := flag.Bool("debug-server", false, "Enable debug HTTP server on port 8888")
	flag.Parse()

	// Set app version for telemetry
	appVersion := version.GetVersion(buildVersion, buildCommit, buildTime)
	if appVersion == "dev-" || appVersion == "-" {
		appVersion = "v0.2.1" // Fallback version
	}
	telemetry.SetAppVersion(appVersion)

	// Initialize telemetry
	telemetry.Initialize(*telemetryEndpoint, "", *telemetryEnabled)

	// Set up panic handler
	defer func() {
		if rec := recover(); rec != nil {
			stack := make([]byte, 32<<10)
			n := runtime.Stack(stack, false)
			telemetry.RecordPanic(rec, stack[:n])
			telemetry.Shutdown()
			panic(rec) // Re-panic to maintain default behavior
		}
	}()

	// Ensure telemetry is flushed on normal exit
	defer telemetry.Shutdown()

	// Handle debug storage flag
	if *debugStorage {
		gui.DebugStorageInfo()
		return 0
	}

	// Check for single instance
	if !gui.CheckSingleInstance() {
		fmt.Println("F.I.R.E. GUI is already running!")
		fmt.Println("Please close the existing instance before starting a new one.")

		// Show a GUI dialog if possible
		myApp := app.NewWithID("com.fire.testbench")
		window := myApp.NewWindow("F.I.R.E. Already Running")
		window.Resize(fyne.NewSize(400, 150))
		window.CenterOnScreen()

		content := widget.NewLabel("F.I.R.E. GUI is already running!\n\nPlease close the existing instance before starting a new one.")
		content.Alignment = fyne.TextAlignCenter

		window.SetContent(container.NewCenter(content))
		window.ShowAndRun()

		return 1
	}

	// Clear logs on startup if requested (default: true)
	if *clearLogs {
		gui.ClearLogs()
	}

	// Set up fmt import
	fmt.Println("Starting F.I.R.E. GUI...")
	fmt.Printf("Starting at: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("Admin mode: %v\n", gui.IsRunningAsAdmin())

	// Initialize debug server if enabled
	if *enableDebugServer {
		debugSrv := gui.NewDebugServer(8888)
		gui.GlobalDebugServer = debugSrv
		go debugSrv.Start()
		gui.DebugLog("INFO", "Debug server started on port 8888")
	}
	gui.DebugLog("INFO", "Starting F.I.R.E. GUI...")
	gui.DebugLog("INFO", fmt.Sprintf("Admin mode: %v", gui.IsRunningAsAdmin()))

	// Add checkpoint
	gui.DebugCheckpoint("startup")

	// Fix locale issue in WSL/minimal environments
	// Set a minimal but valid locale that Fyne will accept
	lang := os.Getenv("LANG")
	if lang == "" || lang == "C" {
		// Use a valid locale format
		_ = os.Setenv("LANG", "en_US.UTF-8")
		_ = os.Setenv("LC_ALL", "en_US.UTF-8")
	}

	gui.DebugLog("INFO", "Creating Fyne application...")
	// Create the application
	myApp := app.NewWithID("com.fire.testbench")
	myApp.SetIcon(theme.ComputerIcon()) // TODO: Use custom icon

	// Apply FIRE theme
	myApp.Settings().SetTheme(gui.FireDarkTheme{})

	// Create main window immediately
	window := myApp.NewWindow("F.I.R.E. System Monitor")
	window.Resize(fyne.NewSize(1600, 900))
	window.CenterOnScreen()

	// Check admin status
	isAdmin := gui.IsRunningAsAdmin()
	if !isAdmin {
		gui.DebugLog("WARNING", "Not running as Administrator - some features will be limited")
	} else {
		gui.DebugLog("INFO", "Running with Administrator privileges")
	}

	var cache *gui.StaticCache

	if *noSplash {
		// No loading screen - create GUI immediately with empty cache
		gui.DebugLog("INFO", "Skipping loading screen...")
		fireGUI := gui.CreateFireGUI(myApp, nil)
		window.SetContent(fireGUI.Content())

		// Attach GUI to debug server if enabled
		if gui.GlobalDebugServer != nil {
			gui.GlobalDebugServer.SetGUI(fireGUI)
		}

		// Set close handler
		window.SetCloseIntercept(func() {
			gui.DebugLog("INFO", "Window close requested")
			fireGUI.GetDashboard().Stop()
			myApp.Quit()
		})

		// Start monitoring
		fireGUI.GetDashboard().Start()

		// Show admin warning after window loads
		go func() {
			time.Sleep(2 * time.Second)
			if !isAdmin {
				fyne.CurrentApp().SendNotification(&fyne.Notification{
					Title:   "Limited Functionality",
					Content: "Running without Administrator privileges. Some features like SPD memory reading will be unavailable.",
				})
			}
		}()
	} else {
		// Create loading overlay
		gui.DebugLog("INFO", "Creating loading overlay...")
		loadingOverlay, loadingLabel, progressBar := gui.CreateLoadingOverlay()
		window.SetContent(loadingOverlay)

		// Show window immediately with loading screen
		window.Show()

		// Create update channel
		updates := make(chan gui.Update)

		// Start background loader
		go func() {
			gui.DebugLog("INFO", "Starting component loading in background...")
			cache = gui.LoadComponentsAsync(updates)
			close(updates)
		}()

		// Consume updates and swap to real UI when done
		go func() {
			// Process progress updates
			for u := range updates {
				gui.DebugLog("LOADING_UI", fmt.Sprintf("Progress update: Step %d/%d - %s", u.Step, u.Total, u.Text))
				fyne.Do(func() {
					// Update RichText with larger font
					loadingLabel.ParseMarkdown("### " + u.Text)
					progressValue := float64(u.Step) / float64(u.Total)
					progressBar.SetValue(progressValue)
					progressBar.Refresh() // Trigger gradient update
					gui.DebugLog("LOADING_UI", fmt.Sprintf("Progress bar set to: %.2f%%", progressValue*100))
				})
			}

			// Small delay to show 100% completion
			time.Sleep(300 * time.Millisecond)

			// Swap in the real UI
			fyne.Do(func() {
				gui.DebugLog("INFO", "Loading complete, creating main GUI...")

				// Create the full GUI with cached data
				fireGUI := gui.CreateFireGUI(myApp, cache)

				// Replace window content with the real GUI
				window.SetContent(fireGUI.Content())

				// Attach GUI to debug server if enabled
				if gui.GlobalDebugServer != nil {
					gui.GlobalDebugServer.SetGUI(fireGUI)
				}

				// Set close handler
				window.SetCloseIntercept(func() {
					gui.DebugLog("INFO", "Window close requested")
					fireGUI.GetDashboard().Stop()
					myApp.Quit()
				})

				// Start dashboard monitoring
				fireGUI.GetDashboard().Start()

				// Show first navigation page
				fireGUI.Navigation().ShowPage(0)

				// Show admin warning if needed
				if !isAdmin {
					time.Sleep(1 * time.Second)
					fyne.CurrentApp().SendNotification(&fyne.Notification{
						Title:   "Limited Functionality",
						Content: "Running without Administrator privileges. Some features like SPD memory reading will be unavailable.",
					})
				}

				gui.DebugLog("INFO", "Main GUI ready")
				fmt.Println("✅ F.I.R.E. GUI ready")
			})
		}()
	}

	// Register debug callbacks if debug server is enabled
	if gui.GlobalDebugServer != nil {
		gui.GlobalDebugServer.RegisterCallback("test", func() {
			gui.DebugLog("INFO", "Test callback executed!")
		})

		gui.GlobalDebugServer.RegisterCallback("update_dashboard", func() {
			// Will be set once GUI is created
			gui.DebugLog("INFO", "Dashboard update requested")
		})
	}

	gui.DebugCheckpoint("pre-run")
	gui.DebugLog("INFO", "Starting main event loop...")

	// This is the ONLY event loop - everything else uses Show()
	window.ShowAndRun()

	gui.DebugLog("INFO", "ShowAndRun returned - GUI window closed")
	gui.DebugLog("INFO", "GUI exited normally")
	fmt.Println("🚪 GUI exited normally")

	return 0
}
