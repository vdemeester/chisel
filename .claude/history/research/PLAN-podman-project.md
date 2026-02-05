# Plan: Podman-Based Tekton Local Executor

## Project Overview

A sister project to Chisel that executes Tekton PipelineRuns locally using Podman as the execution backend instead of Dagger. This project explores a different architectural approach while maintaining compatibility with Tekton specifications.

## Goals

1. **Prove Podman viability** for Tekton local execution
2. **Reduce dependencies** - smaller binary, no BuildKit daemon
3. **Leverage Podman strengths** - rootless execution, Kubernetes integration
4. **Maintain user experience** - same CLI, same output, same examples
5. **Code reuse** - share parser, types, UI, and test infrastructure with Chisel

## Name Suggestions

Following Chisel's naming pattern (builder's tool + pairs with backend):

### Top Candidates

1. **Mallet** ⭐ (Recommended)
   - Pairs with Podman (both are gentler, more precise tools)
   - Builder's tool like Chisel
   - Reflects Podman's rootless, daemonless approach
   - Short, memorable, easy to type

2. **Plane**
   - Woodworking tool that smooths and simplifies
   - Reflects Podman's simpler architecture
   - Good metaphor for container execution

3. **Auger**
   - Drilling tool, creates openings
   - Works with "pods" (drilling into pods)
   - Active, strong name

4. **Gimlet**
   - Small hand drill, precise and simple
   - Reflects focused, minimal approach
   - Unique, memorable

### Other Options

5. **Froe** - Splitting tool (splits workloads into containers)
6. **Adze** - Ancient carpentry tool for shaping
7. **Drawknife** - Two-handled shaping blade
8. **Spokeshave** - For shaping rounded surfaces
9. **Awl** - Pointed tool for precision work
10. **Rasp** - Coarse shaping tool

## Project Structure

### Shared Components (from Chisel)

```
shared/
├── pkg/
│   ├── parser/          # Tekton YAML parsing (100% reuse)
│   ├── types/           # Internal types (100% reuse)
│   ├── ui/              # Output formatting, logging (100% reuse)
│   └── executor/
│       ├── scheduler.go      # DAG scheduling (100% reuse)
│       ├── substitution.go   # Variable substitution (100% reuse)
│       ├── when.go          # Conditional logic (100% reuse)
│       ├── steptemplate.go  # Template merging (100% reuse)
│       ├── matrix.go        # Matrix expansion (100% reuse)
│       ├── retry.go         # Retry logic (100% reuse)
│       ├── timeout.go       # Timeout parsing (100% reuse)
│       └── results.go       # Result types (100% reuse)
├── examples/            # All example YAML files (100% reuse)
└── .github/
    └── workflows/       # Integration tests (adapted)
```

### Project-Specific Components

```
{project-name}/
├── cmd/{project-name}/
│   ├── main.go          # Entry point (adapted from chisel)
│   ├── root.go          # CLI setup (adapted)
│   └── run.go           # Run command (adapted)
├── pkg/
│   ├── podman/          # NEW: Podman abstraction layer
│   │   ├── client.go         # Connection management
│   │   ├── container.go      # Container lifecycle
│   │   ├── volumes.go        # Volume/workspace management
│   │   ├── pods.go           # Pod management for sidecars
│   │   ├── network.go        # Network creation/management
│   │   ├── results.go        # File extraction utilities
│   │   └── types.go          # Type conversions
│   └── executor/        # NEW: Podman-specific executor
│       ├── executor.go       # Main execution logic
│       ├── sidecar.go        # Sidecar/pod management
│       ├── volumes.go        # Volume implementation
│       └── cleanup.go        # Resource cleanup
├── go.mod               # Dependencies (podman bindings)
└── README.md            # Project-specific docs
```

## Architecture Comparison

### Chisel (Dagger)
```
PipelineRun → Parser → Executor → Dagger SDK → Dagger Engine (BuildKit) → Containers
```

### New Project (Podman)
```
PipelineRun → Parser → Executor → Podman Bindings → Podman Service → Containers
```

## Implementation Phases

### Phase 1: Foundation (Week 1-2)

**Goal:** Basic container execution working

**Tasks:**
1. Create new repository/directory structure
2. Set up Go module with Podman bindings dependency
3. Implement `pkg/podman/client.go`:
   - Connection management
   - Service availability checks
   - Platform-specific socket paths
4. Implement `pkg/podman/container.go`:
   - Container creation from SpecGenerator
   - Start/stop/wait lifecycle
   - Stdout/stderr capture
   - Basic error handling
5. Write unit tests for Podman wrapper
6. Create simple CLI that can run a single-step task

**Deliverable:** Can execute `{project} run examples/simple/hello-task.yaml`

### Phase 2: Workspaces & Volumes (Week 2-3)

**Goal:** Workspace management working

**Tasks:**
1. Implement `pkg/podman/volumes.go`:
   - Named volume creation for EmptyDir
   - Bind mount handling for local directories
   - Volume cleanup
2. Implement `pkg/podman/volumes.go` (executor side):
   - Workspace initialization
   - Volume mounting in containers
