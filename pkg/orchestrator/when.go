package orchestrator

import (
	"strings"

	"github.com/vdemeester/chisel/pkg/types"
)

// EvaluateWhen evaluates all when expressions for a task.
// Returns true if the task should run, false if it should be skipped.
// All expressions must evaluate to true for the task to run.
func EvaluateWhen(task *types.ResolvedTask, params map[string]types.ParamValue, results map[string]map[string]string) bool {
	if len(task.When) == 0 {
		return true
	}

	for _, expr := range task.When {
		if !evaluateExpression(expr, params, results) {
			return false
		}
	}

	return true
}

// evaluateExpression evaluates a single when expression.
func evaluateExpression(expr types.WhenExpression, params map[string]types.ParamValue, results map[string]map[string]string) bool {
	// Substitute variables in input
	input := substituteWhenInput(expr.Input, params, results)

	switch expr.Operator {
	case "in":
		return contains(expr.Values, input)
	case "notin":
		return !contains(expr.Values, input)
	default:
		// Unknown operator - treat as false for safety
		return false
	}
}

// substituteWhenInput replaces variable references in the input string.
func substituteWhenInput(input string, params map[string]types.ParamValue, results map[string]map[string]string) string {
	result := input

	// Replace $(params.name) with param values
	for name, value := range params {
		if value.Type == types.ParamTypeString {
			placeholder := "$(params." + name + ")"
			result = strings.ReplaceAll(result, placeholder, value.StringVal)
		}
	}

	// Replace $(tasks.taskname.results.resultname) with result values
	for taskName, taskResults := range results {
		for resultName, resultValue := range taskResults {
			placeholder := "$(tasks." + taskName + ".results." + resultName + ")"
			result = strings.ReplaceAll(result, placeholder, resultValue)
		}
	}

	return result
}

// contains checks if a slice contains a value.
func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}
