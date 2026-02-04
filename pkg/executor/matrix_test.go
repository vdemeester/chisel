package executor

import (
	"testing"

	"github.com/vdemeester/chisel/pkg/types"
)

func TestGenerateMatrixCombinations_Empty(t *testing.T) {
	// No matrix params - should return empty slice
	matrix := types.Matrix{}
	combinations := generateMatrixCombinations(matrix)

	if len(combinations) != 0 {
		t.Errorf("Expected 0 combinations, got %d", len(combinations))
	}
}

func TestGenerateMatrixCombinations_SingleParam(t *testing.T) {
	// Single param with 3 values = 3 combinations
	matrix := types.Matrix{
		Params: []types.MatrixParam{
			{Name: "version", Values: []string{"1.20", "1.21", "1.22"}},
		},
	}
	combinations := generateMatrixCombinations(matrix)

	if len(combinations) != 3 {
		t.Fatalf("Expected 3 combinations, got %d", len(combinations))
	}

	// Check each combination
	expected := []string{"1.20", "1.21", "1.22"}
	for i, combo := range combinations {
		if combo["version"] != expected[i] {
			t.Errorf("Combination %d: version = %q, want %q", i, combo["version"], expected[i])
		}
	}
}

func TestGenerateMatrixCombinations_MultipleParams(t *testing.T) {
	// 2 params: version (3 values) x env (2 values) = 6 combinations
	matrix := types.Matrix{
		Params: []types.MatrixParam{
			{Name: "version", Values: []string{"1.20", "1.21", "1.22"}},
			{Name: "env", Values: []string{"dev", "prod"}},
		},
	}
	combinations := generateMatrixCombinations(matrix)

	if len(combinations) != 6 {
		t.Fatalf("Expected 6 combinations, got %d", len(combinations))
	}

	// Verify all combinations exist
	expectedCombos := []map[string]string{
		{"version": "1.20", "env": "dev"},
		{"version": "1.20", "env": "prod"},
		{"version": "1.21", "env": "dev"},
		{"version": "1.21", "env": "prod"},
		{"version": "1.22", "env": "dev"},
		{"version": "1.22", "env": "prod"},
	}

	for _, expected := range expectedCombos {
		found := false
		for _, combo := range combinations {
			if combo["version"] == expected["version"] && combo["env"] == expected["env"] {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Missing combination: version=%s, env=%s", expected["version"], expected["env"])
		}
	}
}

func TestGenerateMatrixCombinations_ThreeParams(t *testing.T) {
	// 3 params: 2 x 2 x 2 = 8 combinations
	matrix := types.Matrix{
		Params: []types.MatrixParam{
			{Name: "a", Values: []string{"a1", "a2"}},
			{Name: "b", Values: []string{"b1", "b2"}},
			{Name: "c", Values: []string{"c1", "c2"}},
		},
	}
	combinations := generateMatrixCombinations(matrix)

	if len(combinations) != 8 {
		t.Fatalf("Expected 8 combinations, got %d", len(combinations))
	}
}

func TestExpandMatrixTask_NoMatrix(t *testing.T) {
	// Task without matrix should return single task unchanged
	task := types.ResolvedTask{
		Name:     "test",
		TaskName: "test-task",
		Steps: []types.Step{
			{Name: "step1", Image: "alpine"},
		},
	}

	expanded := expandMatrixTask(task)

	if len(expanded) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(expanded))
	}
	if expanded[0].Name != "test" {
		t.Errorf("Task name = %q, want %q", expanded[0].Name, "test")
	}
}

func TestExpandMatrixTask_WithMatrix(t *testing.T) {
	// Task with matrix should be expanded
	task := types.ResolvedTask{
		Name:     "test",
		TaskName: "test-task",
		Matrix: &types.Matrix{
			Params: []types.MatrixParam{
				{Name: "version", Values: []string{"1.20", "1.21"}},
			},
		},
		Params: map[string]types.ParamValue{
			"existing": {Type: types.ParamTypeString, StringVal: "value"},
		},
		Steps: []types.Step{
			{Name: "step1", Image: "alpine"},
		},
	}

	expanded := expandMatrixTask(task)

	if len(expanded) != 2 {
		t.Fatalf("Expected 2 tasks, got %d", len(expanded))
	}

	// Check task names are unique
	if expanded[0].Name == expanded[1].Name {
		t.Error("Expanded tasks should have unique names")
	}

	// Check matrix params are added to task params
	if expanded[0].Params["version"].StringVal != "1.20" {
		t.Errorf("Task 0 version = %q, want %q", expanded[0].Params["version"].StringVal, "1.20")
	}
	if expanded[1].Params["version"].StringVal != "1.21" {
		t.Errorf("Task 1 version = %q, want %q", expanded[1].Params["version"].StringVal, "1.21")
	}

	// Check existing params are preserved
	if expanded[0].Params["existing"].StringVal != "value" {
		t.Errorf("Existing param not preserved in task 0")
	}
	if expanded[1].Params["existing"].StringVal != "value" {
		t.Errorf("Existing param not preserved in task 1")
	}
}

