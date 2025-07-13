package gui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// MotherboardInfo contains motherboard information
type MotherboardInfo struct {
	Manufacturer string
	Model        string
	Version      string
	SerialNumber string
	BIOS         BIOSInfo
	Features     MotherboardFeatures
	ChipsetInfo  ChipsetInfo
}

// MotherboardFeatures contains motherboard feature information
type MotherboardFeatures struct {
	MemorySlots int
	MaxMemory   uint64
	PCIeSlots   int
	M2Slots     int
	SATAPorts   int
	USBPorts    map[string]int // Type -> Count
	FormFactor  string
}

// ChipsetInfo contains chipset information
type ChipsetInfo struct {
	Vendor string
	Model  string
}

// BIOSInfo contains BIOS information
type BIOSInfo struct {
	Vendor      string
	Version     string
	ReleaseDate string
}

// GetMotherboardInfo retrieves motherboard information
func GetMotherboardInfo() (*MotherboardInfo, error) {
	info := &MotherboardInfo{}

	switch runtime.GOOS {
	case "windows":
		return getMotherboardInfoWindows()
	case "linux":
		return getMotherboardInfoLinux()
	case "darwin":
		return getMotherboardInfoDarwin()
	default:
		return info, nil
	}
}

// getMotherboardInfoWindows gets motherboard info on Windows
func getMotherboardInfoWindows() (*MotherboardInfo, error) {
	info := &MotherboardInfo{}

	// Get motherboard info
	cmd := exec.Command("cmd", "/c", "wmic baseboard get manufacturer,product,version,serialnumber /value")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			switch {
			case strings.HasPrefix(line, "Manufacturer="):
				info.Manufacturer = strings.TrimPrefix(line, "Manufacturer=")
			case strings.HasPrefix(line, "Product="):
				info.Model = strings.TrimPrefix(line, "Product=")
			case strings.HasPrefix(line, "Version="):
				info.Version = strings.TrimPrefix(line, "Version=")
			case strings.HasPrefix(line, "SerialNumber="):
				info.SerialNumber = strings.TrimPrefix(line, "SerialNumber=")
			}
		}
	}

	// Get additional motherboard details
	info.Features = GetMotherboardFeatures()
	info.ChipsetInfo = GetChipsetInfo()

	// Get BIOS info
	cmd = exec.Command("cmd", "/c", "wmic bios get manufacturer,version,releasedate /value")
	output, err = cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			switch {
			case strings.HasPrefix(line, "Manufacturer="):
				info.BIOS.Vendor = strings.TrimPrefix(line, "Manufacturer=")
			case strings.HasPrefix(line, "Version="):
				info.BIOS.Version = strings.TrimPrefix(line, "Version=")
			case strings.HasPrefix(line, "ReleaseDate="):
				info.BIOS.ReleaseDate = strings.TrimPrefix(line, "ReleaseDate=")
			}
		}
	}

	return info, nil
}

// getMotherboardInfoLinux gets motherboard info on Linux
func getMotherboardInfoLinux() (*MotherboardInfo, error) {
	info := &MotherboardInfo{}

	// Try to read from DMI
	if data, err := readFile("/sys/class/dmi/id/board_vendor"); err == nil {
		info.Manufacturer = strings.TrimSpace(string(data))
	}
	if data, err := readFile("/sys/class/dmi/id/board_name"); err == nil {
		info.Model = strings.TrimSpace(string(data))
	}
	if data, err := readFile("/sys/class/dmi/id/board_version"); err == nil {
		info.Version = strings.TrimSpace(string(data))
	}
	if data, err := readFile("/sys/class/dmi/id/board_serial"); err == nil {
		info.SerialNumber = strings.TrimSpace(string(data))
	}

	// BIOS info
	if data, err := readFile("/sys/class/dmi/id/bios_vendor"); err == nil {
		info.BIOS.Vendor = strings.TrimSpace(string(data))
	}
	if data, err := readFile("/sys/class/dmi/id/bios_version"); err == nil {
		info.BIOS.Version = strings.TrimSpace(string(data))
	}
	if data, err := readFile("/sys/class/dmi/id/bios_date"); err == nil {
		info.BIOS.ReleaseDate = strings.TrimSpace(string(data))
	}

	// If DMI is not available, try dmidecode
	if info.Model == "" {
		cmd := exec.Command("dmidecode", "-t", "baseboard")
		if output, err := cmd.Output(); err == nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				switch {
				case strings.HasPrefix(line, "Manufacturer:"):
					info.Manufacturer = strings.TrimSpace(strings.TrimPrefix(line, "Manufacturer:"))
				case strings.HasPrefix(line, "Product Name:"):
					info.Model = strings.TrimSpace(strings.TrimPrefix(line, "Product Name:"))
				case strings.HasPrefix(line, "Version:"):
					info.Version = strings.TrimSpace(strings.TrimPrefix(line, "Version:"))
				}
			}
		}
	}

	return info, nil
}

