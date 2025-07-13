package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// FireTheme is a custom dark theme for F.I.R.E. with fire-inspired colors
type FireTheme struct{}

// Color returns the color for the specified theme color name
func (t FireTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		// Dark charcoal background
		return color.NRGBA{R: 0x1a, G: 0x1d, B: 0x21, A: 0xff}
	case theme.ColorNameButton:
		// Slightly lighter for buttons
		return color.NRGBA{R: 0x2d, G: 0x31, B: 0x36, A: 0xff}
	case theme.ColorNameDisabledButton:
		return color.NRGBA{R: 0x1f, G: 0x22, B: 0x26, A: 0xff}
	case theme.ColorNameError:
		// Fire red for errors
		return color.NRGBA{R: 0xff, G: 0x45, B: 0x45, A: 0xff}
	case theme.ColorNameForeground:
		// Off-white text
		return color.NRGBA{R: 0xe0, G: 0xe0, B: 0xe0, A: 0xff}
	case theme.ColorNameHover:
		// Subtle hover effect
		return color.NRGBA{R: 0x3a, G: 0x3f, B: 0x44, A: 0xff}
	case theme.ColorNameInputBackground:
		// Input fields
		return color.NRGBA{R: 0x22, G: 0x26, B: 0x2a, A: 0xff}
	case theme.ColorNamePlaceHolder:
		// Placeholder text
		return color.NRGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff}
	case theme.ColorNamePressed:
		// Pressed state
		return color.NRGBA{R: 0xff, G: 0x57, B: 0x22, A: 0xff}
	case theme.ColorNamePrimary:
		// Fire orange primary color
		return color.NRGBA{R: 0xff, G: 0x57, B: 0x22, A: 0xff}
	case theme.ColorNameScrollBar:
		return color.NRGBA{R: 0x40, G: 0x44, B: 0x48, A: 0xff}
	case theme.ColorNameSeparator:
		return color.NRGBA{R: 0x35, G: 0x39, B: 0x3d, A: 0xff}
	case theme.ColorNameSuccess:
		// Success green
		return color.NRGBA{R: 0x4c, G: 0xaf, B: 0x50, A: 0xff}
	case theme.ColorNameWarning:
		// Warning amber
		return color.NRGBA{R: 0xff, G: 0x98, B: 0x00, A: 0xff}
	case theme.ColorNameShadow:
		return color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x66}
	default:
		// Default fallback color - medium gray
		return color.NRGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff}
	}
}

// Font returns the font resource for the specified text style
func (t FireTheme) Font(style fyne.TextStyle) fyne.Resource {
	// Use default fonts for now
	return theme.DefaultTheme().Font(style)
}

// Icon returns the icon resource for the specified icon name
func (t FireTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	// Use default icons for now, we'll add custom ones later
	return theme.DefaultTheme().Icon(name)
}

// Size returns the size value for the specified size name
func (t FireTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 14 // Standard text
	case theme.SizeNameHeadingText:
		return 18 // Card titles
	case theme.SizeNameSubHeadingText:
		return 16 // Section headers
	case theme.SizeNameCaptionText:
		return 12 // Small labels
	case theme.SizeNamePadding:
		return 8 // More generous padding
	case theme.SizeNameInnerPadding:
		return 6
	default:
		return theme.DefaultTheme().Size(name)
	}
}

// CardBackgroundColor returns the background color for cards
func CardBackgroundColor() color.Color {
	return color.NRGBA{R: 0x22, G: 0x26, B: 0x2a, A: 0xff}
}

// ChartLineColor returns the primary color for chart lines
func ChartLineColor() color.Color {
	return color.NRGBA{R: 0xff, G: 0x57, B: 0x22, A: 0xff}
}

// ChartGridColor returns the color for chart gridlines
func ChartGridColor() color.Color {
	return color.NRGBA{R: 0x35, G: 0x39, B: 0x3d, A: 0x40}
}

// SuccessColor returns the success indicator color
func SuccessColor() color.Color {
	return color.NRGBA{R: 0x4c, G: 0xaf, B: 0x50, A: 0xff}
}

// WarningColor returns the warning indicator color
func WarningColor() color.Color {
	return color.NRGBA{R: 0xff, G: 0x98, B: 0x00, A: 0xff}
}

// ErrorColor returns the error indicator color
func ErrorColor() color.Color {
	return color.NRGBA{R: 0xff, G: 0x45, B: 0x45, A: 0xff}
}
