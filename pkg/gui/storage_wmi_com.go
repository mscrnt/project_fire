//go:build windows
// +build windows

package gui

import (
	"fmt"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

// GetDriveInfoWMI uses COM to query WMI directly
func GetDriveInfoWMI() (map[string]DriveModel, error) {
	models := make(map[string]DriveModel)

	// Initialize COM
	err := ole.CoInitialize(0)
	if err != nil {
		return models, err
	}
	defer ole.CoUninitialize()

	// Connect to WMI
	unknown, err := oleutil.CreateObject("WbemScripting.SWbemLocator")
	if err != nil {
		return models, err
	}
	defer unknown.Release()

	wmi, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return models, err
	}
	defer wmi.Release()

	// Connect to root\Microsoft\Windows\Storage namespace
	serviceRaw, err := oleutil.CallMethod(wmi, "ConnectServer", nil, `root\Microsoft\Windows\Storage`)
	if err != nil {
		return models, err
	}
	service := serviceRaw.ToIDispatch()
	defer service.Release()

	// Query MSFT_Disk
	resultRaw, err := oleutil.CallMethod(service, "ExecQuery", "SELECT * FROM MSFT_Disk")
	if err != nil {
		return models, err
	}
	result := resultRaw.ToIDispatch()
	defer result.Release()

	// Get count
	countVar, err := oleutil.GetProperty(result, "Count")
	if err != nil {
		return models, err
	}
	count := int(countVar.Val)

	// Iterate through results
	for i := 0; i < count; i++ {
		itemRaw, err := oleutil.CallMethod(result, "ItemIndex", i)
		if err != nil {
			continue
		}
		item := itemRaw.ToIDispatch()

		// Get properties
		number, _ := oleutil.GetProperty(item, "Number")
		model, _ := oleutil.GetProperty(item, "Model")
		busType, _ := oleutil.GetProperty(item, "BusType")

		diskNumber := int(number.Val)
		modelStr := model.ToString()
		busTypeInt := int(busType.Val)

		// Determine interface type
		interfaceType := ""
		switch busTypeInt {
		case 17:
			interfaceType = "NVMe"
		case 11:
			interfaceType = "SATA"
		case 8:
			interfaceType = "RAID"
		case 7:
			interfaceType = "USB"
		case 9:
			interfaceType = "iSCSI"
		default:
			interfaceType = fmt.Sprintf("Unknown (%d)", busTypeInt)
		}

		// Get drive letters for this disk
		// This would require additional WMI queries to map disk number to drive letters
		// For now, we'll use a simplified approach

		DebugLog("STORAGE", fmt.Sprintf("WMI COM: Disk %d - Model: %s, BusType: %d (%s)",
			diskNumber, modelStr, busTypeInt, interfaceType))

		item.Release()
	}

	return models, nil
}