// getMotherboardInfoDarwin gets motherboard info on macOS
func getMotherboardInfoDarwin() (*MotherboardInfo, error) {
	info := &MotherboardInfo{}

	// Use system_profiler for hardware info
	cmd := exec.Command("system_profiler", "SPHardwareDataType")
	if output, err := cmd.Output(); err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			switch {
			case strings.HasPrefix(line, "Model Name:"):
				info.Model = strings.TrimSpace(strings.TrimPrefix(line, "Model Name:"))
			case strings.HasPrefix(line, "Model Identifier:"):
				info.Version = strings.TrimSpace(strings.TrimPrefix(line, "Model Identifier:"))
			case strings.HasPrefix(line, "Serial Number"):
				info.SerialNumber = strings.TrimSpace(strings.TrimPrefix(line, "Serial Number (system):"))
			}
		}
		info.Manufacturer = "Apple Inc."
	}

	return info, nil
}

// readFile is a helper to read file contents
func readFile(path string) ([]byte, error) {
	return exec.Command("cat", path).Output()
}

// FormatBIOSDate formats a BIOS date string to a more readable format
func FormatBIOSDate(dateStr string) string {
	// Handle WMI date format: YYYYMMDD000000.000000+000
	if len(dateStr) >= 8 && !strings.Contains(dateStr, "/") && !strings.Contains(dateStr, "-") {
		// Extract YYYYMMDD
		year := dateStr[0:4]
		month := dateStr[4:6]
		day := dateStr[6:8]

		// Convert month number to name
		monthNames := []string{"", "January", "February", "March", "April", "May", "June",
			"July", "August", "September", "October", "November", "December"}

		if monthNum, err := strconv.Atoi(month); err == nil && monthNum >= 1 && monthNum <= 12 {
			return fmt.Sprintf("%s %s, %s", monthNames[monthNum], day, year)
		}

		return fmt.Sprintf("%s-%s-%s", year, month, day)
	}

	// Handle DD/MM/YYYY or MM/DD/YYYY format
	if strings.Contains(dateStr, "/") {
		parts := strings.Split(dateStr, "/")
		if len(parts) == 3 {
			// Assume MM/DD/YYYY for US format
			if len(parts[2]) == 4 {
				month, _ := strconv.Atoi(parts[0])
				day := parts[1]
				year := parts[2]

				monthNames := []string{"", "January", "February", "March", "April", "May", "June",
					"July", "August", "September", "October", "November", "December"}

				if month >= 1 && month <= 12 {
					return fmt.Sprintf("%s %s, %s", monthNames[month], day, year)
				}
			}
		}
	}

	// Return as-is if we can't parse it
	return dateStr
}

// GetMotherboardFeatures gets detailed motherboard features
func GetMotherboardFeatures() MotherboardFeatures {
	features := MotherboardFeatures{
		USBPorts: make(map[string]int),
	}

	if runtime.GOOS == "windows" || isWSL() {
		// Get memory slot information
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/c", "wmic memorychip get DeviceLocator /value | find /c \"DIMM\"")
		} else {
			cmd = exec.Command("cmd.exe", "/c", "wmic memorychip get DeviceLocator /value | find /c \"DIMM\"")
		}

		if output, err := cmd.Output(); err == nil {
			if count, err := strconv.Atoi(strings.TrimSpace(string(output))); err == nil {
				features.MemorySlots = count
			}
		}

		// Get system info for max memory
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/c", "wmic computersystem get TotalPhysicalMemory,MaxCapacity /value")
		} else {
			cmd = exec.Command("cmd.exe", "/c", "wmic computersystem get TotalPhysicalMemory,MaxCapacity /value")
		}

		if output, err := cmd.Output(); err == nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "MaxCapacity=") {
					maxCapStr := strings.TrimPrefix(line, "MaxCapacity=")
					if maxCap, err := strconv.ParseUint(maxCapStr, 10, 64); err == nil {
						features.MaxMemory = maxCap * 1024 // Convert KB to bytes
					}
				}
			}
		}
	}

	return features
}

// GetChipsetInfo gets chipset information
func GetChipsetInfo() ChipsetInfo {
	info := ChipsetInfo{}

	if runtime.GOOS == "windows" || isWSL() {
		// Try to get chipset info from system devices
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/c", "wmic path Win32_IDEController get Name /value")
		} else {
			cmd = exec.Command("cmd.exe", "/c", "wmic path Win32_IDEController get Name /value")
		}

		if output, err := cmd.Output(); err == nil {
			outputStr := string(output)
			// Look for common chipset indicators
			if strings.Contains(outputStr, "Intel") {
				info.Vendor = "Intel"
				// Extract chipset model from controller name
				switch {
				case strings.Contains(outputStr, "Z790"):
					info.Model = "Z790"
				case strings.Contains(outputStr, "Z690"):
					info.Model = "Z690"
				case strings.Contains(outputStr, "B660"):
					info.Model = "B660"
				case strings.Contains(outputStr, "H670"):
					info.Model = "H670"
				case strings.Contains(outputStr, "X670"):
					info.Model = "X670"
				case strings.Contains(outputStr, "B650"):
					info.Model = "B650"
				}
			} else if strings.Contains(outputStr, "AMD") {
				info.Vendor = "AMD"
				// Extract AMD chipset models
				switch {
				case strings.Contains(outputStr, "X670"):
					info.Model = "X670"
				case strings.Contains(outputStr, "B650"):
					info.Model = "B650"
				case strings.Contains(outputStr, "X570"):
					info.Model = "X570"
				case strings.Contains(outputStr, "B550"):
					info.Model = "B550"
				}
			}
		}
	}

	return info
}
