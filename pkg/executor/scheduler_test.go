package executor

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vdemeester/chisel/pkg/types"
)

func TestBuildDAG_NoDependencies(t *testing.T) {
	// Tasks with no runAfter should all be at level 0 (can run in parallel)
	tasks := []types.ResolvedTask{
		{Name: "task-a"},
		{Name: "task-b"},
		{Name: "task-c"},
	}

	dag := BuildDAG(tasks)

	// All tasks should be in the first level (no dependencies)
	level0 := dag.GetLevel(0)
	if len(level0) != 3 {
		t.Errorf("Expected 3 tasks at level 0, got %d", len(level0))
	}
}

func TestBuildDAG_LinearDependencies(t *testing.T) {
	// task-a -> task-b -> task-c (sequential)
	tasks := []types.ResolvedTask{
		{Name: "task-a"},
		{Name: "task-b", RunAfter: []string{"task-a"}},
		{Name: "task-c", RunAfter: []string{"task-b"}},
	}

	dag := BuildDAG(tasks)

	// Each task should be at a different level
	level0 := dag.GetLevel(0)
	level1 := dag.GetLevel(1)
	level2 := dag.GetLevel(2)

	if len(level0) != 1 || level0[0].Name != "task-a" {
		t.Errorf("Level 0 should have task-a, got %v", level0)
	}
	if len(level1) != 1 || level1[0].Name != "task-b" {
		t.Errorf("Level 1 should have task-b, got %v", level1)
	}
	if len(level2) != 1 || level2[0].Name != "task-c" {
		t.Errorf("Level 2 should have task-c, got %v", level2)
	}
}

func TestBuildDAG_DiamondDependencies(t *testing.T) {
	// Diamond pattern:
	//      task-a
	//      /    \
	//  task-b  task-c
	//      \    /
	//      task-d
	tasks := []types.ResolvedTask{
		{Name: "task-a"},
		{Name: "task-b", RunAfter: []string{"task-a"}},
		{Name: "task-c", RunAfter: []string{"task-a"}},
		{Name: "task-d", RunAfter: []string{"task-b", "task-c"}},
	}

	dag := BuildDAG(tasks)

	level0 := dag.GetLevel(0)
	level1 := dag.GetLevel(1)
	level2 := dag.GetLevel(2)

	if len(level0) != 1 {
		t.Errorf("Level 0 should have 1 task (task-a), got %d", len(level0))
	}
	if len(level1) != 2 {
		t.Errorf("Level 1 should have 2 tasks (task-b, task-c), got %d", len(level1))
	}
	if len(level2) != 1 {
		t.Errorf("Level 2 should have 1 task (task-d), got %d", len(level2))
	}
}

func TestBuildDAG_LevelCount(t *testing.T) {
	tasks := []types.ResolvedTask{
		{Name: "task-a"},
		{Name: "task-b", RunAfter: []string{"task-a"}},
		{Name: "task-c", RunAfter: []string{"task-b"}},
	}

	dag := BuildDAG(tasks)

	if dag.LevelCount() != 3 {
		t.Errorf("Expected 3 levels, got %d", dag.LevelCount())
	}
}

func TestBuildDAG_EmptyTasks(t *testing.T) {
	dag := BuildDAG(nil)

	if dag.LevelCount() != 0 {
		t.Errorf("Expected 0 levels for empty tasks, got %d", dag.LevelCount())
	}
}

func TestBuildDAG_MultipleDependencies(t *testing.T) {
	// task-d depends on both task-b AND task-c
	// task-b and task-c both depend on task-a
	// task-d should be at level 2
	tasks := []types.ResolvedTask{
		{Name: "task-a"},
		{Name: "task-b", RunAfter: []string{"task-a"}},
		{Name: "task-c", RunAfter: []string{"task-a"}},
		{Name: "task-d", RunAfter: []string{"task-b", "task-c"}},
	}

	dag := BuildDAG(tasks)

	// task-d should wait for BOTH task-b and task-c
	level2 := dag.GetLevel(2)
	if len(level2) != 1 || level2[0].Name != "task-d" {
		t.Errorf("task-d should be at level 2, got %v", level2)
	}
}