3. Implement workspace CLI flag (`--workspace`)
4. Test with workspace examples

**Deliverable:** Can execute workspace-based tasks with local directory injection

### Phase 3: Core Executor (Week 3-4)

**Goal:** Multi-step tasks and basic pipelines

**Tasks:**
1. Implement `pkg/executor/executor.go`:
   - Task execution loop
   - Step execution with Podman backend
   - Environment variable handling
   - Working directory support
2. Integrate shared components:
   - Variable substitution
   - Timeout handling
   - Retry logic
   - Step template defaults
3. Test with multi-step task examples

**Deliverable:** Can execute multi-step tasks with params, env vars, workspaces

### Phase 4: Results & Pipeline Support (Week 4-5)

**Goal:** Result passing and pipeline execution

**Tasks:**
1. Implement `pkg/podman/results.go`:
   - File extraction via exec or volume
   - Result storage and retrieval
2. Integrate scheduler for parallel task execution
3. Implement result substitution in downstream tasks
4. Test with pipeline examples (results-pipelinerun.yaml)

**Deliverable:** Can execute full pipelines with result passing

### Phase 5: Sidecars (Week 5-6)

**Goal:** Sidecar support via Pods

**Tasks:**
1. Implement `pkg/podman/pods.go`:
   - Pod creation and lifecycle
   - Container-to-pod association
2. Implement `pkg/podman/network.go`:
   - Alternative: shared network approach
   - Network creation and cleanup
3. Implement `pkg/executor/sidecar.go`:
   - Sidecar startup before steps
   - Sidecar cleanup after task
   - Hostname-based service discovery
4. Test with sidecar examples

**Deliverable:** Can execute tasks with sidecars (sidecar-pipelinerun.yaml)

### Phase 6: Advanced Features (Week 6-7)

**Goal:** Complete feature parity

**Tasks:**
1. Implement matrix expansion integration
2. Implement conditional execution (when clauses)
3. Implement finally tasks
4. Add comprehensive error handling
5. Add resource cleanup on interrupt (Ctrl+C)
6. Test all example files

**Deliverable:** All example pipelines work

### Phase 7: Polish & Optimization (Week 7-8)

**Goal:** Production-ready

**Tasks:**
1. Performance optimization:
   - Connection pooling
   - Volume reuse strategies
   - Parallel container cleanup
2. Better error messages and debugging:
   - Clear Podman service errors
   - Container inspection on failure
   - Helpful suggestions
3. Documentation:
   - README with installation
   - Architecture comparison with Chisel
   - Migration guide
   - Contributing guide
4. CI/CD setup:
   - GitHub Actions for tests
   - Cross-platform testing
   - Release automation

**Deliverable:** Production-ready v0.1.0 release

## Code Reuse Strategy

### Direct Reuse (via Go modules)

Option 1: **Separate repositories with shared module**
```
github.com/vdemeester/tekton-local-common
  ├── pkg/parser
  ├── pkg/types
  └── pkg/ui

github.com/vdemeester/chisel (import tekton-local-common)
github.com/vdemeester/{project-name} (import tekton-local-common)
```

Option 2: **Monorepo with multiple commands**
```
github.com/vdemeester/tekton-local
  ├── cmd/
  │   ├── chisel/      # Dagger backend
  │   └── {project}/   # Podman backend
  ├── pkg/
  │   ├── parser/      # Shared
  │   ├── types/       # Shared
  │   ├── ui/          # Shared
  │   ├── dagger/      # Chisel-specific
  │   └── podman/      # Project-specific
  └── examples/        # Shared
```

**Recommendation:** Monorepo (Option 2)
- Easier to keep in sync
- Shared examples and tests
- Single CI/CD pipeline
- Easier code sharing during development

### Test Reuse

Share integration tests with different backends:

```go
// tests/integration/integration_test.go
func TestHelloTask(t *testing.T) {
    backend := os.Getenv("BACKEND") // "chisel" or "mallet"
    runTest(t, backend, "examples/simple/hello-task.yaml")
}
```

Run tests for both:
```bash
BACKEND=chisel go test ./tests/integration/...
BACKEND=mallet go test ./tests/integration/...
```

## Dependencies

### Shared Dependencies
```
github.com/spf13/cobra           # CLI framework
github.com/charmbracelet/lipgloss # UI styling
gopkg.in/yaml.v3                 # YAML parsing
golang.org/x/term                # Terminal detection
```

### Project-Specific Dependencies
```
github.com/containers/podman/v5/pkg/bindings      # Podman Go API
github.com/containers/common/libnetwork/types     # Network types
github.com/opencontainers/runtime-spec/specs-go   # OCI spec
```

### Size Comparison
- Chisel binary: ~21 MB (with Dagger)
- New project binary: ~13-15 MB (estimated, with Podman)

## Feature Matrix

