package gui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
	
	"github.com/mscrnt/project_fire/pkg/telemetry"
)

// GPUInfo holds GPU information
type GPUInfo struct {
	Vendor      string  // NVIDIA, AMD, Intel
	Name        string  // Model name
	Index       int     // GPU index
	Temperature float64 // Celsius
	MemoryUsed  uint64  // Bytes
	MemoryTotal uint64  // Bytes
	Utilization float64 // Percentage 0-100
	PowerDraw   float64 // Watts
	PowerLimit  float64 // Watts
	FanSpeed    float64 // Percentage 0-100
}

// GetGPUInfo returns information about all available GPUs
func GetGPUInfo() ([]GPUInfo, error) {
	var gpus []GPUInfo
	detectedGPUs := make(map[string]bool) // Track detected GPUs to avoid duplicates

	// Check if running on Windows
	if isWindows() || isWSL() {
		// Get Windows GPUs including integrated
		windowsGPUs := getWindowsGPUs()
		for _, gpu := range windowsGPUs {
			key := fmt.Sprintf("%s_%s", gpu.Vendor, gpu.Name)
			if !detectedGPUs[key] {
				gpus = append(gpus, gpu)
				detectedGPUs[key] = true
			}
		}
	} else {
		// Try NVIDIA GPUs first
		nvidiaGPUs := getNVIDIAGPUs()
		for _, gpu := range nvidiaGPUs {
			key := fmt.Sprintf("%s_%s", gpu.Vendor, gpu.Name)
			if !detectedGPUs[key] {
				gpus = append(gpus, gpu)
				detectedGPUs[key] = true
			}
		}

		// Try AMD GPUs
		amdGPUs := getAMDGPUs()
		for _, gpu := range amdGPUs {
			key := fmt.Sprintf("%s_%s", gpu.Vendor, gpu.Name)
			if !detectedGPUs[key] {
				gpus = append(gpus, gpu)
				detectedGPUs[key] = true
			}
		}

		// Get all GPUs from lspci (includes integrated)
		lspciGPUs := getAllGPUsFromLspci()
		for _, gpu := range lspciGPUs {
			key := fmt.Sprintf("%s_%s", gpu.Vendor, gpu.Name)
			if !detectedGPUs[key] {
				gpus = append(gpus, gpu)
				detectedGPUs[key] = true
			}
		}

		// Try Intel GPU detection
		intelGPUs := getIntelGPUs()
		for _, gpu := range intelGPUs {
			key := fmt.Sprintf("%s_%s", gpu.Vendor, gpu.Name)
			if !detectedGPUs[key] {
				gpus = append(gpus, gpu)
				detectedGPUs[key] = true
			}
		}
	}

	// Re-index GPUs
	for i := range gpus {
		gpus[i].Index = i
	}

	return gpus, nil
}

// getNVIDIAGPUs queries NVIDIA GPUs using nvidia-smi
func getNVIDIAGPUs() []GPUInfo {
	var gpus []GPUInfo

	// Check if nvidia-smi is available with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "nvidia-smi", "--query-gpu=index,name,temperature.gpu,memory.used,memory.total,utilization.gpu,power.draw,power.limit,fan.speed", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		return gpus // nvidia-smi not available or no NVIDIA GPU
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, ", ")
		if len(parts) < 9 {
			continue
		}

		gpu := GPUInfo{
			Vendor: "NVIDIA",
		}

		// Parse each field
		if idx, err := strconv.Atoi(parts[0]); err == nil {
			gpu.Index = idx
		}
		gpu.Name = parts[1]
		if temp, err := strconv.ParseFloat(parts[2], 64); err == nil {
			gpu.Temperature = temp
		}
		if memUsed, err := strconv.ParseUint(parts[3], 10, 64); err == nil {
			gpu.MemoryUsed = memUsed * 1024 * 1024 // Convert MB to bytes
		}
		if memTotal, err := strconv.ParseUint(parts[4], 10, 64); err == nil {
			gpu.MemoryTotal = memTotal * 1024 * 1024 // Convert MB to bytes
		}
		if util, err := strconv.ParseFloat(parts[5], 64); err == nil {
			gpu.Utilization = util
		}
		if power, err := strconv.ParseFloat(parts[6], 64); err == nil {
			gpu.PowerDraw = power
		}
		if powerLimit, err := strconv.ParseFloat(parts[7], 64); err == nil {
			gpu.PowerLimit = powerLimit
		}
		if fan, err := strconv.ParseFloat(parts[8], 64); err == nil {
			gpu.FanSpeed = fan
		}

		gpus = append(gpus, gpu)
	}

	return gpus
}

