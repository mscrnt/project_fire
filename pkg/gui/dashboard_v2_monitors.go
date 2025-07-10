package gui

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// monitorCPU monitors CPU usage
func (d *DashboardV2) monitorCPU() {
	for {
		select {
		case <-d.cpuTicker.C:
			d.updateCPU()
		case <-d.stopChan:
			return
		}
	}
}

// updateCPU updates CPU information
func (d *DashboardV2) updateCPU() {
	// Get CPU usage
	cpuPercent, _ := cpu.Percent(0, false)
	// cpuPerCore, _ := cpu.Percent(0, true) // TODO: Use for per-core display

	// Get CPU frequency
	cpuInfo, _ := cpu.Info()
	var currentSpeed float64
	if len(cpuInfo) > 0 {
		currentSpeed = cpuInfo[0].Mhz
	}

	// Get CPU temperature (platform specific)
	temp := getCPUTemperature()

	// Update UI
	if app := fyne.CurrentApp(); app != nil && app.Driver() != nil {
		app.Driver().DoFromGoroutine(func() {
			if len(cpuPercent) > 0 {
				d.cpuPanel.usageLabel.SetText(fmt.Sprintf("%.1f%%", cpuPercent[0]))
				d.cpuPanel.chart.AddValue(cpuPercent[0])

				// Update min/max
				min, max := d.cpuPanel.chart.GetMinMax()
				d.cpuPanel.minMaxLabel.SetText(fmt.Sprintf("Min: %.1f%% | Max: %.1f%%", min, max))
			}

			if currentSpeed > 0 {
				d.cpuPanel.speedLabel.SetText(fmt.Sprintf("Speed: %.2f GHz", currentSpeed/1000))
			}

			if temp > 0 {
				d.cpuPanel.tempLabel.SetText(fmt.Sprintf("%.0f°C", temp))
			}

			d.updateTimestamp()
		}, false)
	}
}

// monitorMemory monitors memory usage
func (d *DashboardV2) monitorMemory() {
	for {
		select {
		case <-d.memoryTicker.C:
			d.updateMemory()
		case <-d.stopChan:
			return
		}
	}
}

// updateMemory updates memory information
func (d *DashboardV2) updateMemory() {
	vmStat, _ := mem.VirtualMemory()
	swapStat, _ := mem.SwapMemory()

	if app := fyne.CurrentApp(); app != nil && app.Driver() != nil {
		app.Driver().DoFromGoroutine(func() {
			if vmStat != nil {
				d.memoryPanel.usageLabel.SetText(fmt.Sprintf("%.1f%%", vmStat.UsedPercent))
				d.memoryPanel.availLabel.SetText(fmt.Sprintf("Free: %.1f GB",
					float64(vmStat.Available)/(1024*1024*1024)))

				// Show WSL vs host memory if applicable
				if d.sysInfo != nil && d.sysInfo.Host.IsWSL && d.sysInfo.Memory.HostTotalGB > d.sysInfo.Memory.TotalGB {
					d.memoryPanel.totalLabel.SetText(fmt.Sprintf("Total: %.1f GB (Host: %.0f GB)",
						float64(vmStat.Total)/(1024*1024*1024), d.sysInfo.Memory.HostTotalGB))
				} else {
					d.memoryPanel.totalLabel.SetText(fmt.Sprintf("Total: %.1f GB",
						float64(vmStat.Total)/(1024*1024*1024)))
				}

				d.memoryPanel.chart.AddValue(vmStat.UsedPercent)

				// Update min/max
				min, max := d.memoryPanel.chart.GetMinMax()
				d.memoryPanel.minMaxLabel.SetText(fmt.Sprintf("Min: %.1f%% | Max: %.1f%%", min, max))
			}

			if swapStat != nil && swapStat.Total > 0 {
				d.memoryPanel.swapLabel.SetText(fmt.Sprintf("Swap: %.1f%% (%.1f GB)",
					swapStat.UsedPercent, float64(swapStat.Used)/(1024*1024*1024)))
			} else {
				d.memoryPanel.swapLabel.SetText("Swap: Not configured")
			}
		}, false)
	}
}

// monitorGPU monitors GPU usage
func (d *DashboardV2) monitorGPU() {
	for {
		select {
		case <-d.gpuTicker.C:
			d.updateGPU()
		case <-d.stopChan:
			return
		}
	}
}

