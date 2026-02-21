package cst

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	container_structure_test "github.com/timo-reymann/ContainerHive/internal/container_structure_test"
	"github.com/timo-reymann/ContainerHive/internal/docker"
)

// Runner wraps the internal container structure test runner and Docker client.
type Runner struct {
	dockerClient *docker.Client
	platform     string
}

// NewRunner creates a Runner with a Docker client for the given platform.
func NewRunner(platform string) (*Runner, error) {
	dc, err := docker.NewClient()
	if err != nil {
		return nil, err
	}
	return &Runner{dockerClient: dc, platform: platform}, nil
}

// Close releases the Docker client resources.
func (r *Runner) Close() error {
	return r.dockerClient.Close()
}

// RunTests executes container structure tests against the given tar file.
func (r *Runner) RunTests(tarFile string, testDefs []string, reportFile string) error {
	runner := &container_structure_test.TestRunner{
		TestDefinitionPaths: testDefs,
		Image:               tarFile,
		Platform:            r.platform,
		ReportFile:          reportFile,
		DockerClient:        r.dockerClient,
	}
	return runner.Run()
}

// CollectTestDefinitions finds test YAML files in a rendered dist directory's
// tests/ subfolder. Only top-level files are included (subdirectories are skipped).
func CollectTestDefinitions(distDir string) []string {
	testsDir := filepath.Join(distDir, "tests")
	entries, err := os.ReadDir(testsDir)
	if err != nil {
		return nil
	}
	var paths []string
	for _, e := range entries {
		if !e.IsDir() {
			paths = append(paths, filepath.Join(testsDir, e.Name()))
		}
	}
	return paths
}

// ReportFileName returns a JUnit report file path for the given image tag,
// replacing colons with hyphens to produce a valid file name.
func ReportFileName(reportDir, imageTag string) string {
	return filepath.Join(reportDir, fmt.Sprintf("%s-cst-report.xml", strings.ReplaceAll(imageTag, ":", "-")))
}
