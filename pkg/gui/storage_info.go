package gui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/disk"
)

// StorageInfo contains information about a storage device
type StorageInfo struct {
	Device      string
	Mountpoint  string
	Filesystem  string
	Type        string // HDD, SSD, NVME, USB
	Size        uint64
	Used        uint64
	Free        uint64
	UsedPercent float64

	// Drive identification
	Model      string
	Serial     string
	Vendor     string
	Controller string
	Firmware   string
	Interface  string // SATA, NVMe, USB, etc.

	// SMART data
	SMART *SMARTData
}

// SMARTData contains SMART attributes for a storage device
type SMARTData struct {
	Temperature    float64 // Celsius
	HealthStatus   string  // Good, Warning, Critical
	PowerOnHours   uint64
	PowerCycles    uint64
	TotalWrittenGB float64
	TotalReadGB    float64
	WearLevel      float64 // Percentage for SSDs
	Available      bool    // Whether SMART data is available
}

// GetStorageInfo returns information about all storage devices
func GetStorageInfo() ([]StorageInfo, error) {
	var storageDevices []StorageInfo

	// Get disk partitions - include all partitions (true) to get Windows drives in WSL
	partitions, err := disk.Partitions(true)
	if err != nil {
		return nil, err
	}

	// Build a map of physical drives first
	driveModels := getDriveModels()

	for _, partition := range partitions {
		// Skip certain filesystems
		if strings.Contains(partition.Fstype, "squashfs") ||
			strings.Contains(partition.Mountpoint, "/snap") ||
			strings.Contains(partition.Mountpoint, "/boot/efi") {
			continue
		}

		// Get usage stats
		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			continue
		}

		// Determine device type and get physical drive info
		deviceType := "HDD"
		physicalDrive := getPhysicalDrive(partition.Device)

		switch {
		case strings.Contains(strings.ToLower(partition.Device), "nvme"):
			deviceType = "NVME"
		case strings.Contains(strings.ToLower(strings.Join(partition.Opts, ",")), "ssd"):
			deviceType = "SSD"
		case strings.Contains(strings.ToLower(partition.Device), "usb") ||
			strings.Contains(strings.ToLower(partition.Mountpoint), "/media") ||
			strings.Contains(strings.ToLower(partition.Mountpoint), "/mnt"):
			deviceType = "USB"
		}

		// In WSL, Windows drives are mounted under /mnt
		isWindowsDrive := false
		if strings.HasPrefix(partition.Mountpoint, "/mnt/") && len(partition.Mountpoint) == 6 {
			isWindowsDrive = true
		}

		// Check if device type is SSD based on rotational flag
		if deviceType == "HDD" && !isWindowsDrive && isNonRotational(physicalDrive) {
			deviceType = "SSD"
		}

		storageInfo := StorageInfo{
			Device:      partition.Device,
			Mountpoint:  partition.Mountpoint,
			Filesystem:  partition.Fstype,
			Type:        deviceType,
			Size:        usage.Total,
			Used:        usage.Used,
			Free:        usage.Free,
			UsedPercent: usage.UsedPercent,
		}

		// Get model information
		if model, ok := driveModels[physicalDrive]; ok {
			storageInfo.Model = model.Model
			storageInfo.Vendor = model.Vendor
			storageInfo.Serial = model.Serial
			storageInfo.Firmware = model.Firmware
			storageInfo.Interface = model.Interface
			// Use MediaType if available
			if model.MediaType != "" {
				storageInfo.Type = model.MediaType
			}
		}

		// Try to get model from mount point for Windows drives
		if storageInfo.Model == "" && len(driveModels) > 0 {
			// For Windows, try the mount point directly (e.g., "C:")
			mountKey := strings.TrimSuffix(partition.Mountpoint, "/")
			if strings.HasPrefix(partition.Mountpoint, "/mnt/") && len(partition.Mountpoint) == 6 {
				// WSL mount point like /mnt/c -> C:
				mountKey = strings.ToUpper(string(partition.Mountpoint[5])) + ":"
			}
			if model, ok := driveModels[mountKey]; ok {
				storageInfo.Model = model.Model
				storageInfo.Vendor = model.Vendor
				storageInfo.Serial = model.Serial
				storageInfo.Firmware = model.Firmware
				storageInfo.Interface = model.Interface
				// Use MediaType if available
				if model.MediaType != "" {
					storageInfo.Type = model.MediaType
				}
			}
		}

		// Determine device type based on model and interface information
		// Only do this if MediaType wasn't already set from PowerShell data
		if storageInfo.Type != "SSD" && storageInfo.Type != "HDD" && isWindowsDrive && storageInfo.Model != "" {
			// For Windows drives, use model and interface info to determine type
			modelLower := strings.ToLower(storageInfo.Model)
			interfaceLower := strings.ToLower(storageInfo.Interface)

			// Check interface type first
			switch {
			case strings.Contains(interfaceLower, "nvme") || storageInfo.Interface == "NVMe" ||
				storageInfo.Interface == "NVMe (RAID)" || strings.Contains(interfaceLower, "pcie") ||
				(strings.Contains(interfaceLower, "raid") && strings.Contains(modelLower, "amd")):
				storageInfo.Type = "NVME"
			case strings.Contains(modelLower, "nvme") || strings.Contains(modelLower, "980 pro") ||
				strings.Contains(modelLower, "970 evo") || strings.Contains(modelLower, "rocket") ||
				strings.Contains(modelLower, "9100 pro") || strings.Contains(modelLower, "9200"):
				// Check model name for NVMe - include common NVMe drive models
				// Samsung 9100 PRO is actually an NVMe drive
				storageInfo.Type = "NVME"
			case strings.Contains(modelLower, "ssd") || strings.Contains(modelLower, "solid state") ||
				strings.Contains(modelLower, "crucial") || strings.Contains(modelLower, "kingston") ||
				strings.Contains(modelLower, "sandisk"):
				// Check model name for SSD
				storageInfo.Type = "SSD"
			case strings.Contains(interfaceLower, "usb"):
				storageInfo.Type = "USB"
			case strings.Contains(modelLower, "st") && len(modelLower) > 2 && modelLower[2] >= '0' && modelLower[2] <= '9':
				// Seagate HDDs often start with ST followed by capacity (e.g., ST10000VN0008)
				storageInfo.Type = "HDD"
			case strings.Contains(modelLower, "wd") && !strings.Contains(modelLower, "ssd") && !strings.Contains(modelLower, "nvme"):
				// Western Digital HDDs
				storageInfo.Type = "HDD"
			case strings.Contains(modelLower, "hgst") || strings.Contains(modelLower, "hitachi") ||
				strings.Contains(modelLower, "toshiba") && !strings.Contains(modelLower, "ssd"):
				// Other HDD manufacturers
				storageInfo.Type = "HDD"
			default:
				// Default to SSD for modern drives if we can't determine
				// Most modern drives are SSDs, especially in systems with multiple drives
				storageInfo.Type = "SSD"
			}
		} else if storageInfo.Type != "SSD" && storageInfo.Type != "HDD" {
			// For non-Windows drives or when MediaType wasn't set, keep the original detection
			storageInfo.Type = deviceType
		}

		// Get SMART data for the physical drive
		storageInfo.SMART = getSMARTData(physicalDrive)

		storageDevices = append(storageDevices, storageInfo)
	}

	return storageDevices, nil
}