// updateGPU updates GPU information
func (d *DashboardV2) updateGPU() {
	gpus, _ := GetGPUInfo()

	if app := fyne.CurrentApp(); app != nil && app.Driver() != nil {
		app.Driver().DoFromGoroutine(func() {
			if len(gpus) > d.gpuPanel.gpuIndex {
				gpu := gpus[d.gpuPanel.gpuIndex]

				d.gpuPanel.nameLabel.SetText(fmt.Sprintf("%s %s", gpu.Vendor, gpu.Name))
				d.gpuPanel.usageLabel.SetText(fmt.Sprintf("%.0f%%", gpu.Utilization))

				if gpu.MemoryTotal > 0 {
					d.gpuPanel.memoryLabel.SetText(fmt.Sprintf("Memory: %s (%.0f%%)",
						FormatGPUMemory(gpu.MemoryUsed, gpu.MemoryTotal),
						float64(gpu.MemoryUsed)/float64(gpu.MemoryTotal)*100))
				}

				if gpu.Temperature > 0 {
					d.gpuPanel.tempLabel.SetText(fmt.Sprintf("%.0f°C", gpu.Temperature))
				}

				if gpu.PowerDraw > 0 {
					d.gpuPanel.powerLabel.SetText(fmt.Sprintf("Power: %s",
						FormatGPUPower(gpu.PowerDraw, gpu.PowerLimit)))
				}

				d.gpuPanel.chart.AddValue(gpu.Utilization)

				// Update min/max
				min, max := d.gpuPanel.chart.GetMinMax()
				d.gpuPanel.minMaxLabel.SetText(fmt.Sprintf("Min: %.0f%% | Max: %.0f%%", min, max))
			} else {
				d.gpuPanel.nameLabel.SetText("No GPU detected")
				d.gpuPanel.usageLabel.SetText("N/A")
			}
		}, false)
	}
}

// monitorStorage monitors storage usage and I/O
func (d *DashboardV2) monitorStorage() {
	var lastDiskIO map[string]disk.IOCountersStat

	for {
		select {
		case <-d.diskTicker.C:
			d.updateStorage(&lastDiskIO)
		case <-d.stopChan:
			return
		}
	}
}

// updateStorage updates storage information
func (d *DashboardV2) updateStorage(lastIO *map[string]disk.IOCountersStat) {
	// Get disk partitions
	partitions, _ := disk.Partitions(true)
	volumes := make([]VolumeInfo, 0)

	for _, partition := range partitions {
		// Skip irrelevant partitions
		if shouldSkipPartition(partition) {
			continue
		}

		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil || usage.Total < 1024*1024*1024 { // Skip < 1GB
			continue
		}

		vol := VolumeInfo{
			MountPoint:  partition.Mountpoint,
			Device:      partition.Device,
			FileSystem:  partition.Fstype,
			TotalGB:     float64(usage.Total) / (1024 * 1024 * 1024),
			UsedGB:      float64(usage.Used) / (1024 * 1024 * 1024),
			FreeGB:      float64(usage.Free) / (1024 * 1024 * 1024),
			UsedPercent: usage.UsedPercent,
		}
		volumes = append(volumes, vol)
	}

	// Get disk I/O stats
	ioCounters, _ := disk.IOCounters()
	var totalReadRate, totalWriteRate float64

	if *lastIO != nil && len(ioCounters) > 0 {
		for name, counter := range ioCounters {
			if lastCounter, ok := (*lastIO)[name]; ok {
				// Calculate rates (bytes per second)
				timeDiff := time.Since(time.Unix(0, int64(lastCounter.ReadTime))).Seconds()
				if timeDiff > 0 {
					readRate := float64(counter.ReadBytes-lastCounter.ReadBytes) / timeDiff
					writeRate := float64(counter.WriteBytes-lastCounter.WriteBytes) / timeDiff
					totalReadRate += readRate
					totalWriteRate += writeRate
				}
			}
		}
	}
	*lastIO = ioCounters

	// Update UI
	if app := fyne.CurrentApp(); app != nil && app.Driver() != nil {
		app.Driver().DoFromGoroutine(func() {
			d.storagePanel.volumes = volumes
			d.storagePanel.volumeList.Refresh()

			// Update I/O stats
			d.storagePanel.ioLabel.SetText(fmt.Sprintf("I/O: Read %.1f MB/s | Write %.1f MB/s",
				totalReadRate/(1024*1024), totalWriteRate/(1024*1024)))

			// Update charts (convert to MB/s)
			d.storagePanel.readChart.AddValue(totalReadRate / (1024 * 1024))
			d.storagePanel.writeChart.AddValue(totalWriteRate / (1024 * 1024))
		}, false)
	}
}