// getAMDGPUs queries AMD GPUs using rocm-smi or radeontop
func getAMDGPUs() []GPUInfo {
	// Try rocm-smi first (for newer AMD GPUs with ROCm support)
	if rocmGPUs := getAMDGPUsROCm(); len(rocmGPUs) > 0 {
		return rocmGPUs
	}

	// Try using radeontop for older AMD GPUs
	if radeonGPUs := getAMDGPUsRadeonTop(); len(radeonGPUs) > 0 {
		return radeonGPUs
	}

	// Try reading from sysfs for basic AMD GPU info
	return getAMDGPUsSysfs()
}

// getAMDGPUsROCm uses rocm-smi to get AMD GPU info
func getAMDGPUsROCm() []GPUInfo {
	var gpus []GPUInfo

	// Check if rocm-smi is available with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "rocm-smi", "--showtemp", "--showuse", "--showmeminfo", "vram", "--json")
	_, err := cmd.Output()
	if err != nil {
		return gpus
	}

	// Try simpler command format with timeout
	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()

	cmd = exec.CommandContext(ctx2, "rocm-smi", "-a")
	output, err := cmd.Output()
	if err != nil {
		return gpus
	}

	// Parse the text output to extract GPU information
	lines := strings.Split(string(output), "\n")
	var currentGPU *GPUInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "GPU[") {
			if currentGPU != nil {
				gpus = append(gpus, *currentGPU)
			}
			currentGPU = &GPUInfo{
				Vendor: "AMD",
				Name:   "AMD GPU",
			}
			// Extract GPU index
			if idx := strings.Index(line, "["); idx >= 0 {
				if endIdx := strings.Index(line[idx:], "]"); endIdx > 0 {
					if num, err := strconv.Atoi(line[idx+1 : idx+endIdx]); err == nil {
						currentGPU.Index = num
					}
				}
			}
		} else if currentGPU != nil {
			switch {
			case strings.Contains(line, "GPU use"):
				if parts := strings.Fields(line); len(parts) >= 3 {
					if util, err := strconv.ParseFloat(strings.TrimSuffix(parts[2], "%"), 64); err == nil {
						currentGPU.Utilization = util
					}
				}
			case strings.Contains(line, "Temperature") && strings.Contains(line, "edge"):
				if parts := strings.Fields(line); len(parts) >= 3 {
					if temp, err := strconv.ParseFloat(strings.TrimSuffix(parts[2], "c"), 64); err == nil {
						currentGPU.Temperature = temp
					}
				}
			case strings.Contains(line, "vram Total"):
				if parts := strings.Fields(line); len(parts) >= 3 {
					if mem, err := strconv.ParseUint(parts[2], 10, 64); err == nil {
						currentGPU.MemoryTotal = mem * 1024 * 1024 // MB to bytes
					}
				}
			case strings.Contains(line, "vram Used"):
				if parts := strings.Fields(line); len(parts) >= 3 {
					if mem, err := strconv.ParseUint(parts[2], 10, 64); err == nil {
						currentGPU.MemoryUsed = mem * 1024 * 1024 // MB to bytes
					}
				}
			}
		}
	}

	if currentGPU != nil {
		gpus = append(gpus, *currentGPU)
	}

	return gpus
}