func TestExpandMatrixTask_MultipleParams(t *testing.T) {
	// Task with 2 matrix params = 4 combinations
	task := types.ResolvedTask{
		Name:     "build",
		TaskName: "build-task",
		Matrix: &types.Matrix{
			Params: []types.MatrixParam{
				{Name: "version", Values: []string{"1.20", "1.21"}},
				{Name: "target", Values: []string{"linux", "darwin"}},
			},
		},
		Steps: []types.Step{
			{Name: "compile", Image: "golang:$(params.version)"},
		},
	}

	expanded := expandMatrixTask(task)

	if len(expanded) != 4 {
		t.Fatalf("Expected 4 tasks, got %d", len(expanded))
	}

	// Verify all have unique names
	names := make(map[string]bool)
	for _, task := range expanded {
		if names[task.Name] {
			t.Errorf("Duplicate task name: %s", task.Name)
		}
		names[task.Name] = true
	}

	// Verify all combinations exist
	foundCombos := make(map[string]bool)
	for _, task := range expanded {
		version := task.Params["version"].StringVal
		target := task.Params["target"].StringVal
		key := version + "-" + target
		foundCombos[key] = true
	}

	expectedKeys := []string{"1.20-linux", "1.20-darwin", "1.21-linux", "1.21-darwin"}
	for _, key := range expectedKeys {
		if !foundCombos[key] {
			t.Errorf("Missing combination: %s", key)
		}
	}
}

func TestExpandAllMatrixTasks(t *testing.T) {
	tasks := []types.ResolvedTask{
		{
			Name:     "no-matrix",
			TaskName: "task1",
			Steps:    []types.Step{{Name: "s1", Image: "alpine"}},
		},
		{
			Name:     "with-matrix",
			TaskName: "task2",
			Matrix: &types.Matrix{
				Params: []types.MatrixParam{
					{Name: "v", Values: []string{"1", "2"}},
				},
			},
			Steps: []types.Step{{Name: "s2", Image: "alpine"}},
		},
		{
			Name:     "also-no-matrix",
			TaskName: "task3",
			Steps:    []types.Step{{Name: "s3", Image: "alpine"}},
		},
	}

	expanded := expandAllMatrixTasks(tasks)

	// Expected: 1 + 2 + 1 = 4 tasks
	if len(expanded) != 4 {
		t.Fatalf("Expected 4 tasks, got %d", len(expanded))
	}

	// First task unchanged
	if expanded[0].Name != "no-matrix" {
		t.Errorf("Task 0 name = %q, want %q", expanded[0].Name, "no-matrix")
	}

	// Middle two are expanded from matrix
	if expanded[1].Name != "with-matrix-0" {
		t.Errorf("Task 1 name = %q, want %q", expanded[1].Name, "with-matrix-0")
	}
	if expanded[2].Name != "with-matrix-1" {
		t.Errorf("Task 2 name = %q, want %q", expanded[2].Name, "with-matrix-1")
	}

	// Last task unchanged
	if expanded[3].Name != "also-no-matrix" {
		t.Errorf("Task 3 name = %q, want %q", expanded[3].Name, "also-no-matrix")
	}
}

func TestExpandMatrixTask_PreservesOtherFields(t *testing.T) {
	// Verify all task fields are copied to expanded tasks
	task := types.ResolvedTask{
		Name:     "test",
		TaskName: "original-task",
		Matrix: &types.Matrix{
			Params: []types.MatrixParam{
				{Name: "v", Values: []string{"1", "2"}},
			},
		},
		RunAfter: []string{"setup"},
		Steps: []types.Step{
			{Name: "step1", Image: "alpine"},
		},
		Workspaces: map[string]string{"source": "shared-ws"},
		Results:    []types.ResultSpec{{Name: "output"}},
	}

	expanded := expandMatrixTask(task)

	for i, exp := range expanded {
		if exp.TaskName != "original-task" {
			t.Errorf("Task %d: TaskName = %q, want %q", i, exp.TaskName, "original-task")
		}
		if len(exp.RunAfter) != 1 || exp.RunAfter[0] != "setup" {
			t.Errorf("Task %d: RunAfter not preserved", i)
		}
		if len(exp.Steps) != 1 {
			t.Errorf("Task %d: Steps not preserved", i)
		}
		if exp.Workspaces["source"] != "shared-ws" {
			t.Errorf("Task %d: Workspaces not preserved", i)
		}
		if len(exp.Results) != 1 {
			t.Errorf("Task %d: Results not preserved", i)
		}
	}
}