| Feature | Chisel (Dagger) | New Project (Podman) |
|---------|-----------------|----------------------|
| Container execution | ✅ | ✅ |
| Workspaces (emptyDir) | ✅ | ✅ (named volumes) |
| Workspaces (local) | ✅ | ✅ (bind mounts) |
| Workspaces (PVC) | ✅ | ✅ (named volumes) |
| Parameters (all types) | ✅ | ✅ |
| Results | ✅ | ✅ (via exec/volume) |
| Sidecars | ✅ | ✅ (via Pods) |
| Parallel tasks | ✅ | ✅ |
| Finally tasks | ✅ | ✅ |
| Timeout | ✅ | ✅ |
| Retry | ✅ | ✅ |
| When clauses | ✅ | ✅ |
| StepTemplate | ✅ | ✅ |
| Matrix builds | ✅ | ✅ |
| Content-addressed cache | ✅ | ❌ |
| Lazy evaluation | ✅ | ❌ |
| Rootless default | ⚠️ | ✅ |
| No daemon required | ❌ | ⚠️ (needs socket) |
| Podman Kube integration | ❌ | ✅ |

## Success Criteria

### Functional
- [ ] All example PipelineRuns execute successfully
- [ ] Output matches Chisel (except timing differences)
- [ ] Error messages are clear and actionable
- [ ] Resource cleanup is complete (no leaked containers/volumes)

### Performance
- [ ] Startup time < 2 seconds (vs Chisel's ~5s first run)
- [ ] Execution time within 20% of Chisel for same workload
- [ ] Memory usage < 500MB for typical pipeline

### Quality
- [ ] >80% test coverage
- [ ] All linters pass (golangci-lint)
- [ ] Documentation complete (README, examples, architecture)
- [ ] CI/CD pipeline green

### User Experience
- [ ] Installation as simple as `go install`
- [ ] Clear error if Podman service not running
- [ ] `--help` documentation complete
- [ ] Examples run out of the box

## Risks & Mitigations

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Sidecar Pod complexity | High | Medium | Fallback to network-based approach |
| File extraction issues | Medium | Medium | Multiple strategies (exec, volume, copy) |
| Cross-platform socket paths | Medium | Low | Platform detection, auto-discovery |
| Podman API changes | Medium | Low | Pin to specific version, abstraction layer |
| Performance regression | Low | Medium | Acceptable for exploration project |
| Service availability | High | Medium | Auto-start, clear error messages |

## Open Questions

1. **Repository structure**: Monorepo or separate repos?
   - **Recommendation**: Monorepo for easier development

2. **Sidecar implementation**: Pods or shared networks?
   - **Recommendation**: Pods (more Kubernetes-like)

3. **File extraction**: Exec cat, volume mount, or copy API?
   - **Recommendation**: Volume mount (cleaner, more reliable)

4. **Binary naming**: Single binary with `--backend` flag or separate binaries?
   - **Recommendation**: Separate binaries (clearer purpose)

5. **Version sync**: Keep versions in sync with Chisel?
   - **Recommendation**: Independent versioning

## Future Enhancements

### Phase 2 Features (Post-v0.1.0)

1. **Podman Kube Integration**
   - Export executed pipeline as Kubernetes YAML
   - `{project} kube generate pipelinerun.yaml`
   - Enables migration path to cluster

2. **Remote Podman Support**
   - Connect to remote Podman instances
   - `--host ssh://user@remote/run/podman/podman.sock`

3. **Quadlet Integration**
   - Generate systemd unit files for pipelines
   - Long-running pipeline services

4. **Better Caching**
   - Implement content-addressed volume naming
   - Reuse volumes based on input hashing

5. **Podman Compose Integration**
   - Multi-container setups via compose files

## Documentation Plan

### README.md
- Project overview and motivation
- Installation instructions (Podman + project)
- Quick start (5-minute tutorial)
- Comparison with Chisel
- Architecture overview

### docs/
- `installation.md` - Detailed installation for all platforms
- `architecture.md` - Deep dive into Podman integration
- `examples.md` - All examples explained
- `migration.md` - Moving from Chisel
- `troubleshooting.md` - Common issues and solutions
- `contributing.md` - Development setup

### Example Comments
- Add comments to all example YAML files
- Explain what each example demonstrates
- Include expected output

## Timeline Summary

| Phase | Duration | Deliverable |
|-------|----------|-------------|
| 1. Foundation | 2 weeks | Basic execution |
| 2. Workspaces | 1 week | Volume management |
| 3. Core Executor | 1 week | Multi-step tasks |
| 4. Results & Pipelines | 1 week | Full pipelines |
| 5. Sidecars | 1 week | Sidecar support |
| 6. Advanced Features | 1 week | Feature parity |
| 7. Polish | 1 week | Production ready |
| **Total** | **8 weeks** | **v0.1.0 release** |

## Conclusion

This project provides:
- **Technical exploration** of Podman as Tekton execution backend
- **Alternative option** for users who prefer Podman
- **Proof of concept** for pluggable backends
- **Learning opportunity** for Podman API and container orchestration

It complements Chisel rather than replacing it, offering users choice based on their environment and preferences.

---

**Next Steps:**
1. Choose project name from suggestions
2. Decide on repository structure (monorepo vs separate)
3. Set up initial project structure
4. Begin Phase 1 implementation
