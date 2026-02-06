package podman

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vdemeester/chisel/pkg/backend"
)

// TestReadResultNotImplemented verifies ReadResult is no longer returning ErrNotImplemented.
func TestReadResultImplemented(t *testing.T) {
	b := NewPodmanBackend()
	ctx := context.Background()

	req := &backend.ResultRequest{
		ContainerID: "nonexistent-container",
		Path:        "/tekton/results/output",
	}

	_, err := b.ReadResult(ctx, req)
	// Should not return ErrNotImplemented anymore
	if err == ErrNotImplemented {
		t.Error("ReadResult should be implemented, not return ErrNotImplemented")
	}
}

// TestReadResultFromTempDir tests reading results from a temporary directory.
func TestReadResultFromTempDir(t *testing.T) {
	// Skip if Podman is not available
	if detectSocketPath() == "" {
		t.Skip("Podman not available")
	}

	// Create a temp directory with a result file
	tmpDir, err := os.MkdirTemp("", "tekton-results-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Write a result file
	resultPath := filepath.Join(tmpDir, "output")
	if err := os.WriteFile(resultPath, []byte("test-result-value"), 0644); err != nil {
		t.Fatalf("Failed to write result file: %v", err)
	}

	// ReadResultFromPath should read from the path directly
	content, err := ReadResultFromPath(resultPath)
	if err != nil {
		t.Fatalf("ReadResultFromPath failed: %v", err)
	}

	expected := "test-result-value"
	if content != expected {
		t.Errorf("expected %q, got %q", expected, content)
	}
}

// TestReadResultFromPathMissing tests handling of missing result files.
func TestReadResultFromPathMissing(t *testing.T) {
	_, err := ReadResultFromPath("/nonexistent/path/to/result")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

// TestExecuteStepCapturesResults tests that ExecuteStep captures results from /tekton/results/.
func TestExecuteStepCapturesResults(t *testing.T) {
	// Skip if Podman is not available
	if detectSocketPath() == "" {
		t.Skip("Podman not available")
	}

	b := NewPodmanBackend()
	ctx := context.Background()

	// Create a step that writes to /tekton/results/
	req := &backend.StepRequest{
		Image: "docker.io/library/alpine:latest",
		Command: []string{"sh", "-c", `
			mkdir -p /tekton/results
			echo -n "hello-world" > /tekton/results/greeting
			echo -n "42" > /tekton/results/count
		`},
	}

	result, err := b.ExecuteStep(ctx, req)
	if err != nil {
		t.Fatalf("ExecuteStep failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	// Check that results were captured
	if result.Results == nil {
		t.Fatal("Results map is nil")
	}

	if greeting, ok := result.Results["greeting"]; !ok || greeting != "hello-world" {
		t.Errorf("expected greeting result 'hello-world', got %q (found: %v)", greeting, ok)
	}

	if count, ok := result.Results["count"]; !ok || count != "42" {
		t.Errorf("expected count result '42', got %q (found: %v)", count, ok)
	}
}

// TestExecuteStepWithResultsDir tests that a temp results dir is mounted.
func TestExecuteStepWithResultsDir(t *testing.T) {
	// Skip if Podman is not available
	if detectSocketPath() == "" {
		t.Skip("Podman not available")
	}

	b := NewPodmanBackend()
	ctx := context.Background()

	// Create a step that checks if /tekton/results exists and is writable
	req := &backend.StepRequest{
		Image: "docker.io/library/alpine:latest",
		Command: []string{"sh", "-c", `
			if [ -d /tekton/results ]; then
				echo "results dir exists"
				touch /tekton/results/test && echo "writable"
			else
				echo "no results dir"
			fi
		`},
	}

	result, err := b.ExecuteStep(ctx, req)
	if err != nil {
		t.Fatalf("ExecuteStep failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d (stderr: %s)", result.ExitCode, result.Stderr)
	}

	if !strings.Contains(result.Stdout, "results dir exists") {
		t.Errorf("expected results dir to exist, got: %s", result.Stdout)
	}

	if !strings.Contains(result.Stdout, "writable") {
		t.Errorf("expected results dir to be writable, got: %s", result.Stdout)
	}
}

// TestExecuteStepNoResults tests that missing results don't cause errors.
func TestExecuteStepNoResults(t *testing.T) {
	// Skip if Podman is not available
	if detectSocketPath() == "" {
		t.Skip("Podman not available")
	}

	b := NewPodmanBackend()
	ctx := context.Background()

	// Create a step that doesn't write any results
	req := &backend.StepRequest{
		Image:   "docker.io/library/alpine:latest",
		Command: []string{"echo", "no results here"},
	}

	result, err := b.ExecuteStep(ctx, req)
	if err != nil {
		t.Fatalf("ExecuteStep failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	// Results should be empty, not nil
	if result.Results == nil {
		t.Error("Results map should not be nil")
	}

	if len(result.Results) != 0 {
		t.Errorf("expected empty results, got %d results", len(result.Results))
	}
}
