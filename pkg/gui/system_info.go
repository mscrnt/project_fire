package gui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

// SystemInfo contains detailed system information
type SystemInfo struct {
	Host   HostInfo
	CPU    CPUInfo
	Memory MemoryInfo
	GPU    []GPUInfo
}

// HostInfo contains host/OS information
type HostInfo struct {
	Hostname             string
	Platform             string
	PlatformFamily       string
	PlatformVersion      string
	KernelVersion        string
	OS                   string
	Architecture         string
	VirtualizationSystem string
	VirtualizationRole   string
	IsWSL                bool
}

// CPUInfo contains CPU information
type CPUInfo struct {
	Model         string
	Vendor        string
	Family        string
	PhysicalCores int
	LogicalCores  int
	MaxFreqMHz    float64
}

// MemoryInfo contains memory information
type MemoryInfo struct {
	TotalGB     float64
	AvailableGB float64
	UsedGB      float64
	UsedPercent float64
	HostTotalGB float64 // For WSL, this is Windows host memory
}

// GetSystemInfo gathers comprehensive system information
func GetSystemInfo() (*SystemInfo, error) {
	info := &SystemInfo{}

	// Get host info
	hostInfo, err := host.Info()
	if err == nil {
		info.Host.Hostname = hostInfo.Hostname
		info.Host.Platform = hostInfo.Platform
		info.Host.PlatformFamily = hostInfo.PlatformFamily
		info.Host.PlatformVersion = hostInfo.PlatformVersion
		info.Host.KernelVersion = hostInfo.KernelVersion
		info.Host.OS = hostInfo.OS
		info.Host.VirtualizationSystem = hostInfo.VirtualizationSystem
		info.Host.VirtualizationRole = hostInfo.VirtualizationRole

		// Check if running in WSL
		if strings.Contains(strings.ToLower(hostInfo.KernelVersion), "microsoft") {
			info.Host.IsWSL = true
		}
	}

	// Architecture
	info.Host.Architecture = runtime.GOARCH

	// Get CPU info
	cpuInfo, err := cpu.Info()
	if err == nil && len(cpuInfo) > 0 {
		info.CPU.Model = cpuInfo[0].ModelName
		info.CPU.Vendor = cpuInfo[0].VendorID
		info.CPU.Family = cpuInfo[0].Family
		info.CPU.MaxFreqMHz = cpuInfo[0].Mhz
	}

	// Get CPU cores
	physicalCores, _ := cpu.Counts(false)
	logicalCores, _ := cpu.Counts(true)
	info.CPU.PhysicalCores = physicalCores
	info.CPU.LogicalCores = logicalCores

	// Get memory info
	vmStat, err := mem.VirtualMemory()
	if err == nil {
		info.Memory.TotalGB = float64(vmStat.Total) / (1024 * 1024 * 1024)
		info.Memory.AvailableGB = float64(vmStat.Available) / (1024 * 1024 * 1024)
		info.Memory.UsedGB = float64(vmStat.Used) / (1024 * 1024 * 1024)
		info.Memory.UsedPercent = vmStat.UsedPercent

		// If in WSL, try to get Windows host memory
		if info.Host.IsWSL {
			hostMem := getWindowsHostMemory()
			if hostMem > 0 {
				info.Memory.HostTotalGB = hostMem
			}
		}
	}

	// Get GPU info
	info.GPU, _ = GetGPUInfo()

	return info, nil
}

// getWindowsHostMemory tries to get Windows host memory when running in WSL
func getWindowsHostMemory() float64 {
	// Try to read from /proc/meminfo which might show host memory in some WSL configs
	// In WSL2, we can try to query Windows through PowerShell
	cmd := exec.Command("powershell.exe", "-Command", "(Get-CimInstance Win32_ComputerSystem).TotalPhysicalMemory")
	output, err := cmd.Output()
	if err == nil {
		var bytes uint64
		_, err = fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &bytes)
		if err == nil {
			return float64(bytes) / (1024 * 1024 * 1024)
		}
	}
	return 0
}

// FormatBytes formats bytes to human readable string
func FormatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
