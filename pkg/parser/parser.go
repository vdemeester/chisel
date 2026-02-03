// Package parser handles parsing Tekton YAML files and resolving references.
package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/vdemeester/chisel/pkg/types"
)

// Options configures the parser
type Options struct {
	// TasksDir is the directory containing Task definitions
	TasksDir string
	// Debug enables debug output
	Debug bool
}

// Parser parses Tekton YAML files
type Parser struct {
	opts      Options
	taskCache map[string]*TektonTask
}

// New creates a new Parser
func New(opts Options) *Parser {
	return &Parser{
		opts:      opts,
		taskCache: make(map[string]*TektonTask),
	}
}

// ParsePipelineRun parses a PipelineRun YAML file and resolves all references
func (p *Parser) ParsePipelineRun(path string) (*types.ResolvedPipelineRun, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Determine base directory for resolving references
	baseDir := filepath.Dir(path)
	if p.opts.TasksDir != "" {
		baseDir = p.opts.TasksDir
	}

	// Parse the YAML - could be PipelineRun, Pipeline, or Task
	var meta struct {
		APIVersion string `yaml:"apiVersion"`
		Kind       string `yaml:"kind"`
	}
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse YAML metadata: %w", err)
	}

	switch meta.Kind {
	case "PipelineRun":
		return p.parsePipelineRun(data, baseDir)
	case "Pipeline":
		return p.parsePipelineAsPipelineRun(data, baseDir)
	case "Task":
		return p.parseTaskAsPipelineRun(data, baseDir)
	default:
		return nil, fmt.Errorf("unsupported kind: %s (expected PipelineRun, Pipeline, or Task)", meta.Kind)
	}
}

// TektonPipelineRun represents a Tekton PipelineRun
type TektonPipelineRun struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec struct {
		PipelineRef *struct {
			Name string `yaml:"name"`
		} `yaml:"pipelineRef"`
		PipelineSpec *TektonPipelineSpec `yaml:"pipelineSpec"`
		Params       []TektonParam       `yaml:"params"`
		Workspaces   []struct {
			Name    string `yaml:"name"`
			SubPath string `yaml:"subPath"`
			// For local execution, we support local directory
			EmptyDir              *struct{} `yaml:"emptyDir"`
			PersistentVolumeClaim *struct {
				ClaimName string `yaml:"claimName"`
			} `yaml:"persistentVolumeClaim"`
		} `yaml:"workspaces"`
	} `yaml:"spec"`
}

// TektonPipeline represents a Tekton Pipeline
type TektonPipeline struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec TektonPipelineSpec `yaml:"spec"`
}

// TektonPipelineSpec is the spec of a Pipeline
type TektonPipelineSpec struct {
	Params     []TektonParamSpec    `yaml:"params"`
	Tasks      []TektonPipelineTask `yaml:"tasks"`
	Finally    []TektonPipelineTask `yaml:"finally"`
	Workspaces []struct {
		Name string `yaml:"name"`
	} `yaml:"workspaces"`
}

// TektonPipelineTask is a task reference within a Pipeline
type TektonPipelineTask struct {
	Name    string `yaml:"name"`
	TaskRef *struct {
		Name string `yaml:"name"`
	} `yaml:"taskRef"`
	TaskSpec   *TektonTaskSpec `yaml:"taskSpec"`
	Params     []TektonParam   `yaml:"params"`
	RunAfter   []string        `yaml:"runAfter"`
	Workspaces []struct {
		Name      string `yaml:"name"`
		Workspace string `yaml:"workspace"`
	} `yaml:"workspaces"`
}

// TektonTask represents a Tekton Task
type TektonTask struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec TektonTaskSpec `yaml:"spec"`
}

// TektonTaskSpec is the spec of a Task
type TektonTaskSpec struct {
	Params     []TektonParamSpec `yaml:"params"`
	Steps      []TektonStep      `yaml:"steps"`
	Sidecars   []TektonSidecar   `yaml:"sidecars"`
	Results    []TektonResult    `yaml:"results"`
	Volumes    []TektonVolume    `yaml:"volumes"`
	Workspaces []struct {
		Name      string `yaml:"name"`
		MountPath string `yaml:"mountPath"`
	} `yaml:"workspaces"`
}

