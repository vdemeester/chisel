package podman

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/vdemeester/chisel/pkg/backend"
)

// Pod API types

type createPodRequest struct {
	Name    string   `json:"name"`
	HostAdd []string `json:"hostadd,omitempty"` // Add entries to /etc/hosts (format: "host:ip")
}

type createPodResponse struct {
	Id string `json:"Id"`
}

// PodOptions configures pod creation.
type PodOptions struct {
	// HostAliases maps hostnames to IP addresses (added to /etc/hosts)
	HostAliases map[string]string
}

// CreatePod creates a new Podman pod.
// Pods provide a shared network namespace for containers.
func CreatePod(ctx context.Context, name string) (string, error) {
	return CreatePodWithOptions(ctx, name, PodOptions{})
}

// CreatePodWithOptions creates a new Podman pod with additional options.
func CreatePodWithOptions(ctx context.Context, name string, opts PodOptions) (string, error) {
	socketPath := detectSocketPath()
	if socketPath == "" {
		return "", ErrNoSocketFound
	}

	client := httpClient(socketPath)

	reqBody := createPodRequest{
		Name: name,
	}

	// Add host aliases (for sidecar hostname resolution)
	for host, ip := range opts.HostAliases {
		reqBody.HostAdd = append(reqBody.HostAdd, fmt.Sprintf("%s:%s", host, ip))
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "http://d/v5.0.0/libpod/pods/create", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to create pod: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create pod: %s (status %d)", string(respBody), resp.StatusCode)
	}

	var result createPodResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Id, nil
}

// StartPod starts a created pod.
func StartPod(ctx context.Context, id string) error {
	socketPath := detectSocketPath()
	if socketPath == "" {
		return ErrNoSocketFound
	}

	client := httpClient(socketPath)

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("http://d/v5.0.0/libpod/pods/%s/start", id), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to start pod: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotModified {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to start pod: %s (status %d)", string(respBody), resp.StatusCode)
	}

	return nil
}

// StopPod stops a running pod.
func StopPod(ctx context.Context, id string) error {
	socketPath := detectSocketPath()
	if socketPath == "" {
		return ErrNoSocketFound
	}

	client := httpClient(socketPath)

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("http://d/v5.0.0/libpod/pods/%s/stop", id), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to stop pod: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotModified {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to stop pod: %s (status %d)", string(respBody), resp.StatusCode)
	}

	return nil
}

// RemovePod removes a pod and all its containers.
func RemovePod(ctx context.Context, id string, force bool) error {
	socketPath := detectSocketPath()
	if socketPath == "" {
		return ErrNoSocketFound
	}

	client := httpClient(socketPath)

	url := fmt.Sprintf("http://d/v5.0.0/libpod/pods/%s?force=%t", id, force)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to remove pod: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to remove pod: %s (status %d)", string(respBody), resp.StatusCode)
	}

	return nil
}

// CreateContainerInPod creates a container inside a pod.
func CreateContainerInPod(ctx context.Context, podID string, spec ContainerSpec) (string, error) {
	if err := spec.Validate(); err != nil {
		return "", fmt.Errorf("invalid container spec: %w", err)
	}

	socketPath := detectSocketPath()
	if socketPath == "" {
		return "", ErrNoSocketFound
	}

	// Pull image if needed
	if err := PullImage(ctx, spec.Image); err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}

	client := httpClient(socketPath)

	// Build the request body with pod
	reqBody := createContainerRequest{
		Image:            spec.Image,
		Command:          spec.Command,
		Entrypoint:       spec.Entrypoint,
		Env:              spec.Env,
		WorkDir:          spec.WorkDir,
		CreateWorkingDir: true, // Create workdir if it doesn't exist
		Name:             spec.Name,
		Pod:              podID, // Join this pod
	}

	// Convert mounts
	for _, m := range spec.Mounts {
		mt := mountSpec{
			Type:        "bind",
			Source:      m.Source,
			Destination: m.Target,
		}
		if m.ReadOnly {
			mt.Options = []string{"ro"}
		}
		reqBody.Mounts = append(reqBody.Mounts, mt)
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create container in pod (pod ID in body, not query param)
	req, err := http.NewRequestWithContext(ctx, "POST", "http://d/v5.0.0/libpod/containers/create", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create container in pod: %s (status %d)", string(respBody), resp.StatusCode)
	}

	var result createResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Id, nil
}

