# Examples

This directory contains example Tekton PipelineRun YAML files for testing chisel and mallet.

## Running Examples

All examples work with both `chisel` (Dagger backend) and `mallet` (Podman backend):

```bash
# With chisel
chisel run examples/simple/hello-pipelinerun.yaml

# With mallet
mallet run examples/simple/hello-pipelinerun.yaml
```

## Simple Examples

Located in `examples/simple/`:

| Example | Description |
|---------|-------------|
| `hello-pipelinerun.yaml` | Basic 3-step pipeline with sequential tasks |
| `hello-pipeline.yaml` | Standalone Pipeline definition |
| `hello-task.yaml` | Standalone Task definition |
| `params-pipelinerun.yaml` | Parameter types: string, array, object |
| `results-pipelinerun.yaml` | Result passing between tasks |
| `parallel-pipelinerun.yaml` | Parallel task execution with DAG |
| `matrix-pipelinerun.yaml` | Matrix builds with multiple parameter combinations |
| `when-pipelinerun.yaml` | Conditional task execution |
| `sidecar-pipelinerun.yaml` | Sidecar containers (Redis example) |
| `timeout-pipelinerun.yaml` | Step timeout handling |
| `retry-pipelinerun.yaml` | Step retry on failure |
| `volumes-pipelinerun.yaml` | Volume sharing between steps |
| `steptemplate-pipelinerun.yaml` | Step template defaults |
| `verbose-pipelinerun.yaml` | Detailed output pipeline |
| `failing-pipelinerun.yaml` | Pipeline that demonstrates failure |
| `workspace-override-pipelinerun.yaml` | Workspace override example |

### Quick Start

```bash
# Most basic example
chisel run examples/simple/hello-pipelinerun.yaml

# Parameters (string, array, object)
mallet run examples/simple/params-pipelinerun.yaml

# Parallel execution
chisel run examples/simple/parallel-pipelinerun.yaml

# Sidecar with Redis
mallet run examples/simple/sidecar-pipelinerun.yaml
```

## Resolver Examples

Located in `examples/resolvers/`:

| Example | Description |
|---------|-------------|
| `http-resolver-pipelinerun.yaml` | Fetch task from HTTP URL |
| `git-resolver-pipelinerun.yaml` | Clone and load task from Git repo |
| `hub-resolver-pipelinerun.yaml` | Fetch task from Artifact Hub |
| `bundles-resolver-pipelinerun.yaml` | Pull task from OCI registry |

### Running Resolver Examples

```bash
# HTTP resolver (fetches from URL)
chisel run examples/resolvers/http-resolver-pipelinerun.yaml

# Git resolver (clones from GitHub)
chisel run examples/resolvers/git-resolver-pipelinerun.yaml

# Hub resolver (Artifact Hub catalog)
chisel run examples/resolvers/hub-resolver-pipelinerun.yaml

# Bundles resolver (OCI artifacts)
chisel run examples/resolvers/bundles-resolver-pipelinerun.yaml
```

## Example Details

### hello-pipelinerun.yaml

Simple 3-step sequential pipeline:
```
step-one → step-two → step-three
```

### parallel-pipelinerun.yaml

Demonstrates DAG-based parallel execution:
```
        ┌─→ lint ────────┐
        │                │
start ──┼─→ test-unit ───┼──→ build → deploy
        │                │
        └─→ test-integ ──┘
```

### results-pipelinerun.yaml

Shows result passing between tasks:
- Task `generate` produces results
- Task `consume` uses `$(tasks.generate.results.*)` to read them

### sidecar-pipelinerun.yaml

Redis sidecar example:
- Starts Redis as a sidecar
- Step connects to Redis by hostname (`redis:6379`)
- Tests SET and GET operations

### matrix-pipelinerun.yaml

Matrix expansion example:
- Runs task with multiple parameter combinations
- E.g., `[compiler: gcc, clang] x [optimization: -O0, -O2]`

### workspace-override-pipelinerun.yaml

Demonstrates the `--workspace` flag:
```bash
# Normal run (clones repo)
mallet run examples/simple/workspace-override-pipelinerun.yaml

# Override with local directory (skips clone)
mallet run examples/simple/workspace-override-pipelinerun.yaml --workspace=source:.
```

## Writing Your Own

Use these examples as templates. Key patterns:

### Inline Pipeline
```yaml
apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  name: my-pipeline
spec:
  pipelineSpec:
    tasks:
      - name: my-task
        taskSpec:
          steps:
            - name: step1
              image: alpine:latest
              script: |
                echo "Hello!"
```

### With Parameters
```yaml
spec:
  pipelineSpec:
    params:
      - name: message
        type: string
        default: "Hello"
    tasks:
      - name: greet
        taskSpec:
          params:
            - name: msg
          steps:
            - script: echo "$(params.msg)"
        params:
          - name: msg
            value: "$(params.message)"
  params:
    - name: message
      value: "Hi from CLI!"
```

### With Sidecars
```yaml
taskSpec:
  sidecars:
    - name: database
      image: postgres:15
      env:
        - name: POSTGRES_PASSWORD
          value: secret
  steps:
    - name: test
      image: postgres:15
      script: |
        psql -h database -U postgres -c "SELECT 1"
```