// getAMDGPUsRadeonTop uses radeontop to get basic AMD GPU usage
func getAMDGPUsRadeonTop() []GPUInfo {
	var gpus []GPUInfo

	// radeontop requires root and may not be available
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "radeontop", "-d", "-", "-l", "1")
	output, err := cmd.Output()
	if err != nil {
		return gpus
	}

	// Parse radeontop output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "gpu") {
			gpu := GPUInfo{
				Vendor: "AMD",
				Name:   "AMD Radeon",
				Index:  0,
			}

			// Extract GPU usage percentage
			if idx := strings.Index(line, "gpu "); idx >= 0 {
				remaining := line[idx+4:]
				if pctIdx := strings.Index(remaining, "%"); pctIdx > 0 {
					if util, err := strconv.ParseFloat(strings.TrimSpace(remaining[:pctIdx]), 64); err == nil {
						gpu.Utilization = util
					}
				}
			}

			gpus = append(gpus, gpu)
			break
		}
	}

	return gpus
}

// getAMDGPUsSysfs reads AMD GPU info from sysfs
func getAMDGPUsSysfs() []GPUInfo {
	var gpus []GPUInfo

	// Look for AMD GPU in /sys/class/drm/
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ls", "/sys/class/drm/")
	output, err := cmd.Output()
	if err != nil {
		return gpus
	}

	cards := strings.Split(string(output), "\n")
	gpuIndex := 0

	for _, card := range cards {
		card = strings.TrimSpace(card)
		if !strings.HasPrefix(card, "card") || strings.Contains(card, "-") {
			continue
		}

		// Check if it's an AMD GPU
		vendorPath := fmt.Sprintf("/sys/class/drm/%s/device/vendor", card)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		vendorCmd := exec.CommandContext(ctx, "cat", vendorPath) // #nosec G204 - vendorPath is constructed from safe directory listing // #nosec G204 - vendorPath is constructed from safe directory listing
		vendorOutput, err := vendorCmd.Output()
		cancel()
		if err != nil {
			continue
		}

		vendor := strings.TrimSpace(string(vendorOutput))
		if vendor != "0x1002" { // AMD vendor ID
			continue
		}

		gpu := GPUInfo{
			Vendor: "AMD",
			Name:   "AMD GPU",
			Index:  gpuIndex,
		}

		// Try to get more specific name from device ID
		devicePath := fmt.Sprintf("/sys/class/drm/%s/device/device", card)
		ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
		deviceCmd := exec.CommandContext(ctx2, "cat", devicePath) // #nosec G204 - devicePath is constructed from safe directory listing
		deviceOutput, err := deviceCmd.Output()
		cancel2()
		if err == nil {
			deviceID := strings.TrimSpace(string(deviceOutput))
			// Check for common AMD APU/integrated GPU device IDs
			switch deviceID {
			case "0x1638", "0x1636": // Cezanne (Ryzen 5000 series)
				gpu.Name = "AMD Radeon Graphics (Cezanne, Integrated)"
			case "0x164c", "0x1681": // Rembrandt (Ryzen 6000 series)
				gpu.Name = "AMD Radeon Graphics (Rembrandt, Integrated)"
			case "0x15d8", "0x15dd": // Raven/Picasso (Ryzen 2000/3000 series)
				gpu.Name = "AMD Radeon Vega Graphics (Integrated)"
			case "0x1506", "0x1507": // Mendocino
				gpu.Name = "AMD Radeon Graphics (Mendocino, Integrated)"
			case "0x15e7", "0x15ff": // Phoenix (Ryzen 7000 series)
				gpu.Name = "AMD Radeon Graphics (Phoenix, Integrated)"
			default:
				// Record unknown AMD GPU device ID
				telemetry.RecordHardwareMiss("AMDGPUDeviceID", map[string]interface{}{
					"device_id": deviceID,
					"vendor": "AMD",
				})
				// Try to get name from lspci for this specific device
				if gpuInfo := getGPUNameFromLspci(card); gpuInfo != "" {
					gpu.Name = gpuInfo
				}
			}
		}

		// Try to get temperature
		hwmonPath := fmt.Sprintf("/sys/class/drm/%s/device/hwmon/", card)
		ctx3, cancel3 := context.WithTimeout(context.Background(), 2*time.Second)
		hwmonCmd := exec.CommandContext(ctx3, "ls", hwmonPath) // #nosec G204 - hwmonPath is constructed from safe directory listing
		hwmonOutput, err := hwmonCmd.Output()
		cancel3()
		if err == nil {
			hwmons := strings.Split(strings.TrimSpace(string(hwmonOutput)), "\n")
			if len(hwmons) > 0 {
				tempPath := fmt.Sprintf("%s%s/temp1_input", hwmonPath, hwmons[0])
				ctx4, cancel4 := context.WithTimeout(context.Background(), 2*time.Second)
				tempCmd := exec.CommandContext(ctx4, "cat", tempPath) // #nosec G204 - tempPath is constructed from safe directory listing
				tempOutput, err := tempCmd.Output()
				cancel4()
				if err == nil {
					if temp, err := strconv.ParseFloat(strings.TrimSpace(string(tempOutput)), 64); err == nil {
						gpu.Temperature = temp / 1000.0 // Convert from millidegrees
					}
				}
			}
		}

		// Try to get memory info
		memInfoPath := fmt.Sprintf("/sys/class/drm/%s/device/mem_info_vram_total", card)
		ctx5, cancel5 := context.WithTimeout(context.Background(), 2*time.Second)
		memCmd := exec.CommandContext(ctx5, "cat", memInfoPath) // #nosec G204 - memInfoPath is constructed from safe directory listing
		memOutput, err := memCmd.Output()
		cancel5()
		if err == nil {
			if mem, err := strconv.ParseUint(strings.TrimSpace(string(memOutput)), 10, 64); err == nil {
				gpu.MemoryTotal = mem
			}
		}

		memUsedPath := fmt.Sprintf("/sys/class/drm/%s/device/mem_info_vram_used", card)
		ctx6, cancel6 := context.WithTimeout(context.Background(), 2*time.Second)
		memUsedCmd := exec.CommandContext(ctx6, "cat", memUsedPath) // #nosec G204 - memUsedPath is constructed from safe directory listing
		memOutput, err = memUsedCmd.Output()
		cancel6()
		if err == nil {
			if mem, err := strconv.ParseUint(strings.TrimSpace(string(memOutput)), 10, 64); err == nil {
				gpu.MemoryUsed = mem
			}
		}

		gpus = append(gpus, gpu)
		gpuIndex++
	}

	return gpus
}

