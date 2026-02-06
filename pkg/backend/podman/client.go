package podman

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/containers/podman/v5/pkg/bindings"
)

// ErrPodmanNotRunning is returned when the Podman service is not available.
var ErrPodmanNotRunning = errors.New("podman service not running: ensure 'podman system service' is active or start it with 'systemctl --user start podman.socket'")

// ErrNoSocketFound is returned when no Podman socket can be found.
var ErrNoSocketFound = errors.New("no podman socket found: check XDG_RUNTIME_DIR/podman/podman.sock or /run/podman/podman.sock")

// Client manages the connection to the Podman service.
type Client struct {
	socketPath string
	ctx        context.Context
}

// NewClient creates a new Podman client using auto-detected socket path.
func NewClient() (*Client, error) {
	socketPath := detectSocketPath()
	if socketPath == "" {
		return nil, ErrNoSocketFound
	}
	return NewClientWithSocket(socketPath)
}

// NewClientWithSocket creates a new Podman client with a specific socket path.
func NewClientWithSocket(socketPath string) (*Client, error) {
	if !socketExists(socketPath) {
		return nil, fmt.Errorf("podman socket not found at %s: %w", socketPath, ErrPodmanNotRunning)
	}

	// Create a context connected to the Podman service
	ctx, err := bindings.NewConnection(context.Background(), "unix://"+socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to podman at %s: %w", socketPath, err)
	}

	return &Client{
		socketPath: socketPath,
		ctx:        ctx,
	}, nil
}

// Close closes the client connection.
func (c *Client) Close() error {
	// The bindings context doesn't require explicit cleanup
	return nil
}

// Context returns the Podman bindings context for API calls.
func (c *Client) Context() context.Context {
	return c.ctx
}

// SocketPath returns the path to the Podman socket.
func (c *Client) SocketPath() string {
	return c.socketPath
}

// IsConnected checks if the Podman service is responding.
func (c *Client) IsConnected(ctx context.Context) bool {
	// Try to dial the socket with a short timeout
	conn, err := net.DialTimeout("unix", c.socketPath, 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// detectSocketPath finds the Podman socket, preferring user socket over system.
func detectSocketPath() string {
	// First try user socket (rootless Podman)
	userSock := userSocketPath()
	if socketExists(userSock) {
		return userSock
	}

	// Fall back to system socket (root Podman)
	sysSock := systemSocketPath()
	if socketExists(sysSock) {
		return sysSock
	}

	return ""
}

// userSocketPath returns the path to the user's Podman socket.
func userSocketPath() string {
	// Check XDG_RUNTIME_DIR first
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		// Fall back to /run/user/<uid>
		runtimeDir = fmt.Sprintf("/run/user/%d", os.Getuid())
	}
	return filepath.Join(runtimeDir, "podman", "podman.sock")
}

// systemSocketPath returns the path to the system Podman socket.
func systemSocketPath() string {
	return "/run/podman/podman.sock"
}

// socketExists checks if a Unix socket exists at the given path.
func socketExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	// Check if it's a socket
	mode := info.Mode()
	return mode&os.ModeSocket != 0 || mode&syscall.S_IFSOCK != 0
}
