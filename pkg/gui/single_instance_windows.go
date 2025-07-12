//go:build windows
// +build windows

package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	// Use existing kernel32 from admin_check_windows.go
	user32   = windows.NewLazySystemDLL("user32.dll")

	procCreateMutex     = kernel32.NewProc("CreateMutexW")
	procGetLastError    = kernel32.NewProc("GetLastError")
	procFindWindow      = user32.NewProc("FindWindowW")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procShowWindow      = user32.NewProc("ShowWindow")
)

const (
	ERROR_ALREADY_EXISTS = 183
	SW_RESTORE          = 9
)

// CheckSingleInstance ensures only one instance of the application is running
// Returns true if this is the first instance, false if another instance is already running
func CheckSingleInstance() bool {
	// Create a unique mutex name for our application
	mutexName := "Global\\FireGUI_SingleInstance_Mutex"
	
	// Convert string to UTF16
	mutexNamePtr, err := windows.UTF16PtrFromString(mutexName)
	if err != nil {
		DebugLog("ERROR", fmt.Sprintf("Failed to create mutex name: %v", err))
		return true // Allow to continue on error
	}

	// Try to create the mutex
	ret, _, err := procCreateMutex.Call(
		0,
		0,
		uintptr(unsafe.Pointer(mutexNamePtr)),
	)

	if ret == 0 {
		DebugLog("ERROR", fmt.Sprintf("Failed to create mutex: %v", err))
		return true // Allow to continue on error
	}

	// Check if mutex already exists
	lastErr, _, _ := procGetLastError.Call()
	if lastErr == ERROR_ALREADY_EXISTS {
		DebugLog("INFO", "Another instance is already running")
		
		// Try to find and bring the existing window to front
		className, _ := windows.UTF16PtrFromString("FyneWindow")
		windowName, _ := windows.UTF16PtrFromString("F.I.R.E. System Monitor")
		
		hwnd, _, _ := procFindWindow.Call(
			uintptr(unsafe.Pointer(className)),
			uintptr(unsafe.Pointer(windowName)),
		)
		
		if hwnd != 0 {
			// Restore window if minimized
			procShowWindow.Call(hwnd, SW_RESTORE)
			// Bring window to foreground
			procSetForegroundWindow.Call(hwnd)
			DebugLog("INFO", "Brought existing instance to foreground")
		}
		
		return false
	}

	DebugLog("INFO", "This is the first instance")
	return true
}

// CreateLockFile creates a lock file to prevent multiple instances (fallback method)
func CreateLockFile() (*os.File, error) {
	lockPath := filepath.Join(os.TempDir(), "fire-gui.lock")
	
	// Try to create the lock file exclusively
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		if os.IsExist(err) {
			// Lock file exists, another instance might be running
			// Try to check if the process is actually running by writing to it
			if testFile, testErr := os.OpenFile(lockPath, os.O_WRONLY, 0600); testErr == nil {
				testFile.Close()
				// We can write to it, so the other process is gone
				os.Remove(lockPath)
				// Try again
				lockFile, err = os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			}
		}
	}
	
	if err != nil {
		return nil, err
	}
	
	// Write our PID to the lock file
	fmt.Fprintf(lockFile, "%d", os.Getpid())
	lockFile.Sync()
	
	return lockFile, nil
}

// RemoveLockFile removes the lock file
func RemoveLockFile(lockFile *os.File) {
	if lockFile != nil {
		lockFile.Close()
		os.Remove(lockFile.Name())
	}
}