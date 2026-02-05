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

## Future Resolvers

The following resolvers are planned:

- **Hub Resolver**: Fetch tasks from Artifact Hub with version constraints
- **Bundles Resolver**: Pull tasks from OCI registries as Tekton Bundles

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
