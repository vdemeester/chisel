package parser

import (
	"bytes"
	"context"
	"io"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

// TestBundlesResolver tests loading tasks from OCI bundles
func TestBundlesResolver(t *testing.T) {
	// Sample Tekton task YAML
	taskYAML := `apiVersion: tekton.dev/v1
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
`

	// Create a mock OCI registry server
	s := httptest.NewServer(registry.New())
	defer s.Close()

	// Parse registry URL
	u, err := url.Parse(s.URL)
	if err != nil {
		t.Fatalf("failed to parse test server URL: %v", err)
	}

	// Create a bundle reference pointing to our mock registry
	bundleRef := u.Host + "/tekton/git-clone:0.9"

	// Create an OCI image with the Tekton task as a layer
	img := empty.Image

	// Add the task YAML as a layer using LayerFromOpener
	layer, err := tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader([]byte(taskYAML))), nil
	})
	if err != nil {
		t.Fatalf("failed to create layer: %v", err)
	}

	img, err = mutate.AppendLayers(img, layer)
	if err != nil {
		t.Fatalf("failed to append layer: %v", err)
	}

	// Push the image to the mock registry
	ref, err := name.ParseReference(bundleRef)
	if err != nil {
		t.Fatalf("failed to parse reference: %v", err)
	}

	err = remote.Write(ref, img)
	if err != nil {
		t.Fatalf("failed to push bundle: %v", err)
	}

	// Now test resolving the bundle
	parser := New(Options{})

	params := []TektonParam{
		{Name: "bundle", Value: bundleRef},
		{Name: "name", Value: "git-clone"},
		{Name: "kind", Value: "Task"},
	}

	task, err := parser.resolveBundles(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if task.Metadata.Name != "git-clone" {
		t.Errorf("expected task name git-clone, got %s", task.Metadata.Name)
	}
}

// TestBundlesResolverWithCache tests that bundles are cached by digest
func TestBundlesResolverWithCache(t *testing.T) {
	// Sample Tekton task YAML
	taskYAML := `apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: buildah
spec:
  steps:
  - name: build
    image: quay.io/buildah/stable
`

	// Create a mock OCI registry server
	s := httptest.NewServer(registry.New())
	defer s.Close()

	// Parse registry URL
	u, err := url.Parse(s.URL)
	if err != nil {
		t.Fatalf("failed to parse test server URL: %v", err)
	}

	// Create and push bundle
	bundleRef := u.Host + "/tekton/buildah:0.6"
	img := empty.Image
	layer, err := tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader([]byte(taskYAML))), nil
	})
	if err != nil {
		t.Fatalf("failed to create layer: %v", err)
	}

	img, err = mutate.AppendLayers(img, layer)
	if err != nil {
		t.Fatalf("failed to append layer: %v", err)
	}

	ref, err := name.ParseReference(bundleRef)
	if err != nil {
		t.Fatalf("failed to parse reference: %v", err)
	}

	err = remote.Write(ref, img)
	if err != nil {
		t.Fatalf("failed to push bundle: %v", err)
	}

	parser := New(Options{})
	params := []TektonParam{
		{Name: "bundle", Value: bundleRef},
		{Name: "name", Value: "buildah"},
	}

	// First call - should fetch from registry
	task1, err := parser.resolveBundles(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error on first call: %v", err)
	}

	if task1.Metadata.Name != "buildah" {
		t.Errorf("expected task name buildah, got %s", task1.Metadata.Name)
	}

	// Second call - should use cache
	task2, err := parser.resolveBundles(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error on cached call: %v", err)
	}

	// Should be the same task (from cache)
	if task1 != task2 {
		t.Error("expected cached task to be same object")
	}
}

// TestBundlesResolverErrors tests error handling
func TestBundlesResolverErrors(t *testing.T) {
	tests := []struct {
		name        string
		params      []TektonParam
		expectError bool
	}{
		{
			name: "missing bundle parameter",
			params: []TektonParam{
				{Name: "name", Value: "git-clone"},
			},
			expectError: true,
		},
		{
			name: "missing name parameter",
			params: []TektonParam{
				{Name: "bundle", Value: "localhost:5000/tekton/git-clone:0.9"},
			},
			expectError: true,
		},
		{
			name: "invalid bundle format",
			params: []TektonParam{
				{Name: "bundle", Value: "not-a-valid-bundle"},
				{Name: "name", Value: "git-clone"},
			},
			expectError: true,
		},
		{
			name: "nonexistent registry",
			params: []TektonParam{
				{Name: "bundle", Value: "nonexistent.registry.io/tekton/task:v1"},
				{Name: "name", Value: "git-clone"},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := New(Options{})

			_, err := parser.resolveBundles(context.Background(), tt.params)

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
		})
	}
}

// TestBundlesResolverAuthentication tests bundle auth handling
func TestBundlesResolverAuthentication(t *testing.T) {
	t.Skip("TODO: Implement authentication test")
}

// TestBundlesResolverMultipleResources tests extracting specific resource by name
func TestBundlesResolverMultipleResources(t *testing.T) {
	t.Skip("TODO: Implement multiple resources test")
}

// TestBundlesResolverKindParameter tests kind parameter handling
func TestBundlesResolverKindParameter(t *testing.T) {
	tests := []struct {
		name       string
		params     []TektonParam
		expectKind string
	}{
		{
			name: "default kind is task",
			params: []TektonParam{
				{Name: "bundle", Value: "localhost:5000/tekton/resource:v1"},
				{Name: "name", Value: "my-resource"},
			},
			expectKind: "task",
		},
		{
			name: "explicit kind pipeline",
			params: []TektonParam{
				{Name: "bundle", Value: "localhost:5000/tekton/resource:v1"},
				{Name: "name", Value: "my-resource"},
				{Name: "kind", Value: "pipeline"},
			},
			expectKind: "pipeline",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test will verify that kind parameter is properly handled
			// For now, skip as implementation doesn't exist yet
			t.Skip("TODO: Implement kind parameter test")
		})
	}
}
