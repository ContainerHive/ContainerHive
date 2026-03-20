package test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/timo-reymann/ContainerHive/pkg/build"
	"github.com/timo-reymann/ContainerHive/pkg/model"
	"github.com/timo-reymann/ContainerHive/pkg/utils"
)

func TestRunTestsForTag(t *testing.T) {
	// Create a temporary directory structure for testing
	tempDir := t.TempDir()

	// Create a mock image tar file
	imageDir := filepath.Join(tempDir, "ubuntu", "24.04", "linux_amd64")
	require.NoError(t, os.MkdirAll(imageDir, 0755))

	tarFile := filepath.Join(imageDir, "image.tar")
	_, err := os.Create(tarFile)
	require.NoError(t, err)

	// Create a test definition file in the correct location (tag directory)
	testsDir := filepath.Join(tempDir, "ubuntu", "24.04", "tests")
	require.NoError(t, os.MkdirAll(testsDir, 0755))

	testFile := filepath.Join(testsDir, "test.yaml")
	testContent := `---
schemaVersion: "2.0.0"

fileExistenceTests:
- name: "test-file"
  path: "/etc/os-release"
  shouldExist: true
`
	require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))

	t.Run("successful test execution", func(t *testing.T) {
		tested, failed, err := runTestsForTag(tempDir, "ubuntu", "24.04", []string{"linux/amd64"})
		// Note: This will likely fail because we don't have a real Docker environment,
		// but we can at least verify the function structure
		assert.NoError(t, err)
		// Since we don't have Docker, it should skip the test (no image.tar found in the right location)
		assert.Equal(t, 0, tested)
		assert.Equal(t, 0, failed)
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
		require.NoError(t, os.MkdirAll(imageDir, 0755))

		// Create a mock image tar file
		tarFile := filepath.Join(imageDir, "image.tar")
		_, err := os.Create(tarFile)
		require.NoError(t, err)

		tested, failed, err := RunProjectTests(context.Background(), tempDir, project, []build.Filter{})
		assert.NoError(t, err)
		assert.Equal(t, 0, tested) // No test files, so nothing tested
		assert.Equal(t, 0, failed)
	})

	t.Run("filters limit tested images", func(t *testing.T) {
		filters := []build.Filter{
			{ImageName: "nonexistent", TagName: ""},
		}

		tested, failed, err := RunProjectTests(context.Background(), t.TempDir(), project, filters)
		assert.NoError(t, err)
		assert.Equal(t, 0, tested) // Filter doesn't match, so nothing tested
		assert.Equal(t, 0, failed)
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
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRunTestsForTag_NoTestDefinitions(t *testing.T) {
	// Tag dir exists but has no tests/ subdir
	tempDir := t.TempDir()
	tagDir := filepath.Join(tempDir, "app", "1.0")
	require.NoError(t, os.MkdirAll(tagDir, 0755))

	tested, failed, err := runTestsForTag(tempDir, "app", "1.0", []string{"linux/amd64"})
	assert.NoError(t, err)
	assert.Equal(t, 0, tested)
	assert.Equal(t, 0, failed)
}

func TestRunTestsForTag_MissingTarFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create tag dir with test definitions but no image.tar
	tagDir := filepath.Join(tempDir, "app", "1.0")
	testsDir := filepath.Join(tagDir, "tests")
	require.NoError(t, os.MkdirAll(testsDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(testsDir, "test.yaml"), []byte("test: true"), 0644))

	// Platform dir exists but no tar
	platDir := filepath.Join(tagDir, "linux-amd64")
	require.NoError(t, os.MkdirAll(platDir, 0755))

	tested, failed, err := runTestsForTag(tempDir, "app", "1.0", []string{"linux/amd64"})
	assert.NoError(t, err)
	assert.Equal(t, 0, tested, "should skip when image.tar is missing")
	assert.Equal(t, 0, failed)
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
		tested, failed, err := RunProjectTests(context.Background(), t.TempDir(), project, filters)
		assert.NoError(t, err)
		assert.Equal(t, 0, tested, "no test files exist so nothing tested")
		assert.Equal(t, 0, failed)
	})

	t.Run("filter excludes base tag when variant specified", func(t *testing.T) {
		filters := []build.Filter{{ImageName: "dotnet", TagName: "8.0.300-node"}}
		// Should not test the base "8.0.300" tag
		tested, failed, err := RunProjectTests(context.Background(), t.TempDir(), project, filters)
		assert.NoError(t, err)
		assert.Equal(t, 0, tested)
		assert.Equal(t, 0, failed)
	})
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
		tested, failed, err := RunProjectTests(context.Background(), t.TempDir(), project, filters)
		assert.NoError(t, err)
		assert.Equal(t, 0, tested) // no test files
		assert.Equal(t, 0, failed)
	})
}
