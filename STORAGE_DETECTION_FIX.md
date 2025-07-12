# Storage Technology Detection Fix

## Problem
All drives were being incorrectly detected as "HDD" technology instead of their actual types (NVMe, SSD, HDD).

## Root Causes
1. Default device type was set to "HDD" for all drives
2. Detection logic was very basic and only checked device names (e.g., looking for "nvme" in device path)
3. In WSL, Windows drives weren't being detected at all because `disk.Partitions(false)` excludes 9p filesystem mounts
4. Model and interface information from Windows wasn't being used to determine drive technology

## Solution

### 1. Include All Partitions
Changed `disk.Partitions(false)` to `disk.Partitions(true)` to include Windows drives mounted via 9p filesystem in WSL.

### 2. Enhanced Drive Type Detection
Added comprehensive detection logic based on:
- Interface type (NVMe, SATA, USB, RAID)
- Model name patterns for common NVMe drives (970 EVO, 980 PRO, Rocket, etc.)
- Vendor-specific patterns (Samsung, Crucial, Kingston, etc.)
- HDD detection patterns (Seagate ST*, Western Digital WD*, etc.)

### 3. PowerShell V2 Implementation
Added an improved PowerShell implementation that:
- Uses Get-PhysicalDisk with proper partition/volume mapping
- Correctly identifies BusType for interface detection
- Handles vendor detection for various manufacturers

### 4. Model-Based Detection for Windows Drives
For Windows drives accessed through WSL, the detection now:
- Checks interface type first (NVMe, SATA, etc.)
- Falls back to model name analysis
- Correctly identifies common NVMe models
- Defaults to SSD for modern drives when uncertain (better than defaulting to HDD)

## Results
- Sabrent Rocket Q4 → Correctly detected as NVME
- Samsung SSD 970 EVO Plus 1TB → Correctly detected as NVME
- Samsung SSD 980 PRO 2TB → Correctly detected as NVME
- Samsung SSD 9100 PRO 2TB → Correctly detected as NVME
- Seagate ST10000VN0008 → Should be detected as HDD (based on model pattern)

## Files Modified
- `/mnt/d/Projects/project_fire/pkg/gui/storage_info.go` - Main storage detection logic
- `/mnt/d/Projects/project_fire/pkg/gui/storage_info_windows.go` - Windows-specific implementation (enhanced vendor detection)

## Future Improvements
1. The Interface field still shows "SCSI" for Windows drives in WSL due to virtualization layer - this could be improved by using the interface data from PowerShell
2. Consider using WMI MediaType field for additional validation
3. Add more drive model patterns as new drives are encountered