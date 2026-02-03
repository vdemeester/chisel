# Local Directory Injection Feature Plan

## Goal

Allow users to "hijack" a task's workspace by injecting the current directory (or any local path) as a volume/workspace. This enables running Tekton tasks against local source code without modifying the task YAML.

## Use Cases

1. **Local Development**: Run a build task against local source code
   ```bash
   chisel run build-task.yaml --workspace=source:.
   ```

2. **Testing Pipeline Changes**: Test a pipeline against a local checkout
   ```bash
   chisel run ci-pipeline.yaml --workspace=source:/path/to/checkout
   ```

3. **Quick Iteration**: Modify code locally, run task, repeat
   ```bash
   chisel run lint-task.yaml -w src:.
   ```

## Proposed CLI Interface

### Option 1: `--workspace` flag (Recommended)

```bash
# Bind current directory to workspace named "source"
chisel run task.yaml --workspace=source:.

# Bind specific path
chisel run task.yaml --workspace=source:/path/to/code

# Short form
chisel run task.yaml -w source:.

# Multiple workspaces
chisel run task.yaml -w source:. -w config:./config
```

### Option 2: `--bind` flag

```bash
chisel run task.yaml --bind=source:.
```

## Implementation Design

### 1. CLI Changes (`cmd/chisel/run.go`)

```go
type runOptions struct {
    // ... existing fields
    Workspaces []string // --workspace or -w flags
}

// Parse workspace bindings
func parseWorkspaceBinding(spec string) (name, path string, err error) {
    parts := strings.SplitN(spec, ":", 2)
    if len(parts) != 2 {
        return "", "", fmt.Errorf("invalid workspace binding: %s (expected name:path)", spec)
    }
    name = parts[0]
    path = parts[1]

    // Resolve relative paths
    if !filepath.IsAbs(path) {
        path, err = filepath.Abs(path)
        if err != nil {
            return "", "", err
        }
    }
    return name, path, nil
}
```

### 2. Parser Changes (`pkg/parser/parser.go`)

Add option to override workspace bindings during parsing:

```go
type Options struct {
    TasksDir           string
    Debug              bool
    WorkspaceOverrides map[string]string // name -> local path
}
```

Or pass overrides to the parse function:

```go
func (p *Parser) ParsePipelineRun(path string, wsOverrides map[string]string) (*types.ResolvedPipelineRun, error)
```

### 3. Types Changes (`pkg/types/types.go`)

Already supports `WorkspaceTypeLocal` which uses `binding.Path` for the local directory. No changes needed.

### 4. Executor Changes (`pkg/executor/executor.go`)

Already handles `WorkspaceTypeLocal`:

```go
case types.WorkspaceTypeLocal:
    return e.client.Host().Directory(binding.Path), nil
```

No changes needed - the existing implementation should work.

## Flow

```
CLI: chisel run task.yaml -w source:.
  │
  ├─► Parse --workspace flags
  │     └─► {source: "/home/user/project"}
  │
  ├─► Parse task.yaml
  │
  ├─► Override workspace bindings
  │     └─► source: {Type: Local, Path: "/home/user/project"}
  │
  └─► Execute
        └─► createWorkspace()
              └─► e.client.Host().Directory("/home/user/project")
```

## Edge Cases

### 1. Workspace doesn't exist in task
- Error: "workspace 'foo' not declared in task"
- Show available workspace names

### 2. Path doesn't exist
- Error: "path '/foo/bar' does not exist"
- Check before execution, not during

### 3. Override PipelineRun's workspace binding
- CLI overrides should take precedence over YAML bindings
- Useful for testing with local code instead of PVC

### 4. Relative paths
- Always resolve to absolute before passing to Dagger
- Use `filepath.Abs()`

## Test Plan

### Unit Tests

1. `TestParseWorkspaceBinding`
   - Valid: `source:.` → `("source", "/abs/path", nil)`
   - Valid: `src:/path/to/code` → `("src", "/path/to/code", nil)`
   - Invalid: `source` → error
   - Invalid: `:::` → error

2. `TestWorkspaceOverride`
   - Override replaces YAML binding
   - Multiple overrides work
   - Unknown workspace returns error

### Integration Tests

1. Run task with `--workspace=source:.`
2. Verify container can read files from local directory
3. Verify container writes are visible locally

## Implementation Steps

1. [ ] Add `--workspace/-w` flag to CLI
2. [ ] Parse workspace binding specs
3. [ ] Pass overrides to parser
4. [ ] Override workspace bindings in parser
5. [ ] Add tests for parsing
6. [ ] Add integration test example
7. [ ] Update README with new flag

## Example

Given task:
```yaml
apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: go-build
spec:
  workspaces:
    - name: source
  steps:
    - name: build
      image: golang:1.21
      workingDir: $(workspaces.source.path)
      script: |
        go build ./...
```

Run with local source:
```bash
cd /home/user/myproject
chisel run go-build.yaml --workspace=source:.
```

The task will build the code in `/home/user/myproject`.