// getAllGPUsFromLspci gets all GPU devices from lspci
func getAllGPUsFromLspci() []GPUInfo {
	var gpus []GPUInfo

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Use lspci -nn to get vendor and device IDs
	cmd := exec.CommandContext(ctx, "lspci", "-nn")
	output, err := cmd.Output()
	if err != nil {
		return gpus
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		lower := strings.ToLower(line)
		// Look for VGA compatible controller or Display controller
		if strings.Contains(lower, "vga compatible controller") ||
			strings.Contains(lower, "display controller") ||
			strings.Contains(lower, "3d controller") {

			gpu := GPUInfo{}

			// Extract vendor and device info
			switch {
			case strings.Contains(lower, "amd") || strings.Contains(lower, "advanced micro devices"):
				gpu.Vendor = "AMD"
			case strings.Contains(lower, "intel"):
				gpu.Vendor = "Intel"
			case strings.Contains(lower, "nvidia"):
				continue // Skip NVIDIA as they're handled by nvidia-smi
			case strings.Contains(lower, "microsoft") || strings.Contains(lower, "basic render driver"):
				continue // Skip WSL virtual display adapters
			default:
				telemetry.RecordHardwareMiss("GPUVendor", map[string]interface{}{
					"name": line,
					"source": "lspci",
				})
				gpu.Vendor = "Unknown"
			}

			// Extract device name
			if idx := strings.LastIndex(line, ":"); idx > 0 {
				deviceInfo := strings.TrimSpace(line[idx+1:])
				// Remove vendor/device IDs in brackets
				if bracketIdx := strings.Index(deviceInfo, "["); bracketIdx > 0 {
					deviceInfo = strings.TrimSpace(deviceInfo[:bracketIdx])
				}
				gpu.Name = deviceInfo

				// Clean up common prefixes
				gpu.Name = strings.TrimPrefix(gpu.Name, "Advanced Micro Devices, Inc. ")
				gpu.Name = strings.TrimPrefix(gpu.Name, "AMD/ATI ")
				gpu.Name = strings.TrimPrefix(gpu.Name, "Intel Corporation ")
			}

			// Identify if it's integrated graphics
			if strings.Contains(lower, "integrated") ||
				strings.Contains(lower, "apu") ||
				(gpu.Vendor == "Intel" && (strings.Contains(lower, "uhd") || strings.Contains(lower, "hd graphics"))) ||
				(gpu.Vendor == "AMD" && strings.Contains(gpu.Name, "Radeon Vega")) ||
				strings.Contains(gpu.Name, "Rembrandt") ||
				strings.Contains(gpu.Name, "Cezanne") ||
				strings.Contains(gpu.Name, "Renoir") ||
				strings.Contains(gpu.Name, "Picasso") ||
				strings.Contains(gpu.Name, "Raven") {
				gpu.Name += " (Integrated)"
			}

			gpus = append(gpus, gpu)
		}
	}

	return gpus
}

