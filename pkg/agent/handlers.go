package agent

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

// SysInfo contains system information
type SysInfo struct {
	Timestamp time.Time     `json:"timestamp"`
	Host      HostInfo      `json:"host"`
	CPU       CPUInfo       `json:"cpu"`
	Memory    MemoryInfo    `json:"memory"`
	Disk      []DiskInfo    `json:"disk"`
	Network   []NetworkInfo `json:"network"`
}

// HostInfo contains host information
type HostInfo struct {
	Hostname        string `json:"hostname"`
	Uptime          uint64 `json:"uptime"`
	BootTime        uint64 `json:"boot_time"`
	OS              string `json:"os"`
	Platform        string `json:"platform"`
	PlatformVersion string `json:"platform_version"`
	KernelVersion   string `json:"kernel_version"`
	Architecture    string `json:"architecture"`
}

// CPUInfo contains CPU information
type CPUInfo struct {
	PhysicalCores int       `json:"physical_cores"`
	LogicalCores  int       `json:"logical_cores"`
	ModelName     string    `json:"model_name"`
	Usage         []float64 `json:"usage_percent"`
	Frequency     []float64 `json:"frequency_mhz"`
}

// MemoryInfo contains memory information
type MemoryInfo struct {
	Total       uint64  `json:"total"`
	Available   uint64  `json:"available"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"used_percent"`
	Free        uint64  `json:"free"`
}

// DiskInfo contains disk information
type DiskInfo struct {
	Path        string  `json:"path"`
	Fstype      string  `json:"fstype"`
	Total       uint64  `json:"total"`
	Free        uint64  `json:"free"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"used_percent"`
}

// NetworkInfo contains network interface information
type NetworkInfo struct {
	Name        string `json:"name"`
	BytesSent   uint64 `json:"bytes_sent"`
	BytesRecv   uint64 `json:"bytes_recv"`
	PacketsSent uint64 `json:"packets_sent"`
	PacketsRecv uint64 `json:"packets_recv"`
}

