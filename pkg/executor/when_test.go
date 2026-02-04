package executor

import (
	"testing"

	"github.com/vdemeester/chisel/pkg/types"
)

func TestEvaluateWhen_Empty(t *testing.T) {
	// No when expressions - should return true (task should run)
	task := &types.ResolvedTask{
		Name: "test-task",
		When: nil,
	}

	result := evaluateWhen(task, nil, nil)
	if !result {
		t.Error("evaluateWhen with no expressions should return true")
	}
}

func TestEvaluateWhen_InOperator_Match(t *testing.T) {
	task := &types.ResolvedTask{
		Name: "test-task",
		When: []types.WhenExpression{
			{
				Input:    "production",
				Operator: "in",
				Values:   []string{"staging", "production"},
			},
		},
	}

	result := evaluateWhen(task, nil, nil)
	if !result {
		t.Error("evaluateWhen should return true when input is in values")
	}
}

func TestEvaluateWhen_InOperator_NoMatch(t *testing.T) {
	task := &types.ResolvedTask{
		Name: "test-task",
		When: []types.WhenExpression{
			{
				Input:    "development",
				Operator: "in",
				Values:   []string{"staging", "production"},
			},
		},
	}

	result := evaluateWhen(task, nil, nil)
	if result {
		t.Error("evaluateWhen should return false when input is not in values")
	}
}

func TestEvaluateWhen_NotInOperator_Match(t *testing.T) {
	task := &types.ResolvedTask{
		Name: "test-task",
		When: []types.WhenExpression{
			{
				Input:    "development",
				Operator: "notin",
				Values:   []string{"staging", "production"},
			},
		},
	}

	result := evaluateWhen(task, nil, nil)
	if !result {
		t.Error("evaluateWhen should return true when input is not in values (notin)")
	}
}

func TestEvaluateWhen_NotInOperator_NoMatch(t *testing.T) {
	task := &types.ResolvedTask{
		Name: "test-task",
		When: []types.WhenExpression{
			{
				Input:    "production",
				Operator: "notin",
				Values:   []string{"staging", "production"},
			},
		},
	}

	result := evaluateWhen(task, nil, nil)
	if result {
		t.Error("evaluateWhen should return false when input is in values (notin)")
	}
}

func TestEvaluateWhen_MultipleExpressions_AllTrue(t *testing.T) {
	task := &types.ResolvedTask{
		Name: "test-task",
		When: []types.WhenExpression{
			{
				Input:    "production",
				Operator: "in",
				Values:   []string{"production"},
			},
			{
				Input:    "passed",
				Operator: "in",
				Values:   []string{"passed", "success"},
			},
		},
	}

	result := evaluateWhen(task, nil, nil)
	if !result {
		t.Error("evaluateWhen should return true when all expressions are true")
	}
}

func TestEvaluateWhen_MultipleExpressions_OneFalse(t *testing.T) {
	task := &types.ResolvedTask{
		Name: "test-task",
		When: []types.WhenExpression{
			{
				Input:    "production",
				Operator: "in",
				Values:   []string{"production"},
			},
			{
				Input:    "failed",
				Operator: "in",
				Values:   []string{"passed", "success"},
			},
		},
	}

	result := evaluateWhen(task, nil, nil)
	if result {
		t.Error("evaluateWhen should return false when any expression is false")
	}
}

func TestEvaluateWhen_WithParamSubstitution(t *testing.T) {
	task := &types.ResolvedTask{
		Name: "test-task",
		Params: map[string]types.ParamValue{
			"env": {Type: types.ParamTypeString, StringVal: "production"},
		},
		When: []types.WhenExpression{
			{
				Input:    "$(params.env)",
				Operator: "in",
				Values:   []string{"staging", "production"},
			},
		},
	}

	result := evaluateWhen(task, task.Params, nil)
	if !result {
		t.Error("evaluateWhen should substitute params and return true")
	}
}

func TestEvaluateWhen_WithResultSubstitution(t *testing.T) {
	task := &types.ResolvedTask{
		Name: "deploy",
		When: []types.WhenExpression{
			{
				Input:    "$(tasks.test.results.status)",
				Operator: "in",
				Values:   []string{"passed"},
			},
		},
	}

	results := map[string]map[string]string{
		"test": {"status": "passed"},
	}

	result := evaluateWhen(task, nil, results)
	if !result {
		t.Error("evaluateWhen should substitute task results and return true")
	}
}

func TestEvaluateWhen_EmptyInput(t *testing.T) {
	// Empty input should not match non-empty values
	task := &types.ResolvedTask{
		Name: "test-task",
		When: []types.WhenExpression{
			{
				Input:    "",
				Operator: "in",
				Values:   []string{"production"},
			},
		},
	}

	result := evaluateWhen(task, nil, nil)
	if result {
		t.Error("evaluateWhen should return false when empty input doesn't match values")
	}
}

func TestEvaluateWhen_EmptyValues(t *testing.T) {
	// No values means nothing can match
	task := &types.ResolvedTask{
		Name: "test-task",
		When: []types.WhenExpression{
			{
				Input:    "production",
				Operator: "in",
				Values:   []string{},
			},
		},
	}

	result := evaluateWhen(task, nil, nil)
	if result {
		t.Error("evaluateWhen should return false when values is empty")
	}
}
