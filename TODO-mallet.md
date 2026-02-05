# TODO: Mallet Implementation

Implement Mallet - a Podman-based Tekton executor as part of the chisel monorepo.

**Updated Plan:** `~/.config/claude/history/research/2026-02/PLAN-mallet-implementation-v2.md`
**Original Research:** `~/.config/claude/history/research/2026-02/RESEARCH-podman-migration.md`

## Overview

Transform chisel into a monorepo supporting two backends:
- **Chisel** - Dagger backend (existing, ~8,135 LOC)
- **Mallet** - Podman backend (new, ~2,850 LOC to add)

**Current Code Reusability:** ~80% of codebase (4,810 LOC) is backend-agnostic
**Must Reimplement:** ~20% of codebase (527 LOC) is Dagger-specific

## Implementation Phases

### Phase 1: Backend Abstraction Layer ⏳
**Timeline:** Week 1
**Status:** Not started
**Estimated LOC:** ~800 LOC (refactoring + new)

- [ ] Create `pkg/backend/` package (~150 LOC)
  - [ ] Define `Backend` interface in `backend.go`
  - [ ] Define `StepRequest`, `StepResult`, `SidecarRequest`, `WorkspaceMount` types in `types.go`
  - [ ] Add comprehensive godoc comments
- [ ] Create `pkg/orchestrator/` package (~600 LOC moved)
  - [ ] Move `pkg/executor/scheduler.go` → `pkg/orchestrator/scheduler.go`
  - [ ] Move `pkg/executor/matrix.go` → `pkg/orchestrator/matrix.go`
  - [ ] Move `pkg/executor/when.go` → `pkg/orchestrator/when.go`
  - [ ] Move `pkg/executor/retry.go` → `pkg/orchestrator/retry.go`
  - [ ] Move `pkg/executor/timeout.go` → `pkg/orchestrator/timeout.go`
  - [ ] Move `pkg/executor/steptemplate.go` → `pkg/orchestrator/steptemplate.go`
  - [ ] Move `pkg/executor/volumes.go` → `pkg/orchestrator/volumes.go`
  - [ ] Move `pkg/executor/results.go` → `pkg/orchestrator/results.go`
  - [ ] Extract substitution logic from executor.go → `substitution.go`
  - [ ] Update package declarations and imports
- [ ] Create `pkg/backend/dagger/` package (~550 LOC refactored)
  - [ ] Move and refactor `pkg/executor/executor.go` to implement `Backend` interface
  - [ ] Move `pkg/executor/sidecar.go` → `pkg/backend/dagger/sidecar.go`
  - [ ] Implement `ExecuteStep()`, `StartSidecar()`, `StopSidecar()`, `ReadResult()`, `Cleanup()`
- [ ] Update Chisel CLI (~50 LOC changed)
  - [ ] Update `cmd/chisel/run.go` to use `pkg/backend/dagger.Backend`
  - [ ] Update imports to use `pkg/orchestrator`
- [ ] **Verify:** All existing tests pass, all examples work, no Chisel regressions

### Phase 2: Mallet CLI Stub ⏳
**Timeline:** Week 2
**Status:** Not started
**Estimated LOC:** ~500 LOC (copied/adapted)

- [ ] Create `cmd/mallet/` directory (~400 LOC copied)
  - [ ] Copy and adapt `cmd/chisel/main.go` → `cmd/mallet/main.go`
  - [ ] Copy and adapt `cmd/chisel/root.go` → `cmd/mallet/root.go`
  - [ ] Copy and adapt `cmd/chisel/run.go` → `cmd/mallet/run.go` (use podman backend)
  - [ ] Copy and adapt `cmd/chisel/workspace.go` → `cmd/mallet/workspace.go`
  - [ ] Update package names and help text
- [ ] Create stub `pkg/backend/podman/` package (~100 LOC)
  - [ ] Create `executor.go` with stub `Backend` implementation
  - [ ] Implement all interface methods returning "not implemented" errors
