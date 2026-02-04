package executor

import (
	"dagger.io/dagger"

	"github.com/vdemeester/chisel/pkg/types"
)

// SidecarConfig holds the processed configuration for a sidecar container.
type SidecarConfig struct {
	Name    string
	Image   string
	Command []string
	Args    []string
	Env     map[string]string
	Ports   []int
}

// SidecarService represents a running sidecar service.
type SidecarService struct {
	Name    string
	Service *dagger.Service
	Ports   []int
}

// startSidecars starts all sidecar containers for a task and returns their services.
// The returned services can be bound to step containers using WithServiceBinding.
func startSidecars(client *dagger.Client, task *types.ResolvedTask) ([]SidecarService, error) {
	if len(task.Sidecars) == 0 {
		return nil, nil
	}

	// For unit tests without a real Dagger client, return configs without services
	if client == nil {
		services := make([]SidecarService, len(task.Sidecars))
		for i, sidecar := range task.Sidecars {
			config := buildSidecarConfig(sidecar)
			services[i] = SidecarService{
				Name:  config.Name,
				Ports: config.Ports,
			}
		}
		return services, nil
	}

	services := make([]SidecarService, 0, len(task.Sidecars))

	for _, sidecar := range task.Sidecars {
		config := buildSidecarConfig(sidecar)

		// Create container from image
		container := client.Container().From(config.Image)

		// Set environment variables
		for name, value := range config.Env {
			container = container.WithEnvVariable(name, value)
		}

		// Set command and args if specified
		if len(config.Command) > 0 || len(config.Args) > 0 {
			cmd := append(config.Command, config.Args...)
			container = container.WithExec(cmd)
		}

		// Expose ports
		for _, port := range config.Ports {
			container = container.WithExposedPort(port)
		}

		// Create service from container
		service := container.AsService()

		services = append(services, SidecarService{
			Name:    config.Name,
			Service: service,
			Ports:   config.Ports,
		})
	}

	return services, nil
}

// buildSidecarConfig processes a Sidecar into a SidecarConfig.
func buildSidecarConfig(sidecar types.Sidecar) SidecarConfig {
	env := make(map[string]string)
	for k, v := range sidecar.Env {
		env[k] = v
	}

	ports := make([]int, len(sidecar.Ports))
	copy(ports, sidecar.Ports)

	return SidecarConfig{
		Name:    sidecar.Name,
		Image:   sidecar.Image,
		Command: sidecar.Command,
		Args:    sidecar.Args,
		Env:     env,
		Ports:   ports,
	}
}

// bindSidecarsToContainer binds sidecar services to a container.
// Each sidecar is accessible by its name as the hostname.
func bindSidecarsToContainer(container *dagger.Container, services []SidecarService) *dagger.Container {
	for _, svc := range services {
		container = container.WithServiceBinding(svc.Name, svc.Service)
	}
	return container
}
