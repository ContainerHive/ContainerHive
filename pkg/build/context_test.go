package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTarFilePath(t *testing.T) {
	tests := []struct {
		distPath string
		name     string
		tag      string
		platform string
		want     string
	}{
		{"dist", "myimg", "1.0", "linux/amd64", filepath.Join("dist", "myimg", "1.0", "linux-amd64", "image.tar")},
		{"/abs/dist", "app", "latest", "linux/arm64", filepath.Join("/abs/dist", "app", "latest", "linux-arm64", "image.tar")},
		{"dist", "img", "1.0-slim", "linux/amd64", filepath.Join("dist", "img", "1.0-slim", "linux-amd64", "image.tar")},
	}

	for _, tt := range tests {
		t.Run(tt.name+":"+tt.tag+":"+tt.platform, func(t *testing.T) {
			got := TarFilePath(tt.distPath, tt.name, tt.tag, tt.platform)
			if got != tt.want {
				t.Errorf("TarFilePath(%q, %q, %q, %q) = %q, want %q", tt.distPath, tt.name, tt.tag, tt.platform, got, tt.want)
			}
		})
	}
}

func TestPatchHiveRefs(t *testing.T) {
	dir := t.TempDir()
	dockerfile := filepath.Join(dir, "Dockerfile")
	content := "FROM __hive__/base:1.0\nRUN echo hello\n"
	if err := os.WriteFile(dockerfile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	patchedPath, cleanup, err := PatchHiveRefs(dockerfile, "127.0.0.1:5000")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	if patchedPath != dockerfile+".patched" {
		t.Errorf("expected patched path %q, got %q", dockerfile+".patched", patchedPath)
	}

	patched, err := os.ReadFile(patchedPath)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(patched), "__hive__") {
		t.Error("patched file still contains __hive__ references")
	}
	if !strings.Contains(string(patched), "127.0.0.1:5000/base:1.0") {
		t.Error("patched file does not contain expected registry reference")
	}

	// Test cleanup removes the file
	cleanup()
	if _, err := os.Stat(patchedPath); !os.IsNotExist(err) {
		t.Error("cleanup did not remove patched file")
	}
}

func TestPatchHiveRefs_InvalidPath(t *testing.T) {
	_, _, err := PatchHiveRefs("/nonexistent/Dockerfile", "127.0.0.1:5000")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
