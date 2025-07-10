package main

import (
	"os"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"
	"github.com/mscrnt/project_fire/pkg/gui"
)

func main() {
	// Fix locale issue in WSL/minimal environments
	// Set a minimal but valid locale that Fyne will accept
	lang := os.Getenv("LANG")
	if lang == "" || lang == "C" {
		// Use a valid locale format
		os.Setenv("LANG", "en_US.UTF-8")
		os.Setenv("LC_ALL", "en_US.UTF-8")
	}

	// Create the application
	myApp := app.NewWithID("com.fire.testbench")
	myApp.SetIcon(theme.ComputerIcon()) // TODO: Use custom icon

	// Create and run the GUI
	fireGUI := gui.NewFireGUI(myApp)
	fireGUI.ShowAndRun()
}
