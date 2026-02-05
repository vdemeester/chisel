# Research: Replacing Dagger with Podman

## Executive Summary

This document analyzes the feasibility and requirements for replacing Dagger with Podman as Chisel's execution backend. Based on comprehensive research of both the current implementation and available technologies, **the migration is technically feasible but requires significant reimplementation** of features that Dagger provides automatically.

**Key Findings:**
- Podman can handle all core container operations currently performed by Dagger
- **Major gap**: Dagger's automatic DAG-based parallelization, caching, and optimization would need manual implementation
- Estimated effort: 2000-3000 lines of new code to replace ~200 lines of Dagger API calls
- Tradeoff: Reduced dependencies and simpler deployment vs. increased maintenance burden

---

## Current Dagger Usage in Chisel

### Dagger API Surface

Chisel uses **13 core Dagger methods** across the codebase:

**Client Methods (6):**
1. `dagger.Connect(ctx, opts)` - Establish connection to Dagger Engine
2. `client.Directory()` - Create empty directory
3. `client.Host().Directory(path)` - Mount host directory
4. `client.Container()` - Create container builder
5. `client.CacheVolume(id)` - Create named cache volume
6. `client.Close()` - Close connection

**Container Methods (7):**
1. `Container.From(image)` - Set base image
2. `Container.WithExec(cmd)` - Execute command
3. `Container.WithEnvVariable(k, v)` - Set environment variable
4. `Container.WithWorkdir(path)` - Set working directory
5. `Container.WithDirectory(path, dir)` - Mount directory
6. `Container.WithMountedCache(path, cache)` - Mount cache volume
7. `Container.Stdout(ctx)` / `Container.Stderr(ctx)` - Capture output

**Sidecar Methods (3):**
1. `Container.AsService()` - Convert container to service
2. `Container.WithServiceBinding(name, svc)` - Bind service to container
3. `Container.WithExposedPort(port)` - Expose port

**File Methods (2):**
1. `Container.File(path)` - Get file reference
2. `File.Contents(ctx)` - Read file contents

### Execution Flow

```
PipelineRun YAML
    ↓
Parser → ResolvedPipelineRun
    ↓
Executor.Execute()
    ├─ Initialize Workspaces (Dagger Directories)
    ├─ Expand Matrix Tasks
    ├─ Build DAG (custom scheduler)
    └─ ExecuteParallel (goroutines)
        ├─ executeTask()
        │   ├─ Start Sidecars (Dagger Services)
        │   └─ executeStep()
        │       ├─ Build Container (Dagger fluent API)
        │       ├─ Execute Command (Dagger WithExec)
        │       ├─ Capture Output (Dagger Stdout/Stderr)
        │       └─ Read Results (Dagger File.Contents)
        └─ FinallyTasks
```

### What Dagger Provides Automatically

1. **Container Layer Caching**: Content-addressed, automatic invalidation
2. **BuildKit Integration**: Advanced build features and optimizations
3. **Lazy Evaluation**: Operations only execute when results are needed
4. **Session Management**: Isolated state per execution
5. **Cross-Platform**: Same code runs on macOS, Linux, Windows
6. **GraphQL API**: Language-agnostic interface
7. **Distributed Caching**: Cache sharing across environments (with Dagger Cloud)

---

## Podman Capabilities Mapping

### Podman Go SDK: `github.com/containers/podman/v5/pkg/bindings`

**Requirements:**
- Podman system service must be running: `systemctl start --user podman.socket`
- Connection: `bindings.NewConnection(ctx, "unix:///run/podman/podman.sock")`

### Direct API Mapping

