//go:build !windows
// +build !windows

package gui

// GetDriveBusTypeEnhanced uses platform-specific methods to detect bus type (stub for non-Windows)
func GetDriveBusTypeEnhanced(driveLetter string) (string, error) {
	// On Linux, we would need to use different methods like:
	// - Reading from /sys/block/*/device/transport
	// - Using lsblk or other system utilities
	// For now, return empty as a stub
	return "", nil
}
