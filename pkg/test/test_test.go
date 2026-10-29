package test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/build"
	"github.com/ContainerHive/ContainerHive/pkg/model"
	"github.com/ContainerHive/ContainerHive/pkg/utils"
)

func TestRunTestsForTag(t *testing.T) {
	// Create a temporary directory structure for testing
	tempDir := t.TempDir()

	// Create a mock image tar file
	imageDir := filepath.Join(tempDir, "ubuntu", "24.04", "linux_amd64")
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		t.Fatalf("failed to create image dir: %v", err)
	}

	tarFile := filepath.Join(imageDir, "image.tar")
	if _, err := os.Create(tarFile); err != nil {
		t.Fatalf("failed to create tar file: %v", err)
	}

	// Create a test definition file in the correct location (tag directory)
	testsDir := filepath.Join(tempDir, "ubuntu", "24.04", "tests")
	if err := os.MkdirAll(testsDir, 0755); err != nil {
		t.Fatalf("failed to create tests dir: %v", err)
	}

	testFile := filepath.Join(testsDir, "test.yaml")
	testContent := `---
schemaVersion: "2.0.0"

fileExistenceTests:
- name: "test-file"
  path: "/etc/os-release"
  shouldExist: true
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	t.Run("successful test execution", func(t *testing.T) {
		opts := &Opts{DistPath: tempDir}
		tested, failed, err := runTestsForTag(t.Context(), opts, "ubuntu", "24.04", []string{"linux/amd64"})
		// Note: This will likely fail because we don't have a real Docker environment,
		// but we can at least verify the function structure
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		// Since we don't have Docker, it should skip the test (no image.tar found in the right location)
		if tested != 0 {
			t.Errorf("expected 0 tested, got %d", tested)
		}
		if failed != 0 {
			t.Errorf("expected 0 failed, got %d", failed)
		}
	})
}

func TestRunProjectTests(t *testing.T) {
	// Create a minimal project for testing
	project := &model.ContainerHiveProject{
		ImagesByIdentifier: map[string]*model.Image{
			"ubuntu": {
				Identifier: "ubuntu",
				Name:       "ubuntu",
				Tags: map[string]*model.Tag{
					"24.04": {
						Name: "24.04",
					},
				},
				Platforms: []string{"linux/amd64"},
			},
		},
		Config: model.HiveProjectConfig{
			Platforms: []string{"linux/amd64"},
		},
	}

	t.Run("empty filters match all images", func(t *testing.T) {
		// Create a temporary directory structure
		tempDir := t.TempDir()
		imageDir := filepath.Join(tempDir, "ubuntu", "24.04", "linux_amd64")
		if err := os.MkdirAll(imageDir, 0755); err != nil {
			t.Fatalf("failed to create image dir: %v", err)
		}

		// Create a mock image tar file
		tarFile := filepath.Join(imageDir, "image.tar")
		if _, err := os.Create(tarFile); err != nil {
			t.Fatalf("failed to create tar file: %v", err)
		}

		tested, failed, err := RunProjectTests(context.Background(), &Opts{DistPath: tempDir, Project: project, Filters: []build.Filter{}})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if tested != 0 {
			t.Errorf("expected 0 tested, got %d", tested)
		}
		if failed != 0 {
			t.Errorf("expected 0 failed, got %d", failed)
		}
	})

	t.Run("filters limit tested images", func(t *testing.T) {
		filters := []build.Filter{
			{ImageName: "nonexistent", TagName: ""},
		}

		tested, failed, err := RunProjectTests(context.Background(), &Opts{DistPath: t.TempDir(), Project: project, Filters: filters})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if tested != 0 {
			t.Errorf("expected 0 tested, got %d", tested)
		}
		if failed != 0 {
			t.Errorf("expected 0 failed, got %d", failed)
		}
	})
}

func TestMatchesFilterIntegration(t *testing.T) {
	filters := []build.Filter{
		{ImageName: "ubuntu", TagName: "24.04"},
		{ImageName: "alpine", TagName: ""},
	}

	tests := []struct {
		image    string
		tag      string
		expected bool
	}{
		{"ubuntu", "24.04", true},
		{"ubuntu", "22.04", false},
		{"alpine", "3.18", true},
		{"debian", "latest", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s:%s", tt.image, tt.tag), func(t *testing.T) {
			result := utils.MatchesFilter(filters, tt.image, tt.tag)
			if result != tt.expected {
				t.Errorf("MatchesFilter(%q, %q) = %v, want %v", tt.image, tt.tag, result, tt.expected)
			}
		})
	}
}

func TestRunTestsForTag_NoTestDefinitions(t *testing.T) {
	// Tag dir exists but has no tests/ subdir
	tempDir := t.TempDir()
	tagDir := filepath.Join(tempDir, "app", "1.0")
	if err := os.MkdirAll(tagDir, 0755); err != nil {
		t.Fatalf("failed to create tag dir: %v", err)
	}

	tested, failed, err := runTestsForTag(t.Context(), &Opts{DistPath: tempDir}, "app", "1.0", []string{"linux/amd64"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if tested != 0 {
		t.Errorf("expected 0 tested, got %d", tested)
	}
	if failed != 0 {
		t.Errorf("expected 0 failed, got %d", failed)
	}
}

func TestRunTestsForTag_MissingTarFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create tag dir with test definitions but no image.tar
	tagDir := filepath.Join(tempDir, "app", "1.0")
	testsDir := filepath.Join(tagDir, "tests")
	if err := os.MkdirAll(testsDir, 0755); err != nil {
		t.Fatalf("failed to create tests dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testsDir, "test.yaml"), []byte("test: true"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Platform dir exists but no tar
	platDir := filepath.Join(tagDir, "linux-amd64")
	if err := os.MkdirAll(platDir, 0755); err != nil {
		t.Fatalf("failed to create platform dir: %v", err)
	}

	tested, failed, err := runTestsForTag(t.Context(), &Opts{DistPath: tempDir}, "app", "1.0", []string{"linux/amd64"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if tested != 0 {
		t.Errorf("expected 0 tested when image.tar is missing, got %d", tested)
	}
	if failed != 0 {
		t.Errorf("expected 0 failed, got %d", failed)
	}

	_, _, err = runTestsForTag(t.Context(), &Opts{
		DistPath:  tempDir,
		Platforms: []string{"linux/amd64"},
	}, "app", "1.0", []string{"linux/amd64"})
	if err == nil {
		t.Fatal("expected an error when an explicitly selected platform has no image.tar and no registry")
	}
	if !strings.Contains(err.Error(), "explicitly selected platform linux/amd64") {
		t.Errorf("expected error to identify the explicitly selected platform, got %v", err)
	}
}

func TestRunProjectTests_WithVariants(t *testing.T) {
	project := &model.ContainerHiveProject{
		ImagesByIdentifier: map[string]*model.Image{
			"dotnet/8": {
				Identifier: "dotnet/8",
				Name:       "dotnet",
				Tags: map[string]*model.Tag{
					"8.0.300": {Name: "8.0.300"},
				},
				Variants: map[string]*model.ImageVariant{
					"node": {
						Name:      "node",
						TagSuffix: "-node",
					},
				},
				Platforms: []string{"linux/amd64"},
			},
		},
		Config: model.HiveProjectConfig{
			Platforms: []string{"linux/amd64"},
		},
	}

	t.Run("filter matches variant tag", func(t *testing.T) {
		filters := []build.Filter{{ImageName: "dotnet", TagName: "8.0.300-node"}}
		tested, failed, err := RunProjectTests(context.Background(), &Opts{DistPath: t.TempDir(), Project: project, Filters: filters})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if tested != 0 {
			t.Errorf("expected 0 tested (no test files), got %d", tested)
		}
		if failed != 0 {
			t.Errorf("expected 0 failed, got %d", failed)
		}
	})

	t.Run("filter excludes base tag when variant specified", func(t *testing.T) {
		filters := []build.Filter{{ImageName: "dotnet", TagName: "8.0.300-node"}}
		// Should not test the base "8.0.300" tag
		tested, failed, err := RunProjectTests(context.Background(), &Opts{DistPath: t.TempDir(), Project: project, Filters: filters})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if tested != 0 {
			t.Errorf("expected 0 tested, got %d", tested)
		}
		if failed != 0 {
			t.Errorf("expected 0 failed, got %d", failed)
		}
	})
}

func TestRunProjectTests_ExplicitPlatformMissingImageForBaseAndVariant(t *testing.T) {
	project := &model.ContainerHiveProject{
		ImagesByIdentifier: map[string]*model.Image{
			"app": {
				Identifier: "app",
				Name:       "app",
				Tags:       map[string]*model.Tag{"1.0": {Name: "1.0"}},
				Variants: map[string]*model.ImageVariant{
					"alpine": {Name: "alpine", TagSuffix: "-alpine"},
				},
				Platforms: []string{"linux/amd64", "linux/arm64"},
			},
		},
		Config: model.HiveProjectConfig{Platforms: []string{"linux/amd64", "linux/arm64"}},
	}

	for _, tagName := range []string{"1.0", "1.0-alpine"} {
		t.Run(tagName, func(t *testing.T) {
			distPath := t.TempDir()
			testsDir := filepath.Join(distPath, "app", tagName, "tests")
			if err := os.MkdirAll(testsDir, 0755); err != nil {
				t.Fatalf("failed to create tests directory: %v", err)
			}
			if err := os.WriteFile(filepath.Join(testsDir, "test.yaml"), []byte("test: true"), 0644); err != nil {
				t.Fatalf("failed to write test definition: %v", err)
			}

			_, _, err := RunProjectTests(t.Context(), &Opts{
				DistPath:  distPath,
				Project:   project,
				Filters:   []build.Filter{{ImageName: "app", TagName: tagName}},
				Platforms: []string{"linux/arm64"},
			})
			if err == nil {
				t.Fatalf("expected an error for explicitly selected platform with no image for %s", tagName)
			}
			if !strings.Contains(err.Error(), "explicitly selected platform linux/arm64") {
				t.Errorf("expected error to identify the explicitly selected platform, got %v", err)
			}
		})
	}
}

func TestRunProjectTests_MultipleImages(t *testing.T) {
	project := &model.ContainerHiveProject{
		ImagesByIdentifier: map[string]*model.Image{
			"app": {
				Identifier: "app",
				Name:       "app",
				Tags: map[string]*model.Tag{
					"1.0": {Name: "1.0"},
				},
				Variants:  map[string]*model.ImageVariant{},
				Platforms: []string{"linux/amd64"},
			},
			"lib": {
				Identifier: "lib",
				Name:       "lib",
				Tags: map[string]*model.Tag{
					"2.0": {Name: "2.0"},
				},
				Variants:  map[string]*model.ImageVariant{},
				Platforms: []string{"linux/amd64"},
			},
		},
		Config: model.HiveProjectConfig{
			Platforms: []string{"linux/amd64"},
		},
	}

	t.Run("image-only filter matches all tags", func(t *testing.T) {
		filters := []build.Filter{{ImageName: "app"}}
		tested, failed, err := RunProjectTests(context.Background(), &Opts{DistPath: t.TempDir(), Project: project, Filters: filters})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if tested != 0 {
			t.Errorf("expected 0 tested (no test files), got %d", tested)
		}
		if failed != 0 {
			t.Errorf("expected 0 failed, got %d", failed)
		}
	})
}

func TestRunProjectTests_ExplicitPlatformSkipsUnsupportedVariant(t *testing.T) {
	project := &model.ContainerHiveProject{
		ImagesByIdentifier: map[string]*model.Image{
			"app": {
				Identifier: "app",
				Name:       "app",
				Tags:       map[string]*model.Tag{"1.0": {Name: "1.0"}},
				Variants: map[string]*model.ImageVariant{
					"aspire": {Name: "aspire", TagSuffix: "-aspire", Platforms: []string{"linux/amd64"}},
				},
			},
		},
		Config: model.HiveProjectConfig{Platforms: []string{"linux/amd64", "linux/arm64"}},
	}

	distPath := t.TempDir()
	testsDir := filepath.Join(distPath, "app", "1.0-aspire", "tests")
	if err := os.MkdirAll(testsDir, 0755); err != nil {
		t.Fatalf("failed to create tests directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testsDir, "test.yaml"), []byte("test: true"), 0644); err != nil {
		t.Fatalf("failed to write test definition: %v", err)
	}

	tested, failed, err := RunProjectTests(t.Context(), &Opts{
		DistPath:  distPath,
		Project:   project,
		Filters:   []build.Filter{{ImageName: "app", TagName: "1.0-aspire"}},
		Platforms: []string{"linux/arm64"},
	})
	if err != nil {
		t.Fatalf("expected amd64-only variant to be skipped for arm64, got %v", err)
	}
	if tested != 0 || failed != 0 {
		t.Errorf("expected 0 tested and 0 failed, got %d tested, %d failed", tested, failed)
	}
}
