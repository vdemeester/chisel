package parser

import (
	"context"
	"fmt"
	"io"
	"net/http"

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

// resolveTaskWithResolver resolves a task using the specified resolver
func (p *Parser) resolveTaskWithResolver(ref *TektonTaskRef) (*TektonTask, error) {
	ctx := context.Background()

	switch ref.Resolver {
	case "http":
		return p.resolveHTTP(ctx, ref.Params)
	case "git":
		return nil, fmt.Errorf("git resolver not yet implemented")
	case "bundles":
		return nil, fmt.Errorf("bundles resolver not yet implemented")
	case "hub":
		return nil, fmt.Errorf("hub resolver not yet implemented")
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