| Dagger Feature | Podman Equivalent | Complexity |
|----------------|-------------------|------------|
| `Connect()` | `bindings.NewConnection()` | ✅ Simple |
| `Container()` | `specgen.NewSpecGenerator()` | ✅ Simple |
| `Container.From()` | `s.Image = "..."` | ✅ Simple |
| `Container.WithExec()` | `s.Command = [...]` + `containers.Start()` | ⚠️ Moderate |
| `Container.WithEnvVariable()` | `s.Env = map[string]string{...}` | ✅ Simple |
| `Container.WithWorkdir()` | `s.WorkDir = "..."` | ✅ Simple |
| `Container.WithDirectory()` | `s.Mounts = []spec.Mount{...}` | ✅ Simple |
| `Container.Stdout()` / `Stderr()` | `containers.Logs()` with channels | ⚠️ Moderate |
| `Directory()` (empty) | Named volume via `volumes.Create()` | ✅ Simple |
| `Host().Directory()` | Bind mount via `s.Mounts` | ✅ Simple |
| `CacheVolume()` | Named volume via `volumes.Create()` | ✅ Simple |
| `Container.AsService()` | Pod with long-running container | ⚠️ Moderate |
| `WithServiceBinding()` | Pod networking / shared network | ⚠️ Moderate |
| `File.Contents()` | `containers.Copy()` or exec `cat` | ⚠️ Moderate |

### Podman Advantages

1. **Rootless by Default**: Better security model
2. **Daemonless Option**: Can run without persistent service (CLI mode)
3. **Kubernetes Compatibility**: `podman generate kube` for K8s manifests
4. **Lighter Weight**: No BuildKit daemon overhead
5. **Red Hat Support**: Strong enterprise backing

### Podman Limitations for Chisel

1. **No Automatic Parallelization**: Must manually orchestrate parallel execution
2. **No Content-Addressed Caching**: Basic layer caching only
3. **No Lazy Evaluation**: Imperative execution model
4. **Service Binding Complexity**: Sidecars require Pod abstraction
5. **File Extraction**: No elegant API for reading files from containers
6. **Requires Service Running**: Go bindings need `podman.socket` active

---

## What Needs to Be Built

### 1. Container Execution Engine

**Current (Dagger):**
```go
container := e.client.Container().From(step.Image)
container = container.WithExec(cmd)
stdout, _ := container.Stdout(ctx)
```

**Replacement (Podman):**
```go
// Create spec
s := specgen.NewSpecGenerator(step.Image, false)
s.Command = cmd
s.Env = envVars
s.WorkDir = workingDir

// Create container
resp, _ := containers.CreateWithSpec(conn, s, nil)

// Start container
containers.Start(conn, resp.ID, nil)

// Wait for completion
exitCode, _ := containers.Wait(conn, resp.ID, nil)

// Capture logs
stdoutChan := make(chan string)
stderrChan := make(chan string)
containers.Logs(conn, resp.ID, opts, stdoutChan, stderrChan)

// Collect output
var stdout, stderr strings.Builder
for line := range stdoutChan { stdout.WriteString(line) }
for line := range stderrChan { stderr.WriteString(line) }

// Cleanup
containers.Remove(conn, resp.ID, nil)
```

**Estimated LOC:** ~150 lines (vs 15 with Dagger)

### 2. Workspace Management

**Current (Dagger):**
```go
// EmptyDir
ws := e.client.Directory()

// Local directory
ws := e.client.Host().Directory(path)

// Cache volume
cache := e.client.CacheVolume(name)
container = container.WithMountedCache(mountPath, cache)
```

**Replacement (Podman):**
```go
// EmptyDir → Named volume
vol, _ := volumes.Create(conn, &volumes.CreateOptions{
    Name: volumeName,
})

s.Volumes = []*specgen.NamedVolume{
    {Name: vol.Name, Dest: mountPath},
}

// Local directory → Bind mount
s.Mounts = []spec.Mount{
    {Source: hostPath, Destination: containerPath, Type: "bind"},
}

// Cache volume → Named volume with persistence
// (same as EmptyDir but with predictable name for reuse)
```

**Estimated LOC:** ~80 lines

### 3. Sidecar/Service Support

**Current (Dagger):**
```go
// Start sidecar as service
sidecarContainer := e.client.Container().From(sidecar.Image)
sidecarContainer = sidecarContainer.WithExposedPort(port)
service := sidecarContainer.AsService()

// Bind to main container
container = container.WithServiceBinding(sidecar.Name, service)
// Sidecar accessible at: http://{sidecar.Name}:{port}/
```

**Replacement (Podman):**

