# Remote Task Resolution Examples

This directory contains examples demonstrating Chisel's remote task resolution capabilities.

## HTTP Resolver

The HTTP resolver fetches task definitions from HTTP/HTTPS URLs.

### Example: Using Tekton Catalog Tasks

```yaml
taskRef:
  resolver: http
  params:
  - name: url
    value: https://raw.githubusercontent.com/tektoncd/catalog/main/task/git-clone/0.9/git-clone.yaml
```

### Run the example:

```bash
chisel run examples/resolvers/http-resolver-pipelinerun.yaml
```

This example:
1. Fetches the `git-clone` task from the Tekton Catalog via HTTP
2. Clones the tektoncd/pipeline repository
3. Lists the files in the cloned repository

### HTTP Resolver Parameters

| Parameter | Required | Description |
|-----------|----------|-------------|
| `url` | Yes | HTTP/HTTPS URL to the task YAML file |
| `http-username` | No | Username for basic authentication |
| `http-password` | No | Password for basic authentication |

### Features

- **Caching**: Tasks are cached in memory to avoid redundant HTTP requests
- **Authentication**: Supports HTTP basic auth for private endpoints
- **Error Handling**: Clear error messages for network failures, 404s, invalid YAML

### Use Cases

1. **Community Tasks**: Use tasks from Tekton Catalog or other public repositories
   ```yaml
   taskRef:
     resolver: http
     params:
     - name: url
       value: https://raw.githubusercontent.com/tektoncd/catalog/main/task/buildpacks/0.6/buildpacks.yaml
   ```

2. **Internal Task Server**: Fetch from internal HTTP server
   ```yaml
   taskRef:
     resolver: http
     params:
     - name: url
       value: https://tasks.company.internal/security-scan.yaml
     - name: http-username
       value: ci-user
     - name: http-password
       value: ${SECRET_PASSWORD}
   ```

3. **GitHub Raw URLs**: Direct links to task files
   ```yaml
   taskRef:
     resolver: http
     params:
     - name: url
       value: https://raw.githubusercontent.com/org/repo/main/tasks/my-task.yaml
   ```

## Git Resolver

The Git resolver clones Git repositories and extracts task definitions from specific paths.

### Example: Using Tekton Catalog via Git

```yaml
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

### Run the example:

```bash
chisel run examples/resolvers/git-resolver-pipelinerun.yaml
```

This example:
1. Fetches the `git-clone` task from Tekton Catalog via Git
2. Fetches the `buildah` task from Tekton Catalog
3. Demonstrates multiple git-resolved tasks in one pipeline

### Git Resolver Parameters

| Parameter | Required | Description |
|-----------|----------|-------------|
| `url` | Yes | Git repository URL (HTTPS or SSH) |
| `revision` | Yes | Branch name, tag, or commit SHA |
| `pathInRepo` | Yes | Path to task YAML within repository |

### Features

- **Caching**: Cloned repositories are cached to avoid redundant clones
- **Shallow Clone**: Uses depth=1 for efficiency (except for commit SHAs)
- **Multiple Revisions**: Supports branches, tags, and commit SHAs
- **Error Handling**: Clear messages for clone failures, missing paths

### Use Cases

1. **Tekton Catalog**: Access community-maintained tasks
   ```yaml
   taskRef:
     resolver: git
     params:
     - name: url
       value: https://github.com/tektoncd/catalog.git
     - name: revision
       value: main
     - name: pathInRepo
       value: task/kaniko/0.6/kaniko.yaml
   ```

2. **Private Repositories**: Use your company's task library
   ```yaml
   taskRef:
     resolver: git
     params:
     - name: url
       value: https://github.com/company/tekton-tasks.git
     - name: revision
       value: v1.2.3
     - name: pathInRepo
       value: tasks/security-scan.yaml
   ```

3. **Version Pinning**: Pin to specific commits for reproducibility
   ```yaml
   taskRef:
     resolver: git
     params:
     - name: url
       value: https://github.com/tektoncd/catalog.git
     - name: revision
       value: abc123def456  # Specific commit SHA
     - name: pathInRepo
       value: task/git-clone/0.9/git-clone.yaml
   ```

## Hub Resolver (Artifact Hub)

The Hub resolver fetches tasks from Artifact Hub, providing catalog discovery and version management.

### Example: Using Artifact Hub Catalog

```yaml
taskRef:
  resolver: hub
  params:
  - name: name
    value: git-clone
  - name: version
    value: "0.9"
  - name: catalog
    value: tekton-catalog-tasks
  - name: type
    value: artifact
