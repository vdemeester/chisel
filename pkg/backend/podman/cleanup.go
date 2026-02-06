package podman

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// ResourceTracker tracks Podman resources for cleanup.
// It is thread-safe and can be used concurrently.
type ResourceTracker struct {
	mu         sync.Mutex
	containers []string
	pods       []string
	volumes    []string
}

// NewResourceTracker creates a new resource tracker.
func NewResourceTracker() *ResourceTracker {
	return &ResourceTracker{
		containers: make([]string, 0),
		pods:       make([]string, 0),
		volumes:    make([]string, 0),
	}
}

// AddContainer adds a container ID to be tracked.
func (rt *ResourceTracker) AddContainer(id string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.containers = append(rt.containers, id)
}

// AddPod adds a pod ID to be tracked.
func (rt *ResourceTracker) AddPod(id string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.pods = append(rt.pods, id)
}

// AddVolume adds a volume name to be tracked.
func (rt *ResourceTracker) AddVolume(name string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.volumes = append(rt.volumes, name)
}

// GetContainers returns a copy of tracked container IDs.
func (rt *ResourceTracker) GetContainers() []string {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	result := make([]string, len(rt.containers))
	copy(result, rt.containers)
	return result
}

// GetPods returns a copy of tracked pod IDs.
func (rt *ResourceTracker) GetPods() []string {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	result := make([]string, len(rt.pods))
	copy(result, rt.pods)
	return result
}

// GetVolumes returns a copy of tracked volume names.
func (rt *ResourceTracker) GetVolumes() []string {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	result := make([]string, len(rt.volumes))
	copy(result, rt.volumes)
	return result
}

// CleanupAll removes all tracked resources.
// It attempts to clean up all resources even if some fail.
// Uses a fresh context with timeout for cleanup.
func (rt *ResourceTracker) CleanupAll(ctx context.Context) error {
	// Use a fresh context for cleanup in case the original is canceled
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	rt.mu.Lock()
	containers := make([]string, len(rt.containers))
	copy(containers, rt.containers)
	pods := make([]string, len(rt.pods))
	copy(pods, rt.pods)
	volumes := make([]string, len(rt.volumes))
	copy(volumes, rt.volumes)
	rt.mu.Unlock()

	// Clean up containers first (stop and remove)
	for _, id := range containers {
		timeout := uint(5)
		_ = StopContainer(cleanupCtx, id, &timeout)
		_ = RemoveContainer(cleanupCtx, id, true)
	}

	// Clean up pods (this also removes containers in the pod)
	for _, id := range pods {
		_ = RemovePod(cleanupCtx, id, true)
	}

	// Clean up volumes
	for _, name := range volumes {
		_ = RemoveVolume(cleanupCtx, name, true)
	}

	// Clear tracked resources
	rt.mu.Lock()
	rt.containers = make([]string, 0)
	rt.pods = make([]string, 0)
	rt.volumes = make([]string, 0)
	rt.mu.Unlock()

	return nil
}

// RemoveVolume removes a Podman volume.
func RemoveVolume(ctx context.Context, name string, force bool) error {
	socketPath := detectSocketPath()
	if socketPath == "" {
		return ErrNoSocketFound
	}

	client := httpClient(socketPath)

	url := "http://d/v5.0.0/libpod/volumes/" + name
	if force {
		url += "?force=true"
	}

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	// 204 = success, 404 = not found (also OK for cleanup)
	if resp.StatusCode != 204 && resp.StatusCode != 404 {
		return fmt.Errorf("failed to remove volume: status %d", resp.StatusCode)
	}

	return nil
}
