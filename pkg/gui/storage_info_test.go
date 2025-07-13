package gui

import (
	"runtime"
	"testing"
)

// TestGetWindowsDriveModelsV2 tests the platform-specific storage detection
func TestGetWindowsDriveModelsV2(t *testing.T) {
	models := GetWindowsDriveModelsV2()
	
	if runtime.GOOS != "windows" {
		// On non-Windows platforms, should return empty map
		if len(models) != 0 {
			t.Errorf("Expected empty map on non-Windows platform, got %d entries", len(models))
		}
		return
	}
	
	// On Windows, the function might return data or empty depending on permissions
	// We just ensure it doesn't panic
	t.Logf("GetWindowsDriveModelsV2 returned %d drive models", len(models))
}

// TestGetDriveBusType tests the bus type detection
func TestGetDriveBusType(t *testing.T) {
	busType, err := GetDriveBusType("C:")
	
	if runtime.GOOS != "windows" {
		// On non-Windows platforms, should return empty string and no error
		if busType != "" || err != nil {
			t.Errorf("Expected empty string and nil error on non-Windows, got busType=%s, err=%v", busType, err)
		}
		return
	}
	
	// On Windows, we might get data or an error depending on the system
	if err != nil {
		t.Logf("GetDriveBusType returned error (might be expected): %v", err)
	} else {
		t.Logf("GetDriveBusType returned: %s", busType)
	}
}

// TestReadMemoryModulesWithSPD tests SPD reading functionality
func TestReadMemoryModulesWithSPD(t *testing.T) {
	modules, err := ReadMemoryModulesWithSPD()
	
	if runtime.GOOS != "windows" {
		// On non-Windows platforms, should return error
		if err == nil {
			t.Error("Expected error on non-Windows platform, got nil")
		}
		if modules != nil {
			t.Error("Expected nil modules on non-Windows platform")
		}
		return
	}
	
	// On Windows, might fail if not admin or driver not available
	if err != nil {
		t.Logf("ReadMemoryModulesWithSPD returned expected error: %v", err)
	} else {
		t.Logf("ReadMemoryModulesWithSPD returned %d modules", len(modules))
	}
}