// sysinfoHandler returns system information as JSON
func sysinfoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	info := SysInfo{
		Timestamp: time.Now(),
	}

	// Get host info
	if hostInfo, err := host.Info(); err == nil {
		info.Host = HostInfo{
			Hostname:        hostInfo.Hostname,
			Uptime:          hostInfo.Uptime,
			BootTime:        hostInfo.BootTime,
			OS:              hostInfo.OS,
			Platform:        hostInfo.Platform,
			PlatformVersion: hostInfo.PlatformVersion,
			KernelVersion:   hostInfo.KernelVersion,
			Architecture:    runtime.GOARCH,
		}
	}

	// Get CPU info
	if cores, err := cpu.Counts(false); err == nil {
		info.CPU.PhysicalCores = cores
	}
	if cores, err := cpu.Counts(true); err == nil {
		info.CPU.LogicalCores = cores
	}
	if cpuInfo, err := cpu.Info(); err == nil && len(cpuInfo) > 0 {
		info.CPU.ModelName = cpuInfo[0].ModelName
		for _, ci := range cpuInfo {
			info.CPU.Frequency = append(info.CPU.Frequency, ci.Mhz)
		}
	}
	if usage, err := cpu.Percent(time.Second, true); err == nil {
		info.CPU.Usage = usage
	}

	// Get memory info
	if vmStat, err := mem.VirtualMemory(); err == nil {
		info.Memory = MemoryInfo{
			Total:       vmStat.Total,
			Available:   vmStat.Available,
			Used:        vmStat.Used,
			UsedPercent: vmStat.UsedPercent,
			Free:        vmStat.Free,
		}
	}

	// Get disk info
	if partitions, err := disk.Partitions(false); err == nil {
		for _, partition := range partitions {
			if usage, err := disk.Usage(partition.Mountpoint); err == nil {
				info.Disk = append(info.Disk, DiskInfo{
					Path:        partition.Mountpoint,
					Fstype:      partition.Fstype,
					Total:       usage.Total,
					Free:        usage.Free,
					Used:        usage.Used,
					UsedPercent: usage.UsedPercent,
				})
			}
		}
	}

	// Get network info
	if interfaces, err := net.IOCounters(true); err == nil {
		for _, iface := range interfaces {
			if iface.Name == "lo" || strings.HasPrefix(iface.Name, "docker") {
				continue // Skip loopback and docker interfaces
			}
			info.Network = append(info.Network, NetworkInfo{
				Name:        iface.Name,
				BytesSent:   iface.BytesSent,
				BytesRecv:   iface.BytesRecv,
				PacketsSent: iface.PacketsSent,
				PacketsRecv: iface.PacketsRecv,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(info); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// LogsResponse contains log data
type LogsResponse struct {
	Lines     []string  `json:"lines"`
	File      string    `json:"file"`
	Timestamp time.Time `json:"timestamp"`
}

// logsHandler returns log file contents
func logsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get query parameters
	logFile := r.URL.Query().Get("file")
	if logFile == "" {
		logFile = "fire.log" // Default log file
	}

	// Security: prevent directory traversal
	if strings.Contains(logFile, "..") || strings.Contains(logFile, "/") || strings.Contains(logFile, "\\") {
		http.Error(w, "Invalid log file name", http.StatusBadRequest)
		return
	}

	tailStr := r.URL.Query().Get("tail")
	tail := 100 // Default to last 100 lines
	if tailStr != "" {
		if n, err := strconv.Atoi(tailStr); err == nil && n > 0 {
			tail = n
		}
	}

	// Read log file
	file, err := os.Open(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Log file not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to open log file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Read lines
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		// Keep only the last N lines
		if len(lines) > tail {
			lines = lines[1:]
		}
	}

	if err := scanner.Err(); err != nil {
		http.Error(w, "Failed to read log file", http.StatusInternalServerError)
		return
	}

	response := LogsResponse{
		Lines:     lines,
		File:      logFile,
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// SensorsInfo contains sensor data
type SensorsInfo struct {
	Timestamp   time.Time         `json:"timestamp"`
	Temperature []TemperatureInfo `json:"temperature"`
	Fans        []FanInfo         `json:"fans"`
	GPU         []GPUInfo         `json:"gpu,omitempty"`
}

// TemperatureInfo contains temperature sensor data
type TemperatureInfo struct {
	Name        string  `json:"name"`
	Temperature float64 `json:"temperature_c"`
	Critical    float64 `json:"critical_c,omitempty"`
}

// FanInfo contains fan sensor data
type FanInfo struct {
	Name  string `json:"name"`
	Speed int    `json:"speed_rpm"`
}

// GPUInfo contains GPU sensor data
type GPUInfo struct {
	Index       int     `json:"index"`
	Name        string  `json:"name"`
	Temperature float64 `json:"temperature_c"`
	MemoryUsed  uint64  `json:"memory_used"`
	MemoryTotal uint64  `json:"memory_total"`
	Utilization int     `json:"utilization_percent"`
	FanSpeed    int     `json:"fan_speed_percent"`
}

// sensorsHandler returns sensor information
func sensorsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	info := SensorsInfo{
		Timestamp:   time.Now(),
		Temperature: []TemperatureInfo{},
		Fans:        []FanInfo{},
		GPU:         []GPUInfo{},
	}

	// Try to get CPU temperature (Linux-specific)
	if runtime.GOOS == "linux" {
		// Check thermal zones
		thermalDir := "/sys/class/thermal"
		if entries, err := os.ReadDir(thermalDir); err == nil {
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), "thermal_zone") {
					zonePath := fmt.Sprintf("%s/%s", thermalDir, entry.Name())

					// Read temperature
					if tempData, err := os.ReadFile(fmt.Sprintf("%s/temp", zonePath)); err == nil {
						if temp, err := strconv.Atoi(strings.TrimSpace(string(tempData))); err == nil {
							// Read type
							typeData, _ := os.ReadFile(fmt.Sprintf("%s/type", zonePath))
							typeName := strings.TrimSpace(string(typeData))
							if typeName == "" {
								typeName = entry.Name()
							}

							info.Temperature = append(info.Temperature, TemperatureInfo{
								Name:        typeName,
								Temperature: float64(temp) / 1000.0, // Convert millidegrees to degrees
							})
						}
					}
				}
			}
		}

		// Check hwmon for fan speeds
		hwmonDir := "/sys/class/hwmon"
		if entries, err := os.ReadDir(hwmonDir); err == nil {
			for _, entry := range entries {
				hwmonPath := fmt.Sprintf("%s/%s", hwmonDir, entry.Name())

				// Look for fan inputs
				for i := 1; i <= 10; i++ {
					fanPath := fmt.Sprintf("%s/fan%d_input", hwmonPath, i)
					if fanData, err := os.ReadFile(fanPath); err == nil {
						if speed, err := strconv.Atoi(strings.TrimSpace(string(fanData))); err == nil {
							// Try to get fan label
							labelPath := fmt.Sprintf("%s/fan%d_label", hwmonPath, i)
							label := fmt.Sprintf("Fan %d", i)
							if labelData, err := os.ReadFile(labelPath); err == nil {
								label = strings.TrimSpace(string(labelData))
							}

							info.Fans = append(info.Fans, FanInfo{
								Name:  label,
								Speed: speed,
							})
						}
					}
				}
			}
		}
	}

	// Note: GPU sensor support would require NVML bindings
	// This is a placeholder that could be extended with proper GPU support

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(info); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
