package podman

import (
	"context"
	"os"
	"testing"
)

// TestNewClientDefaultSocket tests that NewClient can find the default socket.
func TestNewClientDefaultSocket(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		// It's OK if Podman isn't running - we just want to verify the client creation logic
		t.Logf("NewClient returned error (expected if Podman not running): %v", err)
		return
	}
	defer func() { _ = client.Close() }()

	if client.socketPath == "" {
		t.Error("expected socketPath to be set")
	}
}

// TestNewClientWithSocket tests creating a client with a specific socket path.
func TestNewClientWithSocket(t *testing.T) {
	socketPath := "/tmp/test-podman.sock"
	client, err := NewClientWithSocket(socketPath)
	if err != nil {
		// Socket doesn't exist, that's expected
		t.Logf("NewClientWithSocket returned error (expected): %v", err)
		return
	}
	defer func() { _ = client.Close() }()

	if client.socketPath != socketPath {
		t.Errorf("expected socketPath %q, got %q", socketPath, client.socketPath)
	}
}

// TestDetectSocketPath tests socket path detection.
func TestDetectSocketPath(t *testing.T) {
	path := detectSocketPath()

	// Should return a path (either user or system socket)
	if path == "" {
		t.Skip("No Podman socket detected - Podman may not be installed")
	}

	t.Logf("Detected socket path: %s", path)

	// Path should start with / (absolute path) or be a valid XDG path
	if path[0] != '/' {
		t.Errorf("expected absolute path, got %q", path)
	}
}

// TestUserSocketPath tests that we can construct the user socket path.
func TestUserSocketPath(t *testing.T) {
	path := userSocketPath()

	// Should be in XDG_RUNTIME_DIR or /run/user/<uid>
	if path == "" {
		t.Error("userSocketPath returned empty string")
	}

	// Should end with podman/podman.sock
	expected := "podman/podman.sock"
	if len(path) < len(expected) || path[len(path)-len(expected):] != expected {
		t.Errorf("expected path to end with %q, got %q", expected, path)
	}
}

// TestSystemSocketPath tests the system socket path constant.
func TestSystemSocketPath(t *testing.T) {
	path := systemSocketPath()

	expected := "/run/podman/podman.sock"
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}

// TestClientIsConnected tests the connection check.
func TestClientIsConnected(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skip("Podman not available")
	}
	defer func() { _ = client.Close() }()

	connected := client.IsConnected(context.Background())
	t.Logf("IsConnected: %v", connected)
}

// TestErrPodmanNotRunning tests the error message.
func TestErrPodmanNotRunning(t *testing.T) {
	err := ErrPodmanNotRunning
	if err == nil {
		t.Fatal("ErrPodmanNotRunning should not be nil")
	}

	msg := err.Error()
	if msg == "" {
		t.Error("expected non-empty error message")
	}

	// Should mention Podman
	if !contains(msg, "podman") && !contains(msg, "Podman") {
		t.Errorf("expected error message to mention Podman, got: %s", msg)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestSocketExists tests socket existence checking.
func TestSocketExists(t *testing.T) {
	// Test with a path that doesn't exist
	if socketExists("/nonexistent/path/podman.sock") {
		t.Error("expected socketExists to return false for nonexistent path")
	}

	// Test with a path that exists but isn't a socket (use /dev/null as example)
	if socketExists("/dev/null") {
		t.Error("expected socketExists to return false for non-socket file")
	}

	// If XDG_RUNTIME_DIR has a podman socket, test it
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir != "" {
		sockPath := runtimeDir + "/podman/podman.sock"
		result := socketExists(sockPath)
		t.Logf("socketExists(%s) = %v", sockPath, result)
	}
}
