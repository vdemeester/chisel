// Package backend provides an abstraction layer for container execution backends.
// This allows chisel to support multiple container runtimes (Dagger, Podman, etc.)
// through a common interface.
package backend

import (
	"context"
	"time"

	"github.com/vdemeester/chisel/pkg/types"
)

// Backend represents a container execution backend (Dagger, Podman, Docker, etc.)
// Implementations of this interface handle the low-level container operations
// while the orchestrator package handles high-level pipeline logic.
type Backend interface {
	// ExecuteStep runs a single step and returns its result.
	// The step runs in an isolated container with the specified configuration.
	ExecuteStep(ctx context.Context, req *StepRequest) (*StepResult, error)

	// StartSidecar starts a sidecar service container.
	// Sidecars run alongside step containers and provide services like databases.
	StartSidecar(ctx context.Context, req *SidecarRequest) (*SidecarHandle, error)

	// StopSidecar stops a running sidecar service.
	// This is called when the task completes or fails.
	StopSidecar(ctx context.Context, handle *SidecarHandle) error

	// ReadResult reads a result file from a completed step container.
	// Results are written by steps to /tekton/results/ and passed to subsequent steps.
	ReadResult(ctx context.Context, req *ResultRequest) (string, error)

	// Cleanup releases all resources created by this backend.
	// This should be called when pipeline execution completes.
	Cleanup(ctx context.Context) error
}

// StepRequest contains all information needed to execute a single step.
type StepRequest struct {
	// Step is the original Tekton step definition
	Step *types.Step

	// Image is the container image to run (after variable substitution)
	Image string

	// Command is the command to execute (after variable substitution)
	Command []string

	// Args are the command arguments (after variable substitution)
	Args []string

	// Env contains environment variables for the step
	Env map[string]string

	// WorkDir is the working directory inside the container
	WorkDir string

	// Workspaces maps workspace names to their mount configurations
	Workspaces map[string]WorkspaceMount

	// Timeout is the maximum execution time for this step
	Timeout time.Duration

	// Sidecars contains handles to running sidecar services
	Sidecars []SidecarHandle
}

// StepResult contains the outcome of step execution.
type StepResult struct {
	// ExitCode is the container exit code (0 = success)
	ExitCode int

	// Stdout contains the step's standard output
	Stdout string

	// Stderr contains the step's standard error
	Stderr string

	// Results maps result names to their values (from /tekton/results/)
	Results map[string]string

	// Error is set if the step failed to execute
	Error error
}

// SidecarRequest contains configuration for starting a sidecar service.
type SidecarRequest struct {
	// Name is the sidecar name (used for service discovery)
	Name string

	// Image is the container image to run
	Image string

	// Command is the command to execute
	Command []string

	// Env contains environment variables for the sidecar
	Env map[string]string

	// Ports are the ports to expose from the sidecar
	Ports []int32
}

// SidecarHandle references a running sidecar service.
// This is returned by StartSidecar and passed to subsequent operations.
type SidecarHandle struct {
	// ID is the backend-specific identifier for this sidecar
	ID string

	// Name is the sidecar name from the task definition
	Name string

	// Backend identifies which backend created this sidecar
	Backend string

	// Metadata contains backend-specific data about the sidecar
	Metadata map[string]interface{}
}

// WorkspaceMount describes how to mount a workspace into a container.
type WorkspaceMount struct {
	// Name is the workspace name from the task definition
	Name string

	// MountPath is where to mount the workspace inside the container
	MountPath string

	// SourceType indicates the workspace type: emptyDir, local, pvc, configMap, secret
	SourceType string

	// SourcePath is the host path for local workspaces
	SourcePath string

	// VolumeID is the backend-specific volume identifier for emptyDir/pvc workspaces
	VolumeID string
}

// ResultRequest describes a result file to read from a container.
type ResultRequest struct {
	// ContainerID is the backend-specific container identifier
	ContainerID string

	// Path is the full path to the result file inside the container
	Path string
}