// GetUSBDevices returns information about USB devices
func GetUSBDevices() ([]USBDevice, error) {
	// This would require platform-specific implementation
	// For now, return empty list
	return []USBDevice{}, nil
}

// USBDevice represents a USB device
type USBDevice struct {
	Name      string
	Vendor    string
	Product   string
	VendorID  string
	ProductID string
}

// DriveModel holds drive identification info
type DriveModel struct {
	Model     string
	Vendor    string
	Serial    string
	Firmware  string
	Interface string // SATA, NVMe, USB, etc.
	MediaType string // SSD, HDD
}

// getPhysicalDrive extracts the physical drive from a partition device path
func getPhysicalDrive(device string) string {
	// Remove partition numbers from device path
	// e.g., /dev/sda1 -> /dev/sda, /dev/nvme0n1p1 -> /dev/nvme0n1
	if strings.Contains(device, "nvme") {
		// NVMe devices: /dev/nvme0n1p1 -> /dev/nvme0n1
		re := regexp.MustCompile(`^(/dev/nvme\d+n\d+)p?\d*$`)
		matches := re.FindStringSubmatch(device)
		if len(matches) > 1 {
			return matches[1]
		}
	} else {
		// Regular devices: /dev/sda1 -> /dev/sda
		re := regexp.MustCompile(`^(/dev/[a-z]+)\d*$`)
		matches := re.FindStringSubmatch(device)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	return device
}

// getDriveModels returns a map of physical drives to their model information
func getDriveModels() map[string]DriveModel {
	models := make(map[string]DriveModel)

	// Check if running on Windows or WSL
	if isWindows() || isWSL() {
		// Try to get Windows drive info using V2 implementation with better NVMe detection
		if driveInfo := GetWindowsDriveModelsV2(); len(driveInfo) > 0 {
			// Enhance with proper bus type detection on native Windows
			if isWindows() && !isWSL() {
				for driveLetter, model := range driveInfo {
					if len(driveLetter) >= 2 && driveLetter[1] == ':' {
						letter := string(driveLetter[0])
						busType, err := GetDriveBusType(letter)
						if err == nil {
							model.Interface = busType
							driveInfo[driveLetter] = model
							DebugLog("STORAGE", fmt.Sprintf("Enhanced detection: Drive %s is %s", driveLetter, busType))
						}
					}
				}
			}
			return driveInfo
		}
		// If V2 fails, try the original implementation
		if driveInfo := getDriveModelsWindows(); len(driveInfo) > 0 {
			return driveInfo
		}
	}

	// Try multiple methods to get drive information

	// Method 1: Try lsblk with JSON output
	if driveInfo := getDriveModelsFromLsblk(); len(driveInfo) > 0 {
		return driveInfo
	}

	// Method 2: Read from /sys/block
	if driveInfo := getDriveModelsFromSysBlock(); len(driveInfo) > 0 {
		return driveInfo
	}

	// Method 3: Try smartctl if available
	if driveInfo := getDriveModelsFromSmartctl(); len(driveInfo) > 0 {
		return driveInfo
	}

	return models
}

// getDriveModelsFromLsblk uses lsblk to get drive models
func getDriveModelsFromLsblk() map[string]DriveModel {
	models := make(map[string]DriveModel)

	// Try lsblk with specific columns
	cmd := exec.Command("lsblk", "-d", "-n", "-o", "NAME,MODEL,VENDOR,SERIAL")
	output, err := cmd.Output()
	if err != nil {
		return models
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse the output
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			device := "/dev/" + fields[0]
			model := fields[1]
			vendor := ""
			serial := ""

			if len(fields) >= 3 {
				vendor = fields[2]
			}
			if len(fields) >= 4 {
				serial = fields[3]
			}

			// Clean up the model string
			if model != "" && model != "-" {
				models[device] = DriveModel{
					Model:  strings.TrimSpace(model),
					Vendor: strings.TrimSpace(vendor),
					Serial: strings.TrimSpace(serial),
				}
			}
		}
	}

	return models
}

