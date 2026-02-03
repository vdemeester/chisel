package executor

import (
	"testing"

	"github.com/vdemeester/chisel/pkg/types"
)

func TestParseVolumes_EmptyDir(t *testing.T) {
	volumes := []types.Volume{
		{
			Name: "temp-data",
			VolumeSource: types.VolumeSource{
				EmptyDir: &types.EmptyDirVolumeSource{},
			},
		},
	}

	parsed := parseVolumes(volumes)

	if len(parsed) != 1 {
		t.Fatalf("Expected 1 volume, got %d", len(parsed))
	}
	if parsed["temp-data"].Type != VolumeTypeEmptyDir {
		t.Errorf("Expected EmptyDir type, got %v", parsed["temp-data"].Type)
	}
}

func TestParseVolumes_ConfigMap(t *testing.T) {
	volumes := []types.Volume{
		{
			Name: "config",
			VolumeSource: types.VolumeSource{
				ConfigMap: &types.ConfigMapVolumeSource{
					Name: "my-config",
					Items: []types.KeyToPath{
						{Key: "config.yaml", Path: "config.yaml"},
					},
				},
			},
		},
	}

	parsed := parseVolumes(volumes)

	if len(parsed) != 1 {
		t.Fatalf("Expected 1 volume, got %d", len(parsed))
	}
	if parsed["config"].Type != VolumeTypeConfigMap {
		t.Errorf("Expected ConfigMap type, got %v", parsed["config"].Type)
	}
	if parsed["config"].ConfigMapName != "my-config" {
		t.Errorf("Expected configmap name 'my-config', got %s", parsed["config"].ConfigMapName)
	}
}

func TestParseVolumes_Secret(t *testing.T) {
	volumes := []types.Volume{
		{
			Name: "creds",
			VolumeSource: types.VolumeSource{
				Secret: &types.SecretVolumeSource{
					SecretName: "my-secret",
				},
			},
		},
	}

	parsed := parseVolumes(volumes)

	if len(parsed) != 1 {
		t.Fatalf("Expected 1 volume, got %d", len(parsed))
	}
	if parsed["creds"].Type != VolumeTypeSecret {
		t.Errorf("Expected Secret type, got %v", parsed["creds"].Type)
	}
	if parsed["creds"].SecretName != "my-secret" {
		t.Errorf("Expected secret name 'my-secret', got %s", parsed["creds"].SecretName)
	}
}

func TestParseVolumeMount(t *testing.T) {
	mount := types.VolumeMount{
		Name:      "config",
		MountPath: "/etc/app",
		ReadOnly:  true,
	}

	parsed := parseVolumeMount(mount)

	if parsed.Name != "config" {
		t.Errorf("Expected name 'config', got %s", parsed.Name)
	}
	if parsed.MountPath != "/etc/app" {
		t.Errorf("Expected mount path '/etc/app', got %s", parsed.MountPath)
	}
	if !parsed.ReadOnly {
		t.Error("Expected ReadOnly to be true")
	}
}

func TestParseVolumeMount_WithSubPath(t *testing.T) {
	mount := types.VolumeMount{
		Name:      "data",
		MountPath: "/app/data",
		SubPath:   "subdir",
	}

	parsed := parseVolumeMount(mount)

	if parsed.SubPath != "subdir" {
		t.Errorf("Expected subpath 'subdir', got %s", parsed.SubPath)
	}
}
