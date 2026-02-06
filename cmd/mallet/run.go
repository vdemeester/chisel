package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/vdemeester/chisel/pkg/backend"
	"github.com/vdemeester/chisel/pkg/backend/podman"
	"github.com/vdemeester/chisel/pkg/orchestrator"
	"github.com/vdemeester/chisel/pkg/parser"
	"github.com/vdemeester/chisel/pkg/types"
	"github.com/vdemeester/chisel/pkg/ui"
)

var runCmd = &cobra.Command{
	Use:   "run <pipelinerun.yaml>",
	Short: "Run a Tekton PipelineRun locally",
	Long: `Execute a Tekton PipelineRun locally using Podman as the backend.

The command parses the PipelineRun YAML, resolves referenced Pipeline
and Task definitions, and executes them via Podman containers.

Examples:
  # Run a PipelineRun
  mallet run pipelinerun.yaml

  # Specify task definitions directory
  mallet run pipelinerun.yaml --tasks=./tasks/

  # Run with debug output
  mallet run pipelinerun.yaml --debug`,
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runPipeline,
}

var (
	tasksDir   string
	debug      bool
	dryRun     bool
	outputMode string
	workspaces []string
)

func init() {
	runCmd.Flags().StringVarP(&tasksDir, "tasks", "t", "", "Directory containing Task definitions")
	runCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable debug output")
	runCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Parse and validate without executing")
	runCmd.Flags().StringVarP(&outputMode, "output", "o", "", "Output mode: pretty, plain, json (default: auto-detect)")
	runCmd.Flags().StringArrayVarP(&workspaces, "workspace", "w", nil, "Override workspace binding (format: name:path, can be repeated)")
}

func runPipeline(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Determine output mode early so we can use styled errors
	var mode ui.OutputMode
	if outputMode != "" {
		mode = ui.ParseOutputMode(outputMode)
	} else {
		mode = ui.DetectOutputMode()
	}
	log := ui.NewLogger(mode, os.Stdout)

	// Helper to print styled error and return it
	fail := func(err error) error {
		cleaned := ui.CleanError(err)
		log.Error(cleaned.Error())
		return cleaned
	}

	// Handle interrupt signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nInterrupted, cleaning up...")
		cancel()
	}()

	pipelineRunPath := args[0]

	// Parse the PipelineRun and resolve references
	p := parser.New(parser.Options{
		TasksDir: tasksDir,
		Debug:    debug,
		Logger:   log,
	})

	resolved, err := p.ParsePipelineRun(pipelineRunPath)
	if err != nil {
		return fail(fmt.Errorf("failed to parse PipelineRun: %w", err))
	}

	// Apply workspace overrides from CLI flags
	if len(workspaces) > 0 {
		wsOverrides, err := parseWorkspaceBindings(workspaces)
		if err != nil {
			return fail(fmt.Errorf("invalid workspace binding: %w", err))
		}
		if err := applyWorkspaceOverrides(resolved, wsOverrides); err != nil {
			return fail(fmt.Errorf("failed to apply workspace overrides: %w", err))
		}
		if debug {
			for name, path := range wsOverrides {
				log.Debug("Workspace override", "name", name, "path", path)
			}
		}
	}

	if debug {
		log.Debug("Parsed PipelineRun", "name", resolved.Name)
		log.Debug("Pipeline", "name", resolved.PipelineName)
		log.Debug("Tasks", "count", len(resolved.Tasks))
		for _, t := range resolved.Tasks {
			log.Debug("Task", "name", t.Name, "steps", len(t.Steps))
		}
	}

	if dryRun {
		log.Info("Dry run complete. Pipeline parsed successfully.")
		return nil
	}

	// Execute via Podman backend
	be := podman.NewPodmanBackend()
	defer func() { _ = be.Cleanup(ctx) }()

	pipelineStart := time.Now()
	log.PipelineStart(resolved.Name)

	// Results store for passing between tasks
	results := make(map[string]map[string]string)
	var resultsMu sync.Mutex

	// Expand matrix tasks before building DAG
	tasks := orchestrator.ExpandAllMatrixTasks(resolved.Tasks)

	// Build DAG and execute tasks in parallel where possible
	dag := orchestrator.BuildDAG(tasks)
	pipelineErr := dag.ExecuteParallel(func(task *types.ResolvedTask) error {
		// Evaluate when conditions before executing
		resultsMu.Lock()
		shouldRun := orchestrator.EvaluateWhen(task, task.Params, results)
		resultsMu.Unlock()

		if !shouldRun {
			log.Info(fmt.Sprintf("task %s skipped (when conditions not met)", task.Name))
			return nil
		}

		// Get current results snapshot for variable substitution
		resultsMu.Lock()
		resultsCopy := make(map[string]map[string]string)
		for k, v := range results {
			resultsCopy[k] = v
		}
		resultsMu.Unlock()

		taskResults, err := executeTask(ctx, be, *task, resolved, resultsCopy, log)
		if err != nil {
			return err
		}

		// Store task results for dependent tasks
		if len(taskResults) > 0 {
			resultsMu.Lock()
			results[task.Name] = taskResults
			resultsMu.Unlock()
		}

		return nil
	})

	// Execute finally tasks (always run, even on error)
	for _, task := range resolved.FinallyTasks {
		if _, err := executeTask(ctx, be, task, resolved, results, log); err != nil {
			log.Warn(fmt.Sprintf("finally task %s failed: %v", task.Name, err))
		}
	}

	log.PipelineEnd(resolved.Name, time.Since(pipelineStart), pipelineErr)
	if pipelineErr != nil {
		return fail(pipelineErr)
	}
	return nil
}

