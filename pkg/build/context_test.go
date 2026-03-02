package build

import (
	"path/filepath"
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
