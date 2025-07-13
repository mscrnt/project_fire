package gui

import (
	"fmt"
)

// DebugStorageInfo prints detailed debug information about storage detection
func DebugStorageInfo() {
	fmt.Println("=== Storage Debug Info ===")

	// Get drive models
	models := getDriveModels()
	fmt.Printf("\nFound %d drive models:\n", len(models))
	for key, model := range models {
		fmt.Printf("  Key: %s\n", key)
		fmt.Printf("    Model: %s\n", model.Model)
		fmt.Printf("    Serial: %s\n", model.Serial)
		fmt.Printf("    Vendor: %s\n", model.Vendor)
		fmt.Printf("    Firmware: %s\n", model.Firmware)
		fmt.Printf("    Interface: %s\n", model.Interface)
		fmt.Println()
	}

	// Test getDriveLettersForDisk
	fmt.Println("\nTesting getDriveLettersForDisk:")
	for i := 0; i < 5; i++ {
		letters := getDriveLettersForDisk(i)
		fmt.Printf("  Disk %d -> %v\n", i, letters)
	}

	// Get full storage info
	storageDevices, err := GetStorageInfo()
	if err != nil {
		fmt.Printf("\nError getting storage info: %v\n", err)
		return
	}

	fmt.Printf("\nFound %d storage devices:\n", len(storageDevices))
	for i := range storageDevices {
		device := &storageDevices[i]
		fmt.Printf("\nDevice %d:\n", i+1)
		fmt.Printf("  Mount: %s\n", device.Mountpoint)
		fmt.Printf("  Device: %s\n", device.Device)
		fmt.Printf("  Type: %s\n", device.Type)
		fmt.Printf("  Model: %s\n", device.Model)
		fmt.Printf("  Serial: %s\n", device.Serial)
		fmt.Printf("  Vendor: %s\n", device.Vendor)
		fmt.Printf("  Firmware: %s\n", device.Firmware)
		fmt.Printf("  Interface: %s\n", device.Interface)
	}
}
