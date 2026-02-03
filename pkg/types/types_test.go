package types

import "testing"

func TestParamValue_String(t *testing.T) {
	tests := []struct {
		name  string
		param ParamValue
		want  string
	}{
		{
			name: "string type returns StringVal",
			param: ParamValue{
				Type:      ParamTypeString,
				StringVal: "hello",
			},
			want: "hello",
		},
		{
			name: "empty string type",
			param: ParamValue{
				Type:      ParamTypeString,
				StringVal: "",
			},
			want: "",
		},
		{
			name: "array type returns empty string",
			param: ParamValue{
				Type:     ParamTypeArray,
				ArrayVal: []string{"a", "b", "c"},
			},
			want: "",
		},
		{
			name: "object type returns empty string",
			param: ParamValue{
				Type:      ParamTypeObject,
				ObjectVal: map[string]string{"key": "value"},
			},
			want: "",
		},
		{
			name:  "zero value returns empty string",
			param: ParamValue{},
			want:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.param.String()
			if got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParamType_Constants(t *testing.T) {
	// Verify constant values match expected strings
	if ParamTypeString != "string" {
		t.Errorf("ParamTypeString = %q, want %q", ParamTypeString, "string")
	}
	if ParamTypeArray != "array" {
		t.Errorf("ParamTypeArray = %q, want %q", ParamTypeArray, "array")
	}
	if ParamTypeObject != "object" {
		t.Errorf("ParamTypeObject = %q, want %q", ParamTypeObject, "object")
	}
}

func TestWorkspaceType_Constants(t *testing.T) {
	if WorkspaceTypeEmptyDir != "emptyDir" {
		t.Errorf("WorkspaceTypeEmptyDir = %q, want %q", WorkspaceTypeEmptyDir, "emptyDir")
	}
	if WorkspaceTypeLocal != "local" {
		t.Errorf("WorkspaceTypeLocal = %q, want %q", WorkspaceTypeLocal, "local")
	}
	if WorkspaceTypePVC != "pvc" {
		t.Errorf("WorkspaceTypePVC = %q, want %q", WorkspaceTypePVC, "pvc")
	}
}

func TestResolvedPipelineRun_ZeroValue(t *testing.T) {
	// Verify zero value is usable
	var pr ResolvedPipelineRun
	if pr.Name != "" {
		t.Errorf("Name = %q, want empty", pr.Name)
	}
	if pr.Tasks != nil {
		t.Errorf("Tasks = %v, want nil", pr.Tasks)
	}
	if pr.Params != nil {
		t.Errorf("Params = %v, want nil", pr.Params)
	}
}

func TestResolvedTask_ZeroValue(t *testing.T) {
	var task ResolvedTask
	if task.Name != "" {
		t.Errorf("Name = %q, want empty", task.Name)
	}
	if task.Steps != nil {
		t.Errorf("Steps = %v, want nil", task.Steps)
	}
	if len(task.RunAfter) != 0 {
		t.Errorf("RunAfter = %v, want empty", task.RunAfter)
	}
}

func TestStep_Fields(t *testing.T) {
	step := Step{
		Name:       "test-step",
		Image:      "alpine:latest",
		Command:    []string{"sh", "-c"},
		Args:       []string{"echo hello"},
		Script:     "",
		Env:        map[string]string{"FOO": "bar"},
		WorkingDir: "/workspace",
		VolumeMounts: []VolumeMount{
			{Name: "data", MountPath: "/data"},
		},
	}

	if step.Name != "test-step" {
		t.Errorf("Name = %q, want %q", step.Name, "test-step")
	}
	if step.Image != "alpine:latest" {
		t.Errorf("Image = %q, want %q", step.Image, "alpine:latest")
	}
	if len(step.Command) != 2 {
		t.Errorf("len(Command) = %d, want 2", len(step.Command))
	}
	if step.Env["FOO"] != "bar" {
		t.Errorf("Env[FOO] = %q, want %q", step.Env["FOO"], "bar")
	}
	if len(step.VolumeMounts) != 1 {
		t.Errorf("len(VolumeMounts) = %d, want 1", len(step.VolumeMounts))
	}
}

func TestVolume_EmptyDir(t *testing.T) {
	vol := Volume{
		Name: "temp",
		VolumeSource: VolumeSource{
			EmptyDir: &EmptyDirVolumeSource{Medium: "Memory"},
		},
	}

	if vol.Name != "temp" {
		t.Errorf("Name = %q, want %q", vol.Name, "temp")
	}
	if vol.EmptyDir == nil {
		t.Fatal("EmptyDir is nil")
	}
	if vol.EmptyDir.Medium != "Memory" {
		t.Errorf("Medium = %q, want %q", vol.EmptyDir.Medium, "Memory")
	}
	if vol.ConfigMap != nil {
		t.Error("ConfigMap should be nil")
	}
	if vol.Secret != nil {
		t.Error("Secret should be nil")
	}
}

func TestVolume_ConfigMap(t *testing.T) {
	vol := Volume{
		Name: "config",
		VolumeSource: VolumeSource{
			ConfigMap: &ConfigMapVolumeSource{
				Name: "my-config",
				Items: []KeyToPath{
					{Key: "config.yaml", Path: "app/config.yaml"},
				},
			},
		},
	}

	if vol.ConfigMap == nil {
		t.Fatal("ConfigMap is nil")
	}
	if vol.ConfigMap.Name != "my-config" {
		t.Errorf("ConfigMap.Name = %q, want %q", vol.ConfigMap.Name, "my-config")
	}
	if len(vol.ConfigMap.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(vol.ConfigMap.Items))
	}
	if vol.ConfigMap.Items[0].Key != "config.yaml" {
		t.Errorf("Items[0].Key = %q, want %q", vol.ConfigMap.Items[0].Key, "config.yaml")
	}
}

func TestVolume_Secret(t *testing.T) {
	vol := Volume{
		Name: "creds",
		VolumeSource: VolumeSource{
			Secret: &SecretVolumeSource{
				SecretName: "my-secret",
				Items: []KeyToPath{
					{Key: "password", Path: ".password"},
				},
			},
		},
	}

	if vol.Secret == nil {
		t.Fatal("Secret is nil")
	}
	if vol.Secret.SecretName != "my-secret" {
		t.Errorf("SecretName = %q, want %q", vol.Secret.SecretName, "my-secret")
	}
	if len(vol.Secret.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(vol.Secret.Items))
	}
}

func TestVolumeMount_Fields(t *testing.T) {
	mount := VolumeMount{
		Name:      "data",
		MountPath: "/mnt/data",
		SubPath:   "subdir",
		ReadOnly:  true,
	}

	if mount.Name != "data" {
		t.Errorf("Name = %q, want %q", mount.Name, "data")
	}
	if mount.MountPath != "/mnt/data" {
		t.Errorf("MountPath = %q, want %q", mount.MountPath, "/mnt/data")
	}
	if mount.SubPath != "subdir" {
		t.Errorf("SubPath = %q, want %q", mount.SubPath, "subdir")
	}
	if !mount.ReadOnly {
		t.Error("ReadOnly = false, want true")
	}
}

func TestWorkspaceBinding_Fields(t *testing.T) {
	binding := WorkspaceBinding{
		Name:    "source",
		Type:    WorkspaceTypeLocal,
		Path:    "/home/user/project",
		SubPath: "src",
	}

	if binding.Name != "source" {
		t.Errorf("Name = %q, want %q", binding.Name, "source")
	}
	if binding.Type != WorkspaceTypeLocal {
		t.Errorf("Type = %v, want %v", binding.Type, WorkspaceTypeLocal)
	}
	if binding.Path != "/home/user/project" {
		t.Errorf("Path = %q, want %q", binding.Path, "/home/user/project")
	}
	if binding.SubPath != "src" {
		t.Errorf("SubPath = %q, want %q", binding.SubPath, "src")
	}
}

func TestSidecar_Fields(t *testing.T) {
	sidecar := Sidecar{
		Name:    "redis",
		Image:   "redis:latest",
		Command: []string{"redis-server"},
		Args:    []string{"--appendonly", "yes"},
		Env:     map[string]string{"REDIS_PASSWORD": "secret"},
		Ports:   []int{6379},
	}

	if sidecar.Name != "redis" {
		t.Errorf("Name = %q, want %q", sidecar.Name, "redis")
	}
	if sidecar.Image != "redis:latest" {
		t.Errorf("Image = %q, want %q", sidecar.Image, "redis:latest")
	}
	if len(sidecar.Ports) != 1 || sidecar.Ports[0] != 6379 {
		t.Errorf("Ports = %v, want [6379]", sidecar.Ports)
	}
}

func TestResultSpec_Fields(t *testing.T) {
	result := ResultSpec{
		Name:        "version",
		Description: "The build version",
	}

	if result.Name != "version" {
		t.Errorf("Name = %q, want %q", result.Name, "version")
	}
	if result.Description != "The build version" {
		t.Errorf("Description = %q, want %q", result.Description, "The build version")
	}
}