// getIntelGPUs gets Intel integrated GPU info
func getIntelGPUs() []GPUInfo {
	var gpus []GPUInfo

	// Check for Intel GPU in /sys/class/drm/
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ls", "/sys/class/drm/")
	output, err := cmd.Output()
	if err != nil {
		return gpus
	}

	cards := strings.Split(string(output), "\n")

	for _, card := range cards {
		card = strings.TrimSpace(card)
		if !strings.HasPrefix(card, "card") || strings.Contains(card, "-") {
			continue
		}

		// Check if it's an Intel GPU
		vendorPath := fmt.Sprintf("/sys/class/drm/%s/device/vendor", card)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		vendorCmd := exec.CommandContext(ctx, "cat", vendorPath) // #nosec G204 - vendorPath is constructed from safe directory listing
		vendorOutput, err := vendorCmd.Output()
		cancel()
		if err != nil {
			continue
		}

		vendor := strings.TrimSpace(string(vendorOutput))
		if vendor != "0x8086" { // Intel vendor ID
			continue
		}

		gpu := GPUInfo{
			Vendor: "Intel",
			Name:   "Intel Graphics",
		}

		// Try to get more specific name from device ID
		devicePath := fmt.Sprintf("/sys/class/drm/%s/device/device", card)
		ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
		deviceCmd := exec.CommandContext(ctx2, "cat", devicePath) // #nosec G204 - devicePath is constructed from safe directory listing
		deviceOutput, err := deviceCmd.Output()
		cancel2()
		if err == nil {
			deviceID := strings.TrimSpace(string(deviceOutput))
			// Map common Intel GPU device IDs to names
			switch deviceID {
			case "0x0046", "0x0042":
				gpu.Name = "Intel HD Graphics"
			case "0x0166", "0x0156":
				gpu.Name = "Intel HD Graphics 4000"
			case "0x1616", "0x161e":
				gpu.Name = "Intel HD Graphics 5500"
			case "0x5916", "0x5917":
				gpu.Name = "Intel HD Graphics 620"
			case "0x3e92", "0x3e91":
				gpu.Name = "Intel UHD Graphics 630"
			case "0x9a49", "0x9a40":
				gpu.Name = "Intel Iris Xe Graphics"
			case "0x4680", "0x4682":
				gpu.Name = "Intel UHD Graphics 770"
			default:
				// Record unknown Intel GPU device ID
				telemetry.RecordHardwareMiss("IntelGPUDeviceID", map[string]interface{}{
					"device_id": deviceID,
					"vendor": "Intel",
				})
				// Try to get name from lspci for this specific device
				if gpuInfo := getGPUNameFromLspci(card); gpuInfo != "" {
					gpu.Name = gpuInfo
				} else {
					gpu.Name = fmt.Sprintf("Intel Graphics (%s)", deviceID)
				}
			}
		}

		// Mark as integrated
		if !strings.Contains(gpu.Name, "Integrated") {
			gpu.Name += " (Integrated)"
		}

		// Try to get temperature
		hwmonPath := fmt.Sprintf("/sys/class/drm/%s/device/hwmon/", card)
		ctx3, cancel3 := context.WithTimeout(context.Background(), 2*time.Second)
		hwmonCmd := exec.CommandContext(ctx3, "ls", hwmonPath) // #nosec G204 - hwmonPath is constructed from safe directory listing
		hwmonOutput, err := hwmonCmd.Output()
		cancel3()
		if err == nil {
			hwmons := strings.Split(strings.TrimSpace(string(hwmonOutput)), "\n")
			if len(hwmons) > 0 {
				tempPath := fmt.Sprintf("%s%s/temp1_input", hwmonPath, hwmons[0])
				ctx4, cancel4 := context.WithTimeout(context.Background(), 2*time.Second)
				tempCmd := exec.CommandContext(ctx4, "cat", tempPath) // #nosec G204 - tempPath is constructed from safe directory listing
				tempOutput, err := tempCmd.Output()
				cancel4()
				if err == nil {
					if temp, err := strconv.ParseFloat(strings.TrimSpace(string(tempOutput)), 64); err == nil {
						gpu.Temperature = temp / 1000.0 // Convert from millidegrees
					}
				}
			}
		}

		gpus = append(gpus, gpu)
	}

	return gpus
}

