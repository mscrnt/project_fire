//go:build !windows
// +build !windows

// Package gui provides the graphical user interface for the FIRE benchmarking tool.
package gui

// IsRunningAsAdmin checks if the current process is running with administrator privileges
// On non-Windows systems, this returns true as admin checks are Windows-specific
func IsRunningAsAdmin() bool {
	return true
}

// GetAdminRequiredFeatures returns a list of features that require admin privileges
func GetAdminRequiredFeatures() []string {
	return []string{}
}