// TektonStep represents a step in a Task
type TektonStep struct {
	Name         string              `yaml:"name"`
	Image        string              `yaml:"image"`
	Command      []string            `yaml:"command"`
	Args         []string            `yaml:"args"`
	Script       string              `yaml:"script"`
	Env          []TektonEnvVar      `yaml:"env"`
	WorkingDir   string              `yaml:"workingDir"`
	VolumeMounts []TektonVolumeMount `yaml:"volumeMounts"`
}

// TektonVolume represents a volume in a Task
type TektonVolume struct {
	Name     string `yaml:"name"`
	EmptyDir *struct {
		Medium string `yaml:"medium"`
	} `yaml:"emptyDir"`
	ConfigMap *struct {
		Name  string `yaml:"name"`
		Items []struct {
			Key  string `yaml:"key"`
			Path string `yaml:"path"`
		} `yaml:"items"`
	} `yaml:"configMap"`
	Secret *struct {
		SecretName string `yaml:"secretName"`
		Items      []struct {
			Key  string `yaml:"key"`
			Path string `yaml:"path"`
		} `yaml:"items"`
	} `yaml:"secret"`
}

// TektonVolumeMount represents a volume mount in a Step
type TektonVolumeMount struct {
	Name      string `yaml:"name"`
	MountPath string `yaml:"mountPath"`
	SubPath   string `yaml:"subPath"`
	ReadOnly  bool   `yaml:"readOnly"`
}

// TektonSidecar represents a sidecar in a Task
type TektonSidecar struct {
	Name    string         `yaml:"name"`
	Image   string         `yaml:"image"`
	Command []string       `yaml:"command"`
	Args    []string       `yaml:"args"`
	Env     []TektonEnvVar `yaml:"env"`
	Ports   []struct {
		ContainerPort int `yaml:"containerPort"`
	} `yaml:"ports"`
}

// TektonEnvVar represents an environment variable
type TektonEnvVar struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

// TektonParam represents a parameter value
type TektonParam struct {
	Name  string      `yaml:"name"`
	Value interface{} `yaml:"value"`
}

// TektonParamSpec represents a parameter definition
type TektonParamSpec struct {
	Name        string      `yaml:"name"`
	Type        string      `yaml:"type"`
	Default     interface{} `yaml:"default"`
	Description string      `yaml:"description"`
}

// TektonResult represents a result definition
type TektonResult struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

func (p *Parser) parsePipelineRun(data []byte, baseDir string) (*types.ResolvedPipelineRun, error) {
	var pr TektonPipelineRun
	if err := yaml.Unmarshal(data, &pr); err != nil {
		return nil, fmt.Errorf("failed to parse PipelineRun: %w", err)
	}

	resolved := &types.ResolvedPipelineRun{
		Name:       pr.Metadata.Name,
		Params:     make(map[string]types.ParamValue),
		Workspaces: make(map[string]types.WorkspaceBinding),
	}

	// Parse params
	for _, param := range pr.Spec.Params {
		resolved.Params[param.Name] = parseParamValue(param.Value)
	}

	// Parse workspace bindings
	for _, ws := range pr.Spec.Workspaces {
		binding := types.WorkspaceBinding{
			Name:    ws.Name,
			SubPath: ws.SubPath,
		}
		if ws.EmptyDir != nil {
			binding.Type = types.WorkspaceTypeEmptyDir
		} else if ws.PersistentVolumeClaim != nil {
			binding.Type = types.WorkspaceTypePVC
			binding.Path = ws.PersistentVolumeClaim.ClaimName
		} else {
			// Default to emptyDir for local execution
			binding.Type = types.WorkspaceTypeEmptyDir
		}
		resolved.Workspaces[ws.Name] = binding
	}

	// Get the pipeline spec
	var pipelineSpec *TektonPipelineSpec
	if pr.Spec.PipelineSpec != nil {
		pipelineSpec = pr.Spec.PipelineSpec
		resolved.PipelineName = pr.Metadata.Name + "-inline"
	} else if pr.Spec.PipelineRef != nil {
		pipeline, err := p.loadPipeline(pr.Spec.PipelineRef.Name, baseDir)
		if err != nil {
			return nil, fmt.Errorf("failed to load pipeline %s: %w", pr.Spec.PipelineRef.Name, err)
		}
		pipelineSpec = &pipeline.Spec
		resolved.PipelineName = pipeline.Metadata.Name
	} else {
		return nil, fmt.Errorf("PipelineRun must have either pipelineRef or pipelineSpec")
	}

	// Resolve tasks
	for _, pt := range pipelineSpec.Tasks {
		task, err := p.resolveTask(pt, baseDir, resolved.Params)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve task %s: %w", pt.Name, err)
		}
		resolved.Tasks = append(resolved.Tasks, *task)
	}

	// Resolve finally tasks
	for _, pt := range pipelineSpec.Finally {
		task, err := p.resolveTask(pt, baseDir, resolved.Params)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve finally task %s: %w", pt.Name, err)
		}
		resolved.FinallyTasks = append(resolved.FinallyTasks, *task)
	}

	return resolved, nil
}

