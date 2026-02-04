package executor

import (
	"testing"

	"github.com/vdemeester/chisel/pkg/types"
)

func TestApplyStepTemplate_Empty(t *testing.T) {
	// No template - step should remain unchanged
	step := types.Step{
		Name:  "test",
		Image: "alpine:latest",
	}

	result := applyStepTemplate(step, nil)

	if result.Image != "alpine:latest" {
		t.Errorf("Image = %q, want %q", result.Image, "alpine:latest")
	}
}

func TestApplyStepTemplate_ImageDefault(t *testing.T) {
	// Step has no image, template provides default
	step := types.Step{
		Name:   "test",
		Script: "echo hello",
	}
	template := &types.StepTemplate{
		Image: "golang:1.21",
	}

	result := applyStepTemplate(step, template)

	if result.Image != "golang:1.21" {
		t.Errorf("Image = %q, want %q", result.Image, "golang:1.21")
	}
}

func TestApplyStepTemplate_ImageOverride(t *testing.T) {
	// Step has image, should override template
	step := types.Step{
		Name:  "test",
		Image: "alpine:latest",
	}
	template := &types.StepTemplate{
		Image: "golang:1.21",
	}

	result := applyStepTemplate(step, template)

	if result.Image != "alpine:latest" {
		t.Errorf("Image = %q, want %q", result.Image, "alpine:latest")
	}
}

func TestApplyStepTemplate_EnvMerge(t *testing.T) {
	// Template env should be merged with step env
	step := types.Step{
		Name:  "test",
		Image: "alpine:latest",
		Env:   map[string]string{"STEP_VAR": "step_value"},
	}
	template := &types.StepTemplate{
		Env: map[string]string{
			"TEMPLATE_VAR": "template_value",
			"SHARED_VAR":   "from_template",
		},
	}

	result := applyStepTemplate(step, template)

	if result.Env["TEMPLATE_VAR"] != "template_value" {
		t.Errorf("TEMPLATE_VAR = %q, want %q", result.Env["TEMPLATE_VAR"], "template_value")
	}
	if result.Env["STEP_VAR"] != "step_value" {
		t.Errorf("STEP_VAR = %q, want %q", result.Env["STEP_VAR"], "step_value")
	}
}

func TestApplyStepTemplate_EnvOverride(t *testing.T) {
	// Step env should override template env for same key
	step := types.Step{
		Name:  "test",
		Image: "alpine:latest",
		Env:   map[string]string{"SHARED_VAR": "from_step"},
	}
	template := &types.StepTemplate{
		Env: map[string]string{"SHARED_VAR": "from_template"},
	}

	result := applyStepTemplate(step, template)

	if result.Env["SHARED_VAR"] != "from_step" {
		t.Errorf("SHARED_VAR = %q, want %q", result.Env["SHARED_VAR"], "from_step")
	}
}

func TestApplyStepTemplate_WorkingDirDefault(t *testing.T) {
	step := types.Step{
		Name:  "test",
		Image: "alpine:latest",
	}
	template := &types.StepTemplate{
		WorkingDir: "/workspace/source",
	}

	result := applyStepTemplate(step, template)

	if result.WorkingDir != "/workspace/source" {
		t.Errorf("WorkingDir = %q, want %q", result.WorkingDir, "/workspace/source")
	}
}

func TestApplyStepTemplate_WorkingDirOverride(t *testing.T) {
	step := types.Step{
		Name:       "test",
		Image:      "alpine:latest",
		WorkingDir: "/custom/dir",
	}
	template := &types.StepTemplate{
		WorkingDir: "/workspace/source",
	}

	result := applyStepTemplate(step, template)

	if result.WorkingDir != "/custom/dir" {
		t.Errorf("WorkingDir = %q, want %q", result.WorkingDir, "/custom/dir")
	}
}

func TestApplyStepTemplate_VolumeMountsMerge(t *testing.T) {
	// Template volumeMounts should be merged with step volumeMounts
	step := types.Step{
		Name:  "test",
		Image: "alpine:latest",
		VolumeMounts: []types.VolumeMount{
			{Name: "step-vol", MountPath: "/step"},
		},
	}
	template := &types.StepTemplate{
		VolumeMounts: []types.VolumeMount{
			{Name: "template-vol", MountPath: "/template"},
		},
	}

	result := applyStepTemplate(step, template)

	if len(result.VolumeMounts) != 2 {
		t.Fatalf("VolumeMounts count = %d, want 2", len(result.VolumeMounts))
	}

	// Template mounts come first, then step mounts
	if result.VolumeMounts[0].Name != "template-vol" {
		t.Errorf("VolumeMounts[0].Name = %q, want %q", result.VolumeMounts[0].Name, "template-vol")
	}
	if result.VolumeMounts[1].Name != "step-vol" {
		t.Errorf("VolumeMounts[1].Name = %q, want %q", result.VolumeMounts[1].Name, "step-vol")
	}
}

func TestApplyStepTemplate_CommandDefault(t *testing.T) {
	step := types.Step{
		Name:  "test",
		Image: "alpine:latest",
		Args:  []string{"arg1"},
	}
	template := &types.StepTemplate{
		Command: []string{"/bin/custom"},
	}

	result := applyStepTemplate(step, template)

	if len(result.Command) != 1 || result.Command[0] != "/bin/custom" {
		t.Errorf("Command = %v, want [/bin/custom]", result.Command)
	}
}

func TestApplyStepTemplate_CommandOverride(t *testing.T) {
	step := types.Step{
		Name:    "test",
		Image:   "alpine:latest",
		Command: []string{"/bin/step-cmd"},
	}
	template := &types.StepTemplate{
		Command: []string{"/bin/template-cmd"},
	}

	result := applyStepTemplate(step, template)

	if len(result.Command) != 1 || result.Command[0] != "/bin/step-cmd" {
		t.Errorf("Command = %v, want [/bin/step-cmd]", result.Command)
	}
}
