package podman

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vdemeester/chisel/pkg/backend"
)

// TestPodmanBackendImplementsInterface verifies that PodmanBackend implements the Backend interface.
func TestPodmanBackendImplementsInterface(t *testing.T) {
	var _ backend.Backend = (*PodmanBackend)(nil)
}

// TestNewPodmanBackend tests the constructor.
func TestNewPodmanBackend(t *testing.T) {
	b := NewPodmanBackend()
	if b == nil {
		t.Fatal("NewPodmanBackend returned nil")
	}
}

// TestExecuteStep verifies ExecuteStep runs a container.
func TestExecuteStep(t *testing.T) {
	b := NewPodmanBackend()
	ctx := context.Background()

	req := &backend.StepRequest{
		Image:   "docker.io/library/alpine:latest",
		Command: []string{"echo", "hello from mallet"},
	}

	result, err := b.ExecuteStep(ctx, req)
	if err != nil {
		t.Fatalf("ExecuteStep failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Stdout, "hello from mallet") {
		t.Errorf("expected stdout to contain 'hello from mallet', got: %s", result.Stdout)
	}
	t.Logf("ExecuteStep output: %s", result.Stdout)
}

// TestExecuteStepWithEnv verifies ExecuteStep handles environment variables.
func TestExecuteStepWithEnv(t *testing.T) {
	b := NewPodmanBackend()
	ctx := context.Background()

	req := &backend.StepRequest{
		Image:   "docker.io/library/alpine:latest",
		Command: []string{"sh", "-c", "echo $TEST_VAR"},
		Env: map[string]string{
			"TEST_VAR": "mallet-test-value",
		},
	}

	result, err := b.ExecuteStep(ctx, req)
	if err != nil {
		t.Fatalf("ExecuteStep failed: %v", err)
	}
	if !strings.Contains(result.Stdout, "mallet-test-value") {
		t.Errorf("expected stdout to contain 'mallet-test-value', got: %s", result.Stdout)
	}
}

// TestExecuteStepNilRequest verifies ExecuteStep handles nil request.
func TestExecuteStepNilRequest(t *testing.T) {
	b := NewPodmanBackend()
	ctx := context.Background()

	result, err := b.ExecuteStep(ctx, nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

// TestStartSidecarNotImplemented verifies StartSidecar returns ErrNotImplemented.
func TestStartSidecarNotImplemented(t *testing.T) {
	b := NewPodmanBackend()
	ctx := context.Background()

	req := &backend.SidecarRequest{
		Name:  "redis",
		Image: "redis:latest",
	}

	handle, err := b.StartSidecar(ctx, req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotImplemented) {
		t.Errorf("expected ErrNotImplemented, got %v", err)
	}
	if handle != nil {
		t.Errorf("expected nil handle, got %+v", handle)
	}
}

// TestStopSidecarNotImplemented verifies StopSidecar returns ErrNotImplemented.
func TestStopSidecarNotImplemented(t *testing.T) {
	b := NewPodmanBackend()
	ctx := context.Background()

	handle := &backend.SidecarHandle{
		ID:      "test-sidecar-123",
		Name:    "redis",
		Backend: "podman",
	}

	err := b.StopSidecar(ctx, handle)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotImplemented) {
		t.Errorf("expected ErrNotImplemented, got %v", err)
	}
}

// TestReadResultNotImplemented verifies ReadResult returns ErrNotImplemented.
func TestReadResultNotImplemented(t *testing.T) {
	b := NewPodmanBackend()
	ctx := context.Background()

	req := &backend.ResultRequest{
		ContainerID: "container-123",
		Path:        "/tekton/results/output",
	}

	result, err := b.ReadResult(ctx, req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotImplemented) {
		t.Errorf("expected ErrNotImplemented, got %v", err)
	}
	if result != "" {
		t.Errorf("expected empty result, got %q", result)
	}
}

// TestCleanup verifies Cleanup works.
func TestCleanup(t *testing.T) {
	b := NewPodmanBackend()
	ctx := context.Background()

	// Cleanup should work even without any containers
	err := b.Cleanup(ctx)
	if err != nil {
		t.Errorf("Cleanup failed: %v", err)
	}
}

// TestErrNotImplementedMessage verifies the error message is user-friendly.
func TestErrNotImplementedMessage(t *testing.T) {
	expected := "podman backend not yet implemented"
	if ErrNotImplemented.Error() != expected {
		t.Errorf("expected error message %q, got %q", expected, ErrNotImplemented.Error())
	}
}
