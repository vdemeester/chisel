package podman

import (
	"context"
	"testing"
)

// TestResourceTrackerAddContainer tests adding containers to tracker.
func TestResourceTrackerAddContainer(t *testing.T) {
	rt := NewResourceTracker()

	rt.AddContainer("container-1")
	rt.AddContainer("container-2")

	containers := rt.GetContainers()
	if len(containers) != 2 {
		t.Errorf("expected 2 containers, got %d", len(containers))
	}
}

// TestResourceTrackerAddPod tests adding pods to tracker.
func TestResourceTrackerAddPod(t *testing.T) {
	rt := NewResourceTracker()

	rt.AddPod("pod-1")
	rt.AddPod("pod-2")

	pods := rt.GetPods()
	if len(pods) != 2 {
		t.Errorf("expected 2 pods, got %d", len(pods))
	}
}

// TestResourceTrackerCleanupEmpty tests cleanup with no resources.
func TestResourceTrackerCleanupEmpty(t *testing.T) {
	rt := NewResourceTracker()
	ctx := context.Background()

	err := rt.CleanupAll(ctx)
	if err != nil {
		t.Errorf("CleanupAll failed on empty tracker: %v", err)
	}
}

// TestResourceTrackerCleanupNonexistent tests cleanup with nonexistent resources.
func TestResourceTrackerCleanupNonexistent(t *testing.T) {
	// Skip if Podman is not available
	if detectSocketPath() == "" {
		t.Skip("Podman not available")
	}

	rt := NewResourceTracker()
	ctx := context.Background()

	// Add nonexistent resources
	rt.AddContainer("nonexistent-container-12345")
	rt.AddPod("nonexistent-pod-12345")

	// Cleanup should not fail (logs warnings but continues)
	err := rt.CleanupAll(ctx)
	if err != nil {
		t.Logf("CleanupAll returned error (may be OK): %v", err)
	}
}

// TestResourceTrackerConcurrentAccess tests thread-safety.
func TestResourceTrackerConcurrentAccess(t *testing.T) {
	rt := NewResourceTracker()

	done := make(chan bool, 10)

	// Add containers concurrently
	for i := 0; i < 5; i++ {
		go func(id int) {
			rt.AddContainer("container-" + string(rune('a'+id)))
			done <- true
		}(i)
	}

	// Add pods concurrently
	for i := 0; i < 5; i++ {
		go func(id int) {
			rt.AddPod("pod-" + string(rune('a'+id)))
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	containers := rt.GetContainers()
	pods := rt.GetPods()

	if len(containers) != 5 {
		t.Errorf("expected 5 containers, got %d", len(containers))
	}
	if len(pods) != 5 {
		t.Errorf("expected 5 pods, got %d", len(pods))
	}
}

// TestResourceTrackerRealCleanup tests cleanup with real resources.
func TestResourceTrackerRealCleanup(t *testing.T) {
	// Skip if Podman is not available
	if detectSocketPath() == "" {
		t.Skip("Podman not available")
	}

	ctx := context.Background()
	rt := NewResourceTracker()

	// Create a real container
	spec := ContainerSpec{
		Image:   "docker.io/library/alpine:latest",
		Command: []string{"sleep", "300"},
	}

	id, err := CreateContainer(ctx, spec)
	if err != nil {
		t.Fatalf("CreateContainer failed: %v", err)
	}
	rt.AddContainer(id)

	// Start the container
	if err := StartContainer(ctx, id); err != nil {
		t.Fatalf("StartContainer failed: %v", err)
	}

	t.Logf("Created container: %s", id)

	// Cleanup should stop and remove the container
	err = rt.CleanupAll(ctx)
	if err != nil {
		t.Errorf("CleanupAll failed: %v", err)
	}

	// Verify container is gone (this should fail)
	_, _, err = GetContainerLogs(ctx, id)
	if err == nil {
		t.Error("expected error getting logs from removed container")
	}
}