func (p *Parser) parsePipelineAsPipelineRun(data []byte, baseDir string) (*types.ResolvedPipelineRun, error) {
	var pipeline TektonPipeline
	if err := yaml.Unmarshal(data, &pipeline); err != nil {
		return nil, fmt.Errorf("failed to parse Pipeline: %w", err)
	}

	resolved := &types.ResolvedPipelineRun{
		Name:         pipeline.Metadata.Name + "-run",
		PipelineName: pipeline.Metadata.Name,
		Params:       make(map[string]types.ParamValue),
		Workspaces:   make(map[string]types.WorkspaceBinding),
	}

	// Use default params
	for _, param := range pipeline.Spec.Params {
		if param.Default != nil {
			resolved.Params[param.Name] = parseParamValue(param.Default)
		}
	}

	// Create default workspace bindings
	for _, ws := range pipeline.Spec.Workspaces {
		resolved.Workspaces[ws.Name] = types.WorkspaceBinding{
			Name: ws.Name,
			Type: types.WorkspaceTypeEmptyDir,
		}
	}

	// Resolve tasks
	for _, pt := range pipeline.Spec.Tasks {
		task, err := p.resolveTask(pt, baseDir, resolved.Params)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve task %s: %w", pt.Name, err)
		}
		resolved.Tasks = append(resolved.Tasks, *task)
	}

	// Resolve finally tasks
	for _, pt := range pipeline.Spec.Finally {
		task, err := p.resolveTask(pt, baseDir, resolved.Params)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve finally task %s: %w", pt.Name, err)
		}
		resolved.FinallyTasks = append(resolved.FinallyTasks, *task)
	}

	return resolved, nil
}

func (p *Parser) parseTaskAsPipelineRun(data []byte, baseDir string) (*types.ResolvedPipelineRun, error) {
	var task TektonTask
	if err := yaml.Unmarshal(data, &task); err != nil {
		return nil, fmt.Errorf("failed to parse Task: %w", err)
	}

	resolved := &types.ResolvedPipelineRun{
		Name:         task.Metadata.Name + "-run",
		PipelineName: task.Metadata.Name + "-pipeline",
		Params:       make(map[string]types.ParamValue),
		Workspaces:   make(map[string]types.WorkspaceBinding),
	}

	// Use default params
	for _, param := range task.Spec.Params {
		if param.Default != nil {
			resolved.Params[param.Name] = parseParamValue(param.Default)
		}
	}

	// Create default workspace bindings
	for _, ws := range task.Spec.Workspaces {
		resolved.Workspaces[ws.Name] = types.WorkspaceBinding{
			Name: ws.Name,
			Type: types.WorkspaceTypeEmptyDir,
		}
	}

	// Create single task
	resolvedTask := types.ResolvedTask{
		Name:       task.Metadata.Name,
		TaskName:   task.Metadata.Name,
		Params:     resolved.Params,
		Workspaces: make(map[string]string),
	}

	// Convert steps
	for _, step := range task.Spec.Steps {
		resolvedTask.Steps = append(resolvedTask.Steps, convertStep(step))
	}

	// Convert sidecars
	for _, sidecar := range task.Spec.Sidecars {
		resolvedTask.Sidecars = append(resolvedTask.Sidecars, convertSidecar(sidecar))
	}

	// Convert results
	for _, result := range task.Spec.Results {
		resolvedTask.Results = append(resolvedTask.Results, types.ResultSpec{
			Name:        result.Name,
			Description: result.Description,
		})
	}

	resolved.Tasks = append(resolved.Tasks, resolvedTask)
	return resolved, nil
}

