// Package types defines internal types for chisel's pipeline representation.
// These are simplified versions of Tekton types, optimized for local execution.
package types

// ResolvedPipelineRun represents a fully resolved PipelineRun with all
// referenced Pipelines and Tasks inlined.
type ResolvedPipelineRun struct {
	// Name is the PipelineRun name
	Name string

	// PipelineName is the name of the Pipeline being run
	PipelineName string

	// Params are the resolved parameters for the pipeline
	Params map[string]ParamValue

	// Tasks are the resolved tasks in execution order
	Tasks []ResolvedTask

	// FinallyTasks are tasks that run after all other tasks complete
	FinallyTasks []ResolvedTask

	// Workspaces are the workspace bindings
	Workspaces map[string]WorkspaceBinding
}

// ResolvedTask represents a task with all references resolved
type ResolvedTask struct {
	// Name is the task name within the pipeline
	Name string

	// TaskName is the original Task definition name
	TaskName string

	// Steps are the container steps to execute
	Steps []Step

	// Params are the resolved parameters for this task
	Params map[string]ParamValue

	// Workspaces are the workspace bindings for this task
	Workspaces map[string]string // task workspace name -> pipeline workspace name

	// Results are the result definitions
	Results []ResultSpec

	// RunAfter lists tasks that must complete before this one
	RunAfter []string

	// Sidecars are auxiliary containers
	Sidecars []Sidecar

	// Volumes are volumes that can be mounted by steps
	Volumes []Volume

	// When contains expressions that must evaluate to true for the task to run
	When []WhenExpression
}

// WhenExpression defines a condition for task execution.
// All expressions must evaluate to true for the task to run.
type WhenExpression struct {
	// Input is the value to evaluate (supports variable substitution)
	Input string

	// Operator is the comparison operator: "in" or "notin"
	Operator string

	// Values is the list of values to compare against
	Values []string
}

// Step represents a single step within a task
type Step struct {
	// Name is the step name
	Name string

	// Image is the container image to use
	Image string

	// Command is the entrypoint override
	Command []string

	// Args are arguments to the command
	Args []string

	// Script is a script to run (alternative to command/args)
	Script string

	// Env are environment variables
	Env map[string]string

	// WorkingDir is the working directory
	WorkingDir string

	// VolumeMounts are volumes to mount into the container
	VolumeMounts []VolumeMount

	// Timeout is the maximum duration for the step (e.g., "10s", "5m")
	Timeout string

	// Retries is the number of times to retry the step on failure
	Retries int
}

// Sidecar represents a sidecar container
type Sidecar struct {
	// Name is the sidecar name
	Name string

	// Image is the container image
	Image string

	// Command is the entrypoint override
	Command []string

	// Args are arguments to the command
	Args []string

	// Env are environment variables
	Env map[string]string

	// Ports are exposed ports
	Ports []int
}

// ParamValue represents a parameter value (string, array, or object)
type ParamValue struct {
	Type      ParamType
	StringVal string
	ArrayVal  []string
	ObjectVal map[string]string
}

// ParamType indicates the type of a parameter
type ParamType string

const (
	ParamTypeString ParamType = "string"
	ParamTypeArray  ParamType = "array"
	ParamTypeObject ParamType = "object"
)

// String returns the string value or empty string for non-string types
func (p ParamValue) String() string {
	if p.Type == ParamTypeString {
		return p.StringVal
	}
	return ""
}

// ResultSpec defines a result that a task produces
type ResultSpec struct {
	Name        string
	Description string
}

// WorkspaceBinding represents how a workspace is bound
type WorkspaceBinding struct {
	// Name is the workspace name
	Name string

	// Type is the binding type
	Type WorkspaceType

	// Path is the local path (for local directory bindings)
	Path string

	// SubPath is a subdirectory within the workspace
	SubPath string
}

// WorkspaceType indicates how a workspace is provided
type WorkspaceType string

const (
	WorkspaceTypeEmptyDir WorkspaceType = "emptyDir"
	WorkspaceTypeLocal    WorkspaceType = "local"
	WorkspaceTypePVC      WorkspaceType = "pvc"
)

// Volume defines a volume that can be mounted by steps in a task.
type Volume struct {
	// Name is the volume name, referenced by VolumeMounts
	Name string

	// VolumeSource defines where the volume comes from
	VolumeSource
}

// VolumeSource represents the source of a volume.
// Only one of its members may be specified.
type VolumeSource struct {
	// EmptyDir represents an empty directory
	EmptyDir *EmptyDirVolumeSource

	// ConfigMap represents a configMap that should populate this volume
	ConfigMap *ConfigMapVolumeSource

	// Secret represents a secret that should populate this volume
	Secret *SecretVolumeSource
}

// EmptyDirVolumeSource represents an empty directory volume.
type EmptyDirVolumeSource struct {
	// Medium is the storage medium (empty string for default, "Memory" for tmpfs)
	Medium string
}

// ConfigMapVolumeSource represents a configMap volume.
type ConfigMapVolumeSource struct {
	// Name of the configMap
	Name string

	// Items are specific keys to project (if empty, all keys are projected)
	Items []KeyToPath
}

// SecretVolumeSource represents a secret volume.
type SecretVolumeSource struct {
	// SecretName is the name of the secret
	SecretName string

	// Items are specific keys to project (if empty, all keys are projected)
	Items []KeyToPath
}

// KeyToPath maps a key to a file path.
type KeyToPath struct {
	// Key is the key to project
	Key string

	// Path is the relative path of the file
	Path string
}

// VolumeMount describes a mounting of a volume within a container.
type VolumeMount struct {
	// Name must match the Name of a Volume
	Name string

	// MountPath is the path within the container
	MountPath string

	// SubPath is an optional subpath within the volume
	SubPath string

	// ReadOnly defaults to false
	ReadOnly bool
}
