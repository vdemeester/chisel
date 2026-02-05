package orchestrator

import (
	"strings"

	"github.com/vdemeester/chisel/pkg/types"
)

// CaptureResults reads result values from the resultFiles map and stores them
// in the results map. Only results declared in resultSpecs are captured.
// Values are trimmed of leading/trailing whitespace.
func CaptureResults(resultSpecs []types.ResultSpec, resultFiles map[string]string, results map[string]string) {
	for _, spec := range resultSpecs {
		if value, ok := resultFiles[spec.Name]; ok {
			results[spec.Name] = strings.TrimSpace(value)
		}
	}
}
