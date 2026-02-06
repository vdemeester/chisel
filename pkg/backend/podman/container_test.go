package podman

import (
	"strings"
	"testing"
	"time"
)

// TestCreateContainer tests creating a container spec.
func TestCreateContainer(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skip("Podman not available")
	}
	defer client.Close()

	spec := ContainerSpec{
		Image:   "docker.io/library/alpine:latest",
		Command: []string{"echo", "hello"},
	}

	id, err := CreateContainer(client.Context(), spec)
	if err != nil {
		t.Fatalf("CreateContainer failed: %v", err)
	}

	if id == "" {
		t.Error("expected non-empty container ID")
	}

	t.Logf("Created container: %s", id)

	// Cleanup
	if err := RemoveContainer(client.Context(), id, true); err != nil {
		t.Logf("Cleanup warning: %v", err)
	}
}

// TestRunContainer tests running a container and capturing output.
func TestRunContainer(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skip("Podman not available")
	}
	defer client.Close()

	spec := ContainerSpec{
		Image:   "docker.io/library/alpine:latest",
		Command: []string{"echo", "hello from podman"},
	}

	result, err := RunContainer(client.Context(), spec)
	if err != nil {
		t.Fatalf("RunContainer failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	if !strings.Contains(result.Stdout, "hello from podman") {
		t.Errorf("expected stdout to contain 'hello from podman', got: %s", result.Stdout)
	}

	t.Logf("Container output: %s", result.Stdout)
}

// TestRunContainerWithEnv tests running a container with environment variables.
func TestRunContainerWithEnv(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skip("Podman not available")
	}
	defer client.Close()

	spec := ContainerSpec{
		Image:   "docker.io/library/alpine:latest",
		Command: []string{"sh", "-c", "echo $MY_VAR"},
		Env: map[string]string{
			"MY_VAR": "test-value",
		},
	}

	result, err := RunContainer(client.Context(), spec)
	if err != nil {
		t.Fatalf("RunContainer failed: %v", err)
	}

	if !strings.Contains(result.Stdout, "test-value") {
		t.Errorf("expected stdout to contain 'test-value', got: %s", result.Stdout)
	}
}

// TestRunContainerWithWorkDir tests running a container with a working directory.
func TestRunContainerWithWorkDir(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skip("Podman not available")
	}
	defer client.Close()

	spec := ContainerSpec{
		Image:   "docker.io/library/alpine:latest",
		Command: []string{"pwd"},
		WorkDir: "/tmp",
	}

	result, err := RunContainer(client.Context(), spec)
	if err != nil {
		t.Fatalf("RunContainer failed: %v", err)
	}

	if !strings.Contains(result.Stdout, "/tmp") {
		t.Errorf("expected stdout to contain '/tmp', got: %s", result.Stdout)
	}
}

// TestRunContainerWithTimeout tests container timeout handling.
func TestRunContainerWithTimeout(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skip("Podman not available")
	}
	defer client.Close()

	spec := ContainerSpec{
		Image:   "docker.io/library/alpine:latest",
		Command: []string{"sleep", "60"},
		Timeout: 2 * time.Second,
	}

	_, err = RunContainer(client.Context(), spec)
	if err == nil {
		t.Error("expected timeout error")
	}

	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "context") {
		t.Errorf("expected timeout-related error, got: %v", err)
	}

	t.Logf("Got expected timeout error: %v", err)
}

// TestRunContainerFailure tests handling of failed containers.
func TestRunContainerFailure(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skip("Podman not available")
	}
	defer client.Close()

	spec := ContainerSpec{
		Image:   "docker.io/library/alpine:latest",
		Command: []string{"sh", "-c", "echo error >&2; exit 1"},
	}

	result, err := RunContainer(client.Context(), spec)
	if err != nil {
		t.Fatalf("RunContainer failed: %v", err)
	}

	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", result.ExitCode)
	}

	if !strings.Contains(result.Stderr, "error") {
		t.Errorf("expected stderr to contain 'error', got: %s", result.Stderr)
	}
}

// TestContainerSpecValidation tests container spec validation.
func TestContainerSpecValidation(t *testing.T) {
	tests := []struct {
		name    string
		spec    ContainerSpec
		wantErr bool
	}{
		{
			name:    "empty spec",
			spec:    ContainerSpec{},
			wantErr: true,
		},
		{
			name: "no image",
			spec: ContainerSpec{
				Command: []string{"echo"},
			},
			wantErr: true,
		},
		{
			name: "valid minimal",
			spec: ContainerSpec{
				Image:   "alpine",
				Command: []string{"echo"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestRemoveContainer tests container removal.
func TestRemoveContainer(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skip("Podman not available")
	}
	defer client.Close()

	spec := ContainerSpec{
		Image:   "docker.io/library/alpine:latest",
		Command: []string{"echo", "test"},
	}

	id, err := CreateContainer(client.Context(), spec)
	if err != nil {
		t.Fatalf("CreateContainer failed: %v", err)
	}

	err = RemoveContainer(client.Context(), id, true)
	if err != nil {
		t.Errorf("RemoveContainer failed: %v", err)
	}
}

// TestContainerResult tests ContainerResult struct.
func TestContainerResult(t *testing.T) {
	result := ContainerResult{
		ContainerID: "abc123",
		ExitCode:    0,
		Stdout:      "output",
		Stderr:      "",
	}

	if result.ContainerID != "abc123" {
		t.Errorf("unexpected ContainerID: %s", result.ContainerID)
	}

	if result.ExitCode != 0 {
		t.Errorf("unexpected ExitCode: %d", result.ExitCode)
	}
}

// Integration test - skip in CI without Podman
func TestContainerLifecycle(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skip("Podman not available")
	}
	defer client.Close()

	ctx := client.Context()

	// Create
	spec := ContainerSpec{
		Image:   "docker.io/library/alpine:latest",
		Command: []string{"sh", "-c", "echo start; sleep 1; echo done"},
	}

	id, err := CreateContainer(ctx, spec)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	t.Logf("Created: %s", id)

	// Start
	if err := StartContainer(ctx, id); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	t.Log("Started")

	// Wait
	exitCode, err := WaitContainer(ctx, id)
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}
	t.Logf("Exited with code: %d", exitCode)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	// Get logs
	stdout, stderr, err := GetContainerLogs(ctx, id)
	if err != nil {
		t.Fatalf("GetLogs failed: %v", err)
	}
	t.Logf("Stdout: %s", stdout)
	t.Logf("Stderr: %s", stderr)

	// Remove
	if err := RemoveContainer(ctx, id, true); err != nil {
		t.Errorf("Remove failed: %v", err)
	}
	t.Log("Removed")
}
