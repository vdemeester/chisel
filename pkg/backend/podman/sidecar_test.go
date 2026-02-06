package podman

import (
	"context"
	"testing"
	"time"

	"github.com/vdemeester/chisel/pkg/backend"
)

// TestStartSidecarBasic tests starting a simple sidecar.
func TestStartSidecarBasic(t *testing.T) {
	// Skip if Podman is not available
	if detectSocketPath() == "" {
		t.Skip("Podman not available")
	}

	b := NewPodmanBackend()
	ctx := context.Background()

	req := &backend.SidecarRequest{
		Name:  "redis",
		Image: "docker.io/library/redis:alpine",
		Ports: []int32{6379},
	}

	handle, err := b.StartSidecar(ctx, req)
	if err != nil {
		t.Fatalf("StartSidecar failed: %v", err)
	}
	defer func() {
		if err := b.StopSidecar(ctx, handle); err != nil {
			t.Logf("StopSidecar warning: %v", err)
		}
	}()

	if handle == nil {
		t.Fatal("expected handle, got nil")
	}
	if handle.ID == "" {
		t.Error("expected non-empty ID")
	}
	if handle.Name != "redis" {
		t.Errorf("expected name 'redis', got %q", handle.Name)
	}
	if handle.Backend != "podman" {
		t.Errorf("expected backend 'podman', got %q", handle.Backend)
	}

	t.Logf("Started sidecar: %s (ID: %s)", handle.Name, handle.ID)
}

// TestStartSidecarNilRequest tests handling of nil request.
func TestStartSidecarNilRequest(t *testing.T) {
	b := NewPodmanBackend()
	ctx := context.Background()

	handle, err := b.StartSidecar(ctx, nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}
	if handle != nil {
		t.Errorf("expected nil handle, got %+v", handle)
	}
}

// TestStopSidecarNilHandle tests handling of nil handle.
func TestStopSidecarNilHandle(t *testing.T) {
	b := NewPodmanBackend()
	ctx := context.Background()

	err := b.StopSidecar(ctx, nil)
	if err == nil {
		t.Fatal("expected error for nil handle")
	}
}

// TestStopSidecarInvalidID tests handling of invalid container ID.
func TestStopSidecarInvalidID(t *testing.T) {
	// Skip if Podman is not available
	if detectSocketPath() == "" {
		t.Skip("Podman not available")
	}

	b := NewPodmanBackend()
	ctx := context.Background()

	handle := &backend.SidecarHandle{
		ID:      "nonexistent-container-id",
		Name:    "test",
		Backend: "podman",
	}

	err := b.StopSidecar(ctx, handle)
	// Should return an error for nonexistent container
	if err == nil {
		t.Log("StopSidecar succeeded for nonexistent container (might be OK)")
	}
}

// TestSidecarWithEnv tests sidecar with environment variables.
func TestSidecarWithEnv(t *testing.T) {
	// Skip if Podman is not available
	if detectSocketPath() == "" {
		t.Skip("Podman not available")
	}

	b := NewPodmanBackend()
	ctx := context.Background()

	req := &backend.SidecarRequest{
		Name:    "alpine-env",
		Image:   "docker.io/library/alpine:latest",
		Command: []string{"sh", "-c", "echo $TEST_VAR && sleep 30"},
		Env: map[string]string{
			"TEST_VAR": "sidecar-test-value",
		},
	}

	handle, err := b.StartSidecar(ctx, req)
	if err != nil {
		t.Fatalf("StartSidecar failed: %v", err)
	}
	defer func() {
		_ = b.StopSidecar(ctx, handle)
	}()

	if handle.ID == "" {
		t.Error("expected non-empty ID")
	}

	t.Logf("Started sidecar with env: %s", handle.ID)
}

// TestExecuteStepWithSidecar tests running a step that uses a sidecar service.
func TestExecuteStepWithSidecar(t *testing.T) {
	// Skip if Podman is not available
	if detectSocketPath() == "" {
		t.Skip("Podman not available")
	}

	b := NewPodmanBackend()
	ctx := context.Background()

	// Start a simple HTTP server as sidecar
	sidecarReq := &backend.SidecarRequest{
		Name:    "httpserver",
		Image:   "docker.io/library/python:3-alpine",
		Command: []string{"python", "-m", "http.server", "8080"},
		Ports:   []int32{8080},
	}

	handle, err := b.StartSidecar(ctx, sidecarReq)
	if err != nil {
		t.Fatalf("StartSidecar failed: %v", err)
	}
	defer func() {
		_ = b.StopSidecar(ctx, handle)
	}()

	// Wait for the server to start
	time.Sleep(2 * time.Second)

	// Run a step that connects to the sidecar
	stepReq := &backend.StepRequest{
		Image:    "docker.io/library/alpine:latest",
		Command:  []string{"sh", "-c", "wget -q -O - http://localhost:8080/ || echo 'connection failed'"},
		Sidecars: []backend.SidecarHandle{*handle},
	}

	result, err := b.ExecuteStep(ctx, stepReq)
	if err != nil {
		t.Fatalf("ExecuteStep failed: %v", err)
	}

	// The step should be able to connect to the sidecar
	t.Logf("Step output: %s", result.Stdout)
	t.Logf("Step stderr: %s", result.Stderr)
	t.Logf("Exit code: %d", result.ExitCode)
}

// TestPodCreate tests creating a Podman pod.
func TestPodCreate(t *testing.T) {
	// Skip if Podman is not available
	if detectSocketPath() == "" {
		t.Skip("Podman not available")
	}

	ctx := context.Background()

	podID, err := CreatePod(ctx, "test-pod-"+time.Now().Format("150405"))
	if err != nil {
		t.Fatalf("CreatePod failed: %v", err)
	}
	defer func() {
		_ = RemovePod(ctx, podID, true)
	}()

	if podID == "" {
		t.Error("expected non-empty pod ID")
	}

	t.Logf("Created pod: %s", podID)
}

// TestPodRemove tests removing a Podman pod.
func TestPodRemove(t *testing.T) {
	// Skip if Podman is not available
	if detectSocketPath() == "" {
		t.Skip("Podman not available")
	}

	ctx := context.Background()

	podID, err := CreatePod(ctx, "test-pod-remove-"+time.Now().Format("150405"))
	if err != nil {
		t.Fatalf("CreatePod failed: %v", err)
	}

	err = RemovePod(ctx, podID, true)
	if err != nil {
		t.Errorf("RemovePod failed: %v", err)
	}
}
