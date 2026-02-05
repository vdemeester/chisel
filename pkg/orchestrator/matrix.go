package orchestrator

import (
	"fmt"

	"github.com/vdemeester/chisel/pkg/types"
)

// generateMatrixCombinations generates all parameter combinations from a matrix.
// For example, matrix with params [{name: "a", values: ["1", "2"]}, {name: "b", values: ["x", "y"]}]
// produces: [{"a": "1", "b": "x"}, {"a": "1", "b": "y"}, {"a": "2", "b": "x"}, {"a": "2", "b": "y"}]
func generateMatrixCombinations(matrix types.Matrix) []map[string]string {
	if len(matrix.Params) == 0 {
		return nil
	}

	// Start with a single empty combination
	combinations := []map[string]string{{}}

	// For each parameter, expand all existing combinations
	for _, param := range matrix.Params {
		var newCombinations []map[string]string
		for _, combo := range combinations {
			for _, value := range param.Values {
				// Clone the existing combination and add this parameter value
				newCombo := make(map[string]string)
				for k, v := range combo {
					newCombo[k] = v
				}
				newCombo[param.Name] = value
				newCombinations = append(newCombinations, newCombo)
			}
		}
		combinations = newCombinations
	}

	return combinations
}

// expandMatrixTask expands a task with a matrix into multiple task instances.
// Each instance gets a unique name and the matrix parameters added to its params.
// Tasks without a matrix are returned unchanged in a single-element slice.
func expandMatrixTask(task types.ResolvedTask) []types.ResolvedTask {
	if task.Matrix == nil || len(task.Matrix.Params) == 0 {
		return []types.ResolvedTask{task}
	}

	combinations := generateMatrixCombinations(*task.Matrix)
	expanded := make([]types.ResolvedTask, 0, len(combinations))

	for i, combo := range combinations {
		// Clone the task
		newTask := cloneTask(task)

		// Generate unique name
		newTask.Name = fmt.Sprintf("%s-%d", task.Name, i)

		// Add matrix parameters to task params
		if newTask.Params == nil {
			newTask.Params = make(map[string]types.ParamValue)
		}
		for name, value := range combo {
			newTask.Params[name] = types.ParamValue{
				Type:      types.ParamTypeString,
				StringVal: value,
			}
		}

		// Clear the matrix on expanded tasks
		newTask.Matrix = nil

		expanded = append(expanded, newTask)
	}

	return expanded
}

// ExpandAllMatrixTasks expands all tasks with matrices in a slice.
func ExpandAllMatrixTasks(tasks []types.ResolvedTask) []types.ResolvedTask {
	var expanded []types.ResolvedTask
	for _, task := range tasks {
		expanded = append(expanded, expandMatrixTask(task)...)
	}
	return expanded
}

// cloneTask creates a deep copy of a task.
func cloneTask(task types.ResolvedTask) types.ResolvedTask {
	clone := types.ResolvedTask{
		Name:       task.Name,
		TaskName:   task.TaskName,
		RunAfter:   make([]string, len(task.RunAfter)),
		Params:     make(map[string]types.ParamValue),
		Workspaces: make(map[string]string),
	}

	// Copy RunAfter
	copy(clone.RunAfter, task.RunAfter)

	// Copy Params
	for k, v := range task.Params {
		clone.Params[k] = v
	}

	// Copy Workspaces
	for k, v := range task.Workspaces {
		clone.Workspaces[k] = v
	}

	// Copy Steps (shallow copy is fine as we don't modify steps)
	clone.Steps = make([]types.Step, len(task.Steps))
	copy(clone.Steps, task.Steps)

	// Copy Results
	clone.Results = make([]types.ResultSpec, len(task.Results))
	copy(clone.Results, task.Results)

	// Copy Sidecars
	clone.Sidecars = make([]types.Sidecar, len(task.Sidecars))
	copy(clone.Sidecars, task.Sidecars)

	// Copy Volumes
	clone.Volumes = make([]types.Volume, len(task.Volumes))
	copy(clone.Volumes, task.Volumes)

	// Copy When
	clone.When = make([]types.WhenExpression, len(task.When))
	copy(clone.When, task.When)

	// Copy StepTemplate (shallow copy)
	clone.StepTemplate = task.StepTemplate

	// Copy Matrix (will be cleared by caller if needed)
	clone.Matrix = task.Matrix

	return clone
}
