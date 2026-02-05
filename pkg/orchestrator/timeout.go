package orchestrator

import (
	"fmt"
	"time"
)

// ParseTimeout parses a duration string (e.g., "10s", "1m", "5m30s").
// Returns 0 duration for empty string.
// Returns error for invalid formats or negative durations.
func ParseTimeout(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}

	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid timeout %q: %w", s, err)
	}

	if d < 0 {
		return 0, fmt.Errorf("invalid timeout %q: negative duration not allowed", s)
	}

	return d, nil
}
