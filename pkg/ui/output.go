package ui

import (
	"os"

	"golang.org/x/term"
)

// OutputMode determines how output is formatted.
type OutputMode int

const (
	// OutputPretty uses colors, symbols, and hierarchical display.
	// Default for TTY.
	OutputPretty OutputMode = iota
	// OutputPlain uses simple text without colors.
	// Default for non-TTY (pipes, files).
	OutputPlain
	// OutputJSON emits structured JSON lines.
	// Useful for parsing and CI/CD integration.
	OutputJSON
)

// String returns the string representation of the output mode.
func (m OutputMode) String() string {
	switch m {
	case OutputPretty:
		return "pretty"
	case OutputPlain:
		return "plain"
	case OutputJSON:
		return "json"
	default:
		return "unknown"
	}
}

// ParseOutputMode parses a string into an OutputMode.
// Returns OutputPretty if the string is unrecognized.
func ParseOutputMode(s string) OutputMode {
	switch s {
	case "pretty":
		return OutputPretty
	case "plain":
		return OutputPlain
	case "json":
		return OutputJSON
	default:
		return OutputPretty
	}
}

// DetectOutputMode returns the appropriate output mode based on
// whether stdout is a TTY.
func DetectOutputMode() OutputMode {
	if term.IsTerminal(int(os.Stdout.Fd())) {
		return OutputPretty
	}
	return OutputPlain
}
