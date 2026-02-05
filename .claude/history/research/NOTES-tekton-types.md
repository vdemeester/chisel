# Using tektoncd/pipeline Types

## Current Architecture

Chisel currently maintains **two sets of types**:

1. **Parser types** (`pkg/parser/parser.go`)
   - Custom structs: `TektonPipelineRun`, `TektonTask`, `TektonStep`, etc.
   - Used for YAML unmarshaling (~500 lines)
   - Manually kept in sync with Tekton spec

2. **Internal types** (`pkg/types/types.go`)
   - Simplified execution types: `ResolvedPipelineRun`, `ResolvedTask`, `Step`, etc.
   - Optimized for local execution (no cluster-specific fields)
   - Conversion functions bridge parser → internal types

## Option: Use Official tektoncd/pipeline Types

The official `github.com/tektoncd/pipeline/pkg/apis/pipeline/v1` package provides:

- Production-ready types: `PipelineRun`, `Pipeline`, `Task`, `Step`, `Param`, etc.
- Full YAML/JSON unmarshaling support
- Built-in validation and default value setting
- Always in sync with Tekton spec (currently v1.9.0 LTS)

### Tradeoffs

**Pros:**
- ✅ Official types, guaranteed Tekton spec compatibility
- ✅ Eliminates ~500 lines of parser struct definitions
- ✅ Automatic validation and defaults via `SetDefaults()` and `Validate()`
- ✅ Future-proof as Tekton adds features
- ✅ Better ecosystem integration

**Cons:**
- ❌ Heavy dependencies (full Kubernetes stack: k8s.io/api, k8s.io/apimachinery, k8s.io/client-go)
- ❌ Binary size increase (~10-20MB for K8s dependencies)
- ❌ Types include cluster-specific fields (status, conditions, ownerRefs) not needed locally
- ❌ More complex type hierarchy

## Possible Approaches

### Option 1: Hybrid (Recommended for Future)

Use official types for **parsing**, keep simplified types for **execution**:

```go
// Parse with official types
import pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"

var pr pipelinev1.PipelineRun
yaml.Unmarshal(data, &pr)

// Convert to internal simplified types
resolved := convertToResolvedPipelineRun(pr)
```

**Benefits:**
- Official parsing with validation
- Lightweight execution layer
- Balanced dependency footprint

### Option 2: Full Adoption

Use `tektoncd/pipeline` types throughout, skip conversion:

```go
import pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"

func Execute(pr *pipelinev1.PipelineRun) { ... }
```

**Benefits:**
- Maximum compatibility
- Less code to maintain

**Drawbacks:**
- Larger binary
- Heavier memory footprint
- Couples execution to K8s types

### Option 3: Status Quo

Keep custom types for minimal dependencies:

**Benefits:**
- Minimal dependencies (current: gopkg.in/yaml.v3)
- Small binary size
- Full control over types

**Drawbacks:**
- Manual sync with Tekton spec
- Duplicate type definitions
- No automatic validation

## Decision

**Current choice: Option 3 (Status Quo)**

Rationale: Chisel prioritizes minimal dependencies and small binary size for local development use. The parser types are stable and cover the implemented feature set.

**Future consideration: Option 1 (Hybrid)** if:
- Tekton spec evolves significantly
- Validation becomes complex
- Maintaining custom types becomes burdensome

## References

- Official types: `github.com/tektoncd/pipeline/pkg/apis/pipeline/v1`
- Latest version: v1.9.0 LTS (released Jan 2026)
- Docs: https://pkg.go.dev/github.com/tektoncd/pipeline
- Spec: https://tekton.dev/docs/pipelines/
