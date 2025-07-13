package gui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// ShowStorageDetails displays detailed storage information including full SMART data
func (d *Dashboard) ShowStorageDetails(storage *StorageInfo) {
	// Create tabs for different sections
	generalTab := d.createStorageGeneralTab(storage)
	smartTab := d.createStorageSMARTTab(storage)
	capabilitiesTab := d.createStorageCapabilitiesTab(storage)

	tabs := container.NewAppTabs(
		container.NewTabItem("General Information", generalTab),
		container.NewTabItem("S.M.A.R.T. Details", smartTab),
		container.NewTabItem("Capabilities", capabilitiesTab),
	)

	// Create dialog
	title := fmt.Sprintf("Storage Details - %s", storage.Model)
	if storage.Model == "" {
		title = fmt.Sprintf("Storage Details - %s", storage.Mountpoint)
	}

	content := container.NewBorder(
		nil, // top
		nil, // bottom
		nil, // left
		nil, // right
		tabs,
	)

	dialog := dialog.NewCustom(title, "Close", content, d.window)
	dialog.Resize(fyne.NewSize(800, 600))
	dialog.Show()
}

// createStorageGeneralTab creates the general information tab
func (d *Dashboard) createStorageGeneralTab(storage *StorageInfo) fyne.CanvasObject {
	// Device Information Card
	deviceInfo := widget.NewCard("Device Information", "",
		container.NewGridWithColumns(2,
			widget.NewLabel("Model:"),
			widget.NewLabelWithStyle(storage.Model, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabel("Vendor:"),
			widget.NewLabel(storage.Vendor),
			widget.NewLabel("Serial Number:"),
			widget.NewLabel(storage.Serial),
			widget.NewLabel("Firmware:"),
			widget.NewLabel(storage.Firmware),
			widget.NewLabel("Interface:"),
			widget.NewLabel(storage.Interface),
			widget.NewLabel("Technology:"),
			widget.NewLabel(storage.Type),
		),
	)

	// Capacity Information Card
	capacityInfo := widget.NewCard("Capacity Information", "",
		container.NewGridWithColumns(2,
			widget.NewLabel("Total Capacity:"),
			widget.NewLabel(fmt.Sprintf("%.1f GB (%.0f MB)",
				float64(storage.Size)/(1024*1024*1024),
				float64(storage.Size)/(1024*1024))),
			widget.NewLabel("Used Space:"),
			widget.NewLabel(fmt.Sprintf("%.1f GB (%.1f%%)",
				float64(storage.Used)/(1024*1024*1024),
				storage.UsedPercent)),
			widget.NewLabel("Available Space:"),
			widget.NewLabel(fmt.Sprintf("%.1f GB",
				float64(storage.Free)/(1024*1024*1024))),
			widget.NewLabel("File System:"),
			widget.NewLabel(storage.Filesystem),
			widget.NewLabel("Mount Point:"),
			widget.NewLabel(storage.Mountpoint),
		),
	)

	// Usage visualization
	usageBar := widget.NewProgressBar()
	usageBar.SetValue(storage.UsedPercent / 100)

	usageCard := widget.NewCard("Usage Visualization", "",
		container.NewVBox(
			usageBar,
			widget.NewLabelWithStyle(
				fmt.Sprintf("%.1f%% Used", storage.UsedPercent),
				fyne.TextAlignCenter,
				fyne.TextStyle{},
			),
		),
	)

	return container.NewVBox(
		deviceInfo,
		capacityInfo,
		usageCard,
	)
}

// createStorageSMARTTab creates the SMART details tab
func (d *Dashboard) createStorageSMARTTab(storage *StorageInfo) fyne.CanvasObject {
	if storage.SMART == nil || !storage.SMART.Available {
		return container.NewCenter(
			widget.NewLabelWithStyle(
				"S.M.A.R.T. data not available for this device",
				fyne.TextAlignCenter,
				fyne.TextStyle{Italic: true},
			),
		)
	}

	smart := storage.SMART

	// Health Status Card with color coding

	healthLabel := widget.NewLabelWithStyle(
		smart.HealthStatus,
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)
	healthLabel.Importance = widget.HighImportance

	healthCard := widget.NewCard("Health Status", "",
		container.NewHBox(
			widget.NewLabel("Overall Health:"),
			healthLabel,
		),
	)

	// Temperature Card
	tempCard := widget.NewCard("Temperature", "",
		container.NewGridWithColumns(2,
			widget.NewLabel("Current Temperature:"),
			widget.NewLabel(fmt.Sprintf("%.0f°C", smart.Temperature)),
		),
	)

	// Usage Statistics Card
	usageStats := widget.NewCard("Usage Statistics", "",
		container.NewGridWithColumns(2,
			widget.NewLabel("Power On Hours:"),
			widget.NewLabel(fmt.Sprintf("%d hours (%.1f days)",
				smart.PowerOnHours,
				float64(smart.PowerOnHours)/24)),
			widget.NewLabel("Power Cycle Count:"),
			widget.NewLabel(fmt.Sprintf("%d", smart.PowerCycles)),
			widget.NewLabel("Total Data Written:"),
			widget.NewLabel(fmt.Sprintf("%.2f TB", smart.TotalWrittenGB/1024)),
			widget.NewLabel("Total Data Read:"),
			widget.NewLabel(fmt.Sprintf("%.2f TB", smart.TotalReadGB/1024)),
		),
	)

	// Wear Level for SSDs
	var wearCard *widget.Card
	if smart.WearLevel > 0 {
		wearBar := widget.NewProgressBar()
		wearBar.SetValue(smart.WearLevel / 100)

		wearCard = widget.NewCard("SSD Wear Level", "",
			container.NewVBox(
				wearBar,
				widget.NewLabelWithStyle(
					fmt.Sprintf("%.1f%% Wear", smart.WearLevel),
					fyne.TextAlignCenter,
					fyne.TextStyle{},
				),
			),
		)
	}

	// Build the content
	content := container.NewVBox(
		healthCard,
		tempCard,
		usageStats,
	)

	if wearCard != nil {
		content.Add(wearCard)
	}

	// Add raw SMART attributes section
	smartAttrsCard := widget.NewCard("S.M.A.R.T. Attributes", "Self-Monitoring, Analysis and Reporting Technology",
		widget.NewLabelWithStyle(
			"Raw attribute data would be displayed here\n(To be implemented with detailed attribute parsing)",
			fyne.TextAlignLeading,
			fyne.TextStyle{Italic: true},
		),
	)
	content.Add(smartAttrsCard)

	return container.NewScroll(content)
}

// createStorageCapabilitiesTab creates the capabilities tab
func (d *Dashboard) createStorageCapabilitiesTab(storage *StorageInfo) fyne.CanvasObject {
	// I/O Command Sets
	commandSets := []string{}

	// Determine command sets based on interface type
	interfaceLower := strings.ToLower(storage.Interface)
	if strings.Contains(interfaceLower, "nvme") {
		commandSets = append(commandSets,
			"NVM Express 1.4",
			"NVMe Management Interface",
			"Format NVM Command",
			"Security Send/Receive",
			"Firmware Download/Commit",
		)
	} else if strings.Contains(interfaceLower, "sata") || strings.Contains(interfaceLower, "ide") {
		commandSets = append(commandSets,
			"ATA/ATAPI-8",
			"SMART Feature Set",
			"Power Management Feature Set",
			"48-bit Address Feature Set",
			"Native Command Queuing (NCQ)",
		)
	}

	if strings.Contains(strings.ToLower(storage.Type), "ssd") {
		commandSets = append(commandSets,
			"TRIM Support",
			"Garbage Collection",
			"Wear Leveling",
		)
	}

	// Create command sets list
	commandSetsList := widget.NewList(
		func() int { return len(commandSets) },
		func() fyne.CanvasObject {
			return widget.NewLabel("Command Set")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText("• " + commandSets[i])
		},
	)
	commandSetsList.Resize(fyne.NewSize(400, 200))

	commandSetsCard := widget.NewCard("I/O Command Sets", "", commandSetsList)

	// Features Card
	features := []string{}

	if storage.Type == "SSD" || storage.Type == "NVME" {
		features = append(features,
			"No moving parts",
			"Low power consumption",
			"Silent operation",
			"Shock resistant",
		)
	} else if storage.Type == "HDD" {
		features = append(features,
			"High capacity",
			"Cost effective storage",
			"Sequential read/write optimized",
		)
	}

	if storage.Interface != "" {
		features = append(features, fmt.Sprintf("%s Interface", storage.Interface))
	}

	featuresList := widget.NewList(
		func() int { return len(features) },
		func() fyne.CanvasObject {
			return widget.NewLabel("Feature")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText("• " + features[i])
		},
	)
	featuresList.Resize(fyne.NewSize(400, 150))

	featuresCard := widget.NewCard("Device Features", "", featuresList)

	// Performance Characteristics
	perfCard := widget.NewCard("Performance Characteristics", "",
		widget.NewLabel("Performance metrics would be displayed here\n(To be implemented with benchmarking data)"),
	)

	return container.NewVBox(
		commandSetsCard,
		featuresCard,
		perfCard,
	)
}

// Add click handler to storage items to show details
func (d *Dashboard) handleStorageClick(storage *StorageInfo) {
	d.ShowStorageDetails(storage)
}
