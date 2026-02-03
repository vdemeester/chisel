package executor

import (
	"testing"

	"github.com/vdemeester/chisel/pkg/types"
)

func TestCaptureResults_SingleResult(t *testing.T) {
	// Simulate a task that writes a result to /tekton/results/commit-sha
	// The result should be readable via $(tasks.taskname.results.commit-sha)

	results := make(map[string]string)
	resultSpecs := []types.ResultSpec{
		{Name: "commit-sha", Description: "The git commit SHA"},
	}

	// Simulate reading from /tekton/results directory
	resultFiles := map[string]string{
		"commit-sha": "abc123def456",
	}

	captureResults(resultSpecs, resultFiles, results)

	if results["commit-sha"] != "abc123def456" {
		t.Errorf("Expected commit-sha=abc123def456, got %s", results["commit-sha"])
	}
}

func TestCaptureResults_MultipleResults(t *testing.T) {
	results := make(map[string]string)
	resultSpecs := []types.ResultSpec{
		{Name: "image-url"},
		{Name: "image-digest"},
		{Name: "build-time"},
	}

	resultFiles := map[string]string{
		"image-url":    "gcr.io/myproject/myimage:v1.0.0",
		"image-digest": "sha256:abcdef123456",
		"build-time":   "2024-01-15T10:30:00Z",
	}

	captureResults(resultSpecs, resultFiles, results)

	if results["image-url"] != "gcr.io/myproject/myimage:v1.0.0" {
		t.Errorf("image-url mismatch: got %s", results["image-url"])
	}
	if results["image-digest"] != "sha256:abcdef123456" {
		t.Errorf("image-digest mismatch: got %s", results["image-digest"])
	}
	if results["build-time"] != "2024-01-15T10:30:00Z" {
		t.Errorf("build-time mismatch: got %s", results["build-time"])
	}
}

func TestCaptureResults_MissingResult(t *testing.T) {
	// If a declared result is not present in the files, it should not be added
	results := make(map[string]string)
	resultSpecs := []types.ResultSpec{
		{Name: "required-result"},
		{Name: "optional-result"},
	}

	resultFiles := map[string]string{
		"required-result": "value",
		// optional-result is missing
	}

	captureResults(resultSpecs, resultFiles, results)

	if results["required-result"] != "value" {
		t.Errorf("required-result should be captured")
	}
	if _, ok := results["optional-result"]; ok {
		t.Errorf("optional-result should not be present")
	}
}

func TestCaptureResults_TrimWhitespace(t *testing.T) {
	// Result values should have leading/trailing whitespace trimmed
	results := make(map[string]string)
	resultSpecs := []types.ResultSpec{
		{Name: "trimmed"},
	}

	resultFiles := map[string]string{
		"trimmed": "  value with spaces  \n",
	}

	captureResults(resultSpecs, resultFiles, results)

	if results["trimmed"] != "value with spaces" {
		t.Errorf("Expected trimmed value, got %q", results["trimmed"])
	}
}

func TestSubstituteResults(t *testing.T) {
	// Test that $(tasks.taskname.results.resultname) is substituted correctly
	e := &Executor{
		results: map[string]map[string]string{
			"build": {
				"image-url":    "gcr.io/myproject/myimage:v1.0.0",
				"image-digest": "sha256:abc123",
			},
		},
	}

	input := "Image: $(tasks.build.results.image-url)@$(tasks.build.results.image-digest)"
	task := &types.ResolvedTask{}
	pr := &types.ResolvedPipelineRun{}

	result := e.substituteVariables(input, task, pr)
	expected := "Image: gcr.io/myproject/myimage:v1.0.0@sha256:abc123"

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}
