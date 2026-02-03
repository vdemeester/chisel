package main

import (
	"github.com/spf13/cobra"

	"github.com/vdemeester/chisel/internal/version"
)

var rootCmd = &cobra.Command{
	Use:   "chisel",
	Short: "Run Tekton PipelineRuns locally using Dagger",
	Long: `Chisel is a CLI tool that executes Tekton PipelineRuns locally
using Dagger as the execution backend.

It parses Tekton YAML (PipelineRun, Pipeline, Task) and translates
the operations to Dagger, providing fast local execution with
identical behavior to cluster execution.

Named to pair with "Dagger" (both builder's tools), honoring
Tekton's Greek meaning of "builder/carpenter".`,
	Version: version.Version,
}

func init() {
	rootCmd.AddCommand(runCmd)
}
