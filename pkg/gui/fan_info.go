package gui

import (
	"os/exec"
	"strconv"
	"strings"
)

// FanInfo contains information about a system fan
type FanInfo struct {
	Name  string
	Speed int    // RPM
	Type  string // CPU, GPU, Case
}

// GetFanInfo returns information about system fans
func GetFanInfo() ([]FanInfo, error) {
	var fans []FanInfo

	// Try to get fan info from sensors command (lm-sensors)
	cmd := exec.Command("sensors", "-u")
	output, err := cmd.Output()
	if err != nil {
		// If sensors not available, return empty list
		return fans, nil
	}

	// Parse sensors output
	lines := strings.Split(string(output), "\n")
	var currentFan FanInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for fan entries
		if strings.Contains(line, "fan") && strings.Contains(line, "_input:") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				speedStr := strings.TrimSpace(parts[1])
				speed, err := strconv.ParseFloat(speedStr, 64)
				if err == nil {
					fanName := strings.Split(parts[0], "_input")[0]
					currentFan = FanInfo{
						Name:  fanName,
						Speed: int(speed),
						Type:  "System",
					}

					// Determine fan type
					if strings.Contains(strings.ToLower(fanName), "cpu") {
						currentFan.Type = "CPU"
					} else if strings.Contains(strings.ToLower(fanName), "gpu") {
						currentFan.Type = "GPU"
					}

					fans = append(fans, currentFan)
				}
			}
		}
	}

	// Try to get GPU fan info from nvidia-smi
	gpuCmd := exec.Command("nvidia-smi", "--query-gpu=fan.speed", "--format=csv,noheader,nounits")
	gpuOutput, err := gpuCmd.Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(gpuOutput)), "\n")
		for i, line := range lines {
			speed, err := strconv.Atoi(strings.TrimSpace(line))
			if err == nil {
				fans = append(fans, FanInfo{
					Name:  "GPU " + strconv.Itoa(i) + " Fan",
					Speed: speed * 100, // Convert percentage to approximate RPM
					Type:  "GPU",
				})
			}
		}
	}

	return fans, nil
}
