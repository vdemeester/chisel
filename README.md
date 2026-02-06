# Chisel & Mallet

Run Tekton PipelineRuns locally with your choice of container backend.

Named to pair with "Dagger" (both builder's tools), honoring Tekton's Greek meaning of "builder/carpenter".

## Two Tools, One Goal

| Tool | Backend | Best For |
|------|---------|----------|
| **Chisel** | Dagger | Fast iteration with caching, CI environments |
| **Mallet** | Podman | Local development, rootless containers, no daemon |

Both tools share the same parser and orchestration logic - only the container execution differs.

## Features

- Parse and execute Tekton PipelineRun, Pipeline, and Task YAML locally
- Parallel task execution with DAG-based scheduling
- Full parameter support (string, array, object types)
- Workspaces, volumes, sidecars, and result passing
- Identical behavior to cluster execution

## Installation

### From Source

```bash
git clone https://github.com/vdemeester/chisel
cd chisel
make build          # Build both tools
make install        # Install to $GOPATH/bin
```

### Individual Tools

```bash
# Chisel only
go install github.com/vdemeester/chisel/cmd/chisel@latest

# Mallet only
go install github.com/vdemeester/chisel/cmd/mallet@latest
```

## Prerequisites

### Chisel (Dagger Backend)
- [Dagger](https://docs.dagger.io/install) (v0.19+)
- Docker or compatible container runtime

### Mallet (Podman Backend)
- [Podman](https://podman.io/getting-started/installation) (v4.0+)
- Podman socket enabled: `systemctl --user start podman.socket`

## Usage

Both tools use the same CLI interface:

```bash
# Run a PipelineRun
chisel run pipelinerun.yaml
mallet run pipelinerun.yaml

# Run a Task directly
chisel run task.yaml

# Specify task definitions directory
chisel run pipelinerun.yaml --tasks=./tasks/

# Debug mode (verbose output)
mallet run pipelinerun.yaml --debug

# Dry run (parse only, no execution)
chisel run pipelinerun.yaml --dry-run

# Output format (pretty, plain, json)
mallet run pipelinerun.yaml --output=json

# Inject local directory as workspace
chisel run build-task.yaml --workspace=source:.

# Multiple workspace overrides
mallet run pipeline.yaml -w source:. -w config:./config
```

## When to Use Which

### Use Chisel (Dagger) when:
- You want aggressive caching between runs
- Running in CI/CD environments
- You need Dagger's content-addressed caching
- Docker is already running

### Use Mallet (Podman) when:
- You prefer rootless containers
- You don't want a daemon running
- You're on a system without Docker
- You want pure local execution

## Examples

See the `examples/` directory for sample Tekton YAML files.

```bash
# Simple pipeline
chisel run examples/simple/hello-pipelinerun.yaml
mallet run examples/simple/hello-pipelinerun.yaml

# Parallel task execution
chisel run examples/simple/parallel-pipelinerun.yaml

# Result passing between tasks
mallet run examples/simple/results-pipelinerun.yaml

# Volume sharing between steps
chisel run examples/simple/volumes-pipelinerun.yaml

# Sidecar containers (e.g., Redis alongside steps)
mallet run examples/simple/sidecar-pipelinerun.yaml

# Matrix builds (multiple parameter combinations)
chisel run examples/simple/matrix-pipelinerun.yaml

# Conditional execution (when clauses)
mallet run examples/simple/when-pipelinerun.yaml

# Remote task loading via resolvers
chisel run examples/resolvers/git-resolver-pipelinerun.yaml
chisel run examples/resolvers/hub-resolver-pipelinerun.yaml
```

## Supported Features

- [x] Parse PipelineRun, Pipeline, Task YAML
- [x] Execute steps (image, script, command/args, env, workingDir)
- [x] Parameter passing (string, array, object types)
- [x] Inline taskSpec/pipelineSpec support
- [x] Parallel task execution (DAG-based, respects runAfter)
- [x] Result capture and passing between tasks
- [x] Volumes (emptyDir, configMap, secret) with step volumeMounts
- [x] Workspaces (emptyDir, local directory, PVC)
- [x] Finally tasks
- [x] Sidecar execution (auxiliary containers alongside steps)
- [x] Step timeout and retry
- [x] Conditional execution (`when` clauses)
- [x] stepTemplate defaults
- [x] Matrix builds
- [x] Remote resolvers (HTTP, Git, Hub, Bundles)

## Architecture

```
┌──────────────────────────────────────────────────────┐
│  CLI (chisel / mallet)                               │
├──────────────────────────────────────────────────────┤
│  pkg/parser      → Load & validate YAML              │
│  pkg/types       → Tekton data structures            │
│  pkg/orchestrator→ DAG scheduling, matrix, when      │
├──────────────────────────────────────────────────────┤
│  pkg/backend     → Backend interface                 │
│    ├── dagger/   → Chisel: Dagger SDK execution      │
│    └── podman/   → Mallet: Podman API execution      │
└──────────────────────────────────────────────────────┘
         │                           │
         ▼                           ▼
┌─────────────────┐       ┌─────────────────┐
│  Dagger Engine  │       │  Podman Service │
│  (BuildKit)     │       │  (libpod)       │
└─────────────────┘       └─────────────────┘
```

## Tekton Concept Mapping

| Tekton | Chisel (Dagger) | Mallet (Podman) |
|--------|-----------------|-----------------|
| Task | Container.WithExec() | Pod with containers |
| Step | Sequential exec | Sequential containers |
| Workspace | Directory type | Bind mount |
| Parameter | Function argument | Environment/script |
| Result | Return value / file | /tekton/results/ mount |
| Sidecar | Service binding | Container in pod |
| runAfter | DAG dependencies | DAG dependencies |

## Related Projects

- [buildkit-tekton](https://github.com/vdemeester/buildkit-tekton) - BuildKit frontend for Tekton
- [Dagger](https://dagger.io/) - Programmable CI/CD engine
- [Podman](https://podman.io/) - Daemonless container engine
- [Tekton](https://tekton.dev/) - Kubernetes-native CI/CD

## License

Apache 2.0