func TestDAG_ExecuteParallel(t *testing.T) {
	// Test that tasks at the same level execute concurrently
	tasks := []types.ResolvedTask{
		{Name: "task-a"},
		{Name: "task-b"},
		{Name: "task-c"},
	}

	dag := BuildDAG(tasks)

	var maxConcurrent int32
	var currentConcurrent int32
	var mu sync.Mutex
	executionOrder := []string{}

	executor := func(task *types.ResolvedTask) error {
		// Track concurrency
		curr := atomic.AddInt32(&currentConcurrent, 1)
		mu.Lock()
		if curr > maxConcurrent {
			maxConcurrent = curr
		}
		executionOrder = append(executionOrder, task.Name)
		mu.Unlock()

		// Simulate work
		time.Sleep(10 * time.Millisecond)

		atomic.AddInt32(&currentConcurrent, -1)
		return nil
	}

	err := dag.ExecuteParallel(executor)
	if err != nil {
		t.Fatalf("ExecuteParallel failed: %v", err)
	}

	// All 3 tasks should have run concurrently
	if maxConcurrent < 3 {
		t.Errorf("Expected at least 3 concurrent executions, got %d", maxConcurrent)
	}

	// All tasks should have executed
	if len(executionOrder) != 3 {
		t.Errorf("Expected 3 tasks executed, got %d", len(executionOrder))
	}
}

func TestDAG_ExecuteParallel_WithDependencies(t *testing.T) {
	// task-a -> (task-b, task-c) -> task-d
	tasks := []types.ResolvedTask{
		{Name: "task-a"},
		{Name: "task-b", RunAfter: []string{"task-a"}},
		{Name: "task-c", RunAfter: []string{"task-a"}},
		{Name: "task-d", RunAfter: []string{"task-b", "task-c"}},
	}

	dag := BuildDAG(tasks)

	var mu sync.Mutex
	executionOrder := []string{}
	executionTimes := make(map[string]time.Time)

	executor := func(task *types.ResolvedTask) error {
		mu.Lock()
		executionTimes[task.Name] = time.Now()
		executionOrder = append(executionOrder, task.Name)
		mu.Unlock()

		time.Sleep(10 * time.Millisecond)
		return nil
	}

	err := dag.ExecuteParallel(executor)
	if err != nil {
		t.Fatalf("ExecuteParallel failed: %v", err)
	}

	// task-a must start before task-b and task-c
	if !executionTimes["task-a"].Before(executionTimes["task-b"]) {
		t.Error("task-a should execute before task-b")
	}
	if !executionTimes["task-a"].Before(executionTimes["task-c"]) {
		t.Error("task-a should execute before task-c")
	}

	// task-d must start after both task-b and task-c
	if !executionTimes["task-b"].Before(executionTimes["task-d"]) {
		t.Error("task-b should execute before task-d")
	}
	if !executionTimes["task-c"].Before(executionTimes["task-d"]) {
		t.Error("task-c should execute before task-d")
	}
}

func TestDAG_ExecuteParallel_StopsOnError(t *testing.T) {
	tasks := []types.ResolvedTask{
		{Name: "task-a"},
		{Name: "task-b", RunAfter: []string{"task-a"}},
	}

	dag := BuildDAG(tasks)

	expectedErr := fmt.Errorf("task-a failed")
	executor := func(task *types.ResolvedTask) error {
		if task.Name == "task-a" {
			return expectedErr
		}
		return nil
	}

	err := dag.ExecuteParallel(executor)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "task-a") {
		t.Errorf("Expected error to mention task-a, got: %v", err)
	}
}
