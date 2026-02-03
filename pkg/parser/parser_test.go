package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vdemeester/chisel/pkg/types"
)

func TestNew(t *testing.T) {
	p := New(Options{TasksDir: "/tmp", Debug: true})
	if p == nil {
		t.Fatal("New() returned nil")
	}
	if p.opts.TasksDir != "/tmp" {
		t.Errorf("TasksDir = %q, want %q", p.opts.TasksDir, "/tmp")
	}
	if !p.opts.Debug {
		t.Error("Debug = false, want true")
	}
	if p.taskCache == nil {
		t.Error("taskCache is nil")
	}
}

func TestParsePipelineRun_Task(t *testing.T) {
	// Create a temporary task file
	taskYAML := `
apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: hello-task
spec:
  params:
    - name: greeting
      type: string
      default: "Hello"
  steps:
    - name: say-hello
      image: alpine:latest
      script: |
        echo "$(params.greeting), World!"
`
	tmpDir := t.TempDir()
	taskFile := filepath.Join(tmpDir, "task.yaml")
	if err := os.WriteFile(taskFile, []byte(taskYAML), 0644); err != nil {
		t.Fatalf("failed to write task file: %v", err)
	}

	p := New(Options{})
	pr, err := p.ParsePipelineRun(taskFile)
	if err != nil {
		t.Fatalf("ParsePipelineRun() error = %v", err)
	}

	// Task is wrapped as a single-task pipeline
	if pr.Name != "hello-task-run" {
		t.Errorf("Name = %q, want %q", pr.Name, "hello-task-run")
	}
	if pr.PipelineName != "hello-task-pipeline" {
		t.Errorf("PipelineName = %q, want %q", pr.PipelineName, "hello-task-pipeline")
	}
	if len(pr.Tasks) != 1 {
		t.Fatalf("len(Tasks) = %d, want 1", len(pr.Tasks))
	}

	task := pr.Tasks[0]
	if task.Name != "hello-task" {
		t.Errorf("task.Name = %q, want %q", task.Name, "hello-task")
	}
	if len(task.Steps) != 1 {
		t.Fatalf("len(Steps) = %d, want 1", len(task.Steps))
	}
	if task.Steps[0].Name != "say-hello" {
		t.Errorf("step.Name = %q, want %q", task.Steps[0].Name, "say-hello")
	}
	if task.Steps[0].Image != "alpine:latest" {
		t.Errorf("step.Image = %q, want %q", task.Steps[0].Image, "alpine:latest")
	}

	// Check default param was parsed
	if v, ok := pr.Params["greeting"]; !ok {
		t.Error("param 'greeting' not found")
	} else if v.StringVal != "Hello" {
		t.Errorf("param greeting = %q, want %q", v.StringVal, "Hello")
	}
}

func TestParsePipelineRun_Pipeline(t *testing.T) {
	pipelineYAML := `
apiVersion: tekton.dev/v1
kind: Pipeline
metadata:
  name: build-pipeline
spec:
  params:
    - name: repo
      type: string
      default: "https://github.com/example/repo"
  tasks:
    - name: build
      taskSpec:
        steps:
          - name: compile
            image: golang:1.21
            script: go build ./...
    - name: test
      runAfter:
        - build
      taskSpec:
        steps:
          - name: run-tests
            image: golang:1.21
            script: go test ./...
`
	tmpDir := t.TempDir()
	pipelineFile := filepath.Join(tmpDir, "pipeline.yaml")
	if err := os.WriteFile(pipelineFile, []byte(pipelineYAML), 0644); err != nil {
		t.Fatalf("failed to write pipeline file: %v", err)
	}

	p := New(Options{})
	pr, err := p.ParsePipelineRun(pipelineFile)
	if err != nil {
		t.Fatalf("ParsePipelineRun() error = %v", err)
	}

	if pr.Name != "build-pipeline-run" {
		t.Errorf("Name = %q, want %q", pr.Name, "build-pipeline-run")
	}
	if pr.PipelineName != "build-pipeline" {
		t.Errorf("PipelineName = %q, want %q", pr.PipelineName, "build-pipeline")
	}
	if len(pr.Tasks) != 2 {
		t.Fatalf("len(Tasks) = %d, want 2", len(pr.Tasks))
	}

	// Check runAfter is preserved
	testTask := pr.Tasks[1]
	if testTask.Name != "test" {
		t.Errorf("task[1].Name = %q, want %q", testTask.Name, "test")
	}
	if len(testTask.RunAfter) != 1 || testTask.RunAfter[0] != "build" {
		t.Errorf("task[1].RunAfter = %v, want [build]", testTask.RunAfter)
	}
}

