# Chisel

Run Tekton PipelineRuns locally using Dagger as the execution backend.

Named to pair with "Dagger" (both builder's tools), honoring Tekton's Greek meaning of "builder/carpenter".

## Features

- Parse and execute Tekton PipelineRun, Pipeline, and Task YAML locally
- Uses Dagger for container execution (fast, cached, reproducible)
- Supports parameters, workspaces, and basic variable substitution
- Identical behavior to cluster execution

## Installation

```bash
go install github.com/vdemeester/chisel/cmd/chisel@latest
```

Or build from source:

```bash
git clone https://github.com/vdemeester/chisel
cd chisel
go build -o chisel ./cmd/chisel
```

## Prerequisites

- [Dagger](https://docs.dagger.io/install) (v0.19+)
- Docker or compatible container runtime

## Usage

```bash
# Run a PipelineRun
chisel run pipelinerun.yaml

# Run a Task directly
chisel run task.yaml

# Run a Pipeline (uses default params)
chisel run pipeline.yaml

# Specify task definitions directory
chisel run pipelinerun.yaml --tasks=./tasks/

# Debug mode (verbose output)
chisel run pipelinerun.yaml --debug

# Dry run (parse only, no execution)
chisel run pipelinerun.yaml --dry-run

# Output format (pretty, plain, json)
chisel run pipelinerun.yaml --output=json

# Inject local directory as workspace
chisel run build-task.yaml --workspace=source:.

# Multiple workspace overrides
chisel run pipeline.yaml -w source:. -w config:./config
```

## Examples

See the `examples/` directory for sample Tekton YAML files.

```bash
# Run a simple task
chisel run examples/simple/hello-task.yaml

# Run a multi-step pipeline
chisel run examples/simple/hello-pipelinerun.yaml

# Parallel task execution (lint, test-unit, test-integration run concurrently)
chisel run examples/simple/parallel-pipelinerun.yaml

# Result passing between tasks
chisel run examples/simple/results-pipelinerun.yaml

# Volume sharing between steps
chisel run examples/simple/volumes-pipelinerun.yaml

# Array and object parameter types
chisel run examples/simple/params-pipelinerun.yaml

# Workspace override for local development
# First, run with clone (uses emptyDir workspace):
chisel run examples/simple/workspace-override-pipelinerun.yaml
# Then, run with local source (skips clone, uses your directory):
chisel run examples/simple/workspace-override-pipelinerun.yaml --workspace=source:.

# Pipeline that demonstrates failure handling
chisel run examples/simple/failing-pipelinerun.yaml

# Step timeout (steps with time limits)
chisel run examples/simple/timeout-pipelinerun.yaml

# Step retry (automatic retry on failure)
chisel run examples/simple/retry-pipelinerun.yaml

# Sidecar containers (e.g., Redis alongside steps)
chisel run examples/simple/sidecar-pipelinerun.yaml

# Conditional execution (when clauses)
chisel run examples/simple/when-pipelinerun.yaml

# Step template defaults (shared image, env, workingDir)
chisel run examples/simple/steptemplate-pipelinerun.yaml

# Matrix builds (run task with multiple parameter combinations)
chisel run examples/simple/matrix-pipelinerun.yaml

# Remote task loading via HTTP resolver
chisel run examples/resolvers/http-resolver-pipelinerun.yaml
```

## Supported Features

### Implemented

- [x] Parse PipelineRun, Pipeline, Task YAML
- [x] Execute steps (image, script, command/args, env, workingDir)
- [x] Parameter passing (string, array, object types)
- [x] Inline taskSpec/pipelineSpec support
- [x] Parallel task execution (DAG-based, respects runAfter)
- [x] Result capture and passing between tasks (via /tekton/results/)
- [x] Volumes (emptyDir, configMap, secret) with step volumeMounts
- [x] Workspaces (emptyDir, local directory, PVC)
- [x] Finally tasks
- [x] Structured logging (pretty, plain, JSON output modes)
- [x] Step stdout/stderr capture and display
- [x] Variable substitution: `$(params.*)`, `$(params.array[*])`, `$(params.array[N])`, `$(params.object.field)`, `$(workspaces.*.path)`, `$(tasks.*.results.*)`, `$(context.*)`
- [x] CLI workspace override (`--workspace/-w` flag for local development)
- [x] Sidecar execution (run auxiliary containers alongside steps)
- [x] Step timeout (time limits with Go duration format)
- [x] Step retry (automatic retry on failure)
- [x] Conditional execution (`when` clauses with `in`/`notin` operators)
- [x] stepTemplate defaults (image, env, workingDir, command, volumeMounts)
- [x] Matrix builds (run tasks with multiple parameter combinations)
- [x] HTTP resolver (fetch tasks from HTTP/HTTPS URLs)

## Architecture

```
┌─────────────────────────────────────┐
│  chisel CLI                         │
├─────────────────────────────────────┤
│  Parser → Load & validate YAML      │
│  Executor → Run via Dagger SDK      │
└─────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────┐
│  Dagger Engine (BuildKit)           │
└─────────────────────────────────────┘
```

## Tekton → Dagger Mapping

| Tekton     | Dagger                  |
|------------|-------------------------|
| Task Steps | Container.WithExec()    |
| Workspace  | Directory type          |
| Parameter  | Function argument       |
| Result     | Return value / file     |
| Sidecar    | Service type            |
| runAfter   | DAG dependencies (auto) |

## Related Projects

- [buildkit-tekton](https://github.com/vdemeester/buildkit-tekton) - BuildKit frontend for Tekton (by the same author)
- [Dagger](https://dagger.io/) - Programmable CI/CD engine
- [Tekton](https://tekton.dev/) - Kubernetes-native CI/CD

## License

Apache 2.0
