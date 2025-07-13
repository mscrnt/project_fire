package main

import (
	"os"
	"path/filepath"
)

// getDBPath returns the path to the F.I.R.E. database file
func getDBPath() string {
	// Check environment variable first
	if dbPath := os.Getenv("FIRE_DB_PATH"); dbPath != "" {
		return dbPath
	}

	// Default to user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		return "fire.db"
	}

	// Create .fire directory if it doesn't exist
	fireDir := filepath.Join(homeDir, ".fire")
	if err := os.MkdirAll(fireDir, 0o755); err == nil {
		return filepath.Join(fireDir, "fire.db")
	}

	// Fallback to current directory
	return "fire.db"
}