**Option A: Pod-based (Kubernetes-style)**
```go
import "github.com/containers/podman/v5/pkg/bindings/pods"

// Create pod for task
podSpec := specgen.NewPodSpecGenerator()
podSpec.Name = fmt.Sprintf("task-%s", taskName)
podResp, _ := pods.CreatePodFromSpec(conn, podSpec)

// Start sidecar in pod
sidecarSpec := specgen.NewSpecGenerator(sidecar.Image, false)
sidecarSpec.Pod = podResp.ID
sidecarSpec.PortMappings = []nettypes.PortMapping{
    {ContainerPort: port, HostPort: 0}, // Auto-assign host port
}
sidecarResp, _ := containers.CreateWithSpec(conn, sidecarSpec, nil)
containers.Start(conn, sidecarResp.ID, nil)

// Start main container in same pod
mainSpec := specgen.NewSpecGenerator(step.Image, false)
mainSpec.Pod = podResp.ID
// Sidecar accessible via localhost:{port} within pod
```

**Option B: Shared Network**
```go
import "github.com/containers/podman/v5/pkg/bindings/network"

// Create network
netResp, _ := network.Create(conn, &network.CreateOptions{
    Name: fmt.Sprintf("task-%s-net", taskName),
})

// Start sidecar on network
sidecarSpec.Networks = map[string]types.PerNetworkOptions{
    netResp.Name: {},
}
containers.CreateWithSpec(conn, sidecarSpec, nil)

// Connect main container to network
mainSpec.Networks = map[string]types.PerNetworkOptions{
    netResp.Name: {},
}
// Sidecar accessible at: http://{sidecarName}:{port}/
```

**Estimated LOC:** ~200 lines

**Complexity:** HIGH - requires careful lifecycle management (start before step, stop after)

### 4. Result File Extraction

**Current (Dagger):**
```go
resultPath := fmt.Sprintf("/tekton/results/%s", resultName)
content, _ := container.File(resultPath).Contents(ctx)
```

**Replacement (Podman):**

**Option A: Copy from container**
```go
// Not available in bindings - would need direct tar API
```

**Option B: Exec cat command**
```go
execConfig := &handlers.ExecCreateConfig{
    Cmd: []string{"cat", resultPath},
    AttachStdout: true,
}
execID, _ := containers.ExecCreate(conn, containerID, execConfig)

// Attach and capture output
var stdout bytes.Buffer
containers.ExecStartAndAttach(conn, execID, &containers.ExecStartAndAttachOptions{
    OutputStream: &stdout,
})

content := stdout.String()
```

**Option C: Mount volume for results**
```go
// Create volume for results
resultsVol, _ := volumes.Create(conn, &volumes.CreateOptions{
    Name: fmt.Sprintf("task-%s-results", taskName),
})

// Mount to /tekton/results
s.Volumes = []*specgen.NamedVolume{
    {Name: resultsVol.Name, Dest: "/tekton/results"},
}

// After execution, mount volume to temporary container to read files
readSpec := specgen.NewSpecGenerator("alpine", false)
readSpec.Volumes = []*specgen.NamedVolume{
    {Name: resultsVol.Name, Dest: "/data"},
}
readSpec.Command = []string{"cat", fmt.Sprintf("/data/%s", resultName)}
// ... execute and capture stdout
```

**Estimated LOC:** ~120 lines

**Complexity:** MODERATE-HIGH - volume approach is cleanest but verbose

### 5. Timeout Handling

**Current (Dagger):**
```go
ctx, cancel := context.WithTimeout(parentCtx, duration)
defer cancel()
stdout, err := container.Stdout(ctx) // Dagger respects context timeout
```

**Replacement (Podman):**
```go
// Set stop timeout in spec
timeout := uint(timeoutSeconds)
s.StopTimeout = &timeout

// Use context timeout for Wait operation
ctx, cancel := context.WithTimeout(parentCtx, duration)
defer cancel()

// Wait with timeout
exitCode, err := containers.Wait(ctx, containerID, nil)
if err == context.DeadlineExceeded {
    // Force kill
    containers.Stop(conn, containerID, &containers.StopOptions{
        Timeout: &zeroTimeout,
    })
}
```

**Estimated LOC:** ~30 lines (similar to current)

### 6. Retry Logic

**Current:**
- Retry implemented at Go level (no Dagger dependency)
- Re-executes entire step including container creation

**Replacement:**
- No change needed - retry logic is independent of execution backend

**Estimated LOC:** 0 (no change)

