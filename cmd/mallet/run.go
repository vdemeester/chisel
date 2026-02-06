package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/vdemeester/chisel/pkg/backend"
	"github.com/vdemeester/chisel/pkg/backend/podman"
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
	defer be.Cleanup(ctx)

	pipelineStart := time.Now()
	log.PipelineStart(resolved.Name)

	// Execute each task sequentially
	for _, task := range resolved.Tasks {
		if err := executeTask(ctx, be, task, resolved, log); err != nil {
			log.PipelineEnd(resolved.Name, time.Since(pipelineStart), err)
			return fail(fmt.Errorf("task %s failed: %w", task.Name, err))
		}
	}

	log.PipelineEnd(resolved.Name, time.Since(pipelineStart), nil)
	return nil
}

// executeTask runs all steps in a task.
func executeTask(ctx context.Context, be backend.Backend, task types.ResolvedTask, pr *types.ResolvedPipelineRun, log ui.Logger) error {
	taskStart := time.Now()
	log.TaskStart(task.Name)

	for i, step := range task.Steps {
		stepName := step.Name
		if stepName == "" {
			stepName = fmt.Sprintf("step-%d", i)
		}

		stepStart := time.Now()
		log.StepStart(stepName, step.Image)

		// Build the step request
		req := &backend.StepRequest{
			Step:    &step,
			Image:   step.Image,
			Command: step.Command,
			Args:    step.Args,
			Env:     step.Env,
			WorkDir: step.WorkingDir,
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

		// Execute the step
		result, err := be.ExecuteStep(ctx, req)
		if err != nil {
			log.StepEnd(stepName, time.Since(stepStart), err)
			log.TaskEnd(task.Name, time.Since(taskStart), err)
			return fmt.Errorf("step %s failed: %w", stepName, err)
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
			return fmt.Errorf("step %s exited with code %d", stepName, result.ExitCode)
		}

		log.StepEnd(stepName, time.Since(stepStart), nil)
	}

	log.TaskEnd(task.Name, time.Since(taskStart), nil)
	return nil
}
