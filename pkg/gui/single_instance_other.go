//go:build !windows
// +build !windows

package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// CheckSingleInstance ensures only one instance of the application is running
// Returns true if this is the first instance, false if another instance is already running
func CheckSingleInstance() bool {
	// Use file-based locking for Unix-like systems
	lockFile, err := CreateLockFile()
	if err != nil {
		DebugLog("INFO", fmt.Sprintf("Another instance might be running: %v", err))
		return false
	}

	// We successfully created the lock file, so we're the first instance
	// Note: The lock file will be automatically released when the process exits
	_ = lockFile // Keep reference to prevent GC
	DebugLog("INFO", "This is the first instance")
	return true
}

// CreateLockFile creates a lock file to prevent multiple instances
func CreateLockFile() (*os.File, error) {
	lockPath := filepath.Join(os.TempDir(), "fire-gui.lock")

	// Try to create the lock file
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}

	// Try to acquire an exclusive lock
	err = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		lockFile.Close()
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	// Write our PID to the lock file
	lockFile.Truncate(0)
	fmt.Fprintf(lockFile, "%d", os.Getpid())
	lockFile.Sync()

	return lockFile, nil
}

// RemoveLockFile removes the lock file
func RemoveLockFile(lockFile *os.File) {
	if lockFile != nil {
		// Release the lock
		syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
		lockFile.Close()
		os.Remove(lockFile.Name())
	}
}
