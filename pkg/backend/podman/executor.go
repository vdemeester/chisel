// Package podman implements the Backend interface using Podman as the container runtime.
// This allows running Tekton pipelines locally using Podman instead of Dagger.
package podman

import (
	"context"
	"errors"

	"github.com/vdemeester/chisel/pkg/backend"
)

// ErrNotImplemented is returned by all methods until the Podman backend is implemented.
var ErrNotImplemented = errors.New("podman backend not yet implemented")

// PodmanBackend implements the backend.Backend interface using Podman.
type PodmanBackend struct {
	// Fields will be added in Phase 3 when implementing Podman API bindings.
	// Placeholder to satisfy the linter.
	_ struct{}
}

// NewPodmanBackend creates a new Podman backend instance.
func NewPodmanBackend() *PodmanBackend {
	return &PodmanBackend{}
}

// ExecuteStep runs a single step in a Podman container.
// TODO: Implement using Podman API bindings.
func (b *PodmanBackend) ExecuteStep(ctx context.Context, req *backend.StepRequest) (*backend.StepResult, error) {
	return nil, ErrNotImplemented
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
// TODO: Implement using podman exec or container copy.
func (b *PodmanBackend) ReadResult(ctx context.Context, req *backend.ResultRequest) (string, error) {
	return "", ErrNotImplemented
}

// Cleanup releases all resources created by this backend.
// TODO: Implement cleanup of containers, pods, and volumes.
func (b *PodmanBackend) Cleanup(ctx context.Context) error {
	return ErrNotImplemented
}