### 7. Step Template Defaults

**Current:**
- Applied before Dagger container creation
- No Dagger-specific logic

**Replacement:**
- No change needed - applies defaults to SpecGenerator before creation

**Estimated LOC:** 0 (no change)

### 8. Matrix Task Expansion

**Current:**
- Expands tasks before DAG construction
- No Dagger-specific logic

**Replacement:**
- No change needed

**Estimated LOC:** 0 (no change)

### 9. Connection Management

**Current (Dagger):**
```go
client, _ := dagger.Connect(ctx, dagger.WithLogOutput(nil))
defer client.Close()
```

**Replacement (Podman):**
```go
// Ensure service is running
// Option A: Assume user has started it
conn, _ := bindings.NewConnection(ctx, "unix:///run/user/1000/podman/podman.sock")

// Option B: Start service programmatically (requires systemd)
exec.Command("systemctl", "start", "--user", "podman.socket").Run()
conn, _ := bindings.NewConnection(ctx, "unix:///run/user/1000/podman/podman.sock")
```

**Complexity:** MODERATE - need to handle service availability

**Estimated LOC:** ~40 lines (with error handling and service checks)

---

## Architecture Changes Required

### New Package: `pkg/podman`

Create abstraction layer to encapsulate Podman operations:

```
pkg/podman/
├── client.go          # Connection management
├── container.go       # Container lifecycle
├── volumes.go         # Volume/workspace management
├── services.go        # Sidecar/pod management
├── results.go         # File extraction utilities
└── types.go          # Type conversions (types.Step → specgen)
```

### Modified Packages

**`pkg/executor/executor.go`:**
- Replace `*dagger.Client` with `*podman.Client`
- Replace Dagger API calls with Podman abstraction
- Add cleanup logic (containers, volumes, pods, networks)

**`pkg/executor/sidecar.go`:**
- Refactor to use Pod or Network abstraction
- Add lifecycle management (wait for ready, cleanup)

**`pkg/executor/results.go`:**
- Implement file extraction logic

**New: `pkg/executor/cleanup.go`:**
- Resource cleanup (containers, volumes, pods, networks)
- Ensure no resource leaks

### Estimated Code Changes

| Component | Current LOC | New LOC | Delta |
|-----------|-------------|---------|-------|
| Executor core | ~300 | ~450 | +150 |
| Sidecar management | ~80 | ~200 | +120 |
| Workspace mgmt | ~60 | ~120 | +60 |
| Result capture | ~40 | ~140 | +100 |
| Podman abstraction | 0 | ~800 | +800 |
| Connection mgmt | ~10 | ~50 | +40 |
| Cleanup logic | 0 | ~150 | +150 |
| **TOTAL** | **~490** | **~1910** | **+1420** |

**Additional test code:** ~800 LOC

**Total implementation effort:** ~2200 LOC

---

## Feature Parity Analysis

### ✅ Features with Full Parity

- Container execution (image, command, args, env, workingDir)
- Workspace mounting (emptyDir, local paths, persistent volumes)
- Result capture (via volume or exec)
- Timeout handling
- Retry logic
- Environment variable substitution
- Port exposure
- Volume management

### ⚠️ Features with Partial Parity

**Sidecars:**
- **Dagger**: Elegant service binding with automatic hostname resolution
- **Podman**: Requires Pod or shared network; more verbose setup
- **Impact**: More complex code, but functionally equivalent

**Caching:**
- **Dagger**: Content-addressed, automatic invalidation, distributed caching
- **Podman**: Named volumes persist, but no automatic invalidation
- **Impact**: Workspaces work, but no build layer optimization

**Parallel Execution:**
- **Dagger**: Automatic based on DAG
- **Podman**: Already implemented in Chisel's scheduler (no change)
- **Impact**: None - Chisel doesn't use Dagger's parallelization

### ❌ Features Lost

**Content-Addressed Caching:**
- Dagger's automatic layer reuse across builds
- **Impact**: Slower repeat executions (no layer cache)
- **Mitigation**: Podman has basic layer caching for `buildah build`, but not for `podman run`

**Lazy Evaluation:**
- Dagger only executes operations when results are needed
- **Impact**: Podman executes imperatively; minor performance loss
- **Mitigation**: Minimal impact for Chisel's sequential step execution

