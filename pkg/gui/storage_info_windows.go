//go:build windows
// +build windows

package gui

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

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

// GetWindowsDriveMappings uses PowerShell to get accurate drive mappings
func GetWindowsDriveMappings() ([]WindowsDriveMapping, error) {
	// PowerShell script to get drive mappings using proper WMI associations
	psScript := `
$mappings = @()

# Use WMI to get disk drives and their associated logical disks
$wmiServices = Get-WmiObject -Query "SELECT * FROM Win32_DiskDrive"

foreach ($diskDrive in $wmiServices) {
    # Get disk index/number
    $diskNumber = $diskDrive.Index
    
    # Get bus type from MSFT_Disk for accurate detection
    $detectedBusType = "Unknown"
    $mediaType = "HDD"  # Default
    
    # First try to get media type from Get-PhysicalDisk which has accurate info
    try {
        $physicalDisk = Get-PhysicalDisk | Where-Object { $_.DeviceId -eq $diskNumber }
        if ($physicalDisk) {
            
            $pdMediaType = $physicalDisk.MediaType
            if ($pdMediaType -eq "SSD") {
                $mediaType = "SSD"
            } elseif ($pdMediaType -eq "HDD") {
                $mediaType = "HDD"
            } elseif ($pdMediaType -eq "Unspecified" -or $pdMediaType -eq $null -or $pdMediaType -eq "") {
                # For unspecified, check model name
                if ($diskDrive.Model -match "SSD|Solid State|NVMe") {
                    $mediaType = "SSD"
                } else {
                    $mediaType = "HDD"
                }
            } else {
                # Unknown media type value, default based on other indicators
                if ($detectedBusType -eq "NVMe" -or $diskDrive.Model -match "SSD|Solid State") {
                    $mediaType = "SSD"
                }
            }
        } else {
            # No matching PhysicalDisk found, use fallback detection
            if ($detectedBusType -eq "NVMe" -or $diskDrive.Model -match "SSD|Solid State|NVMe") {
                $mediaType = "SSD"
            }
        }
    } catch {
        # Error accessing PhysicalDisk, use fallback detection
        if ($detectedBusType -eq "NVMe" -or $diskDrive.Model -match "SSD|Solid State|NVMe") {
            $mediaType = "SSD"
        }
    }
    
    try {
        $msftDisk = Get-WmiObject -Namespace root\Microsoft\Windows\Storage -Query "SELECT * FROM MSFT_Disk WHERE Number=$diskNumber" -ErrorAction Stop
        if ($msftDisk) {
            # BusType values: 17=NVMe, 11=SATA, 8=RAID, 7=USB, 9=iSCSI, 1=SCSI
            switch ($msftDisk.BusType) {
                17 { $detectedBusType = "NVMe" }
                11 { $detectedBusType = "SATA" }
                8 { 
                    if ($diskDrive.Model -match "AMD-RAID") {
                        $detectedBusType = "NVMe (RAID)"
                    } else {
                        $detectedBusType = "RAID"
                    }
                }
                7 { $detectedBusType = "USB" }
                9 { $detectedBusType = "iSCSI" }
                1 { 
                    # SCSI - check if actually NVMe
                    if ($diskDrive.PNPDeviceID -match "VEN_NVME") {
                        $detectedBusType = "NVMe"
                    } else {
                        $detectedBusType = "SCSI"
                    }
                }
                default { $detectedBusType = "BusType_$($msftDisk.BusType)" }
            }
        }
    } catch {
        # Fallback to checking PNPDeviceID
        if ($diskDrive.PNPDeviceID -match "VEN_NVME") {
            $detectedBusType = "NVMe"
        } elseif ($diskDrive.InterfaceType) {
            $detectedBusType = $diskDrive.InterfaceType
        }
        
        # Fallback media type detection
        if ($diskDrive.Model -match "SSD|Solid State" -or $detectedBusType -eq "NVMe") {
            $mediaType = "SSD"
        }
    }
    
    # Get associated partitions
    $query = "ASSOCIATORS OF {Win32_DiskDrive.DeviceID='$($diskDrive.DeviceID)'} WHERE AssocClass = Win32_DiskDriveToDiskPartition"
    $partitions = Get-WmiObject -Query $query
    
    foreach ($partition in $partitions) {
        # Get associated logical disk
        $query = "ASSOCIATORS OF {Win32_DiskPartition.DeviceID='$($partition.DeviceID)'} WHERE AssocClass = Win32_LogicalDiskToPartition"
        $logicalDisks = Get-WmiObject -Query $query
        
        foreach ($logicalDisk in $logicalDisks) {
            $mapping = @{
                DiskNumber = $diskNumber
                Model = $diskDrive.Model
                SerialNumber = $diskDrive.SerialNumber
                FirmwareVersion = $diskDrive.FirmwareRevision
                MediaType = $mediaType
                BusType = $detectedBusType
                DriveLetter = $logicalDisk.DeviceID
                VolumeName = $logicalDisk.VolumeName
            }
            $mappings += $mapping
        }
    }
}

if ($mappings.Count -eq 0) {
    "[]"
} else {
    $mappings | ConvertTo-Json -Compress
}
`

	var cmd *exec.Cmd
	if isWindows() {
		cmd = exec.Command("powershell", "-NoProfile", "-Command", psScript)
	} else {
		// WSL
		cmd = exec.Command("powershell.exe", "-NoProfile", "-Command", psScript)
	}

	output, err := cmd.CombinedOutput() // Get both stdout and stderr
	if err != nil {
		DebugLog("STORAGE", fmt.Sprintf("PowerShell execution error: %v, output: %s", err, string(output)))
		return nil, fmt.Errorf("failed to execute PowerShell: %w", err)
	}

	// Parse JSON output
	outputStr := strings.TrimSpace(string(output))
	DebugLog("STORAGE", fmt.Sprintf("PowerShell raw output: %s", outputStr))
	
	if outputStr == "" || outputStr == "null" {
		return nil, fmt.Errorf("no drive mappings found")
	}

	// Ensure it's an array
	if !strings.HasPrefix(outputStr, "[") {
		outputStr = "[" + outputStr + "]"
	}

	var mappings []WindowsDriveMapping
	err = json.Unmarshal([]byte(outputStr), &mappings)
	if err != nil {
		DebugLog("STORAGE", fmt.Sprintf("JSON parse error: %v", err))
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	
	if len(mappings) == 0 {
		return nil, fmt.Errorf("no drive mappings found")
	}

	return mappings, nil
}

// GetWindowsDriveModelsV2 uses the new mapping approach
func GetWindowsDriveModelsV2() map[string]DriveModel {
	models := make(map[string]DriveModel)

	mappings, err := GetWindowsDriveMappings()
	if err != nil {
		DebugLog("STORAGE", fmt.Sprintf("GetWindowsDriveMappings error: %v", err))
		return models
	}

	DebugLog("STORAGE", fmt.Sprintf("Found %d drive mappings from V2 method", len(mappings)))

	for _, mapping := range mappings {
		// Determine vendor from model
		vendor := ""
		modelLower := strings.ToLower(mapping.Model)
		if strings.Contains(modelLower, "samsung") {
			vendor = "Samsung"
		} else if strings.Contains(modelLower, "western digital") || strings.Contains(modelLower, "wd") {
			vendor = "Western Digital"
		} else if strings.Contains(modelLower, "seagate") {
			vendor = "Seagate"
		} else if strings.Contains(modelLower, "crucial") {
			vendor = "Crucial"
		} else if strings.Contains(modelLower, "kingston") {
			vendor = "Kingston"
		} else if strings.Contains(modelLower, "sandisk") {
			vendor = "SanDisk"
		} else if strings.Contains(modelLower, "intel") {
			vendor = "Intel"
		} else if strings.Contains(modelLower, "toshiba") {
			vendor = "Toshiba"
		} else if strings.Contains(modelLower, "sabrent") {
			vendor = "Sabrent"
		} else if strings.Contains(modelLower, "micron") {
			vendor = "Micron"
		} else if strings.Contains(modelLower, "corsair") {
			vendor = "Corsair"
		} else if strings.Contains(modelLower, "amd-raid") || strings.Contains(modelLower, "amd raid") {
			vendor = "AMD"
		}

		// Determine interface type
		interfaceType := ""

		switch mapping.BusType {
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
			} else if strings.Contains(modelLower, "ssd") || mapping.MediaType == "SSD" {
				// Likely a SATA SSD
				interfaceType = "SATA"
			} else {
				// Likely a SATA HDD
				interfaceType = "SATA"
			}
		case "RAID":
			// Check for AMD RAID arrays which are typically NVMe
			if strings.Contains(modelLower, "amd-raid") || strings.Contains(modelLower, "array") {
				interfaceType = "NVMe (RAID)"
			} else if mapping.MediaType == "SSD" && strings.Contains(modelLower, "nvme") {
				interfaceType = "NVMe (RAID)"
			} else {
				interfaceType = "RAID"
			}
		case "USB":
			interfaceType = "USB"
		default:
			interfaceType = mapping.BusType
		}

		DebugLog("STORAGE", fmt.Sprintf("Mapping disk %d (%s, Serial: %s) to drive %s",
			mapping.DiskNumber, mapping.Model, mapping.SerialNumber, mapping.DriveLetter))

		models[mapping.DriveLetter] = DriveModel{
			Model:     mapping.Model,
			Vendor:    vendor,
			Serial:    mapping.SerialNumber,
			Firmware:  mapping.FirmwareVersion,
			Interface: interfaceType,
			MediaType: mapping.MediaType,
		}
	}

	return models
}
