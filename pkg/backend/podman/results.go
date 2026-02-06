package podman

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResultsDir is the path where Tekton steps write their results.
const ResultsDir = "/tekton/results"

// ReadResultFromPath reads a result file from a local path.
// This is used to read results from a mounted temp directory after container execution.
func ReadResultFromPath(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read result file %s: %w", path, err)
	}
	return strings.TrimSpace(string(data)), nil
}

// CollectResults reads all result files from a directory.
// Returns a map of result name to value.
func CollectResults(dir string) (map[string]string, error) {
	results := make(map[string]string)

	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return results, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read results directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		path := filepath.Join(dir, name)

		content, err := ReadResultFromPath(path)
		if err != nil {
			// Log warning but continue collecting other results
			continue
		}

		results[name] = content
	}

	return results, nil
}
