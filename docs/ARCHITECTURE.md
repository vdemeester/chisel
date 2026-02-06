# Architecture

This document describes the internal architecture of chisel/mallet, focusing on the backend abstraction layer that enables multiple container runtimes.

## Overview

The codebase is structured as a monorepo with shared packages and two CLI entry points:

```
cmd/
  chisel/          # Dagger backend CLI
  mallet/          # Podman backend CLI

pkg/
  parser/          # YAML parsing and resolution
  types/           # Tekton data structures
  orchestrator/    # Shared pipeline logic
  backend/         # Backend abstraction
    backend.go     # Interface definition
    dagger/        # Dagger implementation
    podman/        # Podman implementation
  ui/              # Logging and output
```

## Package Responsibilities

### pkg/parser

Handles all YAML parsing and reference resolution:

- Parse PipelineRun, Pipeline, Task YAML files
- Resolve inline specs (pipelineSpec, taskSpec)
- Resolve remote references via resolvers (HTTP, Git, Hub, Bundles)
- Variable substitution for parameters
- Produce `ResolvedPipelineRun` with all references inlined

### pkg/types

Defines Tekton-compatible data structures:

- `Step`, `Task`, `Pipeline`, `PipelineRun`
- `ParamValue` (string, array, object types)
- `Workspace`, `Volume`, `VolumeMount`
- `Sidecar`, `StepTemplate`, `Matrix`
- `When` conditions

### pkg/orchestrator

Contains backend-agnostic pipeline execution logic:

| File | Purpose |
|------|---------|
| `scheduler.go` | DAG-based task scheduling with `runAfter` support |
| `matrix.go` | Matrix expansion (parameter combinations) |
| `when.go` | Conditional execution (`when` clauses) |
| `steptemplate.go` | Apply step template defaults |
| `volumes.go` | Parse volume definitions |
| `results.go` | Result collection helpers |
| `timeout.go` | Timeout parsing |
| `retry.go` | Retry logic |

### pkg/backend

The abstraction layer that allows multiple container runtimes.

### pkg/ui

Output formatting and logging:

- Pretty, plain, and JSON output modes
- Step/task/pipeline lifecycle logging
- Error formatting

## Backend Interface

The `Backend` interface (`pkg/backend/backend.go`) defines the contract between orchestration and container execution:

```go
type Backend interface {
    // Execute a single step in a container
    ExecuteStep(ctx context.Context, req *StepRequest) (*StepResult, error)

    // Start a sidecar service container
    StartSidecar(ctx context.Context, req *SidecarRequest) (*SidecarHandle, error)

    // Stop a running sidecar
    StopSidecar(ctx context.Context, handle *SidecarHandle) error

    // Read a result file from a container
    ReadResult(ctx context.Context, req *ResultRequest) (string, error)

    // Release all backend resources
    Cleanup(ctx context.Context) error
}
```

### Request/Response Types

**StepRequest** - Everything needed to run a step:
- Image, command, args, env, workDir
- Workspace mounts
- Timeout duration
- Running sidecar handles

**StepResult** - Step execution outcome:
- Exit code
- Stdout/stderr capture
- Collected results from `/tekton/results/`

**SidecarRequest/Handle** - Sidecar lifecycle management

## Dagger Backend

Location: `pkg/backend/dagger/`

Uses the Dagger SDK to execute containers via BuildKit.

### Key Concepts

- **Container**: Dagger's container type with fluent API
- **Service**: Long-running sidecar containers
- **Directory**: Workspace representation
- **CacheVolume**: Persistent caching between runs

### Execution Model

```
┌─────────────────────────────────────┐
│  DaggerBackend                      │
│  ┌─────────────────────────────────┐│
│  │  client *dagger.Client          ││
│  │  workspaces map[string]*Dir     ││
│  └─────────────────────────────────┘│
└─────────────────────────────────────┘
           │
           ▼
┌─────────────────────────────────────┐
│  Dagger Engine (BuildKit)           │
│  - Content-addressed caching        │
│  - Lazy evaluation                  │
│  - Service bindings                 │
└─────────────────────────────────────┘
```

### Sidecar Implementation

Sidecars are Dagger Services bound to step containers:

```go
service := client.Container().From(image).AsService()
container = container.WithServiceBinding(name, service)
```

## Podman Backend

Location: `pkg/backend/podman/`

