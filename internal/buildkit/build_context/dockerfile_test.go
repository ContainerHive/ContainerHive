package build_context

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDockerfileBuildContext_FileName(t *testing.T) {
	tests := []struct {
		name       string
		dockerfile string
		expected   string
	}{
		{
			name:       "empty dockerfile uses default",
			dockerfile: "",
			expected:   "Dockerfile",
		},
		{
			name:       "simple filename",
			dockerfile: "Dockerfile.prod",
			expected:   "Dockerfile.prod",
		},
		{
			name:       "path with directory",
			dockerfile: "docker/Dockerfile.dev",
			expected:   "docker/Dockerfile.dev",
		},
		{
			name:       "nested path",
			dockerfile: "path/to/custom/Dockerfile",
			expected:   "path/to/custom/Dockerfile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := DockerfileBuildContext{
				Dockerfile: tt.dockerfile,
			}
			got := d.FileName()
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestDockerfileBuildContext_FrontendType(t *testing.T) {
	d := DockerfileBuildContext{}
	expected := "dockerfile.v0"
	got := d.FrontendType()
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestDockerfileBuildContext_ToLocalMounts(t *testing.T) {
	// Create temporary directory structure for testing
	tmpDir := t.TempDir()
	dockerfileDir := filepath.Join(tmpDir, "docker")
	if err := os.MkdirAll(dockerfileDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a Dockerfile
	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte("FROM alpine\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a nested Dockerfile
	nestedDockerfilePath := filepath.Join(dockerfileDir, "Dockerfile.prod")
	if err := os.WriteFile(nestedDockerfilePath, []byte("FROM alpine\n"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		root        string
		dockerfile  string
		expectError bool
	}{
		{
			name:        "default dockerfile",
			root:        tmpDir,
			dockerfile:  "",
			expectError: false,
		},
		{
			name:        "nested dockerfile",
			root:        tmpDir,
			dockerfile:  "docker/Dockerfile.prod",
			expectError: false,
		},
		{
			name:        "invalid root path",
			root:        "/nonexistent/path",
			dockerfile:  "",
			expectError: true,
		},
		{
			name:        "invalid dockerfile path",
			root:        tmpDir,
			dockerfile:  "/nonexistent/Dockerfile",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := DockerfileBuildContext{
				Root:       tt.root,
				Dockerfile: tt.dockerfile,
			}

			mounts, err := d.ToLocalMounts()

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if mounts == nil {
				t.Error("expected mounts map, got nil")
				return
			}

			if len(mounts) != 2 {
				t.Errorf("expected 2 mounts, got %d", len(mounts))
			}

			if _, ok := mounts["context"]; !ok {
				t.Error("missing 'context' mount")
			}

			if _, ok := mounts["dockerfile"]; !ok {
				t.Error("missing 'dockerfile' mount")
			}
		})
	}
}