// getGPUNameFromLspci tries to get GPU name for a specific card from lspci
func getGPUNameFromLspci(card string) string {
	// Get PCI address from sysfs
	pciPath := fmt.Sprintf("/sys/class/drm/%s/device/uevent", card)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "cat", pciPath) // #nosec G204 - pciPath is a fixed system path
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(string(output), "\n")
	var pciAddr string
	for _, line := range lines {
		if strings.HasPrefix(line, "PCI_SLOT_NAME=") {
			pciAddr = strings.TrimPrefix(line, "PCI_SLOT_NAME=")
			break
		}
	}

	if pciAddr == "" {
		return ""
	}

	// Query lspci for this specific device
	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()

	lspciCmd := exec.CommandContext(ctx2, "lspci", "-s", pciAddr) // #nosec G204 -- pciAddr is validated from sysfs enumeration
	lspciOutput, err := lspciCmd.Output()
	if err != nil {
		return ""
	}

	line := strings.TrimSpace(string(lspciOutput))
	if idx := strings.LastIndex(line, ":"); idx > 0 {
		deviceInfo := strings.TrimSpace(line[idx+1:])
		// Clean up common prefixes
		deviceInfo = strings.TrimPrefix(deviceInfo, "Intel Corporation ")
		return deviceInfo
	}

	return ""
}

