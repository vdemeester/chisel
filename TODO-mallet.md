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

### Phase 1: Backend Abstraction Layer ✅
**Timeline:** Week 1
**Status:** COMPLETE (PR #14 merged 2026-02-05)
**Actual LOC:** +811/-164

- [x] Create `pkg/backend/` package
  - [x] Define `Backend` interface in `backend.go`
  - [x] Define `StepRequest`, `StepResult`, `SidecarRequest`, `WorkspaceMount` types
  - [x] Add comprehensive godoc comments
- [x] Create `pkg/orchestrator/` package (~600 LOC moved)
  - [x] Move scheduler, matrix, when, retry, timeout, steptemplate, volumes, results
  - [x] Export all functions for backend use
  - [x] Update package declarations and imports
- [x] Create `pkg/backend/dagger/` package
  - [x] Refactor executor.go to implement `Backend` interface
  - [x] Move sidecar.go
  - [x] All interface methods implemented
- [x] Update Chisel CLI
  - [x] Update imports to use new backend structure
- [x] **Verified:** All 60+ tests pass, all examples work, zero regressions

### Phase 2: Mallet CLI Stub ✅
**Timeline:** Week 2
**Status:** COMPLETE (2026-02-06)
**Actual LOC:** +548 (4 commits)

- [x] Create `cmd/mallet/` directory
  - [x] main.go, root.go, run.go, workspace.go
  - [x] Uses Podman backend, shares parser/types/ui with chisel
- [x] Create stub `pkg/backend/podman/` package
  - [x] TDD: Tests written first (8 tests)
  - [x] All interface methods return ErrNotImplemented
- [x] Create `Makefile` with build targets
  - [x] build, build-chisel, build-mallet, test, lint, clean, install, check
- [x] Update `.goreleaser.yaml` for multi-binary builds
  - [x] Separate builds and archives for chisel and mallet
- [ ] Update root `README.md` to mention both tools
- [ ] Create `cmd/mallet/README.md`
- [x] **Verified:** Both binaries compile, mallet fails gracefully with "not implemented"
  - chisel: 29MB, mallet: 15MB
  - Dry-run works for both
  - mallet execution: "podman backend not yet implemented"

### Phase 3: Podman Core Implementation ✅
**Timeline:** Weeks 3-4
**Status:** COMPLETE (PR #16 merged 2026-02-06)
**Actual LOC:** +1,333/-69

- [x] Add Podman dependencies (pure HTTP API, no CGO)
- [x] Implement `pkg/backend/podman/client.go` (~115 LOC)
  - [x] Connection to Podman socket via HTTP
  - [x] Auto-detect socket path (user vs system)
  - [x] Service availability checks
  - [x] Clear error messages if service not running
- [x] Implement `pkg/backend/podman/container.go` (~479 LOC)
  - [x] Container creation via REST API
  - [x] Container lifecycle: create → start → wait → logs → cleanup
  - [x] Log streaming with multiplexed stdout/stderr parsing
  - [x] Timeout handling with context cancellation
- [x] Implement `pkg/backend/podman/executor.go` (~108 LOC)
  - [x] Build ContainerSpec from StepRequest
  - [x] Execute and capture output
  - [x] Return StepResult
  - [x] Timeout support
- [x] Integration tests with real Podman (skip when unavailable)
- [x] **Verified:** Can execute single-step tasks, hello-pipelinerun works (~700ms)

### Phase 4: Full Orchestration Integration ✅
**Timeline:** Week 5
**Status:** COMPLETE (PR #17, 2026-02-06)
**Actual LOC:** ~270 LOC

- [x] Implement `pkg/backend/podman/results.go` (~50 LOC)
  - [x] ReadResultFromPath and CollectResults helpers
  - [x] Mount temp directory to /tekton/results
  - [x] Collect results after container execution
- [x] Integration with orchestrator (~165 LOC changes)
  - [x] Use BuildDAG and ExecuteParallel for parallel execution
  - [x] Expand matrix tasks before building DAG
  - [x] Evaluate when conditions for conditional execution
  - [x] Pass results between dependent tasks
  - [x] Execute finally tasks after main pipeline
- [x] Add parameter substitution (~60 LOC)
  - [x] $(params.name) with string, array, object support
  - [x] $(tasks.taskname.results.resultname) for result passing
  - [x] $(workspaces.name.path) and $(context.*) variables
- [x] **Verified:** Full pipeline execution with result passing works

### Phase 5: Sidecars & Advanced Features ✅
**Timeline:** Week 6
**Status:** COMPLETE (PR #18, 2026-02-06)
**Actual LOC:** ~540 LOC

- [x] Implement `pkg/backend/podman/sidecar.go` (~285 LOC)
  - [x] Pod operations (CreatePod, StartPod, StopPod, RemovePod)
  - [x] CreateContainerInPod for running containers in pods
  - [x] StartSidecar and StopSidecar implementation
  - [x] Make container command optional (use image default)
- [x] Implement `pkg/backend/podman/cleanup.go` (~150 LOC)
  - [x] ResourceTracker for tracking containers, pods, volumes
  - [x] Thread-safe concurrent access
  - [x] CleanupAll with fresh context for cleanup
  - [x] RemoveVolume function
- [x] Test sidecars, cleanup
- [x] **Verified:** Sidecar lifecycle works, cleanup works

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
