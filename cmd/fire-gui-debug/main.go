// Package main provides the debug build of the FIRE GUI with additional debugging endpoints.
package main

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2/app"
	"github.com/mscrnt/project_fire/pkg/gui"
)

func main() {
	fmt.Println("Starting F.I.R.E. GUI...")
	fmt.Println("==============================================")
	fmt.Println("Changes implemented:")
	fmt.Println("1. ✓ Removed traditional File/Edit/View/Help menu bar")
	fmt.Println("2. ✓ Added red header bar (RGB: 237, 28, 36) with centered 'F.I.R.E.' text")
	fmt.Println("3. ✓ Increased navigation icon size to 48x48 pixels")
	fmt.Println("4. ✓ Updated navigation buttons with vertical icon+text layout")
	fmt.Println("5. ✓ Dark navigation sidebar (RGB: 42, 42, 42)")
	fmt.Println("6. ✓ Navigation buttons: SYSTEM INFO, STABILITY TEST, BENCHMARK, MONITORING, SETTINGS, SUPPORT US")
	fmt.Println("")

	// Create Fyne application
	a := app.New()
	a.SetIcon(nil)

	// Create GUI
	g := gui.CreateFireGUI(a, nil)

	fmt.Println("GUI created successfully!")
	fmt.Println("Window should now display")

	// Add a timer to print status
	go func() {
		time.Sleep(2 * time.Second)
		fmt.Println("\nGUI is running...")
		fmt.Println("The window should show:")
		fmt.Println("- Red header bar at top")
		fmt.Println("- Dark left sidebar with large navigation buttons")
		fmt.Println("- Main content area on the right")
	}()

	// Show and run
	g.ShowAndRun()
}
