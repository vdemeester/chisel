package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vdemeester/chisel/pkg/types"
)

func TestParseWorkspaceBinding(t *testing.T) {
	// Get current directory for relative path tests
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		wantName string
		wantPath string
		wantErr  bool
	}{
		{
			name:     "current directory",
			input:    "source:.",
			wantName: "source",
			wantPath: cwd,
			wantErr:  false,
		},
		{
			name:     "absolute path",
			input:    "source:/tmp/myproject",
			wantName: "source",
			wantPath: "/tmp/myproject",
			wantErr:  false,
		},
		{
			name:     "relative path",
			input:    "config:./config",
			wantName: "config",
			wantPath: filepath.Join(cwd, "config"),
			wantErr:  false,
		},
		{
			name:    "missing colon",
			input:   "source",
			wantErr: true,
		},
		{
			name:    "empty name",
			input:   ":./path",
			wantErr: true,
		},
		{
			name:    "empty path",
			input:   "source:",
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			name, path, err := parseWorkspaceBinding(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("parseWorkspaceBinding(%q) expected error, got nil", tc.input)
				}
				return
			}
			if err != nil {
				t.Errorf("parseWorkspaceBinding(%q) unexpected error: %v", tc.input, err)
				return
			}
			if name != tc.wantName {
				t.Errorf("parseWorkspaceBinding(%q) name = %q, want %q", tc.input, name, tc.wantName)
			}
			if path != tc.wantPath {
				t.Errorf("parseWorkspaceBinding(%q) path = %q, want %q", tc.input, path, tc.wantPath)
			}
		})
	}
}

func TestParseWorkspaceBindings(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tests := []struct {
		name    string
		inputs  []string
		want    map[string]string
		wantErr bool
	}{
		{
			name:   "single binding",
			inputs: []string{"source:."},
			want: map[string]string{
				"source": cwd,
			},
		},
		{
			name:   "multiple bindings",
			inputs: []string{"source:.", "config:/etc/config"},
			want: map[string]string{
				"source": cwd,
				"config": "/etc/config",
			},
		},
		{
			name:   "empty list",
			inputs: []string{},
			want:   map[string]string{},
		},
		{
			name:    "invalid binding",
			inputs:  []string{"invalid"},
			wantErr: true,
		},
		{
			name:    "one valid one invalid",
			inputs:  []string{"source:.", "invalid"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseWorkspaceBindings(tc.inputs)
			if tc.wantErr {
				if err == nil {
					t.Errorf("parseWorkspaceBindings(%v) expected error, got nil", tc.inputs)
				}
				return
			}
			if err != nil {
				t.Errorf("parseWorkspaceBindings(%v) unexpected error: %v", tc.inputs, err)
				return
			}
			if len(got) != len(tc.want) {
				t.Errorf("parseWorkspaceBindings(%v) = %v, want %v", tc.inputs, got, tc.want)
				return
			}
			for k, v := range tc.want {
				if got[k] != v {
					t.Errorf("parseWorkspaceBindings(%v)[%q] = %q, want %q", tc.inputs, k, got[k], v)
				}
			}
		})
	}
}

func TestApplyWorkspaceOverrides(t *testing.T) {
	tests := []struct {
		name      string
		pr        *types.ResolvedPipelineRun
		overrides map[string]string
		wantType  types.WorkspaceType
		wantPath  string
	}{
		{
			name: "override existing workspace",
			pr: &types.ResolvedPipelineRun{
				Workspaces: map[string]types.WorkspaceBinding{
					"source": {Name: "source", Type: types.WorkspaceTypeEmptyDir},
				},
			},
			overrides: map[string]string{"source": "/tmp/mycode"},
			wantType:  types.WorkspaceTypeLocal,
			wantPath:  "/tmp/mycode",
		},
		{
			name: "add new workspace",
			pr: &types.ResolvedPipelineRun{
				Workspaces: map[string]types.WorkspaceBinding{},
			},
			overrides: map[string]string{"source": "/tmp/mycode"},
			wantType:  types.WorkspaceTypeLocal,
			wantPath:  "/tmp/mycode",
		},
		{
			name: "override PVC workspace",
			pr: &types.ResolvedPipelineRun{
				Workspaces: map[string]types.WorkspaceBinding{
					"data": {Name: "data", Type: types.WorkspaceTypePVC, Path: "my-pvc"},
				},
			},
			overrides: map[string]string{"data": "/home/user/data"},
			wantType:  types.WorkspaceTypeLocal,
			wantPath:  "/home/user/data",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := applyWorkspaceOverrides(tc.pr, tc.overrides)
			if err != nil {
				t.Fatalf("applyWorkspaceOverrides() unexpected error: %v", err)
			}

			for wsName, expectedPath := range tc.overrides {
				ws, exists := tc.pr.Workspaces[wsName]
				if !exists {
					t.Errorf("workspace %q not found after override", wsName)
					continue
				}
				if ws.Type != tc.wantType {
					t.Errorf("workspace %q Type = %v, want %v", wsName, ws.Type, tc.wantType)
				}
				if ws.Path != expectedPath {
					t.Errorf("workspace %q Path = %q, want %q", wsName, ws.Path, expectedPath)
				}
			}
		})
	}
}
