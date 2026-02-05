package parser

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHubResolver tests loading tasks from Artifact Hub
func TestHubResolver(t *testing.T) {
	// Mock Artifact Hub API response
	mockPackageResponse := map[string]interface{}{
		"name":    "git-clone",
		"version": "0.9.0",
		"data": map[string]interface{}{
			"task": `apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: git-clone
spec:
  params:
  - name: url
    type: string
  steps:
  - name: clone
    image: gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init:v0.40.2
    script: |
      git clone $(params.url)
`,
		},
	}

	tests := []struct {
		name        string
		params      []TektonParam
		expectError bool
		expectName  string
	}{
		{
			name: "valid task from hub",
			params: []TektonParam{
				{Name: "name", Value: "git-clone"},
				{Name: "version", Value: "0.9.0"},
				{Name: "catalog", Value: "tekton-catalog-tasks"},
				{Name: "kind", Value: "task"},
			},
			expectError: false,
			expectName:  "git-clone",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(mockPackageResponse)
			}))
			defer server.Close()

			parser := New(Options{})

			// Override the API base URL for testing
			task, err := parser.resolveHubWithBaseURL(context.Background(), tt.params, server.URL)

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

// TestHubResolverWithCache tests that Hub resolver caches tasks
func TestHubResolverWithCache(t *testing.T) {
	mockPackageResponse := map[string]interface{}{
		"name":    "buildah",
		"version": "0.6.0",
		"data": map[string]interface{}{
			"task": `apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: buildah
spec:
  steps:
  - name: build
    image: quay.io/buildah/stable
`,
		},
	}

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(mockPackageResponse)
	}))
	defer server.Close()

	parser := New(Options{})
	params := []TektonParam{
		{Name: "name", Value: "buildah"},
		{Name: "version", Value: "0.6.0"},
		{Name: "catalog", Value: "tekton-catalog-tasks"},
	}

	// First call - should hit API
	_, err := parser.resolveHubWithBaseURL(context.Background(), params, server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if callCount != 1 {
		t.Errorf("expected 1 API call, got %d", callCount)
	}

	// Second call - should use cache
	_, err = parser.resolveHubWithBaseURL(context.Background(), params, server.URL)
	if err != nil {
		t.Fatalf("unexpected error on cached call: %v", err)
	}

	if callCount != 1 {
		t.Errorf("expected cache to be used (still 1 call), got %d calls", callCount)
	}
}

// TestHubResolverErrors tests error handling
func TestHubResolverErrors(t *testing.T) {
	tests := []struct {
		name        string
		params      []TektonParam
		statusCode  int
		expectError bool
	}{
		{
			name: "missing name parameter",
			params: []TektonParam{
				{Name: "version", Value: "0.9.0"},
			},
			expectError: true,
		},
		{
			name: "missing version parameter",
			params: []TektonParam{
				{Name: "name", Value: "git-clone"},
			},
			expectError: true,
		},
		{
			name: "404 not found",
			params: []TektonParam{
				{Name: "name", Value: "nonexistent"},
				{Name: "version", Value: "0.0.0"},
			},
			statusCode:  http.StatusNotFound,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.statusCode != 0 {
					w.WriteHeader(tt.statusCode)
				}
			}))
			defer server.Close()

			parser := New(Options{})

			_, err := parser.resolveHubWithBaseURL(context.Background(), tt.params, server.URL)

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
		})
	}
}

// TestHubResolverDefaultCatalog tests default catalog behavior
func TestHubResolverDefaultCatalog(t *testing.T) {
	mockPackageResponse := map[string]interface{}{
		"name":    "kaniko",
		"version": "0.6.0",
		"data": map[string]interface{}{
			"task": `apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: kaniko
spec:
  steps:
  - name: build
    image: gcr.io/kaniko-project/executor
`,
		},
	}

	var requestedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(mockPackageResponse)
	}))
	defer server.Close()

	parser := New(Options{})

	// Test without catalog parameter - should use default
	params := []TektonParam{
		{Name: "name", Value: "kaniko"},
		{Name: "version", Value: "0.6.0"},
	}

	task, err := parser.resolveHubWithBaseURL(context.Background(), params, server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if task.Metadata.Name != "kaniko" {
		t.Errorf("expected task name kaniko, got %s", task.Metadata.Name)
	}

	// Verify default catalog was used in URL
	if requestedPath != "/packages/tekton-catalog-tasks/kaniko/0.6.0" {
		t.Errorf("expected default catalog in path, got %s", requestedPath)
	}
}

// TestHubResolverWithType tests type parameter (artifact vs tekton)
func TestHubResolverWithType(t *testing.T) {
	mockPackageResponse := map[string]interface{}{
		"name":    "golang-test",
		"version": "0.2.0",
		"data": map[string]interface{}{
			"task": `apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: golang-test
spec:
  steps:
  - name: test
    image: golang:1.21
`,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(mockPackageResponse)
	}))
	defer server.Close()

	parser := New(Options{})

	// Test with type=artifact
	params := []TektonParam{
		{Name: "name", Value: "golang-test"},
		{Name: "version", Value: "0.2.0"},
		{Name: "type", Value: "artifact"},
	}

	task, err := parser.resolveHubWithBaseURL(context.Background(), params, server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if task.Metadata.Name != "golang-test" {
		t.Errorf("expected task name golang-test, got %s", task.Metadata.Name)
	}
}

// TestHubResolverInvalidJSON tests handling of invalid JSON responses
func TestHubResolverInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	parser := New(Options{})
	params := []TektonParam{
		{Name: "name", Value: "test"},
		{Name: "version", Value: "1.0.0"},
	}

	_, err := parser.resolveHubWithBaseURL(context.Background(), params, server.URL)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