- [ ] Create `Makefile` with build targets
- [ ] Update `.goreleaser.yaml` for multi-binary builds
- [ ] Update root `README.md` to mention both tools
- [ ] Create `cmd/mallet/README.md`
- [ ] **Verify:** Both binaries compile, mallet fails gracefully with "not implemented"

### Phase 3: Podman Core Implementation ⏳
**Timeline:** Weeks 3-4
**Status:** Not started
**Estimated LOC:** ~900 LOC

- [ ] Add Podman dependencies
  - [ ] `go get github.com/containers/podman/v5/pkg/bindings`
  - [ ] `go get github.com/containers/storage`
  - [ ] `go get github.com/opencontainers/runtime-spec/specs-go`
- [ ] Implement `pkg/backend/podman/client.go` (~150 LOC)
  - [ ] Connection to Podman socket
  - [ ] Auto-detect socket path (user vs system)
  - [ ] Service availability checks
  - [ ] Clear error messages if service not running
- [ ] Implement `pkg/backend/podman/container.go` (~300 LOC)
  - [ ] Container creation using `specgen.NewSpecGenerator()`
  - [ ] Container lifecycle: create → start → wait → cleanup
  - [ ] Log streaming to capture stdout/stderr
  - [ ] Timeout handling with context cancellation
- [ ] Implement `pkg/backend/podman/workspace.go` (~200 LOC)
  - [ ] EmptyDir → Podman named volumes
  - [ ] Local → Bind mounts
  - [ ] PVC → Named volumes
  - [ ] Volume cleanup
- [ ] Implement `pkg/backend/podman/executor.go` (~250 LOC)
  - [ ] Build `SpecGenerator` from `StepRequest`
  - [ ] Execute and capture output
  - [ ] Return `StepResult`
  - [ ] Timeout support
- [ ] Unit tests with mock Podman service
- [ ] Integration tests with real Podman
- [ ] **Verify:** Can execute single-step tasks, workspaces work, no resource leaks

### Phase 4: Full Orchestration Integration ⏳
**Timeline:** Week 5
**Status:** Not started
**Estimated LOC:** ~150 LOC

- [ ] Implement `pkg/backend/podman/results.go` (~100 LOC)
  - [ ] Read files from `/tekton/results/` using `podman exec cat`
  - [ ] Handle missing result files gracefully
- [ ] Integration with orchestrator (~50 LOC changes)
  - [ ] Update `cmd/mallet/run.go` to use full orchestrator
  - [ ] Hook up scheduler for parallel task execution
  - [ ] Integrate matrix expansion, when clauses, retry logic
- [ ] Test multi-step tasks, pipelines, results, parallel execution, matrix builds
- [ ] **Verify:** Full pipeline execution with result passing works

### Phase 5: Sidecars & Advanced Features ⏳
**Timeline:** Week 6
**Status:** Not started
**Estimated LOC:** ~300 LOC

- [ ] Implement `pkg/backend/podman/sidecar.go` (~200 LOC)
  - [ ] Create Podman Pod for task with sidecars
  - [ ] Start sidecar containers in pod
  - [ ] Attach step containers to same pod
  - [ ] Service discovery via pod network
  - [ ] Sidecar cleanup
- [ ] Implement `pkg/backend/podman/cleanup.go` (~100 LOC)
  - [ ] Track created resources (containers, pods, volumes)
  - [ ] Cleanup on success/failure/interrupt (Ctrl+C)
- [ ] Improve error handling
  - [ ] Better error messages for Podman service issues
  - [ ] Container inspection on failure
  - [ ] Helpful suggestions
- [ ] Test sidecars, cleanup, interrupt handling
- [ ] **Verify:** Full feature parity with Chisel, all examples work

### Phase 6: Polish, Documentation & Release ⏳
**Timeline:** Week 7
**Status:** Not started
**Estimated LOC:** ~200 LOC

- [ ] Performance optimization
  - [ ] Connection pooling
  - [ ] Parallel container cleanup
  - [ ] Volume reuse strategies
- [ ] Documentation
  - [ ] Update root `README.md` (overview, when to use each tool, installation)
  - [ ] Update `cmd/mallet/README.md` (Podman setup, troubleshooting)
  - [ ] Create `docs/ARCHITECTURE.md` (backend abstraction design)
  - [ ] Create `docs/MIGRATION.md` (Chisel → Mallet guide)
