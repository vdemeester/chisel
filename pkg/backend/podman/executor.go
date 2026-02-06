// Package podman implements the Backend interface using Podman as the container runtime.
// This allows running Tekton pipelines locally using Podman instead of Dagger.
package podman

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/vdemeester/chisel/pkg/backend"
)

// ErrNotImplemented is returned by methods not yet implemented.
var ErrNotImplemented = errors.New("podman backend not yet implemented")

// PodmanBackend implements the backend.Backend interface using Podman.
type PodmanBackend struct {
	client *Client
}

// NewPodmanBackend creates a new Podman backend instance.
func NewPodmanBackend() *PodmanBackend {
	return &PodmanBackend{}
}

// ensureClient initializes the Podman client if not already done.
func (b *PodmanBackend) ensureClient() error {
	if b.client != nil {
		return nil
	}
	client, err := NewClient()
	if err != nil {
		return err
	}
	b.client = client
	return nil
}

// ExecuteStep runs a single step in a Podman container.
func (b *PodmanBackend) ExecuteStep(ctx context.Context, req *backend.StepRequest) (*backend.StepResult, error) {
	if req == nil {
		return nil, errors.New("step request is nil")
	}

	if err := b.ensureClient(); err != nil {
		return nil, fmt.Errorf("failed to initialize podman client: %w", err)
	}

	// Create a temp directory for Tekton results
	resultsDir, err := os.MkdirTemp("", "tekton-results-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create results directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(resultsDir) }()

	// Build the container spec from the step request
	spec := ContainerSpec{
		Image:   req.Image,
		Timeout: req.Timeout,
	}

	// Build command - combine Command and Args
	if len(req.Command) > 0 {
		spec.Command = append(spec.Command, req.Command...)
	}
	if len(req.Args) > 0 {
		spec.Command = append(spec.Command, req.Args...)
	}

	// If no command specified, try to use the step's script
	if len(spec.Command) == 0 && req.Step != nil && req.Step.Script != "" {
		// For scripts, we wrap in sh -c
		spec.Command = []string{"sh", "-c", req.Step.Script}
	}

	// Fallback to echo if still no command
	if len(spec.Command) == 0 {
		spec.Command = []string{"echo", "no command specified"}
	}

	// Set environment variables
	if len(req.Env) > 0 {
		spec.Env = req.Env
	}

	// Set working directory
	if req.WorkDir != "" {
		spec.WorkDir = req.WorkDir
	}

	// Mount the results directory
	spec.Mounts = append(spec.Mounts, Mount{
		Source: resultsDir,
		Target: ResultsDir,
	})

	// Add workspace mounts
	for name, ws := range req.Workspaces {
		if ws.SourcePath != "" {
			spec.Mounts = append(spec.Mounts, Mount{
				Source: ws.SourcePath,
				Target: ws.MountPath,
			})
		} else if ws.VolumeID != "" {
			// Named volume - for now we treat it as a path
			spec.Mounts = append(spec.Mounts, Mount{
				Source: ws.VolumeID,
				Target: ws.MountPath,
			})
		}
		_ = name // Silence unused warning
	}

	// Run the container
	result, err := RunContainer(ctx, spec)
	if err != nil {
		// Check if it's a timeout
		if strings.Contains(err.Error(), "timeout") {
			return &backend.StepResult{
				ExitCode: -1,
				Error:    err,
			}, nil
		}
		return nil, fmt.Errorf("container execution failed: %w", err)
	}

	// Collect results from the temp directory
	results, err := CollectResults(resultsDir)
	if err != nil {
		// Log warning but don't fail - results are optional
		results = make(map[string]string)
	}

	return &backend.StepResult{
		ExitCode: result.ExitCode,
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
		Results:  results,
	}, nil
}

// StartSidecar starts a sidecar service container.
// TODO: Implement using Podman pods for network sharing.
func (b *PodmanBackend) StartSidecar(ctx context.Context, req *backend.SidecarRequest) (*backend.SidecarHandle, error) {
	return nil, ErrNotImplemented
}

// StopSidecar stops a running sidecar service.
// TODO: Implement container stop and removal.
func (b *PodmanBackend) StopSidecar(ctx context.Context, handle *backend.SidecarHandle) error {
	return ErrNotImplemented
}

// ReadResult reads a result file from a completed step container.
// For the Podman backend, results are captured via mounted temp directories
// during ExecuteStep, so this method is primarily for reading from a path.
func (b *PodmanBackend) ReadResult(ctx context.Context, req *backend.ResultRequest) (string, error) {
	if req == nil {
		return "", errors.New("result request is nil")
	}

	// If the path looks like a local path (starts with /tmp or similar),
	// read directly from the filesystem
	if req.Path != "" {
		return ReadResultFromPath(req.Path)
	}

	return "", fmt.Errorf("cannot read result: container %s no longer exists", req.ContainerID)
}

// Cleanup releases all resources created by this backend.
func (b *PodmanBackend) Cleanup(ctx context.Context) error {
	if b.client != nil {
		return b.client.Close()
	}
	return nil
}
