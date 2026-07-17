package build

import (
	"path/filepath"
	"testing"
)

func TestPushTagFunc(t *testing.T) {
	tests := []struct {
		name      string
		tagName   string
		platform  string
		buildID   string
		want      string
	}{
		{
			name:     "tag without buildID",
			tagName:  "latest",
			platform: "linux/amd64",
			buildID:  "",
			want:     "latest.linux-amd64",
		},
		{
			name:     "tag with buildID",
			tagName:  "latest",
			platform: "linux/amd64",
			buildID:  "abc123",
			want:     "latest.linux-amd64-build.abc123",
		},
		{
			name:     "platform sanitization linux/amd64 becomes linux-amd64",
			tagName:  "v1.0",
			platform: "linux/amd64",
			buildID:  "",
			want:     "v1.0.linux-amd64",
		},
		{
			name:     "platform sanitization linux/arm64 becomes linux-arm64",
			tagName:  "v1.0",
			platform: "linux/arm64",
			buildID:  "",
			want:     "v1.0.linux-arm64",
		},
		{
			name:     "platform sanitization with buildID",
			tagName:  "main",
			platform: "linux/arm64",
			buildID:  "build-42",
			want:     "main.linux-arm64-build.build-42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PushTag(tt.tagName, tt.platform, tt.buildID)
			if got != tt.want {
				t.Errorf("PushTag(%q, %q, %q) = %q, want %q", tt.tagName, tt.platform, tt.buildID, got, tt.want)
			}
		})
	}
}

func TestWithBuildID(t *testing.T) {
	tests := []struct {
		name    string
		tag     string
		buildID string
		want    string
	}{
		{"empty buildID returns tag unchanged", "1.0", "", "1.0"},
		{"appends buildID with -build. separator", "1.0", "abc123", "1.0-build.abc123"},
		{"works with complex tags", "latest.linux-amd64", "build-42", "latest.linux-amd64-build.build-42"},
		{"disambiguates from patch version", "26.06", "22", "26.06-build.22"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WithBuildID(tt.tag, tt.buildID)
			if got != tt.want {
				t.Errorf("WithBuildID(%q, %q) = %q, want %q", tt.tag, tt.buildID, got, tt.want)
			}
		})
	}
}

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
