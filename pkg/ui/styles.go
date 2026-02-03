package ui

import "github.com/charmbracelet/lipgloss"

// Color palette using ANSI 256 colors for broad terminal compatibility.
var (
	ColorSuccess = lipgloss.Color("10") // Green
	ColorFailure = lipgloss.Color("9")  // Red
	ColorWarning = lipgloss.Color("11") // Yellow
	ColorInfo    = lipgloss.Color("12") // Blue
	ColorDim     = lipgloss.Color("8")  // Gray
)

// Styles for different UI elements.
var (
	// Symbol styles
	StyleRunning = lipgloss.NewStyle().Foreground(ColorWarning)
	StyleSuccess = lipgloss.NewStyle().Foreground(ColorSuccess)
	StyleFailure = lipgloss.NewStyle().Foreground(ColorFailure)
	StyleSkipped = lipgloss.NewStyle().Foreground(ColorDim)
	StylePending = lipgloss.NewStyle().Foreground(ColorDim)

	// Text styles
	StyleLabel    = lipgloss.NewStyle().Bold(true)
	StyleDuration = lipgloss.NewStyle().Foreground(ColorDim)
	StyleDim      = lipgloss.NewStyle().Foreground(ColorDim)
	StyleError    = lipgloss.NewStyle().Foreground(ColorFailure)
	StyleDebug    = lipgloss.NewStyle().Foreground(ColorDim)
	StyleInfo     = lipgloss.NewStyle().Foreground(ColorInfo)
	StyleWarn     = lipgloss.NewStyle().Foreground(ColorWarning)
)