func TestParsePipelineRun_InlinePipelineSpec(t *testing.T) {
	prYAML := `
apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  name: my-run
spec:
  pipelineSpec:
    tasks:
      - name: greet
        taskSpec:
          steps:
            - name: echo
              image: alpine
              command: ["echo", "hello"]
  params:
    - name: message
      value: "world"
`
	tmpDir := t.TempDir()
	prFile := filepath.Join(tmpDir, "pr.yaml")
	if err := os.WriteFile(prFile, []byte(prYAML), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	p := New(Options{})
	pr, err := p.ParsePipelineRun(prFile)
	if err != nil {
		t.Fatalf("ParsePipelineRun() error = %v", err)
	}

	if pr.Name != "my-run" {
		t.Errorf("Name = %q, want %q", pr.Name, "my-run")
	}
	if pr.PipelineName != "my-run-inline" {
		t.Errorf("PipelineName = %q, want %q", pr.PipelineName, "my-run-inline")
	}
	if len(pr.Tasks) != 1 {
		t.Fatalf("len(Tasks) = %d, want 1", len(pr.Tasks))
	}

	// Check param from PipelineRun
	if v, ok := pr.Params["message"]; !ok {
		t.Error("param 'message' not found")
	} else if v.StringVal != "world" {
		t.Errorf("param message = %q, want %q", v.StringVal, "world")
	}
}

func TestParsePipelineRun_WithPipelineRef(t *testing.T) {
	// Create pipeline file
	pipelineYAML := `
apiVersion: tekton.dev/v1
kind: Pipeline
metadata:
  name: referenced-pipeline
spec:
  tasks:
    - name: do-thing
      taskSpec:
        steps:
          - name: run
            image: alpine
            script: echo done
`
	// Create pipelinerun that references it
	prYAML := `
apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  name: my-pipeline-run
spec:
  pipelineRef:
    name: referenced-pipeline
`
	tmpDir := t.TempDir()
	pipelineFile := filepath.Join(tmpDir, "referenced-pipeline.yaml")
	if err := os.WriteFile(pipelineFile, []byte(pipelineYAML), 0644); err != nil {
		t.Fatalf("failed to write pipeline file: %v", err)
	}
	prFile := filepath.Join(tmpDir, "pr.yaml")
	if err := os.WriteFile(prFile, []byte(prYAML), 0644); err != nil {
		t.Fatalf("failed to write pr file: %v", err)
	}

	p := New(Options{})
	pr, err := p.ParsePipelineRun(prFile)
	if err != nil {
		t.Fatalf("ParsePipelineRun() error = %v", err)
	}

	if pr.Name != "my-pipeline-run" {
		t.Errorf("Name = %q, want %q", pr.Name, "my-pipeline-run")
	}
	if pr.PipelineName != "referenced-pipeline" {
		t.Errorf("PipelineName = %q, want %q", pr.PipelineName, "referenced-pipeline")
	}
	if len(pr.Tasks) != 1 {
		t.Fatalf("len(Tasks) = %d, want 1", len(pr.Tasks))
	}
}

func TestParsePipelineRun_WithTaskRef(t *testing.T) {
	// Create task file
	taskYAML := `
apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: my-task
spec:
  params:
    - name: input
      default: "default-value"
  steps:
    - name: process
      image: alpine
      script: echo $(params.input)
`
	// Create pipelinerun with taskRef
	prYAML := `
apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  name: task-ref-run
spec:
  pipelineSpec:
    tasks:
      - name: run-task
        taskRef:
          name: my-task
        params:
          - name: input
            value: "override-value"
`
	tmpDir := t.TempDir()
	taskFile := filepath.Join(tmpDir, "my-task.yaml")
	if err := os.WriteFile(taskFile, []byte(taskYAML), 0644); err != nil {
		t.Fatalf("failed to write task file: %v", err)
	}
	prFile := filepath.Join(tmpDir, "pr.yaml")
	if err := os.WriteFile(prFile, []byte(prYAML), 0644); err != nil {
		t.Fatalf("failed to write pr file: %v", err)
	}

	p := New(Options{})
	pr, err := p.ParsePipelineRun(prFile)
	if err != nil {
		t.Fatalf("ParsePipelineRun() error = %v", err)
	}

	if len(pr.Tasks) != 1 {
		t.Fatalf("len(Tasks) = %d, want 1", len(pr.Tasks))
	}

	task := pr.Tasks[0]
	if task.TaskName != "my-task" {
		t.Errorf("TaskName = %q, want %q", task.TaskName, "my-task")
	}

	// Check param override
	if v, ok := task.Params["input"]; !ok {
		t.Error("param 'input' not found")
	} else if v.StringVal != "override-value" {
		t.Errorf("param input = %q, want %q", v.StringVal, "override-value")
	}
}

func TestParsePipelineRun_Workspaces(t *testing.T) {
	prYAML := `
apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  name: workspace-run
spec:
  pipelineSpec:
    workspaces:
      - name: source
      - name: cache
    tasks:
      - name: build
        taskSpec:
          workspaces:
            - name: src
          steps:
            - name: compile
              image: alpine
              script: ls $(workspaces.src.path)
        workspaces:
          - name: src
            workspace: source
  workspaces:
    - name: source
      emptyDir: {}
    - name: cache
      persistentVolumeClaim:
        claimName: my-cache-pvc
`
	tmpDir := t.TempDir()
	prFile := filepath.Join(tmpDir, "pr.yaml")
	if err := os.WriteFile(prFile, []byte(prYAML), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	p := New(Options{})
	pr, err := p.ParsePipelineRun(prFile)
	if err != nil {
		t.Fatalf("ParsePipelineRun() error = %v", err)
	}

	// Check workspace bindings
	if len(pr.Workspaces) != 2 {
		t.Fatalf("len(Workspaces) = %d, want 2", len(pr.Workspaces))
	}

	source, ok := pr.Workspaces["source"]
	if !ok {
		t.Fatal("workspace 'source' not found")
	}
	if source.Type != types.WorkspaceTypeEmptyDir {
		t.Errorf("source.Type = %v, want EmptyDir", source.Type)
	}

	cache, ok := pr.Workspaces["cache"]
	if !ok {
		t.Fatal("workspace 'cache' not found")
	}
	if cache.Type != types.WorkspaceTypePVC {
		t.Errorf("cache.Type = %v, want PVC", cache.Type)
	}
	if cache.Path != "my-cache-pvc" {
		t.Errorf("cache.Path = %q, want %q", cache.Path, "my-cache-pvc")
	}

	// Check task workspace mapping
	if len(pr.Tasks) != 1 {
		t.Fatalf("len(Tasks) = %d, want 1", len(pr.Tasks))
	}
	task := pr.Tasks[0]
	if task.Workspaces["src"] != "source" {
		t.Errorf("task.Workspaces[src] = %q, want %q", task.Workspaces["src"], "source")
	}
}

func TestParsePipelineRun_FinallyTasks(t *testing.T) {
	prYAML := `
apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  name: finally-run
spec:
  pipelineSpec:
    tasks:
      - name: main-task
        taskSpec:
          steps:
            - name: work
              image: alpine
              script: echo "working"
    finally:
      - name: cleanup
        taskSpec:
          steps:
            - name: clean
              image: alpine
              script: echo "cleaning up"
`
	tmpDir := t.TempDir()
	prFile := filepath.Join(tmpDir, "pr.yaml")
	if err := os.WriteFile(prFile, []byte(prYAML), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	p := New(Options{})
	pr, err := p.ParsePipelineRun(prFile)
	if err != nil {
		t.Fatalf("ParsePipelineRun() error = %v", err)
	}

	if len(pr.Tasks) != 1 {
		t.Errorf("len(Tasks) = %d, want 1", len(pr.Tasks))
	}
	if len(pr.FinallyTasks) != 1 {
		t.Fatalf("len(FinallyTasks) = %d, want 1", len(pr.FinallyTasks))
	}
	if pr.FinallyTasks[0].Name != "cleanup" {
		t.Errorf("FinallyTasks[0].Name = %q, want %q", pr.FinallyTasks[0].Name, "cleanup")
	}
}

func TestParsePipelineRun_Volumes(t *testing.T) {
	prYAML := `
apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  name: volume-run
spec:
  pipelineSpec:
    tasks:
      - name: volume-task
        taskSpec:
          volumes:
            - name: temp
              emptyDir: {}
            - name: config
              configMap:
                name: my-config
            - name: creds
              secret:
                secretName: my-secret
          steps:
            - name: use-volumes
              image: alpine
              volumeMounts:
                - name: temp
                  mountPath: /tmp/data
                - name: config
                  mountPath: /etc/config
              script: ls /tmp/data /etc/config
`
	tmpDir := t.TempDir()
	prFile := filepath.Join(tmpDir, "pr.yaml")
	if err := os.WriteFile(prFile, []byte(prYAML), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	p := New(Options{})
	pr, err := p.ParsePipelineRun(prFile)
	if err != nil {
		t.Fatalf("ParsePipelineRun() error = %v", err)
	}

	if len(pr.Tasks) != 1 {
		t.Fatalf("len(Tasks) = %d, want 1", len(pr.Tasks))
	}

	task := pr.Tasks[0]

	// Check volumes
	if len(task.Volumes) != 3 {
		t.Fatalf("len(Volumes) = %d, want 3", len(task.Volumes))
	}

	// Check step volumeMounts
	if len(task.Steps) != 1 {
		t.Fatalf("len(Steps) = %d, want 1", len(task.Steps))
	}
	step := task.Steps[0]
	if len(step.VolumeMounts) != 2 {
		t.Fatalf("len(VolumeMounts) = %d, want 2", len(step.VolumeMounts))
	}
	if step.VolumeMounts[0].Name != "temp" {
		t.Errorf("VolumeMounts[0].Name = %q, want %q", step.VolumeMounts[0].Name, "temp")
	}
	if step.VolumeMounts[0].MountPath != "/tmp/data" {
		t.Errorf("VolumeMounts[0].MountPath = %q, want %q", step.VolumeMounts[0].MountPath, "/tmp/data")
	}
}

func TestParsePipelineRun_Results(t *testing.T) {
	prYAML := `
apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  name: results-run
spec:
  pipelineSpec:
    tasks:
      - name: produce
        taskSpec:
          results:
            - name: version
              description: The version string
            - name: commit
              description: The git commit
          steps:
            - name: generate
              image: alpine
              script: |
                echo "1.0.0" > /tekton/results/version
                echo "abc123" > /tekton/results/commit
`
	tmpDir := t.TempDir()
	prFile := filepath.Join(tmpDir, "pr.yaml")
	if err := os.WriteFile(prFile, []byte(prYAML), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	p := New(Options{})
	pr, err := p.ParsePipelineRun(prFile)
	if err != nil {
		t.Fatalf("ParsePipelineRun() error = %v", err)
	}

	if len(pr.Tasks) != 1 {
		t.Fatalf("len(Tasks) = %d, want 1", len(pr.Tasks))
	}

	task := pr.Tasks[0]
	if len(task.Results) != 2 {
		t.Fatalf("len(Results) = %d, want 2", len(task.Results))
	}
	if task.Results[0].Name != "version" {
		t.Errorf("Results[0].Name = %q, want %q", task.Results[0].Name, "version")
	}
	if task.Results[1].Name != "commit" {
		t.Errorf("Results[1].Name = %q, want %q", task.Results[1].Name, "commit")
	}
}

func TestParseParamValue(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		wantType types.ParamType
		wantStr  string
		wantArr  []string
		wantObj  map[string]string
	}{
		{
			name:     "string value",
			input:    "hello",
			wantType: types.ParamTypeString,
			wantStr:  "hello",
		},
		{
			name:     "array value",
			input:    []interface{}{"a", "b", "c"},
			wantType: types.ParamTypeArray,
			wantArr:  []string{"a", "b", "c"},
		},
		{
			name:     "object value",
			input:    map[string]interface{}{"key": "value", "foo": "bar"},
			wantType: types.ParamTypeObject,
			wantObj:  map[string]string{"key": "value", "foo": "bar"},
		},
		{
			name:     "integer converts to string",
			input:    42,
			wantType: types.ParamTypeString,
			wantStr:  "42",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseParamValue(tc.input)
			if got.Type != tc.wantType {
				t.Errorf("Type = %v, want %v", got.Type, tc.wantType)
			}
			switch tc.wantType {
			case types.ParamTypeString:
				if got.StringVal != tc.wantStr {
					t.Errorf("StringVal = %q, want %q", got.StringVal, tc.wantStr)
				}
			case types.ParamTypeArray:
				if len(got.ArrayVal) != len(tc.wantArr) {
					t.Errorf("len(ArrayVal) = %d, want %d", len(got.ArrayVal), len(tc.wantArr))
				}
				for i, v := range tc.wantArr {
					if got.ArrayVal[i] != v {
						t.Errorf("ArrayVal[%d] = %q, want %q", i, got.ArrayVal[i], v)
					}
				}
			case types.ParamTypeObject:
				if len(got.ObjectVal) != len(tc.wantObj) {
					t.Errorf("len(ObjectVal) = %d, want %d", len(got.ObjectVal), len(tc.wantObj))
				}
				for k, v := range tc.wantObj {
					if got.ObjectVal[k] != v {
						t.Errorf("ObjectVal[%q] = %q, want %q", k, got.ObjectVal[k], v)
					}
				}
			}
		})
	}
}

func TestParsePipelineRun_Errors(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name:    "invalid yaml",
			yaml:    "not: valid: yaml: [",
			wantErr: "failed to parse YAML",
		},
		{
			name: "unsupported kind",
			yaml: `
apiVersion: tekton.dev/v1
kind: TaskRun
metadata:
  name: unsupported
`,
			wantErr: "unsupported kind",
		},
		{
			name: "missing pipelineRef and pipelineSpec",
			yaml: `
apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  name: broken
spec:
  params: []
`,
			wantErr: "must have either pipelineRef or pipelineSpec",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			file := filepath.Join(tmpDir, "test.yaml")
			if err := os.WriteFile(file, []byte(tc.yaml), 0644); err != nil {
				t.Fatalf("failed to write file: %v", err)
			}

			p := New(Options{})
			_, err := p.ParsePipelineRun(file)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !contains(err.Error(), tc.wantErr) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestParsePipelineRun_FileNotFound(t *testing.T) {
	p := New(Options{})
	_, err := p.ParsePipelineRun("/nonexistent/file.yaml")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !contains(err.Error(), "failed to read file") {
		t.Errorf("error = %q, want to contain 'failed to read file'", err.Error())
	}
}

func TestParsePipelineRun_TaskRefNotFound(t *testing.T) {
	prYAML := `
apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  name: missing-task-run
spec:
  pipelineSpec:
    tasks:
      - name: missing
        taskRef:
          name: nonexistent-task
`
	tmpDir := t.TempDir()
	prFile := filepath.Join(tmpDir, "pr.yaml")
	if err := os.WriteFile(prFile, []byte(prYAML), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	p := New(Options{})
	_, err := p.ParsePipelineRun(prFile)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !contains(err.Error(), "not found") {
		t.Errorf("error = %q, want to contain 'not found'", err.Error())
	}
}

func TestParsePipelineRun_StepDetails(t *testing.T) {
	prYAML := `
apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  name: step-details-run
spec:
  pipelineSpec:
    tasks:
      - name: detailed-task
        taskSpec:
          steps:
            - name: full-step
              image: golang:1.21
              command: ["go"]
              args: ["build", "-o", "app", "."]
              workingDir: /workspace/src
              env:
                - name: GOOS
                  value: linux
                - name: GOARCH
                  value: amd64
`
	tmpDir := t.TempDir()
	prFile := filepath.Join(tmpDir, "pr.yaml")
	if err := os.WriteFile(prFile, []byte(prYAML), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	p := New(Options{})
	pr, err := p.ParsePipelineRun(prFile)
	if err != nil {
		t.Fatalf("ParsePipelineRun() error = %v", err)
	}

	step := pr.Tasks[0].Steps[0]

	if step.Name != "full-step" {
		t.Errorf("Name = %q, want %q", step.Name, "full-step")
	}
	if step.Image != "golang:1.21" {
		t.Errorf("Image = %q, want %q", step.Image, "golang:1.21")
	}
	if len(step.Command) != 1 || step.Command[0] != "go" {
		t.Errorf("Command = %v, want [go]", step.Command)
	}
	if len(step.Args) != 4 {
		t.Errorf("len(Args) = %d, want 4", len(step.Args))
	}
	if step.WorkingDir != "/workspace/src" {
		t.Errorf("WorkingDir = %q, want %q", step.WorkingDir, "/workspace/src")
	}
	if step.Env["GOOS"] != "linux" {
		t.Errorf("Env[GOOS] = %q, want %q", step.Env["GOOS"], "linux")
	}
	if step.Env["GOARCH"] != "amd64" {
		t.Errorf("Env[GOARCH] = %q, want %q", step.Env["GOARCH"], "amd64")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
