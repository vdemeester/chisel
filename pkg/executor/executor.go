// Package executor handles executing resolved pipelines via Dagger.
package executor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"dagger.io/dagger"

	"github.com/vdemeester/chisel/pkg/types"
	"github.com/vdemeester/chisel/pkg/ui"
)

// Options configures the executor
type Options struct {
	// Debug enables debug output
	Debug bool
	// Logger for structured output
	Logger ui.Logger
}

// Executor executes resolved pipelines via Dagger
type Executor struct {
	client *dagger.Client
	opts   Options
	log    ui.Logger

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

	log := opts.Logger
	if log == nil {
		log = ui.NewLogger(ui.DetectOutputMode(), nil)
	}

	return &Executor{
		client:     client,
		opts:       opts,
		log:        log,
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
	pipelineStart := time.Now()
	e.log.PipelineStart(pr.Name)

	// Initialize workspaces
	for name, binding := range pr.Workspaces {
		dir, err := e.createWorkspace(ctx, binding)
		if err != nil {
			return fmt.Errorf("failed to create workspace %s: %w", name, err)
		}
		e.workspaces[name] = dir
	}

	var pipelineErr error

	// Build DAG and execute tasks in parallel where possible
	dag := BuildDAG(pr.Tasks)
	pipelineErr = dag.ExecuteParallel(func(task *types.ResolvedTask) error {
		return e.executeTask(ctx, task, pr)
	})

	// Execute finally tasks (always run, even on error)
	for _, task := range pr.FinallyTasks {
		if err := e.executeTask(ctx, &task, pr); err != nil {
			// Log but don't fail on finally task errors
			e.log.Warn("finally task failed", "task", task.Name, "error", err)
		}
	}

	e.log.PipelineEnd(pr.Name, time.Since(pipelineStart), pipelineErr)
	return pipelineErr
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
	taskStart := time.Now()
	e.log.TaskStart(task.Name)

	// Initialize results storage for this task
	e.results[task.Name] = make(map[string]string)

	var taskErr error

	// Execute each step sequentially
	for _, step := range task.Steps {
		if err := e.executeStep(ctx, &step, task, pr); err != nil {
			taskErr = fmt.Errorf("step %s failed: %w", step.Name, err)
			break
		}
	}

	e.log.TaskEnd(task.Name, time.Since(taskStart), taskErr)
	return taskErr
}

func (e *Executor) executeStep(ctx context.Context, step *types.Step, task *types.ResolvedTask, pr *types.ResolvedPipelineRun) error {
	stepStart := time.Now()
	e.log.StepStart(step.Name, step.Image)

	// Create container from image
	container := e.client.Container().From(step.Image)

	// Mount workspaces
	for taskWsName, pipelineWsName := range task.Workspaces {
		if dir, ok := e.workspaces[pipelineWsName]; ok {
			mountPath := fmt.Sprintf("/workspace/%s", taskWsName)
			container = container.WithDirectory(mountPath, dir)
		}
	}

	// Mount volumes
	container = e.mountVolumes(container, step, task)

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
			// Extract shebang and append -c flag for inline script execution
			lines := strings.SplitN(script, "\n", 2)
			shebang := strings.TrimPrefix(lines[0], "#!")
			shebang = strings.TrimSpace(shebang)
			shell = append(strings.Fields(shebang), "-c")
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

	// Execute and capture output
	stdout, err := container.Stdout(ctx)
	if stdout != "" {
		e.log.StepOutput(step.Name, stdout)
	}

	// Also capture stderr on failure
	if err != nil {
		stderr, _ := container.Stderr(ctx)
		if stderr != "" {
			e.log.StepOutput(step.Name, stderr)
		}
	}

	e.log.StepEnd(step.Name, time.Since(stepStart), err)

	if err != nil {
		return err
	}

	// Capture results from /tekton/results/ directory
	if len(task.Results) > 0 {
		resultFiles := make(map[string]string)
		for _, spec := range task.Results {
			// Try to read each result file from the container
			resultPath := fmt.Sprintf("/tekton/results/%s", spec.Name)
			content, readErr := container.File(resultPath).Contents(ctx)
			if readErr == nil {
				resultFiles[spec.Name] = content
			}
		}
		captureResults(task.Results, resultFiles, e.results[task.Name])
	}

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

// mountVolumes mounts task volumes into the container based on step's volumeMounts.
func (e *Executor) mountVolumes(container *dagger.Container, step *types.Step, task *types.ResolvedTask) *dagger.Container {
	if len(step.VolumeMounts) == 0 || len(task.Volumes) == 0 {
		return container
	}

	// Parse task volumes into a lookup map
	volumes := parseVolumes(task.Volumes)

	for _, mount := range step.VolumeMounts {
		vol, ok := volumes[mount.Name]
		if !ok {
			continue
		}

		switch vol.Type {
		case VolumeTypeEmptyDir:
			// EmptyDir is just an empty directory - use a cache volume for persistence within the run
			cache := e.client.CacheVolume(fmt.Sprintf("emptydir-%s-%s", task.Name, mount.Name))
			container = container.WithMountedCache(mount.MountPath, cache)

		case VolumeTypeConfigMap:
			// ConfigMap - in local execution, we'd need to read from a local file
			// For now, create an empty directory as placeholder
			// TODO: Support reading configmaps from local files or environment
			dir := e.client.Directory()
			container = container.WithDirectory(mount.MountPath, dir)

		case VolumeTypeSecret:
			// Secret - in local execution, we'd need to read from a local file
			// For now, create an empty directory as placeholder
			// TODO: Support reading secrets from local files or environment
			dir := e.client.Directory()
			container = container.WithDirectory(mount.MountPath, dir)
		}
	}

	return container
}