// StartSidecar implementation for PodmanBackend
func (b *PodmanBackend) startSidecarImpl(ctx context.Context, req *backend.SidecarRequest) (*backend.SidecarHandle, error) {
	if req == nil {
		return nil, errors.New("sidecar request is nil")
	}

	if err := b.ensureClient(); err != nil {
		return nil, fmt.Errorf("failed to initialize podman client: %w", err)
	}

	// Build container spec from sidecar request
	spec := ContainerSpec{
		Image:   req.Image,
		Command: req.Command,
		Env:     req.Env,
	}

	// For sidecars without a command, we need to use the image's default entrypoint
	if len(spec.Command) == 0 {
		// Don't set any command - use image default
		spec.Command = nil
	}

	// Create the container (not in a pod for now - simpler approach)
	id, err := CreateContainer(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("failed to create sidecar container: %w", err)
	}

	// Start the container
	if err := StartContainer(ctx, id); err != nil {
		// Cleanup on failure
		_ = RemoveContainer(ctx, id, true)
		return nil, fmt.Errorf("failed to start sidecar container: %w", err)
	}

	return &backend.SidecarHandle{
		ID:      id,
		Name:    req.Name,
		Backend: "podman",
		Metadata: map[string]interface{}{
			"ports": req.Ports,
		},
	}, nil
}

// StopSidecar implementation for PodmanBackend
func (b *PodmanBackend) stopSidecarImpl(ctx context.Context, handle *backend.SidecarHandle) error {
	if handle == nil {
		return errors.New("sidecar handle is nil")
	}

	// Stop the container (ignore errors, we'll try removal anyway)
	timeout := uint(10)
	_ = StopContainer(ctx, handle.ID, &timeout)

	// Remove the container
	return RemoveContainer(ctx, handle.ID, true)
}

// RunContainerInPod creates, starts, waits for, and retrieves logs from a container in a pod.
// This is similar to RunContainer but runs the container inside an existing pod.
func RunContainerInPod(ctx context.Context, podID string, spec ContainerSpec) (*ContainerResult, error) {
	// Create container in pod
	id, err := CreateContainerInPod(ctx, podID, spec)
	if err != nil {
		return nil, err
	}

	// Start container
	if err := StartContainer(ctx, id); err != nil {
		_ = RemoveContainer(ctx, id, true)
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// Wait for container (with timeout if specified)
	var waitCtx context.Context
	var cancel context.CancelFunc
	if spec.Timeout > 0 {
		waitCtx, cancel = context.WithTimeout(ctx, spec.Timeout)
		defer cancel()
	} else {
		waitCtx = ctx
	}
	exitCode, err := WaitContainer(waitCtx, id)
	if err != nil {
		// Try to get logs even on error
		stdout, stderr, _ := GetContainerLogs(ctx, id)
		_ = RemoveContainer(ctx, id, true)
		return &ContainerResult{
			ContainerID: id,
			ExitCode:    -1,
			Stdout:      stdout,
			Stderr:      stderr,
		}, err
	}

	// Get logs
	stdout, stderr, err := GetContainerLogs(ctx, id)
	if err != nil {
		_ = RemoveContainer(ctx, id, true)
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}

	// Remove container
	_ = RemoveContainer(ctx, id, true)

	return &ContainerResult{
		ContainerID: id,
		ExitCode:    exitCode,
		Stdout:      stdout,
		Stderr:      stderr,
	}, nil
}
