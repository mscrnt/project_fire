package gui

import (
	"fmt"
	"log"
	"os"
	"time"
)

// GlobalDebugServer is the global debug server instance
var GlobalDebugServer *DebugServer

// DebugLog logs debug messages
func DebugLog(level, format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	message := fmt.Sprintf(format, args...)
	log.Printf("[%s] %s: %s", timestamp, level, message)

	// Write to appropriate log file
	var logFile string
	switch level {
	case "PERF":
		logFile = "perf.log"
	case "SPD", "MEMORY", "STORAGE":
		logFile = "fire-gui.log"
	default:
		logFile = "gui_debug.log"
	}

	logPath := GetLogPath(logFile)
	if f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		fmt.Fprintf(f, "[%s] %s: %s\n", timestamp, level, message)
		f.Close()
	}
}

// DebugCheckpoint logs a checkpoint
func DebugCheckpoint(name string) {
	DebugLog("CHECKPOINT", name)
}