```

### Run the example:

```bash
chisel run examples/resolvers/hub-resolver-pipelinerun.yaml
```

This example:
1. Fetches the `git-clone` task from Artifact Hub
2. Fetches the `buildah` task from Artifact Hub
3. Demonstrates catalog-based task discovery

### Hub Resolver Parameters

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `name` | Yes | - | Task name in the catalog |
| `version` | Yes | - | Task version (exact version string) |
| `catalog` | No | `tekton-catalog-tasks` | Catalog name on Artifact Hub |
| `type` | No | `artifact` | Hub type (`artifact` recommended, `tekton` deprecated) |
| `kind` | No | `task` | Resource kind (`task` or `pipeline`) |

### Features

- **Catalog Discovery**: Browse available tasks on Artifact Hub
- **Version Management**: Specify exact versions
- **Caching**: Tasks cached by catalog/name@version
- **Default Catalog**: Uses `tekton-catalog-tasks` by default

### Use Cases

1. **Community Tasks**: Access curated Tekton tasks
   ```yaml
   taskRef:
     resolver: hub
     params:
     - name: name
       value: kaniko
     - name: version
       value: "0.6"
   ```

2. **Custom Catalogs**: Use your organization's Artifact Hub catalog
   ```yaml
   taskRef:
     resolver: hub
     params:
     - name: name
       value: security-scan
     - name: version
       value: "1.2.0"
     - name: catalog
       value: mycompany-tekton-tasks
   ```

3. **Version Pinning**: Ensure reproducible builds
   ```yaml
   taskRef:
     resolver: hub
     params:
     - name: name
       value: golang-build
     - name: version
       value: "0.3.0"  # Exact version
   ```

### Artifact Hub API

The Hub resolver queries the Artifact Hub API:
```
GET https://artifacthub.io/api/v1/packages/{catalog}/{name}/{version}
```

Tasks are extracted from the `data.task` field in the JSON response.

## Bundles Resolver (OCI Artifacts)

The Bundles resolver fetches tasks from OCI registries as Tekton Bundles. This is the most production-ready approach for versioned task distribution.

### Example: Using Tekton Catalog Bundles

```yaml
taskRef:
  resolver: bundles
  params:
  - name: bundle
    value: ghcr.io/tektoncd/catalog/upstream/tasks/git-clone:656d45176d5dafcfecf8253132f5a8642bb125c3
  - name: name
    value: git-clone
  - name: kind
    value: task
```

### Run the example:

**NOTE:** Tekton Catalog bundles require GHCR authentication even for public packages.

**Option 1: Run with GitHub authentication**
```bash
# Authenticate with GitHub Container Registry
echo $GITHUB_TOKEN | docker login ghcr.io -u <your-username> --password-stdin

# Run the example
chisel run examples/resolvers/bundles-resolver-pipelinerun.yaml
```

**Option 2: Create a local test bundle (no auth required)**
```bash
# Use the helper script to create a bundle in local registry
./examples/resolvers/create-test-bundle.sh localhost:5000 v1

# Then modify the example to use localhost:5000 bundle reference
```

This example demonstrates:
1. Using Tekton Catalog tasks from GHCR
2. Bundle reference format with commit SHA tags
3. Multiple bundle-based tasks in one pipeline

### Bundles Resolver Parameters

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `bundle` | Yes | - | OCI image reference (registry/image:tag or registry/image@digest) |
| `name` | Yes | - | Task name to extract from bundle |
| `kind` | No | `task` | Resource kind (`task` or `pipeline`) |

### Features

- **OCI Registry Support**: Works with any OCI-compliant registry (Docker Hub, GHCR, GCR, ECR, ACR)
- **Authentication**: Uses `~/.docker/config.json` for registry credentials
- **Content Addressing**: Supports both tags and SHA256 digests for immutable references
- **Caching**: Bundles are cached by reference to avoid redundant pulls

### Use Cases

1. **Tekton Catalog Bundles**: Use official Tekton catalog tasks (requires GHCR auth)
   ```yaml
   taskRef:
     resolver: bundles
     params:
     - name: bundle
       # Format: ghcr.io/tektoncd/catalog/upstream/tasks/<task-name>:<commit-sha>
       value: ghcr.io/tektoncd/catalog/upstream/tasks/golang-test:656d45176d5dafcfecf8253132f5a8642bb125c3
     - name: name
       value: golang-test
   ```

   Browse available tasks: https://github.com/orgs/tektoncd/packages?repo_name=catalog

2. **Private Registry**: Use your organization's private bundles
   ```yaml
   taskRef:
     resolver: bundles
     params:
     - name: bundle
       value: ghcr.io/myorg/tekton-tasks/security-scan:v1.2.0
     - name: name
       value: security-scan
   ```

3. **Immutable References**: Pin to specific digests for reproducibility
   ```yaml
   taskRef:
     resolver: bundles
     params:
     - name: bundle
       value: gcr.io/tekton-releases/catalog/upstream/git-clone@sha256:abc123...
     - name: name
       value: git-clone
   ```

### Authentication

The Bundles resolver uses the standard Docker authentication:

1. **Default**: Reads from `~/.docker/config.json`
2. **Login**: Use `docker login registry.io` to authenticate
3. **Environment**: Supports standard Docker credential helpers

For private registries:
```bash
# Docker Hub
docker login

# GHCR (GitHub Container Registry)
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin

# GCR (Google Container Registry)
gcloud auth configure-docker

# ECR (AWS Container Registry)
aws ecr get-login-password | docker login --username AWS --password-stdin $ECR_REGISTRY
```

## Combining Local and Remote Tasks

You can mix local and remote task references in the same pipeline:

```yaml
spec:
  pipelineSpec:
    tasks:
    - name: local-task
      taskRef:
        name: my-local-task  # Loaded from filesystem
    - name: remote-task
      taskRef:
        resolver: http      # Fetched from URL
        params:
        - name: url
          value: https://example.com/task.yaml
```

## Security Considerations

- HTTP resolver validates URLs to prevent SSRF attacks
- HTTPS is recommended for production use
- Be cautious with authentication credentials in YAML (use environment variables or secrets)
- Verify the source of remote tasks before execution
