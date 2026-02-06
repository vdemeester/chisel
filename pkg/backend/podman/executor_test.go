package podman

import (
	"context"
	"errors"
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

// TestExecuteStepNotImplemented verifies ExecuteStep returns ErrNotImplemented.
func TestExecuteStepNotImplemented(t *testing.T) {
	b := NewPodmanBackend()
	ctx := context.Background()

	req := &backend.StepRequest{
		Image:   "alpine:latest",
		Command: []string{"echo", "hello"},
	}

	result, err := b.ExecuteStep(ctx, req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotImplemented) {
		t.Errorf("expected ErrNotImplemented, got %v", err)
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

// TestCleanupNotImplemented verifies Cleanup returns ErrNotImplemented.
func TestCleanupNotImplemented(t *testing.T) {
	b := NewPodmanBackend()
	ctx := context.Background()

	err := b.Cleanup(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotImplemented) {
		t.Errorf("expected ErrNotImplemented, got %v", err)
	}
}

// TestErrNotImplementedMessage verifies the error message is user-friendly.
func TestErrNotImplementedMessage(t *testing.T) {
	expected := "podman backend not yet implemented"
	if ErrNotImplemented.Error() != expected {
		t.Errorf("expected error message %q, got %q", expected, ErrNotImplemented.Error())
	}
}
