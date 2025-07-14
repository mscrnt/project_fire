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
	"github.com/mscrnt/project_fire/pkg/gui"
	"github.com/mscrnt/project_fire/pkg/telemetry"
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
	flag.Parse()

	// Set app version for telemetry
	telemetry.SetAppVersion("v0.1.1") // GUI version

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

	// Initialize debug server first
	debugServer := gui.NewDebugServer(8888)
	gui.GlobalDebugServer = debugServer
	go debugServer.Start()
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

	gui.DebugCheckpoint("pre-gui")
	gui.DebugLog("INFO", "Creating FireGUI...")

	// Create and run the GUI
	fireGUI := gui.NewFireGUI(myApp)

	// Attach GUI to debug server
	fmt.Println("Attaching GUI to debug server...")
	debugServer.SetGUI(fireGUI)

	// Register some useful callbacks
	debugServer.RegisterCallback("test", func() {
		gui.DebugLog("INFO", "Test callback executed!")
	})

	debugServer.RegisterCallback("update_dashboard", func() {
		if fireGUI.GetDashboard() != nil {
			go fireGUI.GetDashboard().UpdateMetrics()
		}
	})

	gui.DebugCheckpoint("pre-run")
	gui.DebugLog("INFO", "Calling ShowAndRun...")
	fmt.Println("About to call ShowAndRun...")

	// Add a goroutine to monitor if the app is hanging
	go func() {
		time.Sleep(5 * time.Second)
		gui.DebugLog("DEBUG", "App still running after 5 seconds...")
		fmt.Println("App still running after 5 seconds...")
	}()

	// Add a panic handler to catch any crashes
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("PANIC: %v\n", r)
			gui.DebugLog("PANIC", fmt.Sprintf("Recovered from panic: %v", r))
		}
	}()

	gui.DebugLog("DEBUG", "Calling fireGUI.ShowAndRun()...")
	fireGUI.ShowAndRun()

	gui.DebugLog("INFO", "GUI exited normally")
	fmt.Println("GUI exited normally")

	return 0
}
