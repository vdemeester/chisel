package parser

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHTTPResolver tests loading tasks from HTTP URLs
func TestHTTPResolver(t *testing.T) {
	sampleTaskYAML := `apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: http-test-task
spec:
  params:
  - name: url
    type: string
  steps:
  - name: fetch
    image: alpine
    script: |
      echo "Fetching $(params.url)"
`

	tests := []struct {
		name        string
		taskYAML    string
		expectError bool
		expectName  string
	}{
		{
			name:        "valid task from HTTP",
			taskYAML:    sampleTaskYAML,
			expectError: false,
			expectName:  "http-test-task",
		},
		{
			name:        "invalid YAML",
			taskYAML:    "invalid: yaml: structure:",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test HTTP server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/x-yaml")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.taskYAML))
			}))
			defer server.Close()

			parser := New(Options{})

			// Create task ref with HTTP resolver
			params := []TektonParam{
				{Name: "url", Value: server.URL},
			}

			task, err := parser.resolveHTTP(context.Background(), params)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
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

// TestHTTPResolverWithCache tests that HTTP resolver caches tasks
func TestHTTPResolverWithCache(t *testing.T) {
	sampleTaskYAML := `apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: cached-task
spec:
  steps:
  - name: test
    image: alpine
`

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/x-yaml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sampleTaskYAML))
	}))
	defer server.Close()

	parser := New(Options{})
	params := []TektonParam{
		{Name: "url", Value: server.URL},
	}

	// First call - should hit server
	_, err := parser.resolveHTTP(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if callCount != 1 {
		t.Errorf("expected 1 HTTP call, got %d", callCount)
	}

	// Second call - should use cache
	_, err = parser.resolveHTTP(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error on cached call: %v", err)
	}

	if callCount != 1 {
		t.Errorf("expected cache to be used (still 1 call), got %d calls", callCount)
	}
}

// TestHTTPResolverErrors tests error handling
func TestHTTPResolverErrors(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectError   bool
		errorContains string
	}{
		{
			name:          "404 not found",
			statusCode:    http.StatusNotFound,
			responseBody:  "Not Found",
			expectError:   true,
			errorContains: "404",
		},
		{
			name:          "500 server error",
			statusCode:    http.StatusInternalServerError,
			responseBody:  "Internal Server Error",
			expectError:   true,
			errorContains: "500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			parser := New(Options{})
			params := []TektonParam{
				{Name: "url", Value: server.URL},
			}

			_, err := parser.resolveHTTP(context.Background(), params)

			if !tt.expectError {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Errorf("expected error containing %q, got none", tt.errorContains)
				return
			}

			// Note: We'll check error message contains expected text once implemented
		})
	}
}

// TestHTTPResolverWithAuth tests basic authentication
func TestHTTPResolverWithAuth(t *testing.T) {
	sampleTaskYAML := `apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: auth-task
spec:
  steps:
  - name: test
    image: alpine
`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for basic auth
		username, password, ok := r.BasicAuth()
		if !ok || username != "testuser" || password != "testpass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/x-yaml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sampleTaskYAML))
	}))
	defer server.Close()

	parser := New(Options{})

	// Test without auth - should fail
	paramsNoAuth := []TektonParam{
		{Name: "url", Value: server.URL},
	}
	_, err := parser.resolveHTTP(context.Background(), paramsNoAuth)
	if err == nil {
		t.Error("expected authentication error, got none")
	}

	// Test with auth - should succeed
	paramsWithAuth := []TektonParam{
		{Name: "url", Value: server.URL},
		{Name: "http-username", Value: "testuser"},
		{Name: "http-password", Value: "testpass"},
	}
	task, err := parser.resolveHTTP(context.Background(), paramsWithAuth)
	if err != nil {
		t.Fatalf("unexpected error with auth: %v", err)
	}

	if task.Metadata.Name != "auth-task" {
		t.Errorf("expected task name auth-task, got %s", task.Metadata.Name)
	}
}
