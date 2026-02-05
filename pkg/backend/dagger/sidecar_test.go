package dagger

import (
	"testing"

	"github.com/vdemeester/chisel/pkg/types"
)

func TestStartSidecars_Empty(t *testing.T) {
	// No sidecars - should return nil services and no error
	task := &types.ResolvedTask{
		Name:     "test-task",
		Sidecars: nil,
	}

	services, err := startSidecars(nil, task)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(services) != 0 {
		t.Errorf("expected 0 services, got %d", len(services))
	}
}

func TestStartSidecars_SingleSidecar(t *testing.T) {
	task := &types.ResolvedTask{
		Name: "test-task",
		Sidecars: []types.Sidecar{
			{
				Name:  "redis",
				Image: "redis:7",
				Ports: []int{6379},
			},
		},
	}

	// This test verifies the sidecar config is correctly processed
	// Full integration test with Dagger would require a running engine
	services, err := startSidecars(nil, task)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(services) != 1 {
		t.Errorf("expected 1 service, got %d", len(services))
	}
}

func TestStartSidecars_MultipleSidecars(t *testing.T) {
	task := &types.ResolvedTask{
		Name: "test-task",
		Sidecars: []types.Sidecar{
			{
				Name:  "redis",
				Image: "redis:7",
				Ports: []int{6379},
			},
			{
				Name:  "postgres",
				Image: "postgres:16",
				Ports: []int{5432},
				Env:   map[string]string{"POSTGRES_PASSWORD": "test"},
			},
		},
	}

	services, err := startSidecars(nil, task)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(services) != 2 {
		t.Errorf("expected 2 services, got %d", len(services))
	}
}

func TestSidecarConfig_WithCommand(t *testing.T) {
	sidecar := types.Sidecar{
		Name:    "custom",
		Image:   "alpine:latest",
		Command: []string{"/bin/sh"},
		Args:    []string{"-c", "while true; do sleep 1; done"},
		Env:     map[string]string{"DEBUG": "true"},
		Ports:   []int{8080},
	}

	config := buildSidecarConfig(sidecar)

	if config.Name != "custom" {
		t.Errorf("Name = %q, want %q", config.Name, "custom")
	}
	if config.Image != "alpine:latest" {
		t.Errorf("Image = %q, want %q", config.Image, "alpine:latest")
	}
	if len(config.Command) != 1 || config.Command[0] != "/bin/sh" {
		t.Errorf("Command = %v, want [/bin/sh]", config.Command)
	}
	if len(config.Args) != 2 {
		t.Errorf("Args = %v, want [-c, while true; do sleep 1; done]", config.Args)
	}
	if config.Env["DEBUG"] != "true" {
		t.Errorf("Env[DEBUG] = %q, want %q", config.Env["DEBUG"], "true")
	}
	if len(config.Ports) != 1 || config.Ports[0] != 8080 {
		t.Errorf("Ports = %v, want [8080]", config.Ports)
	}
}

func TestSidecarConfig_NoCommand(t *testing.T) {
	// When no command is specified, the image's default entrypoint is used
	sidecar := types.Sidecar{
		Name:  "redis",
		Image: "redis:7",
		Ports: []int{6379},
	}

	config := buildSidecarConfig(sidecar)

	if config.Name != "redis" {
		t.Errorf("Name = %q, want %q", config.Name, "redis")
	}
	if len(config.Command) != 0 {
		t.Errorf("Command = %v, want empty", config.Command)
	}
	if len(config.Args) != 0 {
		t.Errorf("Args = %v, want empty", config.Args)
	}
}

func TestBindSidecarsToContainer_Empty(t *testing.T) {
	// No services - container should be returned unchanged
	var services []SidecarService
	result := bindSidecarsToContainer(nil, services)
	if result != nil {
		t.Errorf("expected nil container, got %v", result)
	}
}

func TestBindSidecarsToContainer_NilContainer(t *testing.T) {
	// With nil container and services, should return nil
	services := []SidecarService{
		{Name: "redis", Ports: []int{6379}},
	}
	// This would panic with a real container - test documents behavior
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic with nil container and services")
		}
	}()
	_ = bindSidecarsToContainer(nil, services)
}
