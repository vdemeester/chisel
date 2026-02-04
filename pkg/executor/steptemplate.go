package executor

import "github.com/vdemeester/chisel/pkg/types"

// applyStepTemplate applies default values from a StepTemplate to a Step.
// Step values take precedence over template values.
func applyStepTemplate(step types.Step, template *types.StepTemplate) types.Step {
	if template == nil {
		return step
	}

	result := step

	// Image: use template if step doesn't specify
	if result.Image == "" && template.Image != "" {
		result.Image = template.Image
	}

	// Command: use template if step doesn't specify
	if len(result.Command) == 0 && len(template.Command) > 0 {
		result.Command = make([]string, len(template.Command))
		copy(result.Command, template.Command)
	}

	// WorkingDir: use template if step doesn't specify
	if result.WorkingDir == "" && template.WorkingDir != "" {
		result.WorkingDir = template.WorkingDir
	}

	// Env: merge template env with step env (step takes precedence)
	if len(template.Env) > 0 {
		merged := make(map[string]string)
		// Start with template env
		for k, v := range template.Env {
			merged[k] = v
		}
		// Override with step env
		for k, v := range result.Env {
			merged[k] = v
		}
		result.Env = merged
	}

	// VolumeMounts: prepend template mounts to step mounts
	if len(template.VolumeMounts) > 0 {
		merged := make([]types.VolumeMount, 0, len(template.VolumeMounts)+len(result.VolumeMounts))
		merged = append(merged, template.VolumeMounts...)
		merged = append(merged, result.VolumeMounts...)
		result.VolumeMounts = merged
	}

	return result
}
