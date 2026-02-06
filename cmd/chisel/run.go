package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/vdemeester/chisel/pkg/backend/dagger"
	"github.com/vdemeester/chisel/pkg/parser"
	"github.com/vdemeester/chisel/pkg/types"
	"github.com/vdemeester/chisel/pkg/ui"
)

var runCmd = &cobra.Command{
	Use:   "run <pipelinerun.yaml>",
	Short: "Run a Tekton PipelineRun locally",
	Long: `Execute a Tekton PipelineRun locally using Dagger as the backend.

The command parses the PipelineRun YAML, resolves referenced Pipeline
and Task definitions, and executes them sequentially via Dagger.

Examples:
  # Run a PipelineRun
  chisel run pipelinerun.yaml

  # Specify task definitions directory
  chisel run pipelinerun.yaml --tasks=./tasks/

  # Run with debug output
  chisel run pipelinerun.yaml --debug`,
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
	params     []string
)

func init() {
	runCmd.Flags().StringVarP(&tasksDir, "tasks", "t", "", "Directory containing Task definitions")
	runCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable debug output")
	runCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Parse and validate without executing")
	runCmd.Flags().StringVarP(&outputMode, "output", "o", "", "Output mode: pretty, plain, json (default: auto-detect)")
	runCmd.Flags().StringArrayVarP(&workspaces, "workspace", "w", nil, "Override workspace binding (format: name:path, can be repeated)")
	runCmd.Flags().StringArrayVarP(&params, "param", "p", nil, "Parameter value (format: name=value, can be repeated)")
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

	// Parse CLI params
	cliParams, err := parseParams(params)
	if err != nil {
		return fail(err)
	}

	// Parse the PipelineRun and resolve references
	p := parser.New(parser.Options{
		TasksDir: tasksDir,
		Debug:    debug,
		Logger:   log,
		Params:   cliParams,
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

	// Execute via Dagger backend
	backend, err := dagger.New(ctx, dagger.Options{
		Debug:  debug,
		Logger: log,
	})
	if err != nil {
		return fail(fmt.Errorf("failed to create Dagger backend: %w", err))
	}
	defer backend.Close()

	if err := backend.Execute(ctx, resolved); err != nil {
		return fail(fmt.Errorf("execution failed: %w", err))
	}

	return nil
}

// parseParams parses CLI param strings into a map of ParamValues
func parseParams(params []string) (map[string]types.ParamValue, error) {
	result := make(map[string]types.ParamValue)
	for _, p := range params {
		name, value, err := parser.ParseParamString(p)
		if err != nil {
			return nil, err
		}
		result[name] = value
	}
	return result, nil
}
