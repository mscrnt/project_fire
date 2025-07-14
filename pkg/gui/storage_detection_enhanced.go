//go:build windows
// +build windows

package gui

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
	"github.com/mscrnt/project_fire/pkg/telemetry"
)

// Enhanced storage detection using IOCTL_STORAGE_QUERY_PROPERTY
// Based on Microsoft's documentation for working with NVMe devices

const (
	IOCTL_STORAGE_QUERY_PROPERTY = 0x002D1400

	// Storage bus types from Windows SDK
	BusTypeUnknown           = 0x00
	BusTypeScsi              = 0x01
	BusTypeAtapi             = 0x02
	BusTypeAta               = 0x03
	BusType1394              = 0x04
	BusTypeSsa               = 0x05
	BusTypeFibre             = 0x06
	BusTypeUsb               = 0x07
	BusTypeRAID              = 0x08
	BusTypeiScsi             = 0x09
	BusTypeSas               = 0x0A
	BusTypeSata              = 0x0B
	BusTypeSd                = 0x0C
	BusTypeMmc               = 0x0D
	BusTypeVirtual           = 0x0E
	BusTypeFileBackedVirtual = 0x0F
	BusTypeSpaces            = 0x10
	BusTypeNvme              = 0x11 // This is the key - 0x11 = 17 decimal
	BusTypeSCM               = 0x12
	BusTypeUfs               = 0x13
	BusTypeMax               = 0x14
)

// STORAGE_PROPERTY_ID enumeration
type StoragePropertyId uint32

const (
	StorageDeviceProperty                  StoragePropertyId = 0
	StorageAdapterProperty                 StoragePropertyId = 1
	StorageDeviceIdProperty                StoragePropertyId = 2
	StorageDeviceUniqueIdProperty          StoragePropertyId = 3
	StorageDeviceWriteCacheProperty        StoragePropertyId = 4
	StorageMiniportProperty                StoragePropertyId = 5
	StorageAccessAlignmentProperty         StoragePropertyId = 6
	StorageDeviceSeekPenaltyProperty       StoragePropertyId = 7
	StorageDeviceTrimProperty              StoragePropertyId = 8
	StorageDeviceWriteAggregationProperty  StoragePropertyId = 9
	StorageDeviceDeviceTelemetryProperty   StoragePropertyId = 10
	StorageDeviceLBProvisioningProperty    StoragePropertyId = 11
	StorageDevicePowerProperty             StoragePropertyId = 12
	StorageDeviceCopyOffloadProperty       StoragePropertyId = 13
	StorageDeviceResiliencyProperty        StoragePropertyId = 14
	StorageDeviceMediumProductType         StoragePropertyId = 15
	StorageAdapterRpmbProperty             StoragePropertyId = 16
	StorageAdapterCryptoProperty           StoragePropertyId = 17
	StorageDeviceIoCapabilityProperty      StoragePropertyId = 18
	StorageAdapterProtocolSpecificProperty StoragePropertyId = 19
	StorageDeviceProtocolSpecificProperty  StoragePropertyId = 20
	StorageAdapterTemperatureProperty      StoragePropertyId = 21
	StorageDeviceTemperatureProperty       StoragePropertyId = 22
	StorageAdapterPhysicalTopologyProperty StoragePropertyId = 23
	StorageDevicePhysicalTopologyProperty  StoragePropertyId = 24
	StorageDeviceAttributesProperty        StoragePropertyId = 25
)

// STORAGE_QUERY_TYPE enumeration
type StorageQueryType uint32

const (
	PropertyStandardQuery   StorageQueryType = 0
	PropertyExistsQuery     StorageQueryType = 1
	PropertyMaskQuery       StorageQueryType = 2
	PropertyQueryMaxDefined StorageQueryType = 3
)

// STORAGE_PROPERTY_QUERY structure
type StoragePropertyQuery struct {
	PropertyId           StoragePropertyId
	QueryType            StorageQueryType
	AdditionalParameters [1]byte
}

