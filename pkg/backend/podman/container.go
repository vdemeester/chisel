package podman

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// ContainerSpec describes a container to create and run.
type ContainerSpec struct {
	// Image is the container image to run
	Image string

	// Command is the command to execute
	Command []string

	// Env contains environment variables
	Env map[string]string

	// WorkDir is the working directory inside the container
	WorkDir string

	// Mounts are bind mounts for the container
	Mounts []Mount

	// Timeout is the maximum execution time
	Timeout time.Duration

	// Name is an optional container name
	Name string
}

// Mount describes a bind mount.
type Mount struct {
	Source   string
	Target   string
	ReadOnly bool
}

// ContainerResult contains the result of running a container.
type ContainerResult struct {
	ContainerID string
	ExitCode    int
	Stdout      string
	Stderr      string
}

// Validate checks if the container spec is valid.
func (s *ContainerSpec) Validate() error {
	if s.Image == "" {
		return errors.New("image is required")
	}
	if len(s.Command) == 0 {
		return errors.New("command is required")
	}
	return nil
}

// httpClient returns an HTTP client that connects to the Unix socket.
func httpClient(socketPath string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
		Timeout: 0, // No timeout, we handle it ourselves
	}
}

// createContainerRequest is the JSON body for container creation.
type createContainerRequest struct {
	Image      string            `json:"image"`
	Command    []string          `json:"command,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
	WorkDir    string            `json:"work_dir,omitempty"`
	Name       string            `json:"name,omitempty"`
	Mounts     []mountSpec       `json:"mounts,omitempty"`
	StopSignal string            `json:"stop_signal,omitempty"`
}

type mountSpec struct {
	Type        string   `json:"Type"`
	Source      string   `json:"Source"`
	Destination string   `json:"Destination"`
	Options     []string `json:"Options,omitempty"`
}

type createResponse struct {
	Id       string   `json:"Id"`
	Warnings []string `json:"Warnings"`
}

type waitResponse struct {
	StatusCode int    `json:"StatusCode"`
	Error      string `json:"Error,omitempty"`
}

// PullImage pulls a container image if not present locally.
func PullImage(ctx context.Context, image string) error {
	socketPath := detectSocketPath()
	if socketPath == "" {
		return ErrNoSocketFound
	}

	client := httpClient(socketPath)

	// Check if image exists first
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://d/v5.0.0/libpod/images/%s/exists", image), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to check image: %w", err)
	}
	resp.Body.Close()

	// Image exists
	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	// Pull the image
	req, err = http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("http://d/v5.0.0/libpod/images/pull?reference=%s", image), nil)
	if err != nil {
		return fmt.Errorf("failed to create pull request: %w", err)
	}

	resp, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to pull image %s: %s (status %d)", image, string(respBody), resp.StatusCode)
	}

	// Read and discard the response body (streaming pull progress)
	_, _ = io.Copy(io.Discard, resp.Body)

	return nil
}

// CreateContainer creates a new container without starting it.
func CreateContainer(ctx context.Context, spec ContainerSpec) (string, error) {
	if err := spec.Validate(); err != nil {
		return "", fmt.Errorf("invalid container spec: %w", err)
	}

	// Get socket path from context or use default
	socketPath := detectSocketPath()
	if socketPath == "" {
		return "", ErrNoSocketFound
	}

	// Pull image if needed
	if err := PullImage(ctx, spec.Image); err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}

	client := httpClient(socketPath)

	// Build the request body
	reqBody := createContainerRequest{
		Image:   spec.Image,
		Command: spec.Command,
		Env:     spec.Env,
		WorkDir: spec.WorkDir,
		Name:    spec.Name,
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

	req, err := http.NewRequestWithContext(ctx, "POST", "http://d/v5.0.0/libpod/containers/create", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create container: %s (status %d)", string(respBody), resp.StatusCode)
	}

	var result createResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Id, nil
}

// StartContainer starts a created container.
func StartContainer(ctx context.Context, id string) error {
	socketPath := detectSocketPath()
	if socketPath == "" {
		return ErrNoSocketFound
	}

	client := httpClient(socketPath)

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("http://d/v5.0.0/libpod/containers/%s/start", id), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to start container: %s (status %d)", string(respBody), resp.StatusCode)
	}

	return nil
}

// WaitContainer waits for a container to exit and returns the exit code.
func WaitContainer(ctx context.Context, id string) (int, error) {
	socketPath := detectSocketPath()
	if socketPath == "" {
		return -1, ErrNoSocketFound
	}

	client := httpClient(socketPath)

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("http://d/v5.0.0/libpod/containers/%s/wait", id), nil)
	if err != nil {
		return -1, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return -1, fmt.Errorf("container execution timeout")
		}
		return -1, fmt.Errorf("failed to wait for container: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return -1, fmt.Errorf("failed to wait for container: %s (status %d)", string(respBody), resp.StatusCode)
	}

	// The wait endpoint returns just the exit code as an integer
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return -1, fmt.Errorf("failed to read response: %w", err)
	}

	var exitCode int
	if err := json.Unmarshal(body, &exitCode); err != nil {
		// Try parsing as waitResponse object
		var result waitResponse
		if err2 := json.Unmarshal(body, &result); err2 != nil {
			return -1, fmt.Errorf("failed to decode response: %w (body: %s)", err, string(body))
		}
		return result.StatusCode, nil
	}

	return exitCode, nil
}

// GetContainerLogs retrieves stdout and stderr from a container.
func GetContainerLogs(ctx context.Context, id string) (stdout, stderr string, err error) {
	socketPath := detectSocketPath()
	if socketPath == "" {
		return "", "", ErrNoSocketFound
	}

	client := httpClient(socketPath)

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://d/v5.0.0/libpod/containers/%s/logs?stdout=true&stderr=true", id), nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to get container logs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("failed to get container logs: %s (status %d)", string(respBody), resp.StatusCode)
	}

	// Parse the log stream - Podman uses a multiplexed format
	var stdoutBuf, stderrBuf bytes.Buffer
	buf := make([]byte, 8192)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			// The log stream has an 8-byte header for each frame
			// Byte 0: stream type (1=stdout, 2=stderr)
			// Bytes 4-7: frame size (big endian)
			data := buf[:n]
			for len(data) > 8 {
				streamType := data[0]
				frameSize := int(data[4])<<24 | int(data[5])<<16 | int(data[6])<<8 | int(data[7])
				data = data[8:]
				if frameSize > len(data) {
					frameSize = len(data)
				}
				if streamType == 1 {
					stdoutBuf.Write(data[:frameSize])
				} else if streamType == 2 {
					stderrBuf.Write(data[:frameSize])
				}
				data = data[frameSize:]
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", "", fmt.Errorf("failed to read logs: %w", err)
		}
	}

	return stdoutBuf.String(), stderrBuf.String(), nil
}

// RemoveContainer removes a container.
func RemoveContainer(ctx context.Context, id string, force bool) error {
	socketPath := detectSocketPath()
	if socketPath == "" {
		return ErrNoSocketFound
	}

	client := httpClient(socketPath)

	url := fmt.Sprintf("http://d/v5.0.0/libpod/containers/%s?force=%t", id, force)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to remove container: %s (status %d)", string(respBody), resp.StatusCode)
	}

	return nil
}

// RunContainer creates, starts, waits for, and captures output from a container.
// This is a convenience function for simple use cases.
func RunContainer(ctx context.Context, spec ContainerSpec) (*ContainerResult, error) {
	if err := spec.Validate(); err != nil {
		return nil, fmt.Errorf("invalid container spec: %w", err)
	}

	// Apply timeout if specified
	if spec.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, spec.Timeout)
		defer cancel()
	}

	// Create the container
	id, err := CreateContainer(ctx, spec)
	if err != nil {
		return nil, err
	}

	// Always try to remove the container when done
	defer func() {
		// Use a fresh context for cleanup since the original may be canceled
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = RemoveContainer(cleanupCtx, id, true)
	}()

	// Start the container
	if err := StartContainer(ctx, id); err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// Wait for it to complete
	exitCode, err := WaitContainer(ctx, id)
	if err != nil {
		// Check if it was a timeout
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("container execution timeout after %v", spec.Timeout)
		}
		return nil, err
	}

	// Get the logs
	stdout, stderr, err := GetContainerLogs(ctx, id)
	if err != nil {
		return nil, err
	}

	return &ContainerResult{
		ContainerID: id,
		ExitCode:    exitCode,
		Stdout:      stdout,
		Stderr:      stderr,
	}, nil
}

// StopContainer stops a running container.
func StopContainer(ctx context.Context, id string, timeout *uint) error {
	socketPath := detectSocketPath()
	if socketPath == "" {
		return ErrNoSocketFound
	}

	client := httpClient(socketPath)

	url := "http://d/v5.0.0/libpod/containers/" + id + "/stop"
	if timeout != nil {
		url = fmt.Sprintf("%s?timeout=%d", url, *timeout)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotModified {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to stop container: %s (status %d)", string(respBody), resp.StatusCode)
	}

	return nil
}
