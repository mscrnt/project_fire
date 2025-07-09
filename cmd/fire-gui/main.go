package main

import (
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"
	"github.com/mscrnt/project_fire/pkg/gui"
)

func main() {
	// Create the application
	myApp := app.NewWithID("com.fire.testbench")
	myApp.SetIcon(theme.ComputerIcon()) // TODO: Use custom icon

	// Create and run the GUI
	fireGUI := gui.NewFireGUI(myApp)
	fireGUI.ShowAndRun()
}
