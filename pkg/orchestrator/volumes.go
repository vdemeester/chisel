package orchestrator

import (
	"github.com/vdemeester/chisel/pkg/types"
)

// VolumeType indicates the type of volume.
type VolumeType string

const (
	VolumeTypeEmptyDir  VolumeType = "emptyDir"
	VolumeTypeConfigMap VolumeType = "configMap"
	VolumeTypeSecret    VolumeType = "secret"
)

// ParsedVolume represents a parsed volume ready for mounting.
type ParsedVolume struct {
	Name          string
	Type          VolumeType
	ConfigMapName string
	SecretName    string
	Items         []types.KeyToPath
	Medium        string // for emptyDir
}

// ParsedVolumeMount represents a parsed volume mount.
type ParsedVolumeMount struct {
	Name      string
	MountPath string
	SubPath   string
	ReadOnly  bool
}

// ParseVolumes converts Volume definitions to ParsedVolume map.
func ParseVolumes(volumes []types.Volume) map[string]*ParsedVolume {
	result := make(map[string]*ParsedVolume)

	for _, v := range volumes {
		pv := &ParsedVolume{
			Name: v.Name,
		}

		if v.EmptyDir != nil {
			pv.Type = VolumeTypeEmptyDir
			pv.Medium = v.EmptyDir.Medium
		} else if v.ConfigMap != nil {
			pv.Type = VolumeTypeConfigMap
			pv.ConfigMapName = v.ConfigMap.Name
			pv.Items = v.ConfigMap.Items
		} else if v.Secret != nil {
			pv.Type = VolumeTypeSecret
			pv.SecretName = v.Secret.SecretName
			pv.Items = v.Secret.Items
		}

		result[v.Name] = pv
	}

	return result
}

// parseVolumeMount converts a VolumeMount to ParsedVolumeMount.
func parseVolumeMount(mount types.VolumeMount) *ParsedVolumeMount {
	return &ParsedVolumeMount{
		Name:      mount.Name,
		MountPath: mount.MountPath,
		SubPath:   mount.SubPath,
		ReadOnly:  mount.ReadOnly,
	}
}
