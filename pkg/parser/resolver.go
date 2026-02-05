package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"gopkg.in/yaml.v3"
)

// resolveHTTP fetches a task from an HTTP/HTTPS URL
func (p *Parser) resolveHTTP(ctx context.Context, params []TektonParam) (*TektonTask, error) {
	// Extract URL parameter
	url := getParamValue(params, "url")
	if url == "" {
		return nil, fmt.Errorf("http resolver requires 'url' parameter")
	}

	// Check cache first (cache key is the URL)
	if task, ok := p.taskCache[url]; ok {
		return task, nil
	}

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Add basic authentication if provided
	username := getParamValue(params, "http-username")
	password := getParamValue(params, "http-password")
	if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}

	// Perform the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch task from %s: %w", url, err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: failed to fetch task from %s", resp.StatusCode, url)
	}

	// Read response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse YAML
	var task TektonTask
	if err := yaml.Unmarshal(data, &task); err != nil {
		return nil, fmt.Errorf("failed to parse task YAML from %s: %w", url, err)
	}

	// Cache the task
	p.taskCache[url] = &task

	return &task, nil
}

// resolveGit clones a Git repository and fetches a task from a specific path
func (p *Parser) resolveGit(ctx context.Context, params []TektonParam) (*TektonTask, error) {
	// Extract required parameters
	url := getParamValue(params, "url")
	if url == "" {
		return nil, fmt.Errorf("git resolver requires 'url' parameter")
	}

	revision := getParamValue(params, "revision")
	if revision == "" {
		return nil, fmt.Errorf("git resolver requires 'revision' parameter")
	}

	pathInRepo := getParamValue(params, "pathInRepo")
	if pathInRepo == "" {
		return nil, fmt.Errorf("git resolver requires 'pathInRepo' parameter")
	}

	// Create cache key: url@revision/path
	cacheKey := fmt.Sprintf("%s@%s/%s", url, revision, pathInRepo)

	// Check cache first
	if task, ok := p.taskCache[cacheKey]; ok {
		return task, nil
	}

	// Create temporary directory for cloning
	tmpDir, err := os.MkdirTemp("", "chisel-git-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Clone options with shallow clone for efficiency
	cloneOpts := &git.CloneOptions{
		URL:           url,
		Depth:         1,
		SingleBranch:  true,
		Tags:          git.NoTags,
		Progress:      nil, // Suppress clone progress
	}

	// Try to parse revision as a branch/tag reference first
	// If it's a commit SHA, we'll handle that differently
	if len(revision) == 40 {
		// Likely a commit SHA - need full clone
		cloneOpts.Depth = 0
	} else {
		// Branch or tag
		cloneOpts.ReferenceName = plumbing.NewBranchReferenceName(revision)
	}

	// Perform the clone
	repo, err := git.PlainCloneContext(ctx, tmpDir, false, cloneOpts)
	if err != nil {
		// If branch failed, try as a tag
		cloneOpts.ReferenceName = plumbing.NewTagReferenceName(revision)
		repo, err = git.PlainCloneContext(ctx, tmpDir, false, cloneOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to clone repository %s: %w", url, err)
		}
	}

	// If revision is a commit SHA, checkout that specific commit
	if len(revision) == 40 {
		w, err := repo.Worktree()
		if err != nil {
			return nil, fmt.Errorf("failed to get worktree: %w", err)
		}

		err = w.Checkout(&git.CheckoutOptions{
			Hash: plumbing.NewHash(revision),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to checkout commit %s: %w", revision, err)
		}
	}

	// Read the task file from the cloned repository
	taskPath := filepath.Join(tmpDir, pathInRepo)
	data, err := os.ReadFile(taskPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read task file %s: %w", pathInRepo, err)
	}

	// Parse YAML
	var task TektonTask
	if err := yaml.Unmarshal(data, &task); err != nil {
		return nil, fmt.Errorf("failed to parse task YAML from %s: %w", pathInRepo, err)
	}

	// Cache the task
	p.taskCache[cacheKey] = &task

	return &task, nil
}

// resolveHub fetches a task from Artifact Hub
func (p *Parser) resolveHub(ctx context.Context, params []TektonParam) (*TektonTask, error) {
	return p.resolveHubWithBaseURL(ctx, params, "https://artifacthub.io/api/v1")
}

// resolveHubWithBaseURL fetches a task from Artifact Hub (with configurable base URL for testing)
func (p *Parser) resolveHubWithBaseURL(ctx context.Context, params []TektonParam, baseURL string) (*TektonTask, error) {
	// Extract required parameters
	name := getParamValue(params, "name")
	if name == "" {
		return nil, fmt.Errorf("hub resolver requires 'name' parameter")
	}

	version := getParamValue(params, "version")
	if version == "" {
		return nil, fmt.Errorf("hub resolver requires 'version' parameter")
	}

	// Extract optional parameters
	catalog := getParamValue(params, "catalog")
	if catalog == "" {
		catalog = "tekton-catalog-tasks" // Default catalog
	}

	// type parameter is optional (artifact vs tekton)
	// We'll use artifact type by default as it's the recommended approach
	hubType := getParamValue(params, "type")
	if hubType == "" {
		hubType = "artifact"
	}

	// kind parameter is optional (task vs pipeline)
	kind := getParamValue(params, "kind")
	if kind == "" {
		kind = "task"
	}

	// Create cache key: catalog/name@version
	cacheKey := fmt.Sprintf("hub:%s/%s@%s", catalog, name, version)

	// Check cache first
	if task, ok := p.taskCache[cacheKey]; ok {
		return task, nil
	}

	// Construct Artifact Hub API URL
	// Format: /packages/{catalog}/{name}/{version}
	url := fmt.Sprintf("%s/packages/%s/%s/%s", baseURL, catalog, name, version)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Perform request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from Artifact Hub: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Artifact Hub returned HTTP %d for %s@%s", resp.StatusCode, name, version)
	}

	// Read and parse JSON response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var pkgData struct {
		Data struct {
			Task string `json:"task"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &pkgData); err != nil {
		return nil, fmt.Errorf("failed to parse Artifact Hub response: %w", err)
	}

	// Extract task YAML from the data field
	taskYAML := pkgData.Data.Task
	if taskYAML == "" {
		return nil, fmt.Errorf("no task YAML found in Artifact Hub package %s@%s", name, version)
	}

	// Parse task YAML
	var task TektonTask
	if err := yaml.Unmarshal([]byte(taskYAML), &task); err != nil {
		return nil, fmt.Errorf("failed to parse task YAML from Artifact Hub: %w", err)
	}

	// Cache the task
	p.taskCache[cacheKey] = &task

	return &task, nil
}

// resolveTaskWithResolver resolves a task using the specified resolver
func (p *Parser) resolveTaskWithResolver(ref *TektonTaskRef) (*TektonTask, error) {
	ctx := context.Background()

	switch ref.Resolver {
	case "http":
		return p.resolveHTTP(ctx, ref.Params)
	case "git":
		return p.resolveGit(ctx, ref.Params)
	case "bundles":
		return nil, fmt.Errorf("bundles resolver not yet implemented")
	case "hub":
		return p.resolveHub(ctx, ref.Params)
	default:
		return nil, fmt.Errorf("unknown resolver: %s (supported: http, git, bundles, hub)", ref.Resolver)
	}
}

// getParamValue extracts a parameter value by name from a list of params
func getParamValue(params []TektonParam, name string) string {
	for _, param := range params {
		if param.Name == name {
			if strVal, ok := param.Value.(string); ok {
				return strVal
			}
		}
	}
	return ""
}
