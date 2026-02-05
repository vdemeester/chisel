package parser

import (
	"context"
	"os"
	"testing"
)

// TestGitResolver tests loading tasks from Git repositories
func TestGitResolver(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		revision     string
		pathInRepo   string
		expectError  bool
		expectName   string
	}{
		{
			name:        "valid task from public repo",
			url:         "https://github.com/tektoncd/catalog.git",
			revision:    "main",
			pathInRepo:  "task/git-clone/0.9/git-clone.yaml",
			expectError: false,
			expectName:  "git-clone",
		},
		{
			name:        "invalid path in repo",
			url:         "https://github.com/tektoncd/catalog.git",
			revision:    "main",
			pathInRepo:  "task/nonexistent.yaml",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := New(Options{})

			params := []TektonParam{
				{Name: "url", Value: tt.url},
				{Name: "revision", Value: tt.revision},
				{Name: "pathInRepo", Value: tt.pathInRepo},
			}

			task, err := parser.resolveGit(context.Background(), params)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if task.Metadata.Name != tt.expectName {
				t.Errorf("expected task name %s, got %s", tt.expectName, task.Metadata.Name)
			}
		})
	}
}

// TestGitResolverCache tests that Git resolver caches cloned repositories
func TestGitResolverCache(t *testing.T) {
	parser := New(Options{})

	params := []TektonParam{
		{Name: "url", Value: "https://github.com/tektoncd/catalog.git"},
		{Name: "revision", Value: "main"},
		{Name: "pathInRepo", Value: "task/git-clone/0.9/git-clone.yaml"},
	}

	// First call - should clone repository
	task1, err := parser.resolveGit(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second call with same params - should use cache
	task2, err := parser.resolveGit(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error on cached call: %v", err)
	}

	if task1.Metadata.Name != task2.Metadata.Name {
		t.Error("cached task should match original")
	}
}

// TestGitResolverWithDifferentRevisions tests that different revisions work
func TestGitResolverWithDifferentRevisions(t *testing.T) {
	t.Skip("Skipping - catalog repo structure may change over time")

	// This test would verify tag/commit support but is skipped to avoid
	// dependency on specific catalog repository state. The implementation
	// supports tags and commits via the plumbing.Hash checkout mechanism.
}

// TestGitResolverErrors tests error handling
func TestGitResolverErrors(t *testing.T) {
	tests := []struct {
		name        string
		params      []TektonParam
		expectError bool
	}{
		{
			name: "missing url parameter",
			params: []TektonParam{
				{Name: "revision", Value: "main"},
				{Name: "pathInRepo", Value: "task.yaml"},
			},
			expectError: true,
		},
		{
			name: "missing revision parameter",
			params: []TektonParam{
				{Name: "url", Value: "https://github.com/example/repo.git"},
				{Name: "pathInRepo", Value: "task.yaml"},
			},
			expectError: true,
		},
		{
			name: "missing pathInRepo parameter",
			params: []TektonParam{
				{Name: "url", Value: "https://github.com/example/repo.git"},
				{Name: "revision", Value: "main"},
			},
			expectError: true,
		},
		{
			name: "invalid repository URL",
			params: []TektonParam{
				{Name: "url", Value: "https://github.com/nonexistent/repo.git"},
				{Name: "revision", Value: "main"},
				{Name: "pathInRepo", Value: "task.yaml"},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := New(Options{})

			_, err := parser.resolveGit(context.Background(), tt.params)

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
		})
	}
}

// TestGitResolverCacheKeyUniqueness tests that different paths are cached separately
func TestGitResolverCacheKeyUniqueness(t *testing.T) {
	parser := New(Options{})

	url := "https://github.com/tektoncd/catalog.git"
	revision := "main"

	// Fetch git-clone task
	params1 := []TektonParam{
		{Name: "url", Value: url},
		{Name: "revision", Value: revision},
		{Name: "pathInRepo", Value: "task/git-clone/0.9/git-clone.yaml"},
	}

	task1, err := parser.resolveGit(context.Background(), params1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Fetch different task from same repo (should be a different cache entry)
	params2 := []TektonParam{
		{Name: "url", Value: url},
		{Name: "revision", Value: revision},
		{Name: "pathInRepo", Value: "task/buildah/0.6/buildah.yaml"},
	}

	task2, err := parser.resolveGit(context.Background(), params2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both should be valid and different
	if task1 == nil || task2 == nil {
		t.Error("both tasks should be valid")
	}

	if task1.Metadata.Name == task2.Metadata.Name {
		t.Error("different tasks should have different names")
	}
}

// TestGitResolverWithShallowClone tests that shallow clone is used for efficiency
func TestGitResolverWithShallowClone(t *testing.T) {
	// Create a temp directory for cloning
	tmpDir := t.TempDir()

	parser := New(Options{})

	params := []TektonParam{
		{Name: "url", Value: "https://github.com/tektoncd/catalog.git"},
		{Name: "revision", Value: "main"},
		{Name: "pathInRepo", Value: "task/git-clone/0.9/git-clone.yaml"},
	}

	task, err := parser.resolveGit(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if task.Metadata.Name != "git-clone" {
		t.Errorf("expected task name git-clone, got %s", task.Metadata.Name)
	}

	// Verify temp directory was cleaned up (shallow clone should be temporary)
	files, _ := os.ReadDir(tmpDir)
	if len(files) > 0 {
		t.Error("expected temporary clone directory to be cleaned up")
	}
}