**Cross-Platform Transparency:**
- Dagger runs on macOS, Linux, Windows with same code
- **Impact**: Podman requires platform-specific socket paths
- **Mitigation**: Detect platform and adjust socket path

**BuildKit Integration:**
- Dagger's advanced build features (secrets, cache mounts in builds)
- **Impact**: Not currently used by Chisel (runs existing images)
- **Mitigation**: None needed

---

## Dependency Analysis

### Current Dependencies (with Dagger)

```
dagger.io/dagger v0.19.10
  ├─ BuildKit (indirect)
  ├─ gRPC (indirect)
  ├─ GraphQL client (indirect)
  └─ OpenTelemetry (indirect)
```

**Binary size contribution:** ~15-20 MB

### New Dependencies (with Podman)

```
github.com/containers/podman/v5/pkg/bindings
  ├─ github.com/containers/common
  ├─ github.com/containers/image/v5
  ├─ github.com/opencontainers/runtime-spec
  └─ github.com/containers/storage
```

**Binary size contribution:** ~8-12 MB (smaller than Dagger)

### Runtime Dependencies

**Dagger:**
- Requires Docker daemon OR Dagger Engine container
- ~200MB container image download on first run
- Persistent daemon process

**Podman:**
- Requires Podman installed on host
- Requires `podman.socket` service running (for Go bindings)
- No container daemon overhead

---

## Migration Path

### Phase 1: Abstraction Layer (Week 1-2)

1. Create `pkg/podman` package with abstraction layer
2. Implement core container operations
3. Implement workspace/volume management
4. Write unit tests for Podman wrapper

**Deliverable:** Podman abstraction layer with test coverage

### Phase 2: Executor Refactoring (Week 2-3)

1. Replace Dagger calls in `executor.go` with Podman abstraction
2. Implement result file extraction
3. Add resource cleanup logic
4. Update existing tests

**Deliverable:** Working executor with Podman backend

### Phase 3: Sidecar Support (Week 3-4)

1. Implement Pod-based sidecar management
2. Add network creation/cleanup
3. Test sidecar examples

**Deliverable:** Sidecar functionality with Podman

### Phase 4: Integration Testing (Week 4-5)

1. Run all example pipelines
2. Verify output matches Dagger implementation
3. Performance testing
4. Documentation updates

**Deliverable:** Production-ready Podman backend

### Phase 5: Optimization (Week 5-6)

1. Implement connection pooling
2. Optimize volume reuse
3. Add better error messages
4. Performance tuning

**Deliverable:** Optimized implementation

**Total estimated time:** 6 weeks (1 developer)

---

## Tradeoffs Summary

### Advantages of Switching to Podman

| Benefit | Impact |
|---------|--------|
| **Smaller binary** | -8 MB (15-20 MB → 8-12 MB) |
| **Fewer runtime deps** | No BuildKit daemon required |
| **Better security** | Rootless by default |
| **Enterprise support** | Red Hat backing |
| **Kubernetes integration** | `podman generate kube` |
| **Simpler debugging** | Direct container access |

### Disadvantages of Switching to Podman

| Cost | Impact |
|------|--------|
| **Implementation effort** | ~2200 LOC + tests |
| **Maintenance burden** | More code to maintain |
| **Lost caching** | No content-addressed cache |
| **Complexity** | More verbose API |
| **Service requirement** | Must run `podman.socket` |
| **Sidecar complexity** | Pods are more complex than Dagger services |

---

## Performance Comparison

### Dagger Performance Characteristics

- **First run**: ~5-10s overhead (engine startup)
- **Subsequent runs**: ~1-2s overhead (session reuse)
- **Layer caching**: Excellent (content-addressed)
- **Parallel execution**: Automatic (limited by DAG)

### Estimated Podman Performance

- **First run**: ~0.5-1s overhead (connection only)
- **Subsequent runs**: ~0.5-1s overhead (same)
- **Layer caching**: Basic (image layers only)
- **Parallel execution**: Manual (same as current Chisel scheduler)

**Expected change:**
- Faster for single-run scenarios (no engine overhead)
- Slower for repeated runs (no content-addressed caching)
- Overall: ~10-20% faster for typical usage