func (p *Parser) loadPipeline(name, baseDir string) (*TektonPipeline, error) {
	// Try to find the pipeline file
	patterns := []string{
		filepath.Join(baseDir, name+".yaml"),
		filepath.Join(baseDir, name+".yml"),
		filepath.Join(baseDir, "pipelines", name+".yaml"),
		filepath.Join(baseDir, "pipelines", name+".yml"),
	}

	for _, pattern := range patterns {
		if data, err := os.ReadFile(pattern); err == nil {
			var pipeline TektonPipeline
			if err := yaml.Unmarshal(data, &pipeline); err != nil {
				return nil, err
			}
			return &pipeline, nil
		}
	}

	return nil, fmt.Errorf("pipeline %s not found in %s", name, baseDir)
}

func (p *Parser) loadTask(name, baseDir string) (*TektonTask, error) {
	// Check cache first
	if task, ok := p.taskCache[name]; ok {
		return task, nil
	}

	// Try to find the task file
	patterns := []string{
		filepath.Join(baseDir, name+".yaml"),
		filepath.Join(baseDir, name+".yml"),
		filepath.Join(baseDir, "tasks", name+".yaml"),
		filepath.Join(baseDir, "tasks", name+".yml"),
	}

	for _, pattern := range patterns {
		if data, err := os.ReadFile(pattern); err == nil {
			var task TektonTask
			if err := yaml.Unmarshal(data, &task); err != nil {
				return nil, err
			}
			p.taskCache[name] = &task
			return &task, nil
		}
	}

	return nil, fmt.Errorf("task %s not found in %s", name, baseDir)
}

func (p *Parser) resolveTask(pt TektonPipelineTask, baseDir string, pipelineParams map[string]types.ParamValue) (*types.ResolvedTask, error) {
	resolved := &types.ResolvedTask{
		Name:       pt.Name,
		RunAfter:   pt.RunAfter,
		Params:     make(map[string]types.ParamValue),
		Workspaces: make(map[string]string),
	}

	// Get task spec
	var taskSpec *TektonTaskSpec
	if pt.TaskSpec != nil {
		taskSpec = pt.TaskSpec
		resolved.TaskName = pt.Name + "-inline"
	} else if pt.TaskRef != nil {
		task, err := p.loadTask(pt.TaskRef.Name, baseDir)
		if err != nil {
			return nil, err
		}
		taskSpec = &task.Spec
		resolved.TaskName = task.Metadata.Name
	} else {
		return nil, fmt.Errorf("task %s must have either taskRef or taskSpec", pt.Name)
	}

	// Resolve params - start with defaults
	for _, param := range taskSpec.Params {
		if param.Default != nil {
			resolved.Params[param.Name] = parseParamValue(param.Default)
		}
	}

	// Override with pipeline task params
	for _, param := range pt.Params {
		value := param.Value
		// Handle simple parameter substitution
		if strVal, ok := value.(string); ok {
			if strings.HasPrefix(strVal, "$(params.") && strings.HasSuffix(strVal, ")") {
				paramName := strings.TrimPrefix(strings.TrimSuffix(strVal, ")"), "$(params.")
				if pv, ok := pipelineParams[paramName]; ok {
					resolved.Params[param.Name] = pv
					continue
				}
			}
		}
		resolved.Params[param.Name] = parseParamValue(value)
	}

	// Map workspaces
	for _, ws := range pt.Workspaces {
		resolved.Workspaces[ws.Name] = ws.Workspace
	}

	// Convert steps
	for _, step := range taskSpec.Steps {
		resolved.Steps = append(resolved.Steps, convertStep(step))
	}

	// Convert sidecars
	for _, sidecar := range taskSpec.Sidecars {
		resolved.Sidecars = append(resolved.Sidecars, convertSidecar(sidecar))
	}

	// Convert results
	for _, result := range taskSpec.Results {
		resolved.Results = append(resolved.Results, types.ResultSpec{
			Name:        result.Name,
			Description: result.Description,
		})
	}

	// Convert volumes
	for _, vol := range taskSpec.Volumes {
		resolved.Volumes = append(resolved.Volumes, convertVolume(vol))
	}

	return resolved, nil
}

