package dagger

import (
	"testing"

	"github.com/vdemeester/chisel/pkg/types"
)

func TestSubstituteVariables_StringParam(t *testing.T) {
	e := &DaggerBackend{
		results: make(map[string]map[string]string),
	}
	task := &types.ResolvedTask{
		Name:     "test-task",
		TaskName: "test",
		Params: map[string]types.ParamValue{
			"greeting": {Type: types.ParamTypeString, StringVal: "hello"},
		},
		Workspaces: make(map[string]string),
	}
	pr := &types.ResolvedPipelineRun{
		Name:         "test-run",
		PipelineName: "test-pipeline",
	}

	input := "echo $(params.greeting)"
	got := e.substituteVariables(input, task, pr)
	want := "echo hello"

	if got != want {
		t.Errorf("substituteVariables() = %q, want %q", got, want)
	}
}

func TestSubstituteVariables_ArrayParamStar(t *testing.T) {
	// $(params.myarray[*]) should expand to space-separated values
	e := &DaggerBackend{
		results: make(map[string]map[string]string),
	}
	task := &types.ResolvedTask{
		Name:     "test-task",
		TaskName: "test",
		Params: map[string]types.ParamValue{
			"packages": {Type: types.ParamTypeArray, ArrayVal: []string{"foo", "bar", "baz"}},
		},
		Workspaces: make(map[string]string),
	}
	pr := &types.ResolvedPipelineRun{
		Name:         "test-run",
		PipelineName: "test-pipeline",
	}

	input := "go build $(params.packages[*])"
	got := e.substituteVariables(input, task, pr)
	want := "go build foo bar baz"

	if got != want {
		t.Errorf("substituteVariables() = %q, want %q", got, want)
	}
}

func TestSubstituteVariables_ArrayParamIndex(t *testing.T) {
	// $(params.myarray[0]) should expand to single indexed value
	e := &DaggerBackend{
		results: make(map[string]map[string]string),
	}
	task := &types.ResolvedTask{
		Name:     "test-task",
		TaskName: "test",
		Params: map[string]types.ParamValue{
			"urls": {Type: types.ParamTypeArray, ArrayVal: []string{"https://a.com", "https://b.com", "https://c.com"}},
		},
		Workspaces: make(map[string]string),
	}
	pr := &types.ResolvedPipelineRun{
		Name:         "test-run",
		PipelineName: "test-pipeline",
	}

	tests := []struct {
		input string
		want  string
	}{
		{"curl $(params.urls[0])", "curl https://a.com"},
		{"curl $(params.urls[1])", "curl https://b.com"},
		{"curl $(params.urls[2])", "curl https://c.com"},
	}

	for _, tc := range tests {
		got := e.substituteVariables(tc.input, task, pr)
		if got != tc.want {
			t.Errorf("substituteVariables(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestSubstituteVariables_ArrayParamIndexOutOfBounds(t *testing.T) {
	// $(params.myarray[99]) with only 3 elements should leave placeholder or return empty
	e := &DaggerBackend{
		results: make(map[string]map[string]string),
	}
	task := &types.ResolvedTask{
		Name:     "test-task",
		TaskName: "test",
		Params: map[string]types.ParamValue{
			"items": {Type: types.ParamTypeArray, ArrayVal: []string{"a", "b", "c"}},
		},
		Workspaces: make(map[string]string),
	}
	pr := &types.ResolvedPipelineRun{
		Name:         "test-run",
		PipelineName: "test-pipeline",
	}

	input := "echo $(params.items[99])"
	got := e.substituteVariables(input, task, pr)
	// Out of bounds should expand to empty string
	want := "echo "

	if got != want {
		t.Errorf("substituteVariables() = %q, want %q", got, want)
	}
}

func TestSubstituteVariables_ObjectParamField(t *testing.T) {
	// $(params.myobj.field) should expand to the field value
	e := &DaggerBackend{
		results: make(map[string]map[string]string),
	}
	task := &types.ResolvedTask{
		Name:     "test-task",
		TaskName: "test",
		Params: map[string]types.ParamValue{
			"config": {Type: types.ParamTypeObject, ObjectVal: map[string]string{
				"host": "localhost",
				"port": "8080",
				"path": "/api",
			}},
		},
		Workspaces: make(map[string]string),
	}
	pr := &types.ResolvedPipelineRun{
		Name:         "test-run",
		PipelineName: "test-pipeline",
	}

	tests := []struct {
		input string
		want  string
	}{
		{"http://$(params.config.host):$(params.config.port)$(params.config.path)", "http://localhost:8080/api"},
		{"HOST=$(params.config.host)", "HOST=localhost"},
	}

	for _, tc := range tests {
		got := e.substituteVariables(tc.input, task, pr)
		if got != tc.want {
			t.Errorf("substituteVariables(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestSubstituteVariables_ObjectParamMissingField(t *testing.T) {
	// $(params.myobj.missing) should expand to empty string
	e := &DaggerBackend{
		results: make(map[string]map[string]string),
	}
	task := &types.ResolvedTask{
		Name:     "test-task",
		TaskName: "test",
		Params: map[string]types.ParamValue{
			"config": {Type: types.ParamTypeObject, ObjectVal: map[string]string{
				"host": "localhost",
			}},
		},
		Workspaces: make(map[string]string),
	}
	pr := &types.ResolvedPipelineRun{
		Name:         "test-run",
		PipelineName: "test-pipeline",
	}

	input := "VALUE=$(params.config.missing)"
	got := e.substituteVariables(input, task, pr)
	want := "VALUE="

	if got != want {
		t.Errorf("substituteVariables() = %q, want %q", got, want)
	}
}

func TestSubstituteVariables_MixedParams(t *testing.T) {
	// Mix of string, array, and object params in one input
	e := &DaggerBackend{
		results: make(map[string]map[string]string),
	}
	task := &types.ResolvedTask{
		Name:     "test-task",
		TaskName: "test",
		Params: map[string]types.ParamValue{
			"name":  {Type: types.ParamTypeString, StringVal: "myapp"},
			"tags":  {Type: types.ParamTypeArray, ArrayVal: []string{"v1.0", "latest"}},
			"build": {Type: types.ParamTypeObject, ObjectVal: map[string]string{"target": "linux", "arch": "amd64"}},
		},
		Workspaces: make(map[string]string),
	}
	pr := &types.ResolvedPipelineRun{
		Name:         "test-run",
		PipelineName: "test-pipeline",
	}

	input := "Building $(params.name) for $(params.build.target)/$(params.build.arch) with tags: $(params.tags[*])"
	got := e.substituteVariables(input, task, pr)
	want := "Building myapp for linux/amd64 with tags: v1.0 latest"

	if got != want {
		t.Errorf("substituteVariables() = %q, want %q", got, want)
	}
}
