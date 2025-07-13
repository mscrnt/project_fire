package gui

import (
	"fmt"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// getCPUDynamicMetrics returns dynamic CPU metrics
func (d *Dashboard) getCPUDynamicMetrics() (metrics map[string]float64, additionalInfo map[string]string) {
	metrics = make(map[string]float64)
	additionalInfo = make(map[string]string)

	// CPU usage
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err == nil && len(cpuPercent) > 0 {
		metrics["Overall Usage"] = cpuPercent[0]
	}

	// Per-core usage
	perCore, err := cpu.Percent(time.Second, true)
	if err == nil {
		for i, usage := range perCore {
			metrics[fmt.Sprintf("Core %d Usage", i)] = usage
		}
	}

	// CPU frequency
	cpuInfo, err := cpu.Info()
	if err == nil && len(cpuInfo) > 0 {
		metrics["Current Frequency"] = cpuInfo[0].Mhz / 1000 // Convert to GHz

		// Additional CPU info
		additionalInfo["CPU Family"] = cpuInfo[0].Family
		additionalInfo["Model"] = cpuInfo[0].Model
		additionalInfo["Stepping"] = fmt.Sprintf("%d", cpuInfo[0].Stepping)
		additionalInfo["Microcode"] = cpuInfo[0].Microcode
		if len(cpuInfo[0].Flags) > 0 {
			additionalInfo["CPU Flags"] = fmt.Sprintf("%d features", len(cpuInfo[0].Flags))
		}
	}

	// Temperature
	temp := getCPUTemperature()
	if temp > 0 {
		metrics["Die Temperature"] = temp
	}

	// Power metrics
	metrics["Package Power"] = getCPUPackagePower()
	metrics["Core Voltage"] = getCPUVoltage()

	// CPU times
	times, err := cpu.Times(false)
	if err == nil && len(times) > 0 {
		total := times[0].User + times[0].System + times[0].Idle + times[0].Nice +
			times[0].Iowait + times[0].Irq + times[0].Softirq + times[0].Steal
		if total > 0 {
			additionalInfo["User Time"] = fmt.Sprintf("%.1f%%", (times[0].User/total)*100)
			additionalInfo["System Time"] = fmt.Sprintf("%.1f%%", (times[0].System/total)*100)
			additionalInfo["Idle Time"] = fmt.Sprintf("%.1f%%", (times[0].Idle/total)*100)
			if times[0].Iowait > 0 {
				additionalInfo["I/O Wait"] = fmt.Sprintf("%.1f%%", (times[0].Iowait/total)*100)
			}
		}
	}

	return metrics, additionalInfo
}

// getMemoryDynamicMetrics returns dynamic memory metrics
func (d *Dashboard) getMemoryDynamicMetrics() (metrics map[string]float64, additionalInfo map[string]string) {
	metrics = make(map[string]float64)
	additionalInfo = make(map[string]string)

	// Virtual memory stats
	vmStat, err := mem.VirtualMemory()
	if err == nil {
		metrics["Usage Percent"] = vmStat.UsedPercent
		metrics["Used GB"] = float64(vmStat.Used) / (1024 * 1024 * 1024)
		metrics["Available GB"] = float64(vmStat.Available) / (1024 * 1024 * 1024)
		metrics["Cached GB"] = float64(vmStat.Cached) / (1024 * 1024 * 1024)
		metrics["Buffers GB"] = float64(vmStat.Buffers) / (1024 * 1024 * 1024)

		// Additional info
		additionalInfo["Total Memory"] = fmt.Sprintf("%.1f GB", float64(vmStat.Total)/(1024*1024*1024))
		additionalInfo["Free Memory"] = fmt.Sprintf("%.1f GB", float64(vmStat.Free)/(1024*1024*1024))
		if vmStat.Shared > 0 {
			additionalInfo["Shared Memory"] = fmt.Sprintf("%.1f GB", float64(vmStat.Shared)/(1024*1024*1024))
		}
		if vmStat.Slab > 0 {
			additionalInfo["Kernel Slab"] = fmt.Sprintf("%.1f MB", float64(vmStat.Slab)/(1024*1024))
		}
	}

	// Swap memory stats
	swapStat, err := mem.SwapMemory()
	if err == nil && swapStat.Total > 0 {
		metrics["Swap Usage Percent"] = swapStat.UsedPercent
		metrics["Swap Used GB"] = float64(swapStat.Used) / (1024 * 1024 * 1024)
		additionalInfo["Swap Total"] = fmt.Sprintf("%.1f GB", float64(swapStat.Total)/(1024*1024*1024))
		additionalInfo["Swap Free"] = fmt.Sprintf("%.1f GB", float64(swapStat.Free)/(1024*1024*1024))
	}

	// Memory pressure/temperature (placeholder)
	metrics["Memory Temperature"] = 45.0 // Placeholder - would need specific sensor reading

	return metrics, additionalInfo
}

// getGPUDynamicMetrics returns dynamic GPU metrics
func (d *Dashboard) getGPUDynamicMetrics(comp *Component) (metrics map[string]float64, additionalInfo map[string]string) {
	metrics = make(map[string]float64)
	additionalInfo = make(map[string]string)

	// Get fresh GPU info
	gpus, _ := GetGPUInfo()

	// Find the matching GPU by index
	gpuIndexStr, ok := comp.Details["GPU Index"]
	if !ok {
		return metrics, additionalInfo
	}

	var gpuIndex int
	fmt.Sscanf(gpuIndexStr, "%d", &gpuIndex)

	if gpuIndex >= 0 && gpuIndex < len(gpus) {
		gpu := gpus[gpuIndex]

		// Dynamic metrics
		metrics["GPU Usage"] = gpu.Utilization
		metrics["Temperature"] = gpu.Temperature
		metrics["Power Draw"] = float64(gpu.PowerDraw)
		metrics["Power Limit"] = float64(gpu.PowerLimit)

		if gpu.MemoryTotal > 0 {
			metrics["Memory Used MB"] = float64(gpu.MemoryUsed) / (1024 * 1024)
			metrics["Memory Total MB"] = float64(gpu.MemoryTotal) / (1024 * 1024)
			metrics["Memory Usage Percent"] = float64(gpu.MemoryUsed) / float64(gpu.MemoryTotal) * 100
		}

		// Placeholder metrics for clock speeds (not in current GPUInfo struct)
		metrics["Core Clock MHz"] = 1800.0
		metrics["Memory Clock MHz"] = 7000.0
		metrics["Voltage"] = 0.850

		// Additional info - these fields may not exist in current GPUInfo struct
		// Would need to be added to GPUInfo or fetched separately
		additionalInfo["GPU Index"] = fmt.Sprintf("%d", gpuIndex)
		additionalInfo["Vendor"] = gpu.Vendor
		additionalInfo["Model"] = gpu.Name

		// Power efficiency
		if gpu.PowerDraw > 0 && gpu.Utilization > 0 {
			efficiency := gpu.Utilization / float64(gpu.PowerDraw)
			additionalInfo["Power Efficiency"] = fmt.Sprintf("%.2f %%/W", efficiency)
		}
	}

	return metrics, additionalInfo
}

// getMotherboardDynamicMetrics returns dynamic motherboard metrics
func (d *Dashboard) getMotherboardDynamicMetrics() (metrics map[string]float64, additionalInfo map[string]string) {
	metrics = make(map[string]float64)
	additionalInfo = make(map[string]string)

	// Placeholder sensor readings - in a real implementation these would come from hardware monitoring chips
	metrics["Chipset Temperature"] = 42.0
	metrics["VRM Temperature"] = 55.0
	metrics["System Temperature"] = 38.0

	// Voltages
	metrics["CPU VCore"] = 1.25
	metrics["+12V Rail"] = 12.1
	metrics["+5V Rail"] = 5.05
	metrics["+3.3V Rail"] = 3.31
	metrics["DRAM Voltage"] = 1.35

	// Fan headers
	metrics["CPU Fan RPM"] = 1200.0
	metrics["System Fan 1 RPM"] = 800.0
	metrics["System Fan 2 RPM"] = 900.0

	// Additional system info
	hostInfo, err := host.Info()
	if err == nil {
		additionalInfo["Uptime"] = fmt.Sprintf("%.1f hours", float64(hostInfo.Uptime)/3600)
		additionalInfo["Boot Time"] = time.Unix(int64(hostInfo.BootTime), 0).Format("2006-01-02 15:04:05")
		additionalInfo["Processes"] = fmt.Sprintf("%d", hostInfo.Procs)
	}

	return metrics, additionalInfo
}

// getFanDynamicMetrics returns dynamic fan metrics
func (d *Dashboard) getFanDynamicMetrics(comp *Component) (metrics map[string]float64, additionalInfo map[string]string) {
	metrics = make(map[string]float64)
	additionalInfo = make(map[string]string)

	// Get fresh fan info
	fans, _ := GetFanInfo()

	// Find matching fan by name
	for _, fan := range fans {
		if fan.Name == comp.Details["Name"] {
			metrics["Current Speed RPM"] = float64(fan.Speed)
			metrics["Target Speed RPM"] = float64(fan.Speed) // Placeholder
			metrics["Speed Percent"] = 50.0                  // Placeholder

			additionalInfo["Fan Type"] = fan.Type
			additionalInfo["Control Mode"] = "PWM" // Placeholder
			additionalInfo["Min Speed"] = "600 RPM"
			additionalInfo["Max Speed"] = "2000 RPM"
			break
		}
	}

	return metrics, additionalInfo
}

// getSystemDynamicMetrics returns dynamic system metrics
func (d *Dashboard) getSystemDynamicMetrics() (metrics map[string]float64, additionalInfo map[string]string) {
	metrics = make(map[string]float64)
	additionalInfo = make(map[string]string)

	// Host statistics
	hostInfo, err := host.Info()
	if err == nil {
		metrics["Uptime Hours"] = float64(hostInfo.Uptime) / 3600
		metrics["Process Count"] = float64(hostInfo.Procs)

		additionalInfo["Host ID"] = hostInfo.HostID
		additionalInfo["Boot Time"] = time.Unix(int64(hostInfo.BootTime), 0).Format("2006-01-02 15:04:05")
	}

	// Load average (Unix-like systems)
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		// Load average is not available in gopsutil v3 on all platforms
		// This would need platform-specific implementation
		metrics["Load Average 1m"] = 0.0  // Placeholder
		metrics["Load Average 5m"] = 0.0  // Placeholder
		metrics["Load Average 15m"] = 0.0 // Placeholder
	}

	// Network interfaces
	netStats, err := net.IOCounters(false)
	if err == nil && len(netStats) > 0 {
		metrics["Network Bytes Sent GB"] = float64(netStats[0].BytesSent) / (1024 * 1024 * 1024)
		metrics["Network Bytes Recv GB"] = float64(netStats[0].BytesRecv) / (1024 * 1024 * 1024)
		metrics["Network Packets Sent M"] = float64(netStats[0].PacketsSent) / 1000000
		metrics["Network Packets Recv M"] = float64(netStats[0].PacketsRecv) / 1000000
	}

	// Disk I/O
	diskStats, err := disk.IOCounters()
	if err == nil {
		var totalRead, totalWrite uint64
		for _, stat := range diskStats {
			totalRead += stat.ReadBytes
			totalWrite += stat.WriteBytes
		}
		metrics["Disk Total Read GB"] = float64(totalRead) / (1024 * 1024 * 1024)
		metrics["Disk Total Write GB"] = float64(totalWrite) / (1024 * 1024 * 1024)
	}

	// Process statistics
	procs, err := process.Processes()
	if err == nil {
		additionalInfo["Total Processes"] = fmt.Sprintf("%d", len(procs))

		// Count process states
		running := 0
		sleeping := 0
		for _, p := range procs {
			status, _ := p.Status()
			switch status[0] {
			case "R":
				running++
			case "S":
				sleeping++
			}
		}
		additionalInfo["Running Processes"] = fmt.Sprintf("%d", running)
		additionalInfo["Sleeping Processes"] = fmt.Sprintf("%d", sleeping)
	}

	return metrics, additionalInfo
}
