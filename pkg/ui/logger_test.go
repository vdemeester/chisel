package ui

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestPrettyLogger(t *testing.T) {
	var buf bytes.Buffer
	log := NewLogger(OutputPretty, &buf)

	log.PipelineStart("test-pipeline")
	log.TaskStart("test-task")
	log.StepStart("test-step", "alpine:latest")
	log.StepEnd("test-step", 100*time.Millisecond, nil)
	log.TaskEnd("test-task", 200*time.Millisecond, nil)
	log.PipelineEnd("test-pipeline", 300*time.Millisecond, nil)

	output := buf.String()

	// Check for expected content
	if !strings.Contains(output, "Pipeline:") {
		t.Error("Expected 'Pipeline:' in output")
	}
	if !strings.Contains(output, "Task:") {
		t.Error("Expected 'Task:' in output")
	}
	if !strings.Contains(output, "Step:") {
		t.Error("Expected 'Step:' in output")
	}
	if !strings.Contains(output, "test-pipeline") {
		t.Error("Expected 'test-pipeline' in output")
	}
	if !strings.Contains(output, "100ms") {
		t.Error("Expected '100ms' duration in output")
	}
}

func TestPlainLogger(t *testing.T) {
	var buf bytes.Buffer
	log := NewLogger(OutputPlain, &buf)

	log.PipelineStart("test-pipeline")
	log.Info("test message", "key", "value")
	log.PipelineEnd("test-pipeline", 1*time.Second, nil)

	output := buf.String()

	if !strings.Contains(output, "* Pipeline: test-pipeline") {
		t.Errorf("Expected pipeline start, got: %s", output)
	}
	if !strings.Contains(output, "[INFO] test message key=value") {
		t.Errorf("Expected info message with attrs, got: %s", output)
	}
	if !strings.Contains(output, "+ Pipeline completed [1.0s]") {
		t.Errorf("Expected pipeline end, got: %s", output)
	}
}

func TestJSONLogger(t *testing.T) {
	var buf bytes.Buffer
	log := NewLogger(OutputJSON, &buf)

	log.PipelineStart("test-pipeline")
	log.TaskStart("test-task")
	log.StepStart("test-step", "alpine:latest")
	log.Info("test message", "key", "value")

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 4 {
		t.Fatalf("Expected 4 JSON lines, got %d", len(lines))
	}

	// Parse first line
	var event LogEvent
	if err := json.Unmarshal([]byte(lines[0]), &event); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
	if event.Type != "pipeline_start" {
		t.Errorf("Expected type pipeline_start, got: %s", event.Type)
	}
	if event.Name != "test-pipeline" {
		t.Errorf("Expected name test-pipeline, got: %s", event.Name)
	}
	if event.Time == "" {
		t.Error("Expected time to be set")
	}

	// Parse step start
	var stepEvent LogEvent
	if err := json.Unmarshal([]byte(lines[2]), &stepEvent); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
	if stepEvent.Type != "step_start" {
		t.Errorf("Expected type step_start, got: %s", stepEvent.Type)
	}
	if stepEvent.Image != "alpine:latest" {
		t.Errorf("Expected image alpine:latest, got: %s", stepEvent.Image)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{500 * time.Millisecond, "500ms"},
		{1 * time.Second, "1.0s"},
		{1500 * time.Millisecond, "1.5s"},
		{90 * time.Second, "1.5m"},
	}

	for _, tt := range tests {
		result := formatDuration(tt.duration)
		if result != tt.expected {
			t.Errorf("formatDuration(%v) = %s, expected %s", tt.duration, result, tt.expected)
		}
	}
}

func TestOutputModeParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected OutputMode
	}{
		{"pretty", OutputPretty},
		{"plain", OutputPlain},
		{"json", OutputJSON},
		{"unknown", OutputPretty}, // default
	}

	for _, tt := range tests {
		result := ParseOutputMode(tt.input)
		if result != tt.expected {
			t.Errorf("ParseOutputMode(%s) = %v, expected %v", tt.input, result, tt.expected)
		}
	}
}

func TestOutputModeString(t *testing.T) {
	tests := []struct {
		mode     OutputMode
		expected string
	}{
		{OutputPretty, "pretty"},
		{OutputPlain, "plain"},
		{OutputJSON, "json"},
	}

	for _, tt := range tests {
		result := tt.mode.String()
		if result != tt.expected {
			t.Errorf("OutputMode(%v).String() = %s, expected %s", tt.mode, result, tt.expected)
		}
	}
}
