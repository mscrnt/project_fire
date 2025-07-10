package gui

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
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

	// Try NVIDIA GPUs first
	nvidiaGPUs := getNVIDIAGPUs()
	gpus = append(gpus, nvidiaGPUs...)

	// Try AMD GPUs
	amdGPUs := getAMDGPUs()
	gpus = append(gpus, amdGPUs...)

	// If no dedicated GPUs found, try to get integrated GPU info
	if len(gpus) == 0 {
		if integratedGPU := getIntegratedGPU(); integratedGPU != nil {
			gpus = append(gpus, *integratedGPU)
		}
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
			if strings.Contains(line, "GPU use") {
				if parts := strings.Fields(line); len(parts) >= 3 {
					if util, err := strconv.ParseFloat(strings.TrimSuffix(parts[2], "%"), 64); err == nil {
						currentGPU.Utilization = util
					}
				}
			} else if strings.Contains(line, "Temperature") && strings.Contains(line, "edge") {
				if parts := strings.Fields(line); len(parts) >= 3 {
					if temp, err := strconv.ParseFloat(strings.TrimSuffix(parts[2], "c"), 64); err == nil {
						currentGPU.Temperature = temp
					}
				}
			} else if strings.Contains(line, "vram Total") {
				if parts := strings.Fields(line); len(parts) >= 3 {
					if mem, err := strconv.ParseUint(parts[2], 10, 64); err == nil {
						currentGPU.MemoryTotal = mem * 1024 * 1024 // MB to bytes
					}
				}
			} else if strings.Contains(line, "vram Used") {
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
		defer cancel()
		
		vendorCmd := exec.CommandContext(ctx, "cat", vendorPath)
		vendorOutput, err := vendorCmd.Output()
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

		// Try to get temperature
		hwmonPath := fmt.Sprintf("/sys/class/drm/%s/device/hwmon/", card)
		ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel2()
		
		hwmonCmd := exec.CommandContext(ctx2, "ls", hwmonPath)
		if hwmonOutput, err := hwmonCmd.Output(); err == nil {
			hwmons := strings.Split(strings.TrimSpace(string(hwmonOutput)), "\n")
			if len(hwmons) > 0 {
				tempPath := fmt.Sprintf("%s%s/temp1_input", hwmonPath, hwmons[0])
				ctx3, cancel3 := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel3()
				
				tempCmd := exec.CommandContext(ctx3, "cat", tempPath)
				if tempOutput, err := tempCmd.Output(); err == nil {
					if temp, err := strconv.ParseFloat(strings.TrimSpace(string(tempOutput)), 64); err == nil {
						gpu.Temperature = temp / 1000.0 // Convert from millidegrees
					}
				}
			}
		}

		// Try to get memory info
		memInfoPath := fmt.Sprintf("/sys/class/drm/%s/device/mem_info_vram_total", card)
		ctx4, cancel4 := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel4()
		
		memCmd := exec.CommandContext(ctx4, "cat", memInfoPath)
		if memOutput, err := memCmd.Output(); err == nil {
			if mem, err := strconv.ParseUint(strings.TrimSpace(string(memOutput)), 10, 64); err == nil {
				gpu.MemoryTotal = mem
			}
		}

		memUsedPath := fmt.Sprintf("/sys/class/drm/%s/device/mem_info_vram_used", card)
		ctx5, cancel5 := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel5()
		
		memUsedCmd := exec.CommandContext(ctx5, "cat", memUsedPath)
		if memOutput, err := memUsedCmd.Output(); err == nil {
			if mem, err := strconv.ParseUint(strings.TrimSpace(string(memOutput)), 10, 64); err == nil {
				gpu.MemoryUsed = mem
			}
		}

		gpus = append(gpus, gpu)
		gpuIndex++
	}

	return gpus
}

// getIntegratedGPU attempts to get integrated GPU info
func getIntegratedGPU() *GPUInfo {
	// This is a placeholder for integrated GPU detection
	// In a real implementation, we'd check for Intel/AMD integrated graphics
	return nil
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
