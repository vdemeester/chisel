# TODO: Mallet Implementation

Implement Mallet - a Podman-based Tekton executor as part of the chisel monorepo.

**Full plan:** `.claude/history/research/PLAN-podman-project.md` and `/home/vincent/.claude/plans/floating-coalescing-planet.md`

## Overview

Transform chisel into a monorepo supporting two backends:
- **Chisel** - Dagger backend (existing)
- **Mallet** - Podman backend (new)

Both share ~85% of code (parser, types, UI, orchestration).

## Implementation Phases

### Phase 1: Backend Abstraction & Refactoring ⏳
**Timeline:** Week 1
**Status:** Not started

- [ ] Create `pkg/orchestrator/` package
  - [ ] Move scheduler.go, matrix.go, retry.go, timeout.go, when.go, steptemplate.go, results.go
  - [ ] Extract substitution.go from executor.go
- [ ] Create `pkg/backend/` package
  - [ ] Define Backend interface
  - [ ] Define StepRequest, StepResult, SidecarContext types
- [ ] Refactor Dagger executor
  - [ ] Move to pkg/backend/dagger/
  - [ ] Implement Backend interface
- [ ] Create orchestrator using Backend interface
- [ ] Update cmd/chisel to use new architecture
- [ ] **Verify:** All existing tests pass

### Phase 2: Mallet CLI Stub ⏳
**Timeline:** Week 2
**Status:** Not started

- [ ] Create cmd/mallet/ directory (copy from cmd/chisel/)
- [ ] Create stub pkg/backend/podman/executor.go
- [ ] Update .goreleaser.yaml for both binaries
- [ ] Create Makefile with build-chisel, build-mallet targets
- [ ] **Verify:** Both binaries compile

### Phase 3: Podman Core Implementation ⏳
**Timeline:** Weeks 3-4
**Status:** Not started

- [ ] Add Podman dependencies to go.mod
- [ ] Implement pkg/backend/podman/client.go (connection, service checks)
- [ ] Implement pkg/backend/podman/container.go (create, start, wait, logs, cleanup)
- [ ] Implement pkg/backend/podman/workspace.go (volumes, bind mounts)
- [ ] Implement pkg/backend/podman/executor.go (ExecuteStep)
- [ ] Add timeout support
- [ ] **Verify:** Basic pipelines execute successfully

### Phase 4: Sidecars & Results ⏳
**Timeline:** Week 5
**Status:** Not started

- [ ] Implement pkg/backend/podman/pods.go (Pod-based sidecars)
- [ ] Update ExecuteStep to join pod when sidecars present
- [ ] Implement pkg/backend/podman/results.go (exec-based file extraction)
- [ ] Implement pkg/backend/podman/cleanup.go (resource cleanup)
- [ ] **Verify:** Sidecar and result examples work

### Phase 5: Polish & Documentation ⏳
**Timeline:** Week 6
**Status:** Not started

- [ ] Improve error messages (Podman service not running, etc.)
- [ ] Update CI/CD workflows for both tools
- [ ] Update root README.md for monorepo
- [ ] Create cmd/mallet/README.md
- [ ] Create migration guide
- [ ] Create benchmark comparison script
- [ ] **Verify:** All examples work, CI passes, docs complete

## Success Criteria

- ✅ Chisel continues working with zero regressions
- ✅ Mallet executes all example pipelines successfully
- ✅ Code reuse >80% between tools
- ✅ Both tools pass CI/CD
- ✅ Documentation complete
- ✅ Binary sizes: Chisel ~21MB, Mallet ~13-15MB
- ✅ No resource leaks

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

## Notes

- Research completed in `.claude/history/research/RESEARCH-podman-migration.md`
- Estimated 6 weeks total implementation time
- ~2200 LOC new code (but 85% reused infrastructure)