- [ ] CI/CD updates
  - [ ] Update `.github/workflows/test.yaml` (build/test both)
  - [ ] Update `.github/workflows/release.yaml` (release both)
  - [ ] Update `.goreleaser.yaml` (multi-arch builds)
- [ ] Example updates
  - [ ] Add comments to all YAML files
  - [ ] Create `examples/README.md`
- [ ] Final testing
  - [ ] All examples with both tools
  - [ ] Cross-platform testing
  - [ ] Load testing, memory leak testing
- [ ] **Verify:** Mallet v0.1.0 ready for release

## Success Criteria

### Functional Requirements
- [ ] Chisel continues working with zero regressions
- [ ] All 20+ example PipelineRuns execute successfully with Mallet
- [ ] Output matches Chisel (except execution timing)
- [ ] No resource leaks (containers, pods, volumes cleaned up)
- [ ] Graceful error handling and Ctrl+C cleanup

### Performance Requirements
- [ ] Startup time < 2 seconds (first run)
- [ ] Execution time within 30% of Chisel for same workload
- [ ] Memory usage < 500MB for typical pipeline

### Code Quality Requirements
- [ ] All packages have >70% test coverage
- [ ] All linters pass (`golangci-lint run`)
- [ ] All tests pass (`go test ./...`)
- [ ] Code reuse ~80% between tools

### User Experience Requirements
- [ ] Clear installation instructions
- [ ] Complete `--help` documentation
- [ ] Actionable error messages
- [ ] Examples run out of the box
- [ ] Binary sizes: Chisel ~21MB, Mallet ~15MB

## Quick Reference

**Build:**
```bash
make build          # Build both
make build-chisel   # Build chisel only
make build-mallet   # Build mallet only
```

**Test:**
```bash
./chisel run examples/simple/hello-pipelinerun.yaml
./mallet run examples/simple/hello-pipelinerun.yaml
```

**Architecture:**
```
pkg/
  types/           # Shared: Data structures
  parser/          # Shared: YAML parsing
  ui/              # Shared: Logging
  orchestrator/    # Shared: Orchestration logic
  backend/
    backend.go     # Interface
    dagger/        # Chisel implementation
    podman/        # Mallet implementation

cmd/
  chisel/          # Dagger backend CLI
  mallet/          # Podman backend CLI
```

## Dependencies

### New (Mallet Only)
- `github.com/containers/podman/v5/pkg/bindings` (~3MB)
- `github.com/containers/common/libnetwork/types`
- `github.com/containers/storage`
- `github.com/opencontainers/runtime-spec/specs-go`

### Shared (Already Present)
- `dagger.io/dagger` (Chisel only)
- `github.com/spf13/cobra`
- `gopkg.in/yaml.v3`
- `github.com/go-git/go-git/v5` (Git resolver)
- `github.com/google/go-containerregistry` (Bundles resolver)

## Timeline

| Phase | Duration | LOC | Deliverable |
|-------|----------|-----|-------------|
| 1. Backend Abstraction | Week 1 | ~800 | Chisel refactored |
| 2. Mallet CLI Stub | Week 2 | ~500 | Mallet compiles |
| 3. Podman Core | Weeks 3-4 | ~900 | Basic execution |
| 4. Orchestration | Week 5 | ~150 | Full pipelines |
| 5. Sidecars & Cleanup | Week 6 | ~300 | Feature parity |
| 6. Polish & Release | Week 7 | ~200 | v0.1.0 |
| **Total** | **7 weeks** | **~2,850** | **Production ready** |

## Notes

- **Updated Plan (Feb 5, 2026):** `~/.config/claude/history/research/2026-02/PLAN-mallet-implementation-v2.md`
- **Original Research:** `~/.config/claude/history/research/2026-02/RESEARCH-podman-migration.md`
- **Current Codebase:** 8,135 LOC total (80% reusable, 20% Dagger-specific)
- **Estimated Completion:** 2026-03-26 (7 weeks from start)
