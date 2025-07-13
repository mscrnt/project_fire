package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

func guiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gui",
		Short: "Launch the graphical user interface",
		Long: `Launch the F.I.R.E. graphical user interface.

The GUI provides:
- Live system monitoring dashboard
- Test configuration wizard
- Test history and results
- Run comparisons
- AI-powered test planning
- Certificate management

Note: The GUI requires a graphical environment (X11, Wayland, or Windows/macOS desktop).`,
		RunE: func(_ *cobra.Command, _ []string) error {
			// Check if we're in a GUI environment
			if !hasGUIEnvironment() {
				return fmt.Errorf("GUI environment not detected. The GUI requires a graphical desktop environment")
			}

			// Build the GUI binary path
			guiBinary := "fire-gui"
			if runtime.GOOS == "windows" {
				guiBinary += ".exe"
			}

			// Try to find the GUI binary
			// First check in the same directory as the CLI
			execPath, err := os.Executable()
			if err == nil {
				dir := filepath.Dir(execPath)
				guiPath := filepath.Join(dir, guiBinary)
				if _, err := os.Stat(guiPath); err == nil {
					return runGUI(guiPath)
				}
			}

			// Check in PATH
			guiPath, err := exec.LookPath(guiBinary)
			if err == nil {
				return runGUI(guiPath)
			}

			// GUI not found
			return fmt.Errorf("GUI binary '%s' not found. Please ensure it's built and in your PATH", guiBinary)
		},
	}

	return cmd
}

// hasGUIEnvironment checks if a GUI environment is available
func hasGUIEnvironment() bool {
	switch runtime.GOOS {
	case "windows":
		return true // Windows always has GUI
	case "darwin":
		return true // macOS always has GUI
	case "linux", "freebsd", "openbsd", "netbsd":
		// Check for X11 or Wayland
		display := os.Getenv("DISPLAY")
		waylandDisplay := os.Getenv("WAYLAND_DISPLAY")
		return display != "" || waylandDisplay != ""
	default:
		return false
	}
}

// runGUI launches the GUI binary
func runGUI(path string) error {
	cmd := exec.Command(path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the GUI
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start GUI: %w", err)
	}

	fmt.Printf("GUI launched (PID: %d)\n", cmd.Process.Pid)
	fmt.Println("The GUI is running in a separate window.")

	// Don't wait for the GUI to finish
	return nil
}