// getDriveModelsFromSysBlock reads drive info from /sys/block
func getDriveModelsFromSysBlock() map[string]DriveModel {
	models := make(map[string]DriveModel)

	// List all block devices
	blockPath := "/sys/block"
	entries, err := os.ReadDir(blockPath)
	if err != nil {
		return models
	}

	for _, entry := range entries {
		if entry.IsDir() {
			deviceName := entry.Name()
			// Skip loop devices and ram disks
			if strings.HasPrefix(deviceName, "loop") || strings.HasPrefix(deviceName, "ram") {
				continue
			}

			device := "/dev/" + deviceName
			devicePath := filepath.Join(blockPath, deviceName)

			// Read model
			model := readSysFile(filepath.Join(devicePath, "device", "model"))
			vendor := readSysFile(filepath.Join(devicePath, "device", "vendor"))
			serial := readSysFile(filepath.Join(devicePath, "device", "serial"))

			// For NVMe devices, try different paths
			if model == "" && strings.HasPrefix(deviceName, "nvme") {
				model = readSysFile(filepath.Join(devicePath, "device", "device", "model"))
				vendor = readSysFile(filepath.Join(devicePath, "device", "device", "vendor"))
				serial = readSysFile(filepath.Join(devicePath, "device", "device", "serial"))
			}

			if model != "" {
				models[device] = DriveModel{
					Model:  strings.TrimSpace(model),
					Vendor: strings.TrimSpace(vendor),
					Serial: strings.TrimSpace(serial),
				}
			}
		}
	}

	return models
}

// getDriveModelsFromSmartctl uses smartctl to get drive information
func getDriveModelsFromSmartctl() map[string]DriveModel {
	models := make(map[string]DriveModel)

	// List all drives using smartctl --scan
	cmd := exec.Command("smartctl", "--scan")
	output, err := cmd.Output()
	if err != nil {
		return models
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		// Parse device from scan output
		fields := strings.Fields(line)
		if len(fields) >= 1 {
			device := fields[0]

			// Get device info
			infoCmd := exec.Command("smartctl", "-i", device) // #nosec G204 -- device comes from trusted smartctl --scan output
			infoOutput, err := infoCmd.Output()
			if err == nil {
				model := extractSmartctlField(string(infoOutput), "Device Model:")
				if model == "" {
					model = extractSmartctlField(string(infoOutput), "Model Number:")
				}
				vendor := extractSmartctlField(string(infoOutput), "Vendor:")
				serial := extractSmartctlField(string(infoOutput), "Serial Number:")

				if model != "" {
					models[device] = DriveModel{
						Model:  model,
						Vendor: vendor,
						Serial: serial,
					}
				}
			}
		}
	}

	return models
}

// isNonRotational checks if a drive is non-rotational (SSD)
func isNonRotational(device string) bool {
	// Extract device name from path
	deviceName := filepath.Base(device)

	// Check rotational flag in sysfs
	rotationalPath := filepath.Join("/sys/block", deviceName, "queue", "rotational")
	data, err := os.ReadFile(rotationalPath) // nolint:gosec // G304 - sysfs path constructed from device name
	if err == nil {
		rotational := strings.TrimSpace(string(data))
		return rotational == "0"
	}

	return false
}

// getSMARTData retrieves SMART data for a physical drive
func getSMARTData(device string) *SMARTData {
	smart := &SMARTData{
		Available: false,
	}

	// Try smartctl first
	cmd := exec.Command("smartctl", "-A", "-H", device)
	output, err := cmd.Output()
	if err != nil {
		// smartctl returns non-zero exit code even on success sometimes
		// Check if we got any output
		if len(output) == 0 {
			return smart
		}
	}

	outputStr := string(output)

	// Check health status
	switch {
	case strings.Contains(outputStr, "SMART overall-health self-assessment test result: PASSED"):
		smart.HealthStatus = "Good"
	case strings.Contains(outputStr, "SMART overall-health self-assessment test result: FAILED"):
		smart.HealthStatus = "Critical"
	default:
		smart.HealthStatus = "Unknown"
	}

	// Extract temperature
	if temp := extractSmartAttribute(outputStr, "194", "Temperature_Celsius"); temp != "" {
		if val, err := strconv.ParseFloat(temp, 64); err == nil {
			smart.Temperature = val
			smart.Available = true
		}
	} else if temp := extractSmartAttribute(outputStr, "190", "Airflow_Temperature_Cel"); temp != "" {
		if val, err := strconv.ParseFloat(temp, 64); err == nil {
			smart.Temperature = val
			smart.Available = true
		}
	}

	// Extract power-on hours
	if hours := extractSmartAttribute(outputStr, "9", "Power_On_Hours"); hours != "" {
		if val, err := strconv.ParseUint(hours, 10, 64); err == nil {
			smart.PowerOnHours = val
			smart.Available = true
		}
	}

	// Extract power cycles
	if cycles := extractSmartAttribute(outputStr, "12", "Power_Cycle_Count"); cycles != "" {
		if val, err := strconv.ParseUint(cycles, 10, 64); err == nil {
			smart.PowerCycles = val
			smart.Available = true
		}
	}

	// Extract wear level for SSDs
	if wear := extractSmartAttribute(outputStr, "177", "Wear_Leveling_Count"); wear != "" {
		if val, err := strconv.ParseFloat(wear, 64); err == nil {
			smart.WearLevel = 100 - val // Convert to percentage used
			smart.Available = true
		}
	} else if wear := extractSmartAttribute(outputStr, "231", "SSD_Life_Left"); wear != "" {
		if val, err := strconv.ParseFloat(wear, 64); err == nil {
			smart.WearLevel = 100 - val
			smart.Available = true
		}
	}

	// Extract total written (LBAs)
	if written := extractSmartAttribute(outputStr, "241", "Total_LBAs_Written"); written != "" {
		if val, err := strconv.ParseFloat(written, 64); err == nil {
			// Convert LBAs to GB (assuming 512 bytes per LBA)
			smart.TotalWrittenGB = val * 512 / (1024 * 1024 * 1024)
			smart.Available = true
		}
	}

	// Extract total read (LBAs)
	if read := extractSmartAttribute(outputStr, "242", "Total_LBAs_Read"); read != "" {
		if val, err := strconv.ParseFloat(read, 64); err == nil {
			// Convert LBAs to GB (assuming 512 bytes per LBA)
			smart.TotalReadGB = val * 512 / (1024 * 1024 * 1024)
			smart.Available = true
		}
	}

	return smart
}

