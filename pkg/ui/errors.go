package ui

import (
	"errors"
	"regexp"
	"strings"
)

// traceparentPattern matches Dagger's trace span annotations like [traceparent:abc-123]
var traceparentPattern = regexp.MustCompile(`\s*\[traceparent:[^\]]+\]`)

// CleanErrorMessage removes Dagger trace spans from error messages.
// These spans are useful for debugging Dagger internals but clutter user-facing output.
func CleanErrorMessage(msg string) string {
	cleaned := traceparentPattern.ReplaceAllString(msg, "")
	// Clean up any double spaces left behind
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	return cleaned
}

// CleanError wraps an error with a cleaned message.
// Returns nil if the input error is nil.
func CleanError(err error) error {
	if err == nil {
		return nil
	}
	cleaned := CleanErrorMessage(err.Error())
	return errors.New(cleaned)
}
