# Research: Remote Task Resolution in Chisel

## Current State

Chisel currently supports **only local filesystem-based task loading**:

- ✅ Inline `taskSpec` (tasks defined directly in pipeline YAML)
- ✅ Local `taskRef` (simple name lookup in filesystem)
- ❌ No remote resolution capabilities
- ❌ No HTTP/HTTPS URLs
- ❌ No Git repository references
- ❌ No OCI bundles (Tekton Bundles)
- ❌ No Artifact Hub integration

### Current Task Loading Logic

From `pkg/parser/parser.go:470-496`:

```go
func (p *Parser) loadTask(name, baseDir string) (*TektonTask, error) {
    // Check cache first
    if task, ok := p.taskCache[name]; ok {
        return task, nil
    }

    // Try file patterns in order
    patterns := []string{
        filepath.Join(baseDir, name+".yaml"),
        filepath.Join(baseDir, name+".yml"),
        filepath.Join(baseDir, "tasks", name+".yaml"),
        filepath.Join(baseDir, "tasks", name+".yml"),
    }

    // Read and unmarshal from filesystem
    for _, pattern := range patterns {
        if data, err := os.ReadFile(pattern); err == nil {
            var task TektonTask
            if err := yaml.Unmarshal(data, &task); err != nil {
                return nil, err
            }
            p.taskCache[name] = &task
            return &task, nil
        }
    }

    return nil, fmt.Errorf("task %s not found in %s", name, baseDir)
}
```

**Limitations:**
- Only filesystem paths (no URLs)
- No authentication
- No versioning
- No digest verification
- Simple in-memory cache only

### Current Data Structure

```go
// Line 119-121
TaskRef *struct {
    Name string `yaml:"name"`
} `yaml:"taskRef"`
```

This minimal structure only supports simple name-based references.

---

## What Tekton Pipelines Supports

Tekton Pipelines (v1.9.0+) includes four built-in resolvers enabled by default:

### 1. Bundles Resolver (OCI Registry)

Fetches tasks/pipelines from OCI image registries as Tekton Bundles.

**Example:**
```yaml
apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  name: bundle-demo
spec:
  pipelineRef:
    resolver: bundles
    params:
    - name: bundle
      value: gcr.io/tekton-releases/catalog/upstream/git-clone:0.9
    - name: name
      value: git-clone
    - name: kind
      value: task
```

**Parameters:**
- `bundle` (required): OCI image URL with digest or tag
- `name` (required): Resource name within the bundle
- `kind` (required): `task` or `pipeline`
- `cache` (optional): `always`, `never`, or `auto` (default: auto, caches for 1 hour)

**Benefits:**
- Content-addressable with SHA256 digests
- Registry authentication via docker config
- Immutable artifacts for supply chain security
- Works with any OCI-compliant registry (Docker Hub, GHCR, GCR, etc.)

### 2. Git Resolver

Clones Git repositories and fetches resources from specific paths.

**Example:**
```yaml
apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  name: git-demo
spec:
  taskRef:
    resolver: git
    params:
    - name: url
      value: https://github.com/tektoncd/catalog.git
    - name: revision
      value: main
    - name: pathInRepo
      value: task/git-clone/0.9/git-clone.yaml
```

**Parameters:**
- `url` (required): Git repository URL (HTTPS or SSH)
- `revision` (required): Branch, tag, or commit SHA
- `pathInRepo` (required): Path to task within repository
- `token` (optional): Personal access token for private repos
- `tokenKey` (optional): Key in secret for token
- `scmType` (optional): `github`, `gitlab`, `gitea`, `bitbucket-server`, `bitbucket-cloud`