// executeTask runs all steps in a task and returns task results.
func executeTask(ctx context.Context, be backend.Backend, task types.ResolvedTask, pr *types.ResolvedPipelineRun, taskResults map[string]map[string]string, log ui.Logger) (map[string]string, error) {
	taskStart := time.Now()
	log.TaskStart(task.Name)

	// Create temp directories for task volumes (emptyDir)
	volumeDirs := make(map[string]string)
	volumes := orchestrator.ParseVolumes(task.Volumes)
	for name, vol := range volumes {
		if vol.Type == orchestrator.VolumeTypeEmptyDir {
			dir, err := os.MkdirTemp("", fmt.Sprintf("tekton-vol-%s-%s-*", task.Name, name))
			if err != nil {
				return nil, fmt.Errorf("failed to create volume directory: %w", err)
			}
			volumeDirs[name] = dir
			defer func(d string) { _ = os.RemoveAll(d) }(dir)
		}
	}

	// Collect results from all steps
	stepResults := make(map[string]string)

	for i, originalStep := range task.Steps {
		// Apply stepTemplate defaults
		step := orchestrator.ApplyStepTemplate(originalStep, task.StepTemplate)

		stepName := step.Name
		if stepName == "" {
			stepName = fmt.Sprintf("step-%d", i)
		}

		stepStart := time.Now()
		log.StepStart(stepName, step.Image)

		// Substitute variables in script
		script := step.Script
		if script != "" {
			script = substituteVariables(script, &task, pr, taskResults)
		}

		// Substitute variables in command
		var command []string
		for _, c := range step.Command {
			command = append(command, substituteVariables(c, &task, pr, taskResults))
		}

		// Substitute variables in args
		var args []string
		for _, a := range step.Args {
			args = append(args, substituteVariables(a, &task, pr, taskResults))
		}

		// Substitute variables in env values
		env := make(map[string]string)
		for k, v := range step.Env {
			env[k] = substituteVariables(v, &task, pr, taskResults)
		}

		// Substitute variables in workingDir
		workDir := step.WorkingDir
		if workDir != "" {
			workDir = substituteVariables(workDir, &task, pr, taskResults)
		}

		// Create a copy of step with substituted values
		substitutedStep := step
		substitutedStep.Script = script
		substitutedStep.Command = command
		substitutedStep.Args = args
		substitutedStep.Env = env
		substitutedStep.WorkingDir = workDir

		// Build the step request
		req := &backend.StepRequest{
			Step:    &substitutedStep,
			Image:   step.Image,
			Command: command,
			Args:    args,
			Env:     env,
			WorkDir: workDir,
		}

		// Convert workspace bindings
		req.Workspaces = make(map[string]backend.WorkspaceMount)
		for name, ws := range pr.Workspaces {
			req.Workspaces[name] = backend.WorkspaceMount{
				Name:       ws.Name,
				MountPath:  "/workspace/" + name,
				SourceType: string(ws.Type),
				SourcePath: ws.Path,
			}
		}

		// Add volume mounts from the step
		for _, vm := range step.VolumeMounts {
			if dir, ok := volumeDirs[vm.Name]; ok {
				req.Workspaces["vol-"+vm.Name] = backend.WorkspaceMount{
					Name:       vm.Name,
					MountPath:  vm.MountPath,
					SourceType: "emptyDir",
					SourcePath: dir,
				}
			}
		}

		// Execute the step
		result, err := be.ExecuteStep(ctx, req)
		if err != nil {
			log.StepEnd(stepName, time.Since(stepStart), err)
			log.TaskEnd(task.Name, time.Since(taskStart), err)
			return nil, fmt.Errorf("step %s failed: %w", stepName, err)
		}

		// Log output
		if result.Stdout != "" {
			log.StepOutput(stepName, result.Stdout)
		}
		if result.Stderr != "" {
			log.Warn(fmt.Sprintf("Step %s stderr: %s", stepName, result.Stderr))
		}

		// Check exit code
		if result.ExitCode != 0 {
			err := fmt.Errorf("step exited with code %d", result.ExitCode)
			log.StepEnd(stepName, time.Since(stepStart), err)
			log.TaskEnd(task.Name, time.Since(taskStart), err)
			return nil, fmt.Errorf("step %s exited with code %d", stepName, result.ExitCode)
		}

		// Collect results from this step
		for name, value := range result.Results {
			stepResults[name] = value
		}

		log.StepEnd(stepName, time.Since(stepStart), nil)
	}

	log.TaskEnd(task.Name, time.Since(taskStart), nil)
	return stepResults, nil
}

