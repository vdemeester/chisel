package backend_test

import (
	"context"
	"testing"
	"time"

	"github.com/vdemeester/chisel/pkg/backend"
	"github.com/vdemeester/chisel/pkg/types"
)

// TestBackendInterface verifies that the Backend interface is properly defined
// This test will compile only if the interface exists with correct method signatures
func TestBackendInterface(t *testing.T) {
	// This test verifies that any Backend implementation has the required methods
	var _ backend.Backend = (*mockBackend)(nil)
}

// mockBackend is a minimal implementation for testing the interface
type mockBackend struct {
	executeStepFunc  func(context.Context, *backend.StepRequest) (*backend.StepResult, error)
	startSidecarFunc func(context.Context, *backend.SidecarRequest) (*backend.SidecarHandle, error)
	stopSidecarFunc  func(context.Context, *backend.SidecarHandle) error
	readResultFunc   func(context.Context, *backend.ResultRequest) (string, error)
	cleanupFunc      func(context.Context) error
}

func (m *mockBackend) ExecuteStep(ctx context.Context, req *backend.StepRequest) (*backend.StepResult, error) {
	if m.executeStepFunc != nil {
		return m.executeStepFunc(ctx, req)
	}
	return &backend.StepResult{}, nil
}

func (m *mockBackend) StartSidecar(ctx context.Context, req *backend.SidecarRequest) (*backend.SidecarHandle, error) {
	if m.startSidecarFunc != nil {
		return m.startSidecarFunc(ctx, req)
	}
	return &backend.SidecarHandle{}, nil
}

func (m *mockBackend) StopSidecar(ctx context.Context, handle *backend.SidecarHandle) error {
	if m.stopSidecarFunc != nil {
		return m.stopSidecarFunc(ctx, handle)
	}
	return nil
}

func (m *mockBackend) ReadResult(ctx context.Context, req *backend.ResultRequest) (string, error) {
	if m.readResultFunc != nil {
		return m.readResultFunc(ctx, req)
	}
	return "", nil
}

func (m *mockBackend) Cleanup(ctx context.Context) error {
	if m.cleanupFunc != nil {
		return m.cleanupFunc(ctx)
	}
	return nil
}

// TestStepRequestValidation tests that StepRequest contains all necessary fields
func TestStepRequestValidation(t *testing.T) {
	req := &backend.StepRequest{
		Step:       &types.Step{Name: "test-step"},
		Image:      "alpine:latest",
		Command:    []string{"/bin/sh"},
		Args:       []string{"-c", "echo hello"},
		Env:        map[string]string{"FOO": "bar"},
		WorkDir:    "/workspace",
		Workspaces: map[string]backend.WorkspaceMount{},
		Timeout:    30 * time.Second,
		Sidecars:   []backend.SidecarHandle{},
	}

	if req.Image == "" {
		t.Error("StepRequest should have Image field")
	}
	if req.Command == nil {
		t.Error("StepRequest should have Command field")
	}
	if req.Env == nil {
		t.Error("StepRequest should have Env field")
	}
}

// TestStepResultStructure tests that StepResult has all necessary fields
func TestStepResultStructure(t *testing.T) {
	result := &backend.StepResult{
		ExitCode: 0,
		Stdout:   "output",
		Stderr:   "",
		Results:  map[string]string{"key": "value"},
		Error:    nil,
	}

	if result.ExitCode != 0 {
		t.Error("StepResult should have ExitCode field")
	}
	if result.Stdout == "" {
		t.Error("StepResult should have Stdout field")
	}
	if result.Results == nil {
		t.Error("StepResult should have Results map")
	}
}

// TestSidecarRequestStructure tests that SidecarRequest has required fields
func TestSidecarRequestStructure(t *testing.T) {
	req := &backend.SidecarRequest{
		Name:    "test-sidecar",
		Image:   "nginx:latest",
		Command: []string{"nginx"},
		Env:     map[string]string{"PORT": "8080"},
		Ports:   []int32{80, 443},
	}

	if req.Name == "" {
		t.Error("SidecarRequest should have Name field")
	}
	if req.Image == "" {
		t.Error("SidecarRequest should have Image field")
	}
}

// TestSidecarHandleStructure tests that SidecarHandle has required fields
func TestSidecarHandleStructure(t *testing.T) {
	handle := &backend.SidecarHandle{
		ID:       "sidecar-123",
		Name:     "test-sidecar",
		Backend:  "dagger",
		Metadata: map[string]interface{}{"key": "value"},
	}

	if handle.ID == "" {
		t.Error("SidecarHandle should have ID field")
	}
	if handle.Name == "" {
		t.Error("SidecarHandle should have Name field")
	}
	if handle.Backend == "" {
		t.Error("SidecarHandle should have Backend field")
	}
}

// TestWorkspaceMountStructure tests that WorkspaceMount has required fields
func TestWorkspaceMountStructure(t *testing.T) {
	mount := &backend.WorkspaceMount{
		Name:       "source",
		MountPath:  "/workspace/source",
		SourceType: "emptyDir",
		SourcePath: "",
		VolumeID:   "vol-123",
	}

	if mount.Name == "" {
		t.Error("WorkspaceMount should have Name field")
	}
	if mount.MountPath == "" {
		t.Error("WorkspaceMount should have MountPath field")
	}
	if mount.SourceType == "" {
		t.Error("WorkspaceMount should have SourceType field")
	}
}

// TestResultRequestStructure tests that ResultRequest has required fields
func TestResultRequestStructure(t *testing.T) {
	req := &backend.ResultRequest{
		ContainerID: "container-123",
		Path:        "/tekton/results/output",
	}

	if req.ContainerID == "" {
		t.Error("ResultRequest should have ContainerID field")
	}
	if req.Path == "" {
		t.Error("ResultRequest should have Path field")
	}
}