**Alternative API-based syntax** (doesn't clone, uses GitHub API):
```yaml
params:
- name: org
  value: tektoncd
- name: repo
  value: catalog
- name: revision
  value: main
- name: pathInRepo
  value: task/git-clone/0.9/git-clone.yaml
- name: token
  value: github_pat_...
```

### 3. HTTP Resolver

Fetches task definitions directly from HTTP/HTTPS URLs.

**Example:**
```yaml
apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  name: http-demo
spec:
  taskRef:
    resolver: http
    params:
    - name: url
      value: https://raw.githubusercontent.com/tektoncd/catalog/main/task/git-clone/0.9/git-clone.yaml
```

**Parameters:**
- `url` (required): HTTP/HTTPS URL to task YAML
- `http-username` (optional): Basic auth username
- `http-password-secret` (optional): Secret containing password
- `http-password-secret-key` (optional): Key within secret (default: `password`)

**Use cases:**
- Quick prototyping
- Internal task servers
- Static file hosting (S3, GCS, GitHub raw URLs)

### 4. Hub Resolver (Artifact Hub)

Discovers and fetches tasks from Artifact Hub or Tekton Hub catalogs.

**Example:**
```yaml
apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  name: hub-demo
spec:
  taskRef:
    resolver: hub
    params:
    - name: name
      value: git-clone
    - name: version
      value: "0.9"
    - name: type
      value: artifact
    - name: catalog
      value: tekton-catalog-tasks
```

**Parameters:**
- `name` (required): Task identifier
- `version` (required): Version string or constraint
- `type` (optional): `artifact` (Artifact Hub, recommended) or `tekton` (deprecated Tekton Hub)
- `catalog` (optional): Source catalog name
- `kind` (optional): `task` or `pipeline`

**Version constraints** (when using Artifact Hub):
- Exact: `"0.9"`
- Range: `">=0.7.0, <2.0.0"`
- Wildcard: `"0.9.x"`

---

## Implementation Options for Chisel

### Option 1: Full Resolver Support

Implement all four Tekton resolvers for maximum compatibility.

**Pros:**
- Full Tekton compatibility
- Supports all community patterns
- Enables supply chain security (Bundles with digests)
- Access to Tekton Catalog via Hub

**Cons:**
- Significant implementation effort (~1500-2000 LOC)
- Adds dependencies (OCI client, git client, HTTP client)
- Requires authentication handling
- Caching complexity

**Estimated effort:** 3-4 weeks

### Option 2: HTTP + Git Only

Implement the two simplest resolvers.

**Pros:**
- Covers most use cases
- Simpler than OCI/Hub
- Enables remote task reuse
- Lower dependency footprint

**Cons:**
- No Artifact Hub integration
- No OCI bundle support (less secure)
- Community tasks require raw GitHub URLs

**Estimated effort:** 1-2 weeks

### Option 3: HTTP Only

Minimal remote resolution support.

**Pros:**
- Simplest implementation (~200-300 LOC)
- No external dependencies beyond net/http
- Covers basic remote task loading

**Cons:**
- Manual URL management (no catalog discovery)
- No versioning support
- No git integration

**Estimated effort:** 3-5 days

### Option 4: Bundle Resolver via Dagger

Leverage Dagger's existing OCI support.

**Pros:**
- Dagger already handles OCI registries
- Reuses existing authentication
- Natural fit for Chisel's architecture

**Cons:**
- Dagger-specific (won't work for Mallet)
- Still requires task extraction logic
- Limited to OCI bundles only

**Estimated effort:** 1 week

---

## Recommended Approach

### Phase 1: HTTP Resolver (Quick Win)

Start with HTTP resolver as proof of concept:

**Benefits:**
- Minimal code (~200 LOC)
- Immediate value for remote tasks
- No new dependencies
- Easy to test

**Implementation:**

1. **Extend TaskRef struct:**
```go
type TektonTaskRef struct {
    Name     string        `yaml:"name"`     // For simple name lookup
    Resolver string        `yaml:"resolver"` // "http", "git", "bundles", "hub"
    Params   []TektonParam `yaml:"params"`   // Resolver parameters
}
```

2. **Add HTTP resolver:**
```go
func (p *Parser) resolveHTTP(params []TektonParam) (*TektonTask, error) {
    url := getParamValue(params, "url")

    resp, err := http.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    data, _ := io.ReadAll(resp.Body)

    var task TektonTask
    yaml.Unmarshal(data, &task)
    return &task, nil
}
```

3. **Update loadTask to check resolver:**
```go
if pt.TaskRef.Resolver == "http" {
    return p.resolveHTTP(pt.TaskRef.Params)
}
// Fall back to filesystem
```

### Phase 2: Git Resolver

Add Git support using `go-git`:

```go
import "github.com/go-git/go-git/v5"

func (p *Parser) resolveGit(params []TektonParam) (*TektonTask, error) {
    url := getParamValue(params, "url")
    revision := getParamValue(params, "revision")
    path := getParamValue(params, "pathInRepo")

    // Clone to temp dir
    tmpDir, _ := os.MkdirTemp("", "chisel-git-")
    defer os.RemoveAll(tmpDir)

    repo, _ := git.PlainClone(tmpDir, false, &git.CloneOptions{
        URL:           url,
        ReferenceName: plumbing.NewBranchReferenceName(revision),
        Depth:         1,
    })

    // Read task file
    taskPath := filepath.Join(tmpDir, path)
    data, _ := os.ReadFile(taskPath)

    var task TektonTask
    yaml.Unmarshal(data, &task)
    return &task, nil
}
```

### Phase 3: Hub Resolver (Optional)

Integrate with Artifact Hub API:

```go
func (p *Parser) resolveHub(params []TektonParam) (*TektonTask, error) {
    name := getParamValue(params, "name")
    version := getParamValue(params, "version")
    catalog := getParamValue(params, "catalog", "tekton-catalog-tasks")

    // Query Artifact Hub API
    url := fmt.Sprintf("https://artifacthub.io/api/v1/packages/%s/%s/%s",
        catalog, name, version)

    resp, _ := http.Get(url)
    var pkg ArtifactHubPackage
    json.NewDecoder(resp.Body).Decode(&pkg)

    // Fetch task YAML from content URL
    taskResp, _ := http.Get(pkg.ContentURL)
    data, _ := io.ReadAll(taskResp.Body)

    var task TektonTask
    yaml.Unmarshal(data, &task)
    return &task, nil
}
```

### Phase 4: Bundles Resolver (Advanced)

Use containerd or OCI client libraries:

```go
import "github.com/google/go-containerregistry/pkg/v1/remote"

func (p *Parser) resolveBundle(params []TektonParam) (*TektonTask, error) {
    bundle := getParamValue(params, "bundle")
    name := getParamValue(params, "name")

    // Fetch OCI image
    ref, _ := remote.Get(bundle)
    img, _ := ref.Image()

    // Extract manifest, find task layer
    manifest, _ := img.Manifest()
    for _, layer := range manifest.Layers {
        if layer.Annotations["dev.tekton.image.name"] == name {
            rc, _ := layer.Compressed()
            data, _ := io.ReadAll(rc)

            var task TektonTask
            yaml.Unmarshal(data, &task)
            return &task, nil
        }
    }

    return nil, fmt.Errorf("task %s not found in bundle", name)
}
```

---

## Authentication Handling

### HTTP Basic Auth
```go
req, _ := http.NewRequest("GET", url, nil)
req.SetBasicAuth(username, password)
resp, _ := http.DefaultClient.Do(req)
```

### Git Authentication
```go
git.PlainClone(tmpDir, false, &git.CloneOptions{
    URL: url,
    Auth: &http.BasicAuth{
        Username: "git",
        Password: token,
    },
})
```

### OCI Registry Auth
```go
import "github.com/google/go-containerregistry/pkg/authn"

ref, _ := remote.Get(bundle, remote.WithAuth(authn.DefaultKeychain))
```

---

## Caching Strategy

### Current: In-Memory Only
```go
taskCache map[string]*TektonTask
```

### Proposed: Persistent Cache

```go
type CacheEntry struct {
    Task       *TektonTask
    URL        string
    FetchedAt  time.Time
    Digest     string // For bundles
}

type TaskCache struct {
    entries map[string]*CacheEntry
    dir     string // ~/.cache/chisel/tasks/
}

func (c *TaskCache) Get(key string) (*TektonTask, bool) {
    entry, ok := c.entries[key]
    if !ok {
        return nil, false
    }

    // Check if cached entry is still valid (1 hour TTL)
    if time.Since(entry.FetchedAt) > time.Hour {
        return nil, false
    }

    return entry.Task, true
}
```

**Cache key strategies:**
- HTTP: URL as key
- Git: `{url}@{revision}/{path}` as key
- Bundles: Digest as key (immutable)
- Hub: `{catalog}/{name}@{version}` as key

---

## Testing Strategy

### Unit Tests
```go
func TestHTTPResolver(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte(sampleTaskYAML))
    }))
    defer server.Close()

    parser := New(Options{})
    task, err := parser.resolveHTTP([]TektonParam{
        {Name: "url", Value: server.URL},
    })

    assert.NoError(t, err)
    assert.Equal(t, "sample-task", task.Metadata.Name)
}
```

### Integration Tests
```bash
# Test HTTP resolver
./chisel run examples/remote/http-taskref.yaml

# Test Git resolver
./chisel run examples/remote/git-taskref.yaml

# Test Hub resolver
./chisel run examples/remote/hub-taskref.yaml
```

---

## Example YAMLs

### HTTP Resolver Example
```yaml
# examples/remote/http-taskref.yaml
apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  name: http-example
spec:
  pipelineSpec:
    tasks:
    - name: clone
      taskRef:
        resolver: http
        params:
        - name: url
          value: https://raw.githubusercontent.com/tektoncd/catalog/main/task/git-clone/0.9/git-clone.yaml
      params:
      - name: url
        value: https://github.com/example/repo
```

### Git Resolver Example
```yaml
# examples/remote/git-taskref.yaml
apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  name: git-example
spec:
  pipelineSpec:
    tasks:
    - name: clone
      taskRef:
        resolver: git
        params:
        - name: url
          value: https://github.com/tektoncd/catalog.git
        - name: revision
          value: main
        - name: pathInRepo
          value: task/git-clone/0.9/git-clone.yaml
```

---

## Dependencies

### HTTP Resolver
- None (uses stdlib `net/http`)

### Git Resolver
- `github.com/go-git/go-git/v5` (~5MB)

### Hub Resolver
- None (uses `net/http` + `encoding/json`)

### Bundles Resolver
- `github.com/google/go-containerregistry` (~10MB)
- OR leverage Dagger's OCI support (Chisel only)

---

## Migration Path

### Step 1: Add resolver field support
- Update TaskRef struct
- Parser handles resolver field
- Backward compatible (resolver is optional)

### Step 2: Implement HTTP resolver
- Add resolveHTTP function
- Basic caching
- Error handling

### Step 3: Add Git resolver
- Import go-git
- Clone + extract logic
- Auth support

### Step 4: (Optional) Hub/Bundles
- Based on user demand
- More complex, defer to later

---

## Security Considerations

1. **URL Validation**: Prevent SSRF attacks
   - Reject localhost URLs
   - Reject private IP ranges
   - Configurable allow-list

2. **YAML Validation**: Prevent injection
   - Validate task structure
   - Reject malformed YAML
   - Size limits (prevent DoS)

3. **Credential Storage**: Secure handling
   - Never log credentials
   - Use OS keychain where possible
   - Support Kubernetes secrets (future)

4. **Digest Verification**: For bundles
   - Require SHA256 digests in production
   - Warn on tag-based references

---

## Recommendation

**Implement in this order:**

1. **Phase 1:** HTTP Resolver (1 week)
   - Quick win, immediate value
   - No dependencies
   - Enables remote task loading

2. **Phase 2:** Git Resolver (1 week)
   - Access to Tekton Catalog
   - Versioned task definitions
   - Community standard

3. **Phase 3:** Hub Resolver (Optional, 3-5 days)
   - If Artifact Hub becomes popular
   - Catalog discovery UX
   - Version constraints

4. **Phase 4:** Bundles (Future)
   - When supply chain security becomes priority
   - Requires OCI infrastructure
   - Can leverage Dagger for Chisel

**Total effort:** 2-3 weeks for HTTP + Git resolvers

---

## References

- [TEP-0060: Remote Resource Resolution](https://github.com/tektoncd/community/blob/main/teps/0060-remote-resource-resolution.md)
- [TEP-0005: Tekton OCI Bundles](https://github.com/tektoncd/community/blob/main/teps/0005-tekton-oci-bundles.md)
- [Tekton Bundles Resolver Docs](https://tekton.dev/docs/pipelines/bundle-resolver/)
- [Tekton Git Resolver Docs](https://tekton.dev/docs/pipelines/git-resolver/)
- [Tekton HTTP Resolver Docs](https://tekton.dev/docs/pipelines/http-resolver/)
- [Tekton Hub Resolver Docs](https://tekton.dev/docs/pipelines/hub-resolver/)
- [Artifact Hub API](https://artifacthub.io/docs/api/)
