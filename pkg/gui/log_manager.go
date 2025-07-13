package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ClearLogs clears or archives old log files
func ClearLogs() {
	logFiles := []string{
		"gui_debug.log",
		"perf.log",
		"fire-gui.log",
	}

	// Get the directory where the executable is located
	exePath, err := os.Executable()
	if err != nil {
		fmt.Printf("Error getting executable path: %v\n", err)
		return
	}
	exeDir := filepath.Dir(exePath)

	// Create logs directory if it doesn't exist
	logsDir := filepath.Join(exeDir, "logs")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		fmt.Printf("Error creating logs directory: %v\n", err)
	}

	// Archive old logs with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")

	for _, logFile := range logFiles {
		logPath := filepath.Join(exeDir, logFile)

		// Check if log file exists
		if info, err := os.Stat(logPath); err == nil && info.Size() > 0 {
			// Archive the log file
			archivePath := filepath.Join(logsDir, fmt.Sprintf("%s_%s.log",
				logFile[:len(logFile)-4], timestamp))

			// Move or copy the file
			if err := os.Rename(logPath, archivePath); err != nil {
				// If rename fails (cross-device), try copy
				if err := copyFile(logPath, archivePath); err == nil {
					os.Remove(logPath)
					fmt.Printf("Archived %s to %s\n", logFile, archivePath)
				} else {
					// If copy also fails, just truncate the file
					if file, err := os.OpenFile(logPath, os.O_TRUNC, 0o644); err == nil {
						file.Close()
						fmt.Printf("Cleared %s\n", logFile)
					}
				}
			} else {
				fmt.Printf("Archived %s to %s\n", logFile, archivePath)
			}
		}
	}

	// Create new empty log files
	for _, logFile := range logFiles {
		logPath := filepath.Join(exeDir, logFile)
		if file, err := os.Create(logPath); err == nil {
			fmt.Fprintf(file, "# %s - Created %s\n", logFile, time.Now().Format("2006-01-02 15:04:05"))
			file.Close()
		}
	}
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	err = os.WriteFile(dst, input, 0600)
	if err != nil {
		return err
	}

	return nil
}

// GetLogPath returns the path to a log file
func GetLogPath(filename string) string {
	exePath, err := os.Executable()
	if err != nil {
		return filename
	}
	return filepath.Join(filepath.Dir(exePath), filename)
}