// monitorNetwork monitors network usage
func (d *DashboardV2) monitorNetwork() {
	for {
		select {
		case <-d.networkTicker.C:
			d.updateNetwork()
		case <-d.stopChan:
			return
		}
	}
}

// updateNetwork updates network information
func (d *DashboardV2) updateNetwork() {
	// Get primary network interface info from first non-zero counter
	// gopsutil doesn't provide interface details like standard net package
	ioCountersPerNic, _ := net.IOCounters(true)
	var primaryInterface string

	for _, counter := range ioCountersPerNic {
		if counter.BytesSent > 0 || counter.BytesRecv > 0 {
			primaryInterface = counter.Name
			break
		}
	}

	// For IP, we'll just show the interface name for now
	primaryIP := "N/A"

	// Get network I/O stats
	ioCounters, _ := net.IOCounters(false)

	if app := fyne.CurrentApp(); app != nil && app.Driver() != nil {
		app.Driver().DoFromGoroutine(func() {
			// Update interface info
			if primaryInterface != "" {
				d.networkPanel.interfaceLabel.SetText(fmt.Sprintf("Interface: %s", primaryInterface))
				d.networkPanel.ipLabel.SetText(fmt.Sprintf("IP: %s", primaryIP))
			}

			// Calculate rates
			if len(ioCounters) > 0 {
				current := ioCounters[0]

				// Calculate rates if we have previous data
				if d.networkPanel.lastBytes.BytesSent > 0 {
					timeDiff := 1.0 // 1 second update interval
					uploadRate := float64(current.BytesSent-d.networkPanel.lastBytes.BytesSent) / timeDiff
					downloadRate := float64(current.BytesRecv-d.networkPanel.lastBytes.BytesRecv) / timeDiff

					d.networkPanel.uploadLabel.SetText(fmt.Sprintf("↑ %.1f MB/s", uploadRate/(1024*1024)))
					d.networkPanel.downloadLabel.SetText(fmt.Sprintf("↓ %.1f MB/s", downloadRate/(1024*1024)))

					// Update charts (MB/s)
					d.networkPanel.uploadChart.AddValue(uploadRate / (1024 * 1024))
					d.networkPanel.downloadChart.AddValue(downloadRate / (1024 * 1024))
				}

				d.networkPanel.totalLabel.SetText(fmt.Sprintf("Total: ↑ %s ↓ %s",
					formatBytes(current.BytesSent), formatBytes(current.BytesRecv)))

				d.networkPanel.lastBytes = current
			}
		}, false)
	}
}

// monitorSystem monitors system information
func (d *DashboardV2) monitorSystem() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.updateSystem()
		case <-d.stopChan:
			return
		}
	}
}

// updateSystem updates system information
func (d *DashboardV2) updateSystem() {
	// Get uptime
	uptime, _ := host.Uptime()

	// Get process count
	pids, _ := process.Pids()

	if app := fyne.CurrentApp(); app != nil && app.Driver() != nil {
		app.Driver().DoFromGoroutine(func() {
			// Format uptime
			hours := uptime / 3600
			minutes := (uptime % 3600) / 60
			d.systemPanel.uptimeLabel.SetText(fmt.Sprintf("Uptime: %dh %dm", hours, minutes))

			// Update process count
			d.systemPanel.processLabel.SetText(fmt.Sprintf("Processes: %d", len(pids)))
		}, false)
	}
}

// getCPUTemperature attempts to get CPU temperature (platform specific)
func getCPUTemperature() float64 {
	// This is platform specific and would need different implementations
	// for Windows, Linux, macOS
	// For now, return 0 to indicate not available
	return 0
}

// Platform specific temperature reading would go here
// Linux: Read from /sys/class/thermal/thermal_zone*/temp
// Windows: Use WMI
// macOS: Use IOKit

