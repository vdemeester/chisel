// Package executor handles executing resolved pipelines via Dagger.
package executor

import (
	"context"
	"fmt"
	"strings"

	"dagger.io/dagger"
	"github.com/vdemeester/chisel/pkg/types"
)

// Options configures the executor
type Options struct {
	// Debug enables debug output
	Debug bool
}

// Executor executes resolved pipelines via Dagger
type Executor struct {
	client *dagger.Client
	opts   Options

	// workspaces maps workspace names to directories
	workspaces map[string]*dagger.Directory

	// results stores task results for variable substitution
	results map[string]map[string]string
}

// New creates a new Executor
func New(ctx context.Context, opts Options) (*Executor, error) {
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(nil))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Dagger: %w", err)
	}

	return &Executor{
		client:     client,
		opts:       opts,
		workspaces: make(map[string]*dagger.Directory),
		results:    make(map[string]map[string]string),
	}, nil
}

// Close closes the Dagger client
func (e *Executor) Close() {
	if e.client != nil {
		_ = e.client.Close()
	}
}

// Execute runs the resolved pipeline
func (e *Executor) Execute(ctx context.Context, pr *types.ResolvedPipelineRun) error {
	// Initialize workspaces
	for name, binding := range pr.Workspaces {
		dir, err := e.createWorkspace(ctx, binding)
		if err != nil {
			return fmt.Errorf("failed to create workspace %s: %w", name, err)
		}
		e.workspaces[name] = dir
	}

	// Execute tasks in order
	// TODO: Implement parallel execution based on runAfter dependencies
	for _, task := range pr.Tasks {
		if e.opts.Debug {
			fmt.Printf("Executing task: %s\n", task.Name)
		}

		if err := e.executeTask(ctx, &task, pr); err != nil {
			return fmt.Errorf("task %s failed: %w", task.Name, err)
		}

		if e.opts.Debug {
			fmt.Printf("Task %s completed\n", task.Name)
		}
	}

	// Execute finally tasks (always run, even on error - simplified for now)
	for _, task := range pr.FinallyTasks {
		if e.opts.Debug {
			fmt.Printf("Executing finally task: %s\n", task.Name)
		}

		if err := e.executeTask(ctx, &task, pr); err != nil {
			// Log but don't fail on finally task errors
			fmt.Printf("Warning: finally task %s failed: %v\n", task.Name, err)
		}
	}

	return nil
}

func (e *Executor) createWorkspace(ctx context.Context, binding types.WorkspaceBinding) (*dagger.Directory, error) {
	switch binding.Type {
	case types.WorkspaceTypeEmptyDir:
		return e.client.Directory(), nil
	case types.WorkspaceTypeLocal:
		return e.client.Host().Directory(binding.Path), nil
	case types.WorkspaceTypePVC:
		// For PVC, use a cache volume mounted to a directory
		// This provides persistence across runs
		cache := e.client.CacheVolume(binding.Path)
		return e.client.Container().
			From("alpine:latest").
			WithMountedCache("/workspace", cache).
			Directory("/workspace"), nil
	default:
		return e.client.Directory(), nil
	}
}

func (e *Executor) executeTask(ctx context.Context, task *types.ResolvedTask, pr *types.ResolvedPipelineRun) error {
	// Initialize results storage for this task
	e.results[task.Name] = make(map[string]string)

	// Execute each step sequentially
	for i, step := range task.Steps {
		if e.opts.Debug {
			fmt.Printf("  Step %d: %s\n", i+1, step.Name)
		}

		if err := e.executeStep(ctx, &step, task, pr); err != nil {
			return fmt.Errorf("step %s failed: %w", step.Name, err)
		}
	}

	return nil
}

func (e *Executor) executeStep(ctx context.Context, step *types.Step, task *types.ResolvedTask, pr *types.ResolvedPipelineRun) error {
	// Create container from image
	container := e.client.Container().From(step.Image)

	// Mount workspaces
	for taskWsName, pipelineWsName := range task.Workspaces {
		if dir, ok := e.workspaces[pipelineWsName]; ok {
			mountPath := fmt.Sprintf("/workspace/%s", taskWsName)
			container = container.WithDirectory(mountPath, dir)
		}
	}

	// Set environment variables
	for name, value := range step.Env {
		resolvedValue := e.substituteVariables(value, task, pr)
		container = container.WithEnvVariable(name, resolvedValue)
	}

	// Set working directory
	if step.WorkingDir != "" {
		workingDir := e.substituteVariables(step.WorkingDir, task, pr)
		container = container.WithWorkdir(workingDir)
	}

	// Execute command or script
	if step.Script != "" {
		// Script execution - write script to file and execute
		script := e.substituteVariables(step.Script, task, pr)

		// Detect shebang or default to sh
		shell := []string{"sh", "-c"}
		if strings.HasPrefix(script, "#!") {
			// Extract shebang
			lines := strings.SplitN(script, "\n", 2)
			shebang := strings.TrimPrefix(lines[0], "#!")
			shebang = strings.TrimSpace(shebang)
			shell = strings.Fields(shebang)
			if len(lines) > 1 {
				script = lines[1]
			}
		}

		container = container.WithExec(append(shell, script))
	} else if len(step.Command) > 0 {
		// Command execution
		cmd := make([]string, 0, len(step.Command)+len(step.Args))
		for _, c := range step.Command {
			cmd = append(cmd, e.substituteVariables(c, task, pr))
		}
		for _, a := range step.Args {
			cmd = append(cmd, e.substituteVariables(a, task, pr))
		}
		container = container.WithExec(cmd)
	}

	// Sync to execute and get output
	_, err := container.Sync(ctx)
	if err != nil {
		return err
	}

	// TODO: Capture results from /tekton/results/ directory

	return nil
}

// substituteVariables replaces Tekton variable references with actual values
func (e *Executor) substituteVariables(input string, task *types.ResolvedTask, pr *types.ResolvedPipelineRun) string {
	result := input

	// Replace $(params.name) with task params
	for name, value := range task.Params {
		placeholder := fmt.Sprintf("$(params.%s)", name)
		result = strings.ReplaceAll(result, placeholder, value.String())
	}

	// Replace $(workspaces.name.path) with workspace paths
	for name := range task.Workspaces {
		placeholder := fmt.Sprintf("$(workspaces.%s.path)", name)
		path := fmt.Sprintf("/workspace/%s", name)
		result = strings.ReplaceAll(result, placeholder, path)
	}

	// Replace $(tasks.taskname.results.resultname) with stored results
	for taskName, taskResults := range e.results {
		for resultName, resultValue := range taskResults {
			placeholder := fmt.Sprintf("$(tasks.%s.results.%s)", taskName, resultName)
			result = strings.ReplaceAll(result, placeholder, resultValue)
		}
	}

	// Replace $(context.*) variables
	result = strings.ReplaceAll(result, "$(context.pipelineRun.name)", pr.Name)
	result = strings.ReplaceAll(result, "$(context.pipeline.name)", pr.PipelineName)
	result = strings.ReplaceAll(result, "$(context.task.name)", task.TaskName)

	return result
}