// getWindowsGPUs gets all GPUs on Windows including integrated
func getWindowsGPUs() []GPUInfo {
	var gpus []GPUInfo

	// Use WMI to get all video controllers
	var cmd *exec.Cmd
	if isWindows() {
		cmd = exec.Command("cmd", "/c", "wmic path Win32_VideoController get Name,AdapterRAM,VideoProcessor,Status /format:csv")
	} else {
		// WSL
		cmd = exec.Command("cmd.exe", "/c", "wmic path Win32_VideoController get Name,AdapterRAM,VideoProcessor,Status /format:csv")
	}

	output, err := cmd.Output()
	if err != nil {
		return gpus
	}

	lines := strings.Split(string(output), "\n")
	var headers []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.Trim(line, "\r")
		if line == "" {
			continue
		}

		fields := strings.Split(line, ",")

		// First line with multiple fields is headers
		if len(headers) == 0 && len(fields) > 1 && strings.Contains(line, "Name") {
			headers = fields
			continue
		}

		// Skip if not a data line
		if len(fields) < 3 || strings.Contains(line, "Node") {
			continue
		}

		// Create a map for easier field access
		fieldMap := make(map[string]string)
		for j, header := range headers {
			if j < len(fields) {
				fieldMap[strings.TrimSpace(header)] = strings.TrimSpace(fields[j])
			}
		}

		name := fieldMap["Name"]
		status := fieldMap["Status"]

		// Skip if disabled or not OK
		if status != "OK" && status != "" {
			continue
		}

		// Skip virtual display adapters
		if strings.Contains(strings.ToLower(name), "microsoft basic") ||
			strings.Contains(strings.ToLower(name), "remote") ||
			strings.Contains(strings.ToLower(name), "virtual") {
			continue
		}

		gpu := GPUInfo{
			Name: name,
		}

		// Parse memory
		if ramStr := fieldMap["AdapterRAM"]; ramStr != "" && ramStr != "0" {
			if ram, err := strconv.ParseUint(ramStr, 10, 64); err == nil {
				gpu.MemoryTotal = ram
			}
		}

		// Determine vendor from name
		lowerName := strings.ToLower(name)
		switch {
		case strings.Contains(lowerName, "nvidia"):
			gpu.Vendor = "NVIDIA"
		case strings.Contains(lowerName, "amd") || strings.Contains(lowerName, "radeon"):
			gpu.Vendor = "AMD"
			// Check if it's integrated (APU)
			if strings.Contains(lowerName, "graphics") && !strings.Contains(lowerName, "radeon") {
				gpu.Name += " (Integrated)"
			}
		case strings.Contains(lowerName, "intel"):
			gpu.Vendor = "Intel"
			gpu.Name += " (Integrated)"
		default:
			telemetry.RecordHardwareMiss("GPUVendor", map[string]interface{}{
				"name": gpu.Name,
				"source": "wmi",
			})
			gpu.Vendor = "Unknown"
		}

		gpus = append(gpus, gpu)
	}

	// Also try to get NVIDIA GPU stats if available
	nvidiaGPUs := getNVIDIAGPUs()
	for _, nGPU := range nvidiaGPUs {
		// Update existing NVIDIA GPU with live stats
		for i := range gpus {
			if gpus[i].Vendor != "NVIDIA" || !strings.Contains(gpus[i].Name, nGPU.Name) {
				continue
			}
			gpus[i].Temperature = nGPU.Temperature
			gpus[i].MemoryUsed = nGPU.MemoryUsed
			gpus[i].Utilization = nGPU.Utilization
			gpus[i].PowerDraw = nGPU.PowerDraw
			gpus[i].PowerLimit = nGPU.PowerLimit
			gpus[i].FanSpeed = nGPU.FanSpeed
			break
		}
	}

	return gpus
}

// FormatGPUMemory formats GPU memory usage as a human-readable string
func FormatGPUMemory(used, total uint64) string {
	usedGB := float64(used) / (1024 * 1024 * 1024)
	totalGB := float64(total) / (1024 * 1024 * 1024)
	return fmt.Sprintf("%.1f / %.1f GB", usedGB, totalGB)
}

// FormatGPUPower formats GPU power usage
func FormatGPUPower(draw, limit float64) string {
	if limit > 0 {
		return fmt.Sprintf("%.0f / %.0f W", draw, limit)
	}
	return fmt.Sprintf("%.0f W", draw)
}

// isWindows checks if running on Windows
func isWindows() bool {
	return strings.Contains(strings.ToLower(os.Getenv("OS")), "windows")
}

// isWSL checks if running in WSL
func isWSL() bool {
	if data, err := exec.Command("uname", "-r").Output(); err == nil {
		return strings.Contains(strings.ToLower(string(data)), "microsoft")
	}
	return false
}