// Helper functions

func readSysFile(path string) string {
	data, err := os.ReadFile(path) // nolint:gosec // G304 - internal helper for reading sysfs files
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func extractSmartctlField(output, field string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, field) {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

func extractSmartAttribute(output, id, name string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		// SMART attributes are formatted with fixed columns
		fields := strings.Fields(line)
		if len(fields) >= 10 {
			// Check if this line has the attribute ID we're looking for
			if fields[0] == id || strings.Contains(fields[1], name) {
				// RAW_VALUE is typically in the last column
				return fields[len(fields)-1]
			}
		}
	}
	return ""
}

// getDriveModelsWindows gets drive models on Windows using multiple methods
func getDriveModelsWindows() map[string]DriveModel {
	startTime := time.Now()
	defer func() {
		DebugLog("PERF", fmt.Sprintf("getDriveModelsWindows took %v", time.Since(startTime)))
	}()

	models := make(map[string]DriveModel)

	// Method 0: Try the improved PowerShell implementation first (most accurate)
	if v2Models := getDriveModelsFromPowerShellV2(); len(v2Models) > 0 {
		DebugLog("STORAGE", fmt.Sprintf("Using V2 models, found %d drives", len(v2Models)))
		return v2Models
	}

	// Method 1: Try PowerShell Get-PhysicalDisk first (better for NVMe)
	if psModels := getDriveModelsFromPowerShell(); len(psModels) > 0 {
		// Merge with existing models
		for k, v := range psModels {
			models[k] = v
		}
	}

	// Method 1b: Try Storage Spaces/MSFT_Disk for additional info
	if msftModels := getDriveModelsFromMSFTDisk(); len(msftModels) > 0 {
		// Merge/update with existing models
		for k, v := range msftModels {
			if existing, ok := models[k]; ok {
				// Update with better info if available
				if existing.Model == "" || strings.Contains(strings.ToLower(existing.Model), "raid") {
					models[k] = v
				}
			} else {
				models[k] = v
			}
		}
	}

	// Method 2: Traditional WMI diskdrive query
	// Build the wmic command - get more detailed drive info
	var cmd *exec.Cmd
	if isWindows() {
		cmd = exec.Command("cmd", "/c", "wmic diskdrive get Model,Size,InterfaceType,MediaType,SerialNumber,FirmwareRevision,Index,Caption /format:csv")
	} else {
		// WSL
		cmd = exec.Command("cmd.exe", "/c", "wmic diskdrive get Model,Size,InterfaceType,MediaType,SerialNumber,FirmwareRevision,Index,Caption /format:csv")
	}

	output, err := cmd.Output()
	if err != nil {
		return models
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
		if len(headers) == 0 && len(fields) > 1 && strings.Contains(line, "Model") {
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

		// Get the index to map to drive letters later
		indexStr := fieldMap["Index"]
		model := fieldMap["Model"]
		caption := fieldMap["Caption"]
		serial := fieldMap["SerialNumber"]
		firmware := fieldMap["FirmwareRevision"]
		interfaceType := fieldMap["InterfaceType"]

		driveIndex, _ := strconv.Atoi(indexStr)

		// Skip RAID controller entries if we already have better info from PowerShell
		if model != "" && (strings.Contains(strings.ToLower(model), "raid") ||
			strings.Contains(strings.ToLower(model), "scsi") ||
			strings.Contains(strings.ToLower(model), "controller")) {
			// Check if we already have this drive from PowerShell
			driveLetters := getDriveLettersForDisk(driveIndex)
			if len(driveLetters) > 0 {
				if _, exists := models[driveLetters[0]]; exists {
					continue // Skip this RAID entry
				}
			}
			// Try to extract actual drive model from caption
			if caption != "" && !strings.Contains(strings.ToLower(caption), model) {
				model = caption
			}
		}

		// Clean up model name - remove common prefixes
		model = strings.TrimPrefix(model, "WDC ")
		model = strings.TrimPrefix(model, "ST")
		if strings.HasPrefix(model, "Samsung ") {
			model = strings.TrimPrefix(model, "Samsung ")
			model = "Samsung " + model
		}

		if model != "" {

			// Get drive letters for this physical disk
			driveLetters := getDriveLettersForDisk(driveIndex)

			// Determine vendor from model
			vendor := ""
			modelLower := strings.ToLower(model)
			switch {
			case strings.Contains(modelLower, "samsung"):
				vendor = "Samsung"
			case strings.Contains(modelLower, "western digital") || strings.Contains(modelLower, "wd"):
				vendor = "Western Digital"
			case strings.Contains(modelLower, "seagate"):
				vendor = "Seagate"
			case strings.Contains(modelLower, "crucial"):
				vendor = "Crucial"
			case strings.Contains(modelLower, "kingston"):
				vendor = "Kingston"
			case strings.Contains(modelLower, "sandisk"):
				vendor = "SanDisk"
			case strings.Contains(modelLower, "toshiba"):
				vendor = "Toshiba"
			}

			for _, driveLetter := range driveLetters {
				// Debug log
				DebugLog("STORAGE", fmt.Sprintf("WMI: Mapping disk %d (%s) to drive %s", driveIndex, model, driveLetter))

				driveModel := DriveModel{
					Model:  model,
					Vendor: vendor,
					Serial: serial,
				}

				// Add firmware if available
				if firmware != "" {
					driveModel.Firmware = firmware
				}

				// Add interface type
				if interfaceType != "" {
					driveModel.Interface = interfaceType
				}

				models[driveLetter] = driveModel
			}
		}
	}

	return models
}

// getDriveLettersForDisk gets all drive letters associated with a physical disk
func getDriveLettersForDisk(diskIndex int) []string {
	var driveLetters []string

	// Method 1: Try to get logical disks directly from disk index using associations
	var assocCmd *exec.Cmd
	if isWindows() {
		// Query for logical disks associated with this physical disk
		assocCmd = exec.Command("cmd", "/c", fmt.Sprintf("wmic path Win32_DiskDriveToDiskPartition where Antecedent='Win32_DiskDrive.DeviceID=\"\\\\\\\\.\\\\PHYSICALDRIVE%d\"' get Dependent /value", diskIndex)) // #nosec G204 - diskIndex is a validated integer from WMI query
	} else {
		assocCmd = exec.Command("cmd.exe", "/c", fmt.Sprintf("wmic path Win32_DiskDriveToDiskPartition where Antecedent='Win32_DiskDrive.DeviceID=\"\\\\\\\\.\\\\PHYSICALDRIVE%d\"' get Dependent /value", diskIndex)) // #nosec G204 - diskIndex is a validated integer from WMI query
	}

	output, err := assocCmd.Output()
	if err == nil && len(output) > 0 {
		// Parse partition associations
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "Disk #") && strings.Contains(line, "Partition #") {
				// Extract partition info
				// Format: Dependent=\\COMPUTER\root\cimv2:Win32_DiskPartition.DeviceID="Disk #0, Partition #1"
				start := strings.Index(line, "Disk #")
				end := strings.Index(line[start:], "\"") + start
				if start > -1 && end > start {
					partitionID := line[start:end]

					// Now get logical disk for this partition
					var logicalCmd *exec.Cmd
					if isWindows() {
						logicalCmd = exec.Command("cmd", "/c", fmt.Sprintf("wmic path Win32_LogicalDiskToPartition where Antecedent='Win32_DiskPartition.DeviceID=\"%s\"' get Dependent /value", partitionID)) // #nosec G204 - partitionID is validated from WMI output
					} else {
						logicalCmd = exec.Command("cmd.exe", "/c", fmt.Sprintf("wmic path Win32_LogicalDiskToPartition where Antecedent='Win32_DiskPartition.DeviceID=\"%s\"' get Dependent /value", partitionID)) // #nosec G204 - partitionID is validated from WMI output
					}

					logicalOutput, err := logicalCmd.Output()
					if err == nil {
						logicalLines := strings.Split(string(logicalOutput), "\n")
						for _, logicalLine := range logicalLines {
							// Extract drive letter from Dependent=\\COMPUTER\root\cimv2:Win32_LogicalDisk.DeviceID="C:"
							if strings.Contains(logicalLine, "DeviceID=") {
								start := strings.Index(logicalLine, "DeviceID=\"")
								if start > -1 {
									start += len("DeviceID=\"")
									end := strings.Index(logicalLine[start:], "\"")
									if end > -1 {
										driveLetter := logicalLine[start : start+end]
										if len(driveLetter) == 2 && driveLetter[1] == ':' {
											driveLetters = append(driveLetters, driveLetter)
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Method 2: If the above didn't work, try a simpler approach
	if len(driveLetters) == 0 {
		// Get all logical disks and their associated disk indices
		var cmd *exec.Cmd
		if isWindows() {
			cmd = exec.Command("cmd", "/c", "wmic logicaldisk where DriveType=3 get DeviceID,Size /format:csv")
		} else {
			cmd = exec.Command("cmd.exe", "/c", "wmic logicaldisk where DriveType=3 get DeviceID,Size /format:csv")
		}

		output, err := cmd.Output()
		if err == nil {
			// Parse and find which logical disks exist
			existingDrives := make(map[string]bool)
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.Contains(line, ":") && !strings.Contains(line, "DeviceID") {
					fields := strings.Split(line, ",")
					if len(fields) >= 2 {
						driveLetter := fields[len(fields)-2] // DeviceID is second to last
						driveLetter = strings.TrimSpace(driveLetter)
						driveLetter = strings.Trim(driveLetter, "\r")
						if len(driveLetter) == 2 && driveLetter[1] == ':' {
							existingDrives[driveLetter] = true
						}
					}
				}
			}

			// Make educated guesses based on disk index
			if diskIndex == 0 && existingDrives["C:"] {
				driveLetters = append(driveLetters, "C:")
			} else if diskIndex > 0 {
				// Try common drive letters in order
				letters := []string{"D:", "E:", "F:", "G:", "H:", "I:", "J:", "K:", "L:", "M:", "N:", "O:", "P:"}
				assigned := 0
				for _, letter := range letters {
					if existingDrives[letter] && assigned < diskIndex {
						assigned++
						if assigned == diskIndex {
							driveLetters = append(driveLetters, letter)
							break
						}
					}
				}
			}
		}
	}

	// Fallback: If we still couldn't map it, make a basic guess
	if len(driveLetters) == 0 {
		if diskIndex == 0 {
			driveLetters = append(driveLetters, "C:")
		} else {
			// Try to find the next available drive letter
			letter := byte('D' + diskIndex - 1)
			if letter <= 'Z' {
				driveLetters = append(driveLetters, fmt.Sprintf("%c:", letter))
			}
		}
	}

	return driveLetters
}

// getDriveModelsFromPowerShell uses PowerShell Get-PhysicalDisk for better NVMe detection
func getDriveModelsFromPowerShell() map[string]DriveModel {
	startTime := time.Now()
	defer func() {
		DebugLog("PERF", fmt.Sprintf("getDriveModelsFromPowerShell took %v", time.Since(startTime)))
	}()

	models := make(map[string]DriveModel)

	// PowerShell command to get physical disk info with disk number
	psCmd := `Get-PhysicalDisk | Select-Object @{Name='DeviceId';Expression={$_.DeviceId}}, ` +
		`@{Name='DiskNumber';Expression={$_.DeviceId -replace '.*physicaldrive(\d+).*','$1'}}, ` +
		`@{Name='FriendlyName';Expression={$_.FriendlyName}}, ` +
		`@{Name='SerialNumber';Expression={$_.SerialNumber}}, ` +
		`@{Name='MediaType';Expression={$_.MediaType}}, ` +
		`@{Name='BusType';Expression={$_.BusType}}, ` +
		`@{Name='Size';Expression={$_.Size}}, ` +
		`@{Name='Model';Expression={$_.Model}}, ` +
		`@{Name='FirmwareVersion';Expression={$_.FirmwareVersion}} | ` +
		`ConvertTo-Json -Compress`

	var cmd *exec.Cmd
	if isWindows() {
		cmd = exec.Command("powershell", "-NoProfile", "-Command", psCmd)
	} else {
		// WSL
		cmd = exec.Command("powershell.exe", "-NoProfile", "-Command", psCmd)
	}

	output, err := cmd.Output()
	if err != nil {
		return models
	}

	// Parse JSON output
	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" {
		return models
	}

	// PowerShell might return a single object or array
	if !strings.HasPrefix(outputStr, "[") {
		outputStr = "[" + outputStr + "]"
	}

	// Simple JSON parsing for the fields we need
	// This is a simplified parser - in production you'd use encoding/json
	disks := strings.Split(outputStr, "},{")

	for _, disk := range disks {
		disk = strings.Trim(disk, "[]{}")

		// Extract fields
		friendlyName := extractJSONField(disk, "FriendlyName")
		serialNumber := extractJSONField(disk, "SerialNumber")
		mediaType := extractJSONField(disk, "MediaType")
		busType := extractJSONField(disk, "BusType")
		model := extractJSONField(disk, "Model")
		firmwareVersion := extractJSONField(disk, "FirmwareVersion")
		deviceID := extractJSONField(disk, "DeviceId")
		diskNumberStr := extractJSONField(disk, "DiskNumber")

		// Use Model if available, otherwise FriendlyName
		if model == "" {
			model = friendlyName
		}

		// Skip if no useful model info
		if model == "" || strings.Contains(strings.ToLower(model), "microsoft") {
			continue
		}

		// Try to map to drive letters
		// First, try to get disk number from DiskNumber field
		diskNum := -1
		if diskNumberStr != "" {
			if num, err := strconv.Atoi(diskNumberStr); err == nil {
				diskNum = num
				DebugLog("STORAGE", fmt.Sprintf("PowerShell: Using DiskNumber %d for %s", diskNum, model))
			}
		}

		// If that didn't work, try to extract from DeviceId
		if diskNum < 0 {
			diskNum = extractDiskNumber(deviceID)
		}
		if diskNum >= 0 {
			driveLetters := getDriveLettersForDisk(diskNum)

			// Determine vendor first (we'll need modelLower for interface detection)
			vendor := ""
			modelLower := strings.ToLower(model)

			// Determine interface type from BusType
			interfaceType := busType
			switch busType {
			case "NVMe":
				interfaceType = "NVMe"
			case "SATA", "ATA":
				interfaceType = "SATA"
			case "SCSI":
				// SCSI can be reported for NVMe drives behind certain controllers
				// Check model name for NVMe indicators
				if strings.Contains(modelLower, "nvme") ||
					strings.Contains(modelLower, "980 pro") ||
					strings.Contains(modelLower, "970 evo") ||
					strings.Contains(modelLower, "rocket") ||
					strings.Contains(modelLower, "9100 pro") ||
					strings.Contains(modelLower, "9200") ||
					strings.Contains(modelLower, "9300") ||
					strings.Contains(modelLower, "9400") ||
					strings.Contains(modelLower, "9500") ||
					strings.Contains(modelLower, "sn850") ||
					strings.Contains(modelLower, "sn770") ||
					strings.Contains(modelLower, "sn750") {
					interfaceType = "NVMe"
				} else {
					// Likely a SATA drive (SSD or HDD)
					interfaceType = "SATA"
				}
			case "RAID":
				// This is an NVMe behind RAID
				if mediaType == "SSD" && strings.Contains(strings.ToLower(model), "nvme") {
					interfaceType = "NVMe (RAID)"
				} else {
					interfaceType = "RAID"
				}
			}
			switch {
			case strings.Contains(modelLower, "samsung"):
				vendor = "Samsung"
			case strings.Contains(modelLower, "western digital") || strings.Contains(modelLower, "wd"):
				vendor = "Western Digital"
			case strings.Contains(modelLower, "seagate"):
				vendor = "Seagate"
			case strings.Contains(modelLower, "crucial"):
				vendor = "Crucial"
			case strings.Contains(modelLower, "kingston"):
				vendor = "Kingston"
			case strings.Contains(modelLower, "sandisk"):
				vendor = "SanDisk"
			case strings.Contains(modelLower, "intel"):
				vendor = "Intel"
			}

			for _, driveLetter := range driveLetters {
				// Debug log
				DebugLog("STORAGE", fmt.Sprintf("PowerShell: Mapping disk %d (%s, Serial: %s) to drive %s", diskNum, model, serialNumber, driveLetter))

				models[driveLetter] = DriveModel{
					Model:     model,
					Vendor:    vendor,
					Serial:    serialNumber,
					Firmware:  firmwareVersion,
					Interface: interfaceType,
				}
			}
		}
	}

	return models
}

// getDriveModelsFromPowerShellV2 uses an improved PowerShell approach for accurate drive mappings
func getDriveModelsFromPowerShellV2() map[string]DriveModel {
	models := make(map[string]DriveModel)

	// PowerShell script to get drive mappings
	psScript := `
$mappings = @()

# Get all physical disks
$disks = Get-PhysicalDisk

foreach ($disk in $disks) {
    # Get disk number from DeviceId
    $diskNumber = $null
    if ($disk.DeviceId -match 'physicaldrive(\d+)') {
        $diskNumber = [int]$matches[1]
    }
    
    if ($diskNumber -eq $null) {
        continue
    }
    
    # Get partitions for this disk
    $partitions = Get-Partition -DiskNumber $diskNumber -ErrorAction SilentlyContinue
    
    foreach ($partition in $partitions) {
        # Get volume for this partition
        $volume = Get-Volume -Partition $partition -ErrorAction SilentlyContinue
        
        if ($volume -and $volume.DriveLetter) {
            $mapping = @{
                DiskNumber = $diskNumber
                Model = if ($disk.Model) { $disk.Model } else { $disk.FriendlyName }
                SerialNumber = $disk.SerialNumber
                FirmwareVersion = $disk.FirmwareVersion
                MediaType = $disk.MediaType
                BusType = $disk.BusType
                DriveLetter = $volume.DriveLetter + ":"
                VolumeName = $volume.FileSystemLabel
            }
            $mappings += $mapping
        }
    }
}

$mappings | ConvertTo-Json -Compress
`

	var cmd *exec.Cmd
	if isWindows() {
		cmd = exec.Command("powershell", "-NoProfile", "-Command", psScript)
	} else {
		// WSL
		cmd = exec.Command("powershell.exe", "-NoProfile", "-Command", psScript)
	}

	output, err := cmd.Output()
	if err != nil {
		DebugLog("STORAGE", fmt.Sprintf("PowerShell V2 error: %v", err))
		return models
	}

	// Parse JSON output
	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" || outputStr == "null" {
		return models
	}

	// Ensure it's an array
	if !strings.HasPrefix(outputStr, "[") {
		outputStr = "[" + outputStr + "]"
	}

	// Simple JSON parsing
	disks := strings.Split(outputStr, "},{")

	for _, disk := range disks {
		disk = strings.Trim(disk, "[]{}")

		// Extract fields
		driveLetter := extractJSONField(disk, "DriveLetter")
		model := extractJSONField(disk, "Model")
		serialNumber := extractJSONField(disk, "SerialNumber")
		firmwareVersion := extractJSONField(disk, "FirmwareVersion")
		mediaType := extractJSONField(disk, "MediaType")
		busType := extractJSONField(disk, "BusType")

		if driveLetter == "" || model == "" {
			continue
		}

		// Skip Microsoft virtual disks
		if strings.Contains(strings.ToLower(model), "microsoft") {
			continue
		}

		// Determine vendor from model
		vendor := ""
		modelLower := strings.ToLower(model)
		switch {
		case strings.Contains(modelLower, "samsung"):
			vendor = "Samsung"
		case strings.Contains(modelLower, "western digital") || strings.Contains(modelLower, "wd"):
			vendor = "Western Digital"
		case strings.Contains(modelLower, "seagate"):
			vendor = "Seagate"
		case strings.Contains(modelLower, "crucial"):
			vendor = "Crucial"
		case strings.Contains(modelLower, "kingston"):
			vendor = "Kingston"
		case strings.Contains(modelLower, "sandisk"):
			vendor = "SanDisk"
		case strings.Contains(modelLower, "intel"):
			vendor = "Intel"
		case strings.Contains(modelLower, "toshiba"):
			vendor = "Toshiba"
		case strings.Contains(modelLower, "sabrent"):
			vendor = "Sabrent"
		case strings.Contains(modelLower, "micron"):
			vendor = "Micron"
		case strings.Contains(modelLower, "corsair"):
			vendor = "Corsair"
		}

		// Determine interface type
		interfaceType := ""
		switch busType {
		case "NVMe":
			interfaceType = "NVMe"
		case "SATA", "ATA":
			interfaceType = "SATA"
		case "SCSI":
			// SCSI can be reported for NVMe drives behind certain controllers
			// Check model name for NVMe indicators
			if strings.Contains(modelLower, "nvme") ||
				strings.Contains(modelLower, "980 pro") ||
				strings.Contains(modelLower, "970 evo") ||
				strings.Contains(modelLower, "rocket") ||
				strings.Contains(modelLower, "9100 pro") ||
				strings.Contains(modelLower, "9200") ||
				strings.Contains(modelLower, "sn850") ||
				strings.Contains(modelLower, "sn770") ||
				strings.Contains(modelLower, "sn750") {
				interfaceType = "NVMe"
			} else {
				// Likely a SATA drive (SSD or HDD)
				interfaceType = "SATA"
			}
		case "RAID":
			if mediaType == "SSD" && strings.Contains(strings.ToLower(model), "nvme") {
				interfaceType = "NVMe (RAID)"
			} else {
				interfaceType = "RAID"
			}
		case "USB":
			interfaceType = "USB"
		default:
			interfaceType = busType
		}

		DebugLog("STORAGE", fmt.Sprintf("PowerShell V2: Mapping %s (Serial: %s, BusType: %s) to drive %s",
			model, serialNumber, busType, driveLetter))

		models[driveLetter] = DriveModel{
			Model:     model,
			Vendor:    vendor,
			Serial:    serialNumber,
			Firmware:  firmwareVersion,
			Interface: interfaceType,
		}
	}

	return models
}

// extractJSONField extracts a field value from a JSON string
func extractJSONField(json, field string) string {
	// Look for "field":"value" pattern
	pattern := `"` + field + `":"([^"]*)"`
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(json)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// extractDiskNumber extracts disk number from DeviceId like "\\?\scsi#disk&ven..."
func extractDiskNumber(deviceID string) int {
	// Debug log the deviceID
	DebugLog("STORAGE", fmt.Sprintf("extractDiskNumber: deviceID=%s", deviceID))

	// Try to extract from the deviceID
	// Look for patterns like "physicaldrive0", "physicaldrive1", etc
	re := regexp.MustCompile(`physicaldrive(\d+)`)
	matches := re.FindStringSubmatch(strings.ToLower(deviceID))
	if len(matches) > 1 {
		if num, err := strconv.Atoi(matches[1]); err == nil {
			DebugLog("STORAGE", fmt.Sprintf("extractDiskNumber: found physicaldrive%d", num))
			return num
		}
	}

	// Look for disk number in format "disk&ven_...&prod_...&*_N"
	// where N is the disk number at the end
	re2 := regexp.MustCompile(`[&_](\d+)$`)
	matches2 := re2.FindStringSubmatch(deviceID)
	if len(matches2) > 1 {
		if num, err := strconv.Atoi(matches2[1]); err == nil {
			DebugLog("STORAGE", fmt.Sprintf("extractDiskNumber: found disk number %d at end", num))
			return num
		}
	}

	// Look for patterns like "#disk&ven_..._0" or "#disk&ven_..._1"
	parts := strings.Split(deviceID, "&")
	for i, part := range parts {
		// Check if this is the last part and contains a number
		if i == len(parts)-1 {
			// Extract trailing number
			re3 := regexp.MustCompile(`_(\d+)$`)
			matches3 := re3.FindStringSubmatch(part)
			if len(matches3) > 1 {
				if num, err := strconv.Atoi(matches3[1]); err == nil {
					DebugLog("STORAGE", fmt.Sprintf("extractDiskNumber: found disk number %d in last part", num))
					return num
				}
			}
		}
	}

	DebugLog("STORAGE", "extractDiskNumber: no disk number found, returning -1")
	return -1
}

// getDriveModelsFromMSFTDisk uses WMI MSFT_Disk for additional storage info
func getDriveModelsFromMSFTDisk() map[string]DriveModel {
	models := make(map[string]DriveModel)

	// PowerShell command to get MSFT_Disk info with drive letter mapping
	psCmd := `
		$disks = Get-WmiObject -Namespace root\Microsoft\Windows\Storage -Class MSFT_Disk
		$results = @()
		foreach ($disk in $disks) {
			$partitions = Get-WmiObject -Namespace root\Microsoft\Windows\Storage -Query "ASSOCIATORS OF {MSFT_Disk.ObjectId='$($disk.ObjectId)'} WHERE AssocClass=MSFT_DiskToPartition"
			foreach ($partition in $partitions) {
				$volumes = Get-WmiObject -Namespace root\Microsoft\Windows\Storage -Query "ASSOCIATORS OF {MSFT_Partition.ObjectId='$($partition.ObjectId)'} WHERE AssocClass=MSFT_PartitionToVolume"
				foreach ($volume in $volumes) {
					if ($volume.DriveLetter) {
						$results += @{
							DriveLetter = $volume.DriveLetter
							FriendlyName = $disk.FriendlyName
							SerialNumber = $disk.SerialNumber
							Model = $disk.Model
							FirmwareVersion = $disk.FirmwareRevision
							BusType = $disk.BusType
							MediaType = $disk.MediaType
						}
					}
				}
			}
		}
		$results | ConvertTo-Json -Compress
	`

	var cmd *exec.Cmd
	if isWindows() {
		cmd = exec.Command("powershell", "-NoProfile", "-Command", psCmd)
	} else {
		// WSL
		cmd = exec.Command("powershell.exe", "-NoProfile", "-Command", psCmd)
	}

	output, err := cmd.Output()
	if err != nil {
		return models
	}

	// Parse JSON output
	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" || outputStr == "null" {
		return models
	}

	// PowerShell might return a single object or array
	if !strings.HasPrefix(outputStr, "[") {
		outputStr = "[" + outputStr + "]"
	}

	// Simple JSON parsing
	disks := strings.Split(outputStr, "},{")

	for _, disk := range disks {
		disk = strings.Trim(disk, "[]{}")

		// Extract fields
		driveLetter := extractJSONField(disk, "DriveLetter")
		friendlyName := extractJSONField(disk, "FriendlyName")
		serialNumber := extractJSONField(disk, "SerialNumber")
		model := extractJSONField(disk, "Model")
		firmwareVersion := extractJSONField(disk, "FirmwareVersion")
		busType := extractJSONField(disk, "BusType")

		if driveLetter == "" {
			continue
		}

		// Add colon if not present
		if !strings.HasSuffix(driveLetter, ":") {
			driveLetter += ":"
		}

		// Use Model if available, otherwise FriendlyName
		displayModel := model
		if displayModel == "" {
			displayModel = friendlyName
		}

		// Skip if still no useful info
		if displayModel == "" || strings.Contains(strings.ToLower(displayModel), "microsoft") {
			continue
		}

		// Determine interface type
		interfaceType := ""
		switch busType {
		case "17": // NVMe
			interfaceType = "NVMe"
		case "11": // SATA
			interfaceType = "SATA"
		case "7": // USB
			interfaceType = "USB"
		case "8": // RAID
			interfaceType = "RAID"
		default:
			interfaceType = busType
		}

		// Extract vendor from model
		vendor := ""
		modelLower := strings.ToLower(displayModel)
		switch {
		case strings.Contains(modelLower, "samsung"):
			vendor = "Samsung"
		case strings.Contains(modelLower, "western digital") || strings.Contains(modelLower, "wd"):
			vendor = "Western Digital"
		case strings.Contains(modelLower, "seagate"):
			vendor = "Seagate"
		case strings.Contains(modelLower, "crucial"):
			vendor = "Crucial"
		case strings.Contains(modelLower, "intel"):
			vendor = "Intel"
		}

		// Debug log
		DebugLog("STORAGE", fmt.Sprintf("MSFT_Disk: Mapping %s (Serial: %s) to drive %s", displayModel, serialNumber, driveLetter))

		models[driveLetter] = DriveModel{
			Model:     displayModel,
			Vendor:    vendor,
			Serial:    serialNumber,
			Firmware:  firmwareVersion,
			Interface: interfaceType,
		}
	}

	return models
}
