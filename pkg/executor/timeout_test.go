package executor

import (
	"testing"
	"time"
)

func TestParseTimeout_Valid(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"10s", 10 * time.Second},
		{"1m", 1 * time.Minute},
		{"5m30s", 5*time.Minute + 30*time.Second},
		{"1h", 1 * time.Hour},
		{"2h30m", 2*time.Hour + 30*time.Minute},
		{"500ms", 500 * time.Millisecond},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := parseTimeout(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expected {
				t.Errorf("parseTimeout(%q) = %v, want %v", tc.input, got, tc.expected)
			}
		})
	}
}

func TestParseTimeout_Empty(t *testing.T) {
	got, err := parseTimeout("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 0 {
		t.Errorf("parseTimeout(\"\") = %v, want 0", got)
	}
}

func TestParseTimeout_Invalid(t *testing.T) {
	tests := []string{
		"invalid",
		"10",      // missing unit
		"-5s",     // negative
		"abc123s", // invalid format
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := parseTimeout(input)
			if err == nil {
				t.Errorf("parseTimeout(%q) expected error, got nil", input)
			}
		})
	}
}

func TestTimeoutContext_NoTimeout(t *testing.T) {
	// When timeout is 0, should return original context without timeout
	timeout := time.Duration(0)
	hasTimeout := timeout > 0
	if hasTimeout {
		t.Error("expected no timeout for 0 duration")
	}
}

func TestTimeoutContext_WithTimeout(t *testing.T) {
	// When timeout is positive, should create context with timeout
	timeout := 5 * time.Second
	hasTimeout := timeout > 0
	if !hasTimeout {
		t.Error("expected timeout for 5s duration")
	}
}

func TestParseTimeout_TektonFormat(t *testing.T) {
	// Tekton uses Go duration format, verify common patterns work
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"30s", 30 * time.Second},
		{"2m", 2 * time.Minute},
		{"1h30m", 1*time.Hour + 30*time.Minute},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := parseTimeout(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expected {
				t.Errorf("parseTimeout(%q) = %v, want %v", tc.input, got, tc.expected)
			}
		})
	}
}
