package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// FireDarkTheme implements a dark theme for F.I.R.E. System Monitor
type FireDarkTheme struct{}

func (m FireDarkTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.RGBA{0x11, 0x11, 0x11, 0xff} // Very dark grey
	case theme.ColorNameButton:
		return color.RGBA{0x21, 0x21, 0x21, 0xff}
	case theme.ColorNameDisabledButton:
		return color.RGBA{0x15, 0x15, 0x15, 0xff}
	case theme.ColorNameForeground:
		return color.RGBA{0xff, 0xff, 0xff, 0xff} // White text
	case theme.ColorNameHover:
		return color.RGBA{0x31, 0x31, 0x31, 0xff}
	case theme.ColorNameInputBackground:
		return color.RGBA{0x21, 0x21, 0x21, 0xff}
	case theme.ColorNamePlaceHolder:
		return color.RGBA{0x88, 0x88, 0x88, 0xff}
	case theme.ColorNamePressed:
		return color.RGBA{0x41, 0x41, 0x41, 0xff}
	case theme.ColorNameScrollBar:
		return color.RGBA{0x31, 0x31, 0x31, 0xff}
	case theme.ColorNameSelection:
		return color.RGBA{0x11, 0x11, 0x11, 0xff} // Same as background to hide selection
	case theme.ColorNameShadow:
		return color.RGBA{0x00, 0x00, 0x00, 0x66}
	case theme.ColorNameDisabled:
		return color.RGBA{0x55, 0x55, 0x55, 0xff}
	case theme.ColorNameError:
		return color.RGBA{0xf4, 0x43, 0x36, 0xff}
	case theme.ColorNameFocus:
		return color.RGBA{0xe3, 0x06, 0x13, 0xff} // FIRE red
	case theme.ColorNameInputBorder:
		return color.RGBA{0x31, 0x31, 0x31, 0xff}
	case theme.ColorNameMenuBackground:
		return color.RGBA{0x21, 0x21, 0x21, 0xff}
	case theme.ColorNameOverlayBackground:
		return color.RGBA{0x11, 0x11, 0x11, 0xcc}
	case theme.ColorNamePrimary:
		return color.RGBA{0xe3, 0x06, 0x13, 0xff} // FIRE red
	case theme.ColorNameSeparator:
		return color.RGBA{0x31, 0x31, 0x31, 0xff}
	case theme.ColorNameSuccess:
		return color.RGBA{0x4c, 0xaf, 0x50, 0xff}
	case theme.ColorNameWarning:
		return color.RGBA{0xff, 0x98, 0x00, 0xff}
	}
	return theme.DefaultTheme().Color(name, variant)
}

func (m FireDarkTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (m FireDarkTheme) Font(style fyne.TextStyle) fyne.Resource {
	// Use monospace for certain elements
	if style.Monospace {
		return theme.DefaultTheme().Font(style)
	}
	return theme.DefaultTheme().Font(style)
}

func (m FireDarkTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 14 // Increased from 12
	case theme.SizeNameHeadingText:
		return 24 // Increased from 16 for header
	case theme.SizeNameSubHeadingText:
		return 16 // Increased from 14
	case theme.SizeNamePadding:
		return 4 // Reduced for compact display
	case theme.SizeNameInnerPadding:
		return 2 // Reduced for compact display
	case theme.SizeNameScrollBar:
		return 14 // Increased from 12
	case theme.SizeNameScrollBarSmall:
		return 3
	case theme.SizeNameSeparatorThickness:
		return 1
	case theme.SizeNameLineSpacing:
		return 3 // Increased from 2
	case theme.SizeNameInputBorder:
		return 1
	}
	return theme.DefaultTheme().Size(name)
}

// Metric colors for bars
var (
	ColorCPUUsage    = color.RGBA{0x00, 0x7a, 0xcc, 0xff} // Blue
	ColorMemoryUsage = color.RGBA{0x00, 0xcc, 0x44, 0xff} // Green
	ColorGPUUsage    = color.RGBA{0xff, 0x66, 0x00, 0xff} // Orange
	ColorTemperature = color.RGBA{0xff, 0xcc, 0x00, 0xff} // Yellow
	ColorPower       = color.RGBA{0x00, 0xcc, 0x88, 0xff} // Teal
	ColorVoltage     = color.RGBA{0xcc, 0x00, 0xcc, 0xff} // Purple
	ColorFrequency   = color.RGBA{0x00, 0xcc, 0xcc, 0xff} // Cyan

	// Dynamic status colors
	ColorGood     = color.RGBA{0x00, 0xcc, 0x44, 0xff} // Green - all good
	ColorWarning  = color.RGBA{0xff, 0xcc, 0x00, 0xff} // Yellow - warning
	ColorCaution  = color.RGBA{0xff, 0x66, 0x00, 0xff} // Orange - caution
	ColorCritical = color.RGBA{0xff, 0x00, 0x00, 0xff} // Red - critical/danger

	// UI colors
	ColorSunset = color.RGBA{0xff, 0x7f, 0x50, 0xff} // Coral/sunset - subtle selection
	ColorEmber  = color.RGBA{0xff, 0x6b, 0x6b, 0xff} // Soft ember red
)

// UI colors
var ColorCardBackground = color.RGBA{0x22, 0x22, 0x22, 0xff} // #222222
