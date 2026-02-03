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
```

## Examples

See the `examples/` directory for sample Tekton YAML files.

```bash
# Run a simple task
chisel run examples/simple/hello-task.yaml

# Run a multi-step pipeline
chisel run examples/simple/hello-pipelinerun.yaml --debug
```

## Supported Features

### Phase 1 (MVP) - Current

- [x] Parse PipelineRun, Pipeline, Task YAML
- [x] Execute steps (image, script, command/args, env)
- [x] Basic parameter passing
- [x] Sequential task execution
- [x] Inline taskSpec/pipelineSpec support

### Phase 2 (Planned)

- [ ] Full parameter types (string, array, object)
- [ ] Result passing between tasks
- [ ] Complete variable substitution
- [ ] Multiple workspace types
- [ ] Parallel task execution (runAfter)
- [ ] Finally tasks

### Phase 3 (Planned)

- [ ] Sidecar support
- [ ] Conditional execution (when)
- [ ] Secret management
- [ ] Code generation mode
- [ ] Matrix builds

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
