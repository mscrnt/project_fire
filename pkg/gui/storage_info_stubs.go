//go:build !windows
// +build !windows

package gui

import "fmt"

// GetWindowsDriveModelsV2 stub for non-Windows platforms
func GetWindowsDriveModelsV2() map[string]DriveModel {
	return make(map[string]DriveModel)
}

// GetDriveBusType stub for non-Windows platforms
func GetDriveBusType(_ string) (string, error) {
	return "", nil
}

// WindowsDriveMapping represents the mapping between physical disks and logical drives
type WindowsDriveMapping struct {
	DiskNumber      int    `json:"DiskNumber"`
	Model           string `json:"Model"`
	SerialNumber    string `json:"SerialNumber"`
	FirmwareVersion string `json:"FirmwareVersion"`
	MediaType       string `json:"MediaType"`
	BusType         string `json:"BusType"`
	DriveLetter     string `json:"DriveLetter"`
	VolumeName      string `json:"VolumeName"`
}

// GetWindowsDriveMappings stub for non-Windows platforms
func GetWindowsDriveMappings() ([]WindowsDriveMapping, error) {
	return nil, fmt.Errorf("drive mappings not supported on this platform")
}
