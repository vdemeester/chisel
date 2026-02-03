package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/vdemeester/chisel/pkg/executor"
	"github.com/vdemeester/chisel/pkg/parser"
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
	Args: cobra.ExactArgs(1),
	RunE: runPipeline,
}

var (
	tasksDir string
	debug    bool
	dryRun   bool
)

func init() {
	runCmd.Flags().StringVarP(&tasksDir, "tasks", "t", "", "Directory containing Task definitions")
	runCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable debug output")
	runCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Parse and validate without executing")
}

func runPipeline(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
	})

	resolved, err := p.ParsePipelineRun(pipelineRunPath)
	if err != nil {
		return fmt.Errorf("failed to parse PipelineRun: %w", err)
	}

	if debug {
		fmt.Printf("Parsed PipelineRun: %s\n", resolved.Name)
		fmt.Printf("Pipeline: %s\n", resolved.PipelineName)
		fmt.Printf("Tasks: %d\n", len(resolved.Tasks))
		for _, t := range resolved.Tasks {
			fmt.Printf("  - %s (%d steps)\n", t.Name, len(t.Steps))
		}
	}

	if dryRun {
		fmt.Println("Dry run complete. Pipeline parsed successfully.")
		return nil
	}

	// Execute via Dagger
	exec, err := executor.New(ctx, executor.Options{
		Debug: debug,
	})
	if err != nil {
		return fmt.Errorf("failed to create executor: %w", err)
	}
	defer exec.Close()

	if err := exec.Execute(ctx, resolved); err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	fmt.Println("Pipeline completed successfully.")
	return nil
}
