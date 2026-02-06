# Mallet

Mallet runs Tekton PipelineRuns locally using Podman as the container backend.

## Overview

Mallet is part of the [chisel](../../README.md) project. It provides an alternative to chisel for users who prefer Podman over Dagger/Docker.

**Key differences from Chisel:**
- Uses Podman instead of Dagger
- No daemon required (Podman is daemonless)
- Rootless container support
- Each task runs in a Podman pod (like Tekton on Kubernetes)

## Prerequisites

### Podman Installation

**Fedora/RHEL/CentOS:**
```bash
sudo dnf install podman
```

**Ubuntu/Debian:**
```bash
sudo apt install podman
```

**macOS (via Homebrew):**
```bash
brew install podman
podman machine init
podman machine start
```

### Enable Podman Socket

Mallet communicates with Podman via its REST API socket:

```bash
# Enable and start the user socket
systemctl --user enable podman.socket
systemctl --user start podman.socket

# Verify it's running
podman info
```

The socket is typically at:
- User: `$XDG_RUNTIME_DIR/podman/podman.sock` (e.g., `/run/user/1000/podman/podman.sock`)
- System: `/run/podman/podman.sock`

## Usage

```bash
# Run a PipelineRun
mallet run pipelinerun.yaml

# Run a Task directly
mallet run task.yaml

# With debug output
mallet run pipelinerun.yaml --debug

# Dry run (parse only)
mallet run pipelinerun.yaml --dry-run

# Override workspace with local directory
mallet run build-pipeline.yaml --workspace=source:.
```

## How It Works

Mallet executes Tekton tasks using Podman pods:

1. **Pod Creation**: Each task creates a Podman pod with shared network namespace
2. **Sidecars**: Started as containers in the pod before steps run
3. **Steps**: Run sequentially as containers in the same pod
4. **Networking**: All containers share localhost; sidecars accessible by hostname
5. **Results**: Captured via mounted `/tekton/results/` directory
6. **Cleanup**: Pod and all containers removed after task completion

```
┌─────────────────────────────────────────────────┐
│  Podman Pod (task-name)                         │
│  ┌─────────────┐  ┌─────────────┐               │
│  │  Sidecar    │  │  Sidecar    │               │
│  │  (redis)    │  │  (postgres) │   Running     │
│  └─────────────┘  └─────────────┘               │
│                                                 │
│  ┌─────────────┐  ┌─────────────┐               │
│  │  Step 1     │  │  Step 2     │   Sequential  │
│  │  (clone)    │→ │  (build)    │→  ...         │
│  └─────────────┘  └─────────────┘               │
│                                                 │
│  Shared: network namespace, /tekton/results/    │
└─────────────────────────────────────────────────┘
```

## Sidecar Support

Sidecars are fully supported. They run alongside steps and are accessible by hostname:

```yaml
taskSpec:
  sidecars:
    - name: redis
      image: redis:7-alpine
  steps:
    - name: test
      image: redis:7-alpine
      script: |
        redis-cli -h redis ping  # Works! Resolves to 127.0.0.1
```

Mallet configures host aliases so sidecars can be reached by name (e.g., `redis:6379`).

## Troubleshooting

### "podman socket not found"

The Podman socket isn't running:
```bash
systemctl --user start podman.socket
```

Or check if it exists:
```bash
ls -la $XDG_RUNTIME_DIR/podman/podman.sock
```

### "permission denied" on socket

Ensure you're running as the same user who owns the socket:
```bash
ls -la /run/user/$(id -u)/podman/podman.sock
```

### Container name conflicts

If you see "name already in use" errors, clean up leftover containers:
```bash
podman container prune -f
podman pod prune -f
```

### Slow image pulls

Mallet pulls images on demand. Pre-pull frequently used images:
```bash
podman pull alpine:latest
podman pull golang:1.23
```

### Debug mode

Use `--debug` to see detailed execution information:
```bash
mallet run pipeline.yaml --debug
```

## Differences from Chisel

| Feature | Mallet (Podman) | Chisel (Dagger) |
|---------|-----------------|-----------------|
| Caching | None (fresh each run) | Content-addressed cache |
| Daemon | Daemonless | Requires Docker daemon |
| Rootless | Yes | Depends on Docker config |
| Pod model | Native Podman pods | Dagger containers |
| Binary size | ~15MB | ~29MB |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `CONTAINER_HOST` | Override Podman socket path |
| `XDG_RUNTIME_DIR` | Used to find user socket |

## See Also

- [Main README](../../README.md) - Overview of chisel/mallet
- [Examples](../../examples/) - Sample Tekton YAML files
- [Podman Documentation](https://docs.podman.io/)
