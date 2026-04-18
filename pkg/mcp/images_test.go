package mcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/timo-reymann/ContainerHive/pkg/model"
)

func getTestDataPath(t *testing.T, name string) string {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	absPath, err := filepath.Abs(filepath.Join(wd, "..", "..", name))
	if err != nil {
		t.Fatal(err)
	}
	return absPath
}

func TestListImages(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping filesystem test in short mode")
	}

	tests := []struct {
		name        string
		projectRoot string
		wantErr     bool
	}{
		{
			name:        "simple project",
			projectRoot: getTestDataPath(t, "pkg/testdata/simple-project"),
			wantErr:     false,
		},
		{
			name:        "multi variant project",
			projectRoot: getTestDataPath(t, "pkg/testdata/multi-variant-project"),
			wantErr:     false,
		},
		{
			name:        "invalid path",
			projectRoot: "/nonexistent/path",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			images, err := listImages(context.Background(), tt.projectRoot)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListImages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(images) == 0 {
				t.Error("ListImages() expected images, got none")
			}
		})
	}
}

func TestGetImage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping filesystem test in short mode")
	}

	simpleProjectPath := getTestDataPath(t, "pkg/testdata/simple-project")

	tests := []struct {
		name        string
		projectRoot string
		imageName   string
		wantErr     bool
	}{
		{
			name:        "python image",
			projectRoot: simpleProjectPath,
			imageName:   "python",
			wantErr:     false,
		},
		{
			name:        "nonexistent image",
			projectRoot: simpleProjectPath,
			imageName:   "nonexistent",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			image, err := getImage(context.Background(), tt.projectRoot, tt.imageName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetImage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && image == nil {
				t.Error("GetImage() expected image, got nil")
			}
		})
	}
}

func TestGetDependencies_InvalidProjectRoot(t *testing.T) {
	_, err := getDependencies(context.Background(), "/nonexistent/path", "python", "forward")
	if err == nil {
		t.Error("getDependencies() expected error for invalid project root, got nil")
	}
}

func TestGetDependencies_MissingDist(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping filesystem test in short mode")
	}

	// getDependencies requires a rendered dist/ directory; without it ResolveOrder returns an error.
	simpleProjectPath := getTestDataPath(t, "pkg/testdata/simple-project")
	_, err := getDependencies(context.Background(), simpleProjectPath, "python", "forward")
	if err == nil {
		t.Error("getDependencies() expected error when dist/ is missing, got nil")
	}
}

func TestAddImage(t *testing.T) {
	tmpDir := t.TempDir()

	err := addImage(context.Background(), tmpDir, "test-image", "Test image description", "ubuntu:22.04", "")
	if err != nil {
		t.Fatalf("AddImage() error = %v", err)
	}

	dockerfilePath := filepath.Join(tmpDir, "images", "test-image", "Dockerfile")
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		t.Error("AddImage() expected Dockerfile to exist")
	}

	imageYAMLPath := filepath.Join(tmpDir, "images", "test-image", "image.yml")
	if _, err := os.Stat(imageYAMLPath); os.IsNotExist(err) {
		t.Error("AddImage() expected image.yml to exist")
	}

	data, err := os.ReadFile(imageYAMLPath)
	if err != nil {
		t.Fatalf("Failed to read image.yml: %v", err)
	}

	var config model.ImageDefinitionConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatalf("Failed to parse image.yml: %v", err)
	}

	if config.Description != "Test image description" {
		t.Errorf("AddImage() description = %v, want %v", config.Description, "Test image description")
	}

	if len(config.Tags) != 1 || config.Tags[0].Name != "ubuntu:22.04" {
		t.Errorf("AddImage() tags = %v, want [ubuntu:22.04]", config.Tags)
	}
}

func TestAddImage_SpecialCharsInDescription(t *testing.T) {
	tmpDir := t.TempDir()

	err := addImage(context.Background(), tmpDir, "test-image", "Description: with colon & \"quotes\"", "ubuntu:22.04", "")
	if err != nil {
		t.Fatalf("AddImage() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "images", "test-image", "image.yml"))
	if err != nil {
		t.Fatalf("Failed to read image.yml: %v", err)
	}

	var config model.ImageDefinitionConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatalf("Failed to parse image.yml with special chars: %v", err)
	}

	if config.Description != "Description: with colon & \"quotes\"" {
		t.Errorf("AddImage() description = %v, want original", config.Description)
	}
}

func TestAddImageVariant(t *testing.T) {
	tmpDir := t.TempDir()

	err := addImage(context.Background(), tmpDir, "test-image", "Test", "ubuntu:22.04", "")
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	err = addImageVariant(context.Background(), tmpDir, "test-image", "slim", "-slim", nil, nil)
	if err != nil {
		t.Fatalf("AddImageVariant() error = %v", err)
	}

	imageYAMLPath := filepath.Join(tmpDir, "images", "test-image", "image.yml")
	data, err := os.ReadFile(imageYAMLPath)
	if err != nil {
		t.Fatalf("Failed to read image.yml: %v", err)
	}

	var config model.ImageDefinitionConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatalf("Failed to parse image.yml: %v", err)
	}

	if len(config.Variants) != 1 {
		t.Errorf("AddImageVariant() variants count = %v, want 1", len(config.Variants))
		return
	}

	variant := config.Variants[0]
	if variant.Name != "slim" {
		t.Errorf("AddImageVariant() variant name = %v, want slim", variant.Name)
	}
	if variant.TagSuffix != "-slim" {
		t.Errorf("AddImageVariant() tag suffix = %v, want -slim", variant.TagSuffix)
	}

	variantDockerfilePath := filepath.Join(tmpDir, "images", "test-image", "slim", "Dockerfile")
	if _, err := os.Stat(variantDockerfilePath); os.IsNotExist(err) {
		t.Error("AddImageVariant() expected variant Dockerfile to exist")
	}

	variantImageYAMLPath := filepath.Join(tmpDir, "images", "test-image", "slim", "image.yml")
	if _, err := os.Stat(variantImageYAMLPath); os.IsNotExist(err) {
		t.Error("AddImageVariant() expected variant image.yml to exist")
	}
}