// STORAGE_DEVICE_DESCRIPTOR structure
type StorageDeviceDescriptor struct {
	Version               uint32
	Size                  uint32
	DeviceType            byte
	DeviceTypeModifier    byte
	RemovableMedia        byte
	CommandQueueing       byte
	VendorIdOffset        uint32
	ProductIdOffset       uint32
	ProductRevisionOffset uint32
	SerialNumberOffset    uint32
	BusType               uint32 // This is what we need!
	RawPropertiesLength   uint32
	RawDeviceProperties   [1]byte
}

// GetStorageDeviceDescriptor retrieves the storage device descriptor
func GetStorageDeviceDescriptor(devicePath string) (*StorageDeviceDescriptor, error) {
	// Open the device
	pathPtr, err := windows.UTF16PtrFromString(devicePath)
	if err != nil {
		return nil, err
	}

	handle, err := windows.CreateFile(
		pathPtr,
		windows.GENERIC_READ,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		0,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open device: %w", err)
	}
	defer windows.CloseHandle(handle)

	// Prepare the query
	query := StoragePropertyQuery{
		PropertyId: StorageDeviceProperty,
		QueryType:  PropertyStandardQuery,
	}

	// Allocate buffer for the descriptor
	bufferSize := uint32(4096) // Should be enough for most devices
	buffer := make([]byte, bufferSize)

	var bytesReturned uint32
	err = windows.DeviceIoControl(
		handle,
		IOCTL_STORAGE_QUERY_PROPERTY,
		(*byte)(unsafe.Pointer(&query)),
		uint32(unsafe.Sizeof(query)),
		&buffer[0],
		bufferSize,
		&bytesReturned,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("DeviceIoControl failed: %w", err)
	}

	// Cast the buffer to our structure
	descriptor := (*StorageDeviceDescriptor)(unsafe.Pointer(&buffer[0]))

	return descriptor, nil
}

// GetDriveBusType returns the bus type for a given drive letter
func GetDriveBusType(driveLetter string) (string, error) {
	// Format the device path correctly
	devicePath := fmt.Sprintf("\\\\.\\%s:", driveLetter)

	descriptor, err := GetStorageDeviceDescriptor(devicePath)
	if err != nil {
		return "", err
	}

	// Convert bus type to string
	switch descriptor.BusType {
	case BusTypeScsi:
		return "SCSI", nil
	case BusTypeAtapi:
		return "ATAPI", nil
	case BusTypeAta:
		return "ATA", nil
	case BusType1394:
		return "IEEE1394", nil
	case BusTypeSsa:
		return "SSA", nil
	case BusTypeFibre:
		return "Fibre", nil
	case BusTypeUsb:
		return "USB", nil
	case BusTypeRAID:
		return "RAID", nil
	case BusTypeiScsi:
		return "iSCSI", nil
	case BusTypeSas:
		return "SAS", nil
	case BusTypeSata:
		return "SATA", nil
	case BusTypeSd:
		return "SD", nil
	case BusTypeMmc:
		return "MMC", nil
	case BusTypeVirtual:
		return "Virtual", nil
	case BusTypeFileBackedVirtual:
		return "FileBackedVirtual", nil
	case BusTypeSpaces:
		return "Spaces", nil
	case BusTypeNvme:
		return "NVMe", nil
	case BusTypeSCM:
		return "SCM", nil
	case BusTypeUfs:
		return "UFS", nil
	default:
		// Record hardware miss for unknown bus type
		telemetry.RecordHardwareMiss("StorageBusType", map[string]interface{}{
			"bus_type": descriptor.BusType,
			"drive":    driveLetter,
		})
		return fmt.Sprintf("Unknown(%d)", descriptor.BusType), nil
	}
}

// IsNVMeDrive checks if a drive is NVMe using the proper Windows API
func IsNVMeDrive(driveLetter string) bool {
	busType, err := GetDriveBusType(driveLetter)
	if err != nil {
		DebugLog("STORAGE", fmt.Sprintf("Failed to get bus type for drive %s: %v", driveLetter, err))
		return false
	}

	return busType == "NVMe"
}
