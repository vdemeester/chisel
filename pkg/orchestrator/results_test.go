package orchestrator

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

	CaptureResults(resultSpecs, resultFiles, results)

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

	CaptureResults(resultSpecs, resultFiles, results)

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

	CaptureResults(resultSpecs, resultFiles, results)

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

	CaptureResults(resultSpecs, resultFiles, results)

	if results["trimmed"] != "value with spaces" {
		t.Errorf("Expected trimmed value, got %q", results["trimmed"])
	}
}

// TestSubstituteResults moved to executor package temporarily
// Will be refactored when extracting substitution logic to orchestrator