Uses the Podman REST API directly (no CGO dependencies).

### Key Files

| File | Purpose |
|------|---------|
| `client.go` | Socket detection, HTTP client setup |
| `container.go` | Container lifecycle (create, start, wait, logs, remove) |
| `executor.go` | Backend interface implementation |
| `sidecar.go` | Pod and sidecar operations |
| `cleanup.go` | Resource tracking and cleanup |
| `results.go` | Result file collection |

### Execution Model

Each task runs in a Podman pod (similar to Kubernetes):

```
┌─────────────────────────────────────────────────┐
│  Podman Pod (task-name-timestamp)               │
│  ┌─────────────┐  ┌─────────────┐               │
│  │  Sidecar    │  │  Sidecar    │   (parallel)  │
│  │  container  │  │  container  │               │
│  └─────────────┘  └─────────────┘               │
│                                                 │
│  ┌─────────────┐  ┌─────────────┐               │
│  │  Step 1     │→ │  Step 2     │   (sequential)│
│  │  container  │  │  container  │               │
│  └─────────────┘  └─────────────┘               │
│                                                 │
│  Shared: network namespace, /tekton/results/    │
└─────────────────────────────────────────────────┘
```

### Sidecar Implementation

1. Create pod with host aliases (sidecar names → 127.0.0.1)
2. Start sidecar containers in the pod
3. Run step containers in the same pod
4. All containers share network namespace

```go
// Pod creation with host aliases
podman.CreatePodWithOptions(ctx, name, PodOptions{
    HostAliases: map[string]string{"redis": "127.0.0.1"},
})

// Containers join the pod
podman.CreateContainerInPod(ctx, podID, spec)
```

### API Communication

Direct HTTP to Podman socket (no external dependencies):

```go
client := &http.Client{
    Transport: &http.Transport{
        DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
            return net.Dial("unix", socketPath)
        },
    },
}
```

## Data Flow

```
┌──────────────────┐
│  YAML Files      │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  pkg/parser      │  Parse, resolve references, substitute variables
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  ResolvedPipeline│  Fully resolved pipeline with all tasks inlined
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  pkg/orchestrator│  Build DAG, expand matrix, evaluate when
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  CLI (run.go)    │  Execute tasks via backend
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  pkg/backend     │  Container execution
│  ├── dagger/     │
│  └── podman/     │
└──────────────────┘
```

## Adding a New Backend

To add a new container backend:

1. Create `pkg/backend/newbackend/` package
2. Implement the `Backend` interface
3. Create CLI entry point in `cmd/newtool/`

Required interface methods:

```go
type NewBackend struct {
    // Backend-specific state
}

func (b *NewBackend) ExecuteStep(ctx context.Context, req *StepRequest) (*StepResult, error) {
    // 1. Create container from req.Image
    // 2. Set environment, working directory
    // 3. Mount workspaces
    // 4. Execute command or script
    // 5. Capture stdout/stderr
    // 6. Collect results from /tekton/results/
    // 7. Return StepResult
}

func (b *NewBackend) StartSidecar(ctx context.Context, req *SidecarRequest) (*SidecarHandle, error) {
    // Start long-running service container
}

func (b *NewBackend) StopSidecar(ctx context.Context, handle *SidecarHandle) error {
    // Stop and remove sidecar container
}

func (b *NewBackend) ReadResult(ctx context.Context, req *ResultRequest) (string, error) {
    // Read result file from container filesystem
}

func (b *NewBackend) Cleanup(ctx context.Context) error {
    // Release all resources
}
```

## Design Decisions

### Why Two Backends?

- **Dagger**: Best for CI with aggressive caching
- **Podman**: Best for local dev, rootless, daemonless

### Why Pods in Podman?

Tekton runs each TaskRun in a Kubernetes pod. Using Podman pods:
- Matches Tekton's execution model
- Enables sidecar networking (shared localhost)
- Provides natural isolation between tasks

### Why Direct HTTP for Podman?

- No CGO dependencies (pure Go binary)
- Smaller binary size
- Full control over API communication
- Works with both user and system sockets

### Why Shared Orchestrator?

DAG scheduling, matrix expansion, when conditions, and result passing are container-runtime-agnostic. Sharing this code:
- Ensures identical behavior between backends
- Reduces maintenance burden
- Makes adding new backends easier
