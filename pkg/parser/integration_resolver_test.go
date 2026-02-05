package parser

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestHTTPResolverIntegration tests the HTTP resolver with a full pipeline
func TestHTTPResolverIntegration(t *testing.T) {
	// Create a test task YAML
	taskYAML := `apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: http-task
spec:
  params:
  - name: message
    type: string
    default: "hello"
  steps:
  - name: echo
    image: alpine
    script: |
      echo $(params.message)
`

	// Create HTTP server to serve the task
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-yaml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(taskYAML))
	}))
	defer server.Close()

	// Create a pipeline that references the task via HTTP
	pipelineYAML := `apiVersion: tekton.dev/v1
kind: Pipeline
metadata:
  name: http-resolver-pipeline
spec:
  tasks:
  - name: fetch-task
    taskRef:
      resolver: http
      params:
      - name: url
        value: ` + server.URL + `
    params:
    - name: message
      value: "Hello from HTTP resolver!"
`

	// Write pipeline to temp file
	tmpDir := t.TempDir()
	pipelinePath := filepath.Join(tmpDir, "pipeline.yaml")
	err := os.WriteFile(pipelinePath, []byte(pipelineYAML), 0644)
	if err != nil {
		t.Fatalf("failed to write pipeline file: %v", err)
	}

	// Parse the pipeline
	parser := New(Options{})
	resolved, err := parser.ParsePipelineRun(pipelinePath)
	if err != nil {
		t.Fatalf("failed to parse pipeline: %v", err)
	}

	// Verify the pipeline was resolved correctly
	if resolved.PipelineName != "http-resolver-pipeline" {
		t.Errorf("expected pipeline name http-resolver-pipeline, got %s", resolved.PipelineName)
	}

	if len(resolved.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(resolved.Tasks))
	}

	task := resolved.Tasks[0]
	if task.Name != "fetch-task" {
		t.Errorf("expected task name fetch-task, got %s", task.Name)
	}

	if task.TaskName != "http-task" {
		t.Errorf("expected task definition name http-task, got %s", task.TaskName)
	}

	if len(task.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(task.Steps))
	}

	if task.Steps[0].Name != "echo" {
		t.Errorf("expected step name echo, got %s", task.Steps[0].Name)
	}

	// Verify parameter was passed
	if messageParam, ok := task.Params["message"]; ok {
		if messageParam.StringVal != "Hello from HTTP resolver!" {
			t.Errorf("expected param value 'Hello from HTTP resolver!', got %s", messageParam.StringVal)
		}
	} else {
		t.Error("expected message parameter to be set")
	}
}

// TestMixedResolvers tests using both local and HTTP resolvers in the same pipeline
func TestMixedResolvers(t *testing.T) {
	// Create HTTP task
	httpTaskYAML := `apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: remote-task
spec:
  steps:
  - name: remote-step
    image: alpine
    script: echo "from HTTP"
`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-yaml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(httpTaskYAML))
	}))
	defer server.Close()

	// Create local task file
	tmpDir := t.TempDir()
	tasksDir := filepath.Join(tmpDir, "tasks")
	os.MkdirAll(tasksDir, 0755)

	localTaskYAML := `apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: local-task
spec:
  steps:
  - name: local-step
    image: alpine
    script: echo "from local file"
`
	localTaskPath := filepath.Join(tasksDir, "local-task.yaml")
	os.WriteFile(localTaskPath, []byte(localTaskYAML), 0644)

	// Create pipeline using both resolvers
	pipelineYAML := `apiVersion: tekton.dev/v1
kind: Pipeline
metadata:
  name: mixed-pipeline
spec:
  tasks:
  - name: local
    taskRef:
      name: local-task
  - name: remote
    taskRef:
      resolver: http
      params:
      - name: url
        value: ` + server.URL + `
    runAfter:
    - local
`

	pipelinePath := filepath.Join(tmpDir, "pipeline.yaml")
	os.WriteFile(pipelinePath, []byte(pipelineYAML), 0644)

	// Parse the pipeline
	parser := New(Options{TasksDir: tasksDir})
	resolved, err := parser.ParsePipelineRun(pipelinePath)
	if err != nil {
		t.Fatalf("failed to parse pipeline: %v", err)
	}

	if len(resolved.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(resolved.Tasks))
	}

	// Verify local task
	localTask := resolved.Tasks[0]
	if localTask.TaskName != "local-task" {
		t.Errorf("expected local-task, got %s", localTask.TaskName)
	}

	// Verify remote task
	remoteTask := resolved.Tasks[1]
	if remoteTask.TaskName != "remote-task" {
		t.Errorf("expected remote-task, got %s", remoteTask.TaskName)
	}

	// Verify runAfter dependency
	if len(remoteTask.RunAfter) != 1 || remoteTask.RunAfter[0] != "local" {
		t.Errorf("expected remote task to run after local, got %v", remoteTask.RunAfter)
	}
}
