package executor

import (
	"fmt"
	"sync"

	"github.com/vdemeester/chisel/pkg/types"
)

// DAG represents a directed acyclic graph of tasks organized by execution levels.
// Tasks at the same level can be executed in parallel.
type DAG struct {
	levels [][]types.ResolvedTask
}

// BuildDAG analyzes task dependencies and organizes them into execution levels.
// Tasks with no dependencies are at level 0, tasks depending on level 0 are at level 1, etc.
func BuildDAG(tasks []types.ResolvedTask) *DAG {
	if len(tasks) == 0 {
		return &DAG{levels: nil}
	}

	// Build a map of task name to task
	taskMap := make(map[string]*types.ResolvedTask)
	for i := range tasks {
		taskMap[tasks[i].Name] = &tasks[i]
	}

	// Calculate the level for each task
	taskLevels := make(map[string]int)
	for i := range tasks {
		calculateLevel(tasks[i].Name, taskMap, taskLevels)
	}

	// Find max level
	maxLevel := 0
	for _, level := range taskLevels {
		if level > maxLevel {
			maxLevel = level
		}
	}

	// Organize tasks into levels
	levels := make([][]types.ResolvedTask, maxLevel+1)
	for i := range levels {
		levels[i] = []types.ResolvedTask{}
	}

	for i := range tasks {
		level := taskLevels[tasks[i].Name]
		levels[level] = append(levels[level], tasks[i])
	}

	return &DAG{levels: levels}
}

// calculateLevel recursively calculates the level of a task based on its dependencies.
func calculateLevel(taskName string, taskMap map[string]*types.ResolvedTask, levels map[string]int) int {
	// Already calculated
	if level, ok := levels[taskName]; ok {
		return level
	}

	task, ok := taskMap[taskName]
	if !ok {
		return 0
	}

	// No dependencies means level 0
	if len(task.RunAfter) == 0 {
		levels[taskName] = 0
		return 0
	}

	// Level is 1 + max level of all dependencies
	maxDepLevel := 0
	for _, dep := range task.RunAfter {
		depLevel := calculateLevel(dep, taskMap, levels)
		if depLevel > maxDepLevel {
			maxDepLevel = depLevel
		}
	}

	level := maxDepLevel + 1
	levels[taskName] = level
	return level
}

// GetLevel returns all tasks at the given level.
func (d *DAG) GetLevel(level int) []types.ResolvedTask {
	if level < 0 || level >= len(d.levels) {
		return nil
	}
	return d.levels[level]
}

// LevelCount returns the total number of levels in the DAG.
func (d *DAG) LevelCount() int {
	return len(d.levels)
}

// TaskExecutor is a function that executes a single task.
type TaskExecutor func(task *types.ResolvedTask) error

// ExecuteParallel executes tasks in parallel within each level,
// but sequentially between levels. It stops on the first error.
func (d *DAG) ExecuteParallel(executor TaskExecutor) error {
	for level := 0; level < d.LevelCount(); level++ {
		tasks := d.GetLevel(level)
		if len(tasks) == 0 {
			continue
		}

		// Execute all tasks at this level in parallel
		var wg sync.WaitGroup
		errChan := make(chan error, len(tasks))

		for i := range tasks {
			wg.Add(1)
			go func(task *types.ResolvedTask) {
				defer wg.Done()
				if err := executor(task); err != nil {
					errChan <- fmt.Errorf("task %s failed: %w", task.Name, err)
				}
			}(&tasks[i])
		}

		wg.Wait()
		close(errChan)

		// Check for errors
		for err := range errChan {
			return err // Return first error
		}
	}

	return nil
}