// substituteVariables replaces Tekton variable references with actual values
func substituteVariables(input string, task *types.ResolvedTask, pr *types.ResolvedPipelineRun, results map[string]map[string]string) string {
	result := input

	// Replace $(params.name) with task params
	for name, value := range task.Params {
		switch value.Type {
		case types.ParamTypeArray:
			// Handle $(params.name[*]) - expand to space-separated values
			starPlaceholder := fmt.Sprintf("$(params.%s[*])", name)
			result = strings.ReplaceAll(result, starPlaceholder, strings.Join(value.ArrayVal, " "))

			// Handle $(params.name[N]) - expand to indexed value
			for i, v := range value.ArrayVal {
				indexPlaceholder := fmt.Sprintf("$(params.%s[%d])", name, i)
				result = strings.ReplaceAll(result, indexPlaceholder, v)
			}

		case types.ParamTypeObject:
			// Handle $(params.name.field) - expand to field value
			for field, fieldValue := range value.ObjectVal {
				fieldPlaceholder := fmt.Sprintf("$(params.%s.%s)", name, field)
				result = strings.ReplaceAll(result, fieldPlaceholder, fieldValue)
			}

		default:
			// String param - simple replacement
			placeholder := fmt.Sprintf("$(params.%s)", name)
			result = strings.ReplaceAll(result, placeholder, value.String())
		}
	}

	// Replace $(workspaces.name.path) with workspace paths
	for name := range task.Workspaces {
		placeholder := fmt.Sprintf("$(workspaces.%s.path)", name)
		path := fmt.Sprintf("/workspace/%s", name)
		result = strings.ReplaceAll(result, placeholder, path)
	}

	// Replace $(tasks.taskname.results.resultname) with stored results
	for taskName, taskResults := range results {
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
