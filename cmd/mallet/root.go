package main

import (
	"github.com/spf13/cobra"

	"github.com/vdemeester/chisel/internal/version"
)

var rootCmd = &cobra.Command{
	Use:   "mallet",
	Short: "Run Tekton PipelineRuns locally using Podman",
	Long: `Mallet is a CLI tool that executes Tekton PipelineRuns locally
using Podman as the execution backend.

It parses Tekton YAML (PipelineRun, Pipeline, Task) and translates
the operations to Podman containers, providing fast local execution
without requiring Kubernetes or Dagger.

Mallet is the Podman-based companion to Chisel (which uses Dagger).
Both tools share the same parser, orchestration logic, and examples.

Named to pair with "Chisel" (both builder's tools), honoring
Tekton's Greek meaning of "builder/carpenter".`,
	Version: version.Version,
}

func init() {
	rootCmd.AddCommand(runCmd)
}
