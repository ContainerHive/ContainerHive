package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/model"
	"github.com/ContainerHive/ContainerHive/pkg/shard"
)

func sbomTestProject() *model.ContainerHiveProject {
	return &model.ContainerHiveProject{
		ImagesByIdentifier: map[string]*model.Image{
			"app": {
				Identifier: "app",
				Name:       "app",
				Tags:       map[string]*model.Tag{"1.0": {Name: "1.0"}},
				Platforms:  []string{"linux/amd64"},
				Variants: map[string]*model.ImageVariant{
					"riscv": {Name: "riscv", TagSuffix: "-riscv", Platforms: []string{"linux/riscv64"}},
				},
			},
		},
	}
}

func writeImageTar(t *testing.T, distPath, imageName, tagName, platformDir string) {
	t.Helper()
	dir := filepath.Join(distPath, imageName, tagName, platformDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create platform dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "image.tar"), []byte("tar"), 0644); err != nil {
		t.Fatalf("failed to write image.tar: %v", err)
	}
}

func TestCollectSBOMWorkItems(t *testing.T) {
	allTags := func(string, string) bool { return true }
	baseOnly := func(_, tagName string) bool { return tagName == "1.0" }

	tests := []struct {
		name      string
		owns      func(identifier, tagName string) bool
		platforms []string
		shard     shard.Shard
		tars      [][2]string
		wantItems int
		wantErr   bool
	}{
		{
			name:      "variant platform selected",
			owns:      allTags,
			platforms: []string{"riscv64"},
			tars:      [][2]string{{"1.0-riscv", "linux-riscv64"}},
			wantItems: 1,
		},
		{
			name:      "shard owns only tags skipping the platform",
			owns:      baseOnly,
			platforms: []string{"riscv64"},
			shard:     shard.Shard{Current: 1, Max: 2},
			wantItems: 0,
		},
		{
			name:      "no platform selected tags skipping the platform",
			owns:      baseOnly,
			platforms: []string{"riscv64"},
			wantItems: 0,
		},
		{
			name:      "supported platform without image tar",
			owns:      allTags,
			platforms: []string{"amd64"},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			distPath := t.TempDir()
			for _, tar := range tt.tars {
				writeImageTar(t, distPath, "app", tar[0], tar[1])
			}

			items, err := collectSBOMWorkItems(sbomTestProject(), distPath, nil, tt.owns, tt.platforms, tt.shard)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %d items", len(items))
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if len(items) != tt.wantItems {
				t.Errorf("expected %d work items, got %d", tt.wantItems, len(items))
			}
		})
	}
}