---

## Compatibility Matrix

| Feature | Dagger | Podman | Notes |
|---------|--------|--------|-------|
| Linux AMD64 | ✅ | ✅ | Full support |
| Linux ARM64 | ✅ | ✅ | Full support |
| macOS AMD64 | ✅ | ✅ | Podman Desktop required |
| macOS ARM64 | ✅ | ✅ | Podman Desktop required |
| Windows | ✅ | ⚠️ | WSL2 required for Podman |
| Rootless | ✅ | ✅ | Podman advantage |
| Alpine Linux | ✅ | ✅ | Both work |
| Container images | ✅ | ✅ | Full compatibility |

---

## Risk Assessment

### High Risks

1. **Sidecar complexity**: Pod management is significantly more complex than Dagger's service binding
   - **Mitigation**: Thorough testing, fallback to network-based approach

2. **File extraction**: No elegant API for reading files from containers
   - **Mitigation**: Use volume-based approach or exec workaround

3. **Cross-platform support**: Socket path differences, macOS Podman Desktop quirks
   - **Mitigation**: Platform detection, comprehensive testing

### Medium Risks

1. **Service availability**: Requires `podman.socket` to be running
   - **Mitigation**: Auto-start service, clear error messages

2. **Resource cleanup**: Pods, networks, volumes, containers must be cleaned up
   - **Mitigation**: Defer cleanup, context-based cancellation

3. **Performance regression**: Loss of content-addressed caching
   - **Mitigation**: Acceptable for local dev use case

### Low Risks

1. **API changes**: Podman bindings API might change
   - **Mitigation**: Pin to specific version, abstraction layer isolates changes

2. **Debugging**: Different tooling than Dagger
   - **Mitigation**: Better tooling (`podman ps`, `podman logs` are familiar)

---

## Recommendation

### Decision Matrix

| Criteria | Weight | Dagger Score | Podman Score |
|----------|--------|--------------|--------------|
| Implementation effort | 20% | 10/10 (done) | 3/10 (6 weeks) |
| Runtime dependencies | 15% | 6/10 (engine) | 9/10 (socket only) |
| Performance | 15% | 9/10 (caching) | 7/10 (basic) |
| Maintainability | 20% | 9/10 (simple) | 5/10 (complex) |
| Security | 10% | 8/10 | 10/10 (rootless) |
| Binary size | 10% | 6/10 | 8/10 |
| Ecosystem fit | 10% | 7/10 | 8/10 (Tekton/K8s) |
| **Weighted Total** | | **8.0/10** | **6.2/10** |

### Recommendation: **Keep Dagger**

**Rationale:**

1. **Current implementation works well**: Dagger provides exactly what Chisel needs with minimal code
2. **Hidden costs**: Podman migration requires ~2200 LOC of complex logic for sidecars, file extraction, cleanup
3. **Lost capabilities**: Content-addressed caching is valuable for repeated executions
4. **Maintenance burden**: More code = more bugs, more testing, more documentation

**When to reconsider:**

- If Dagger becomes unmaintained or significantly changes licensing
- If binary size becomes a critical constraint (embedded systems)
- If enterprise customers demand Podman-only environments
- If Chisel needs features Podman provides (e.g., `podman generate kube` for migration path)

### Alternative: **Make Backend Pluggable**

If flexibility is desired, create an abstraction layer:

```go
type ContainerBackend interface {
    Execute(ctx context.Context, spec ContainerSpec) (*ExecutionResult, error)
    CreateWorkspace(spec WorkspaceSpec) (Workspace, error)
    StartSidecar(spec SidecarSpec) (Service, error)
}

type DaggerBackend struct { client *dagger.Client }
type PodmanBackend struct { conn context.Context }
```

This allows:
- Users to choose backend via CLI flag: `--backend=dagger|podman`
- Gradual migration or side-by-side testing
- Future backends (Docker, containerd, etc.)

**Effort:** +500 LOC for abstraction + full Podman implementation

---

## Appendix: Code Examples

### A. Current Dagger Implementation