func convertStep(step TektonStep) types.Step {
	env := make(map[string]string)
	for _, e := range step.Env {
		env[e.Name] = e.Value
	}

	var volumeMounts []types.VolumeMount
	for _, vm := range step.VolumeMounts {
		volumeMounts = append(volumeMounts, types.VolumeMount{
			Name:      vm.Name,
			MountPath: vm.MountPath,
			SubPath:   vm.SubPath,
			ReadOnly:  vm.ReadOnly,
		})
	}

	return types.Step{
		Name:         step.Name,
		Image:        step.Image,
		Command:      step.Command,
		Args:         step.Args,
		Script:       step.Script,
		Env:          env,
		WorkingDir:   step.WorkingDir,
		VolumeMounts: volumeMounts,
	}
}

func convertSidecar(sidecar TektonSidecar) types.Sidecar {
	env := make(map[string]string)
	for _, e := range sidecar.Env {
		env[e.Name] = e.Value
	}
	var ports []int
	for _, p := range sidecar.Ports {
		ports = append(ports, p.ContainerPort)
	}
	return types.Sidecar{
		Name:    sidecar.Name,
		Image:   sidecar.Image,
		Command: sidecar.Command,
		Args:    sidecar.Args,
		Env:     env,
		Ports:   ports,
	}
}

func convertVolume(vol TektonVolume) types.Volume {
	v := types.Volume{
		Name: vol.Name,
	}

	if vol.EmptyDir != nil {
		v.EmptyDir = &types.EmptyDirVolumeSource{
			Medium: vol.EmptyDir.Medium,
		}
	}

	if vol.ConfigMap != nil {
		var items []types.KeyToPath
		for _, item := range vol.ConfigMap.Items {
			items = append(items, types.KeyToPath{
				Key:  item.Key,
				Path: item.Path,
			})
		}
		v.ConfigMap = &types.ConfigMapVolumeSource{
			Name:  vol.ConfigMap.Name,
			Items: items,
		}
	}

	if vol.Secret != nil {
		var items []types.KeyToPath
		for _, item := range vol.Secret.Items {
			items = append(items, types.KeyToPath{
				Key:  item.Key,
				Path: item.Path,
			})
		}
		v.Secret = &types.SecretVolumeSource{
			SecretName: vol.Secret.SecretName,
			Items:      items,
		}
	}

	return v
}

func parseParamValue(value interface{}) types.ParamValue {
	switch v := value.(type) {
	case string:
		return types.ParamValue{
			Type:      types.ParamTypeString,
			StringVal: v,
		}
	case []interface{}:
		var arr []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				arr = append(arr, s)
			}
		}
		return types.ParamValue{
			Type:     types.ParamTypeArray,
			ArrayVal: arr,
		}
	case map[string]interface{}:
		obj := make(map[string]string)
		for k, val := range v {
			if s, ok := val.(string); ok {
				obj[k] = s
			}
		}
		return types.ParamValue{
			Type:      types.ParamTypeObject,
			ObjectVal: obj,
		}
	default:
		return types.ParamValue{
			Type:      types.ParamTypeString,
			StringVal: fmt.Sprintf("%v", value),
		}
	}
}
