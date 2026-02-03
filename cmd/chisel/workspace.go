package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vdemeester/chisel/pkg/types"
)

// parseWorkspaceBinding parses a workspace binding spec of the form "name:path"
// and returns the workspace name and absolute path.
func parseWorkspaceBinding(spec string) (name, path string, err error) {
	if spec == "" {
		return "", "", fmt.Errorf("empty workspace binding")
	}

	parts := strings.SplitN(spec, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid workspace binding %q: expected format name:path", spec)
	}

	name = parts[0]
	path = parts[1]

	if name == "" {
		return "", "", fmt.Errorf("invalid workspace binding %q: name cannot be empty", spec)
	}
	if path == "" {
		return "", "", fmt.Errorf("invalid workspace binding %q: path cannot be empty", spec)
	}

	// Resolve relative paths to absolute
	if !filepath.IsAbs(path) {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return "", "", fmt.Errorf("failed to resolve path %q: %w", path, err)
		}
		path = absPath
	}

	return name, path, nil
}

// parseWorkspaceBindings parses multiple workspace binding specs and returns
// a map of workspace name to absolute path.
func parseWorkspaceBindings(specs []string) (map[string]string, error) {
	result := make(map[string]string)

	for _, spec := range specs {
		name, path, err := parseWorkspaceBinding(spec)
		if err != nil {
			return nil, err
		}
		result[name] = path
	}

	return result, nil
}

// applyWorkspaceOverrides applies workspace binding overrides to a resolved pipeline run.
// It replaces existing workspace bindings with local directory bindings.
func applyWorkspaceOverrides(pr *types.ResolvedPipelineRun, overrides map[string]string) error {
	for name, path := range overrides {
		// Check if the workspace exists in the pipeline
		if _, exists := pr.Workspaces[name]; !exists {
			// Workspace doesn't exist yet - add it (useful for Tasks that declare workspaces)
			pr.Workspaces[name] = types.WorkspaceBinding{
				Name: name,
				Type: types.WorkspaceTypeLocal,
				Path: path,
			}
		} else {
			// Override existing workspace with local binding
			pr.Workspaces[name] = types.WorkspaceBinding{
				Name: name,
				Type: types.WorkspaceTypeLocal,
				Path: path,
			}
		}
	}
	return nil
}