```go
// From pkg/executor/executor.go
func (e *Executor) executeStep(ctx context.Context, step types.Step, ...) error {
    container := e.client.Container().From(step.Image)

    // Mount workspace
    if wsDir, ok := e.workspaces["source"]; ok {
        container = container.WithDirectory("/workspace/source", wsDir)
    }

    // Set environment
    for k, v := range step.Env {
        container = container.WithEnvVariable(k, v)
    }

    // Execute
    container = container.WithExec(step.Command)

    // Capture output
    stdout, _ := container.Stdout(ctx)

    return nil
}
```

**Lines of code:** ~15

### B. Proposed Podman Implementation

```go
// Proposed pkg/podman/executor.go
func (e *Executor) executeStep(ctx context.Context, step types.Step, ...) error {
    // Create container spec
    s := specgen.NewSpecGenerator(step.Image, false)
    s.Name = fmt.Sprintf("step-%s-%d", taskName, stepIndex)
    s.Command = step.Command
    s.Env = step.Env
    s.WorkDir = step.WorkingDir

    // Mount workspaces via bind mount
    if wsPath, ok := e.workspaces["source"]; ok {
        s.Mounts = append(s.Mounts, spec.Mount{
            Source:      wsPath,
            Destination: "/workspace/source",
            Type:        "bind",
        })
    }

    // Create container
    resp, err := containers.CreateWithSpec(e.conn, s, nil)
    if err != nil {
        return fmt.Errorf("create container: %w", err)
    }
    containerID := resp.ID

    // Ensure cleanup
    defer func() {
        timeout := uint(1)
        containers.Stop(e.conn, containerID, &containers.StopOptions{
            Timeout: &timeout,
        })
        containers.Remove(e.conn, containerID, &containers.RemoveOptions{
            Force: true,
        })
    }()

    // Start container
    if err := containers.Start(e.conn, containerID, nil); err != nil {
        return fmt.Errorf("start container: %w", err)
    }

    // Wait for completion
    exitCode, err := containers.Wait(ctx, containerID, nil)
    if err != nil {
        return fmt.Errorf("wait for container: %w", err)
    }

    if exitCode != 0 {
        return fmt.Errorf("container exited with code %d", exitCode)
    }

    // Capture logs
    stdoutChan := make(chan string, 100)
    stderrChan := make(chan string, 100)

    logsDone := make(chan struct{})
    var stdout, stderr strings.Builder

    go func() {
        for line := range stdoutChan {
            stdout.WriteString(line)
        }
        close(logsDone)
    }()

    go func() {
        for line := range stderrChan {
            stderr.WriteString(line)
        }
    }()

    follow := false
    fullLogs := true
    containers.Logs(e.conn, containerID, &containers.LogOptions{
        Stdout: &fullLogs,
        Stderr: &fullLogs,
        Follow: &follow,
    }, stdoutChan, stderrChan)

    <-logsDone

    // Store output
    e.logger.LogStepOutput(taskName, step.Name, stdout.String(), stderr.String())

    return nil
}
```

**Lines of code:** ~75

**Ratio:** 5x more code for equivalent functionality

---

## Appendix: Service Availability Checks

```go
// pkg/podman/client.go
func EnsureServiceRunning() error {
    // Try to connect
    conn, err := bindings.NewConnection(context.Background(), getSocketPath())
    if err == nil {
        return nil // Already running
    }

    // Try to start service
    cmd := exec.Command("systemctl", "start", "--user", "podman.socket")
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("podman service not running and failed to start: %w", err)
    }

    // Wait for socket
    for i := 0; i < 10; i++ {
        time.Sleep(500 * time.Millisecond)
        if _, err := bindings.NewConnection(context.Background(), getSocketPath()); err == nil {
            return nil
        }
    }

    return fmt.Errorf("podman service did not start within 5 seconds")
}

func getSocketPath() string {
    if runtime.GOOS == "darwin" {
        return "unix:///Users/" + os.Getenv("USER") + "/.local/share/containers/podman/machine/podman.sock"
    }
    uid := os.Getuid()
    return fmt.Sprintf("unix:///run/user/%d/podman/podman.sock", uid)
}
```

---

## Document Metadata

- **Author**: Research synthesis from Dagger/Podman analysis
- **Date**: 2026-02-05
- **Version**: 1.0
- **Status**: Research Complete
- **Decision**: Recommend keeping Dagger (can be revisited)
