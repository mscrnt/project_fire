package gui

import (
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
)

var (
	// Icon paths - only using SVG now
	svgPath string

	// Icon cache
	iconCache = make(map[string]fyne.Resource)
)

func init() {
	// Get the executable path to find icons relative to it
	exePath, err := os.Executable()
	if err != nil {
		// Fallback to working directory
		svgPath = filepath.Join(".", "icons", "svg")
	} else {
		// Icons are in the same directory as the executable or in project root
		exeDir := filepath.Dir(exePath)
		svgPath = filepath.Join(exeDir, "icons", "svg")

		// Check if icons exist at exe location, otherwise try project root
		if _, err := os.Stat(svgPath); os.IsNotExist(err) {
			// Try project root (for development)
			svgPath = filepath.Join(exeDir, "..", "..", "icons", "svg")
			if _, err := os.Stat(svgPath); os.IsNotExist(err) {
				// Last resort - current directory
				svgPath = filepath.Join(".", "icons", "svg")
			}
		}
	}

	// Clean the path to ensure it's proper for the OS
	svgPath = filepath.Clean(svgPath)
}

// LoadIconFromFile loads an icon from the file system (deprecated - use SVG)
func LoadIconFromFile(filename string) fyne.Resource {
	// Return nil since we removed PNG files
	return nil
}

// LoadSVGIcon loads an SVG icon from the svg directory
func LoadSVGIcon(filename string) fyne.Resource {
	return LoadIconFromPath(filepath.Join(svgPath, filename))
}

// LoadIconFromPath loads an icon from a full path
func LoadIconFromPath(fullPath string) fyne.Resource {
	// Check cache first
	if cached, ok := iconCache[fullPath]; ok {
		return cached
	}

	// Read file
	data, err := os.ReadFile(fullPath)
	if err != nil {
		DebugLog("ERROR", "Failed to load icon %s: %v", fullPath, err)
		return nil
	}

	// Create resource
	resource := &fyne.StaticResource{
		StaticName:    filepath.Base(fullPath),
		StaticContent: data,
	}

	// Cache it
	iconCache[fullPath] = resource

	return resource
}

// Icon getters
func GetCPUIcon() fyne.Resource {
	// Use monitoring.svg for monitoring
	return LoadSVGIcon("monitoring.svg")
}

func GetMemoryIcon() fyne.Resource {
	// TODO: Add memory.svg
	return nil
}

func GetGPUIcon() fyne.Resource {
	// TODO: Add gpu.svg
	return nil
}

func GetStorageIcon() fyne.Resource {
	// Use system.svg for storage temporarily
	return LoadSVGIcon("system.svg")
}

func GetNetworkIcon() fyne.Resource {
	// TODO: Add network.svg
	return nil
}

func GetSystemIcon() fyne.Resource {
	return LoadSVGIcon("system.svg")
}

func GetGaugeIcon() fyne.Resource {
	// Use benchmark.svg for benchmarks
	return LoadSVGIcon("benchmark.svg")
}

func GetFanIcon() fyne.Resource {
	// TODO: Add fan.svg
	return nil
}

func GetPowerIcon() fyne.Resource {
	// TODO: Add power.svg
	return nil
}

func GetTestIcon() fyne.Resource {
	return LoadSVGIcon("tests.svg")
}

func GetReportIcon() fyne.Resource {
	// TODO: Add report.svg
	return nil
}

func GetSettingsIcon() fyne.Resource {
	// TODO: Add settings.svg
	return nil
}

func GetSupportIcon() fyne.Resource {
	// Use coffee.svg for Buy Me a Coffee
	return LoadSVGIcon("coffee.svg")
}
