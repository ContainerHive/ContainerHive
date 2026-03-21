package container_structure_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/container-structure-test/cmd/container-structure-test/app/cmd/test"
	"github.com/GoogleContainerTools/container-structure-test/pkg/config"
	"github.com/GoogleContainerTools/container-structure-test/pkg/drivers"
	"github.com/GoogleContainerTools/container-structure-test/pkg/types/unversioned"
	"github.com/timo-reymann/ContainerHive/internal/docker"
)

// TestRunner executes container structure tests against a Docker image and produces JUnit reports.
type TestRunner struct {
	TestDefinitionPaths []string
	Image               string
	Platform            string
	ReportFile          string
	DockerClient        *docker.Client
}

func (t *TestRunner) getOptions(output unversioned.OutputValue) *config.StructureTestOptions {
	return &config.StructureTestOptions{
		ImagePath:           t.Image,
		IgnoreRefAnnotation: false,
		ConfigFiles:         t.TestDefinitionPaths,
		Platform:            t.Platform,
		JSON:                true,
		Output:              output,
		NoColor:             false,
		Driver:              "docker",
		Quiet:               true,
	}
}

func (t *TestRunner) isTar() bool {
	return filepath.Ext(t.Image) == ".tar"
}

func (t *TestRunner) resolveImageName(ctx context.Context) (string, error) {
	if t.isTar() {
		return t.DockerClient.LoadImageFromTar(ctx, t.Image)
	}
	if t.DockerClient.HasImage(ctx, t.Image) {
		return t.Image, nil
	}
	return t.DockerClient.PullImage(ctx, t.Image)
}

func (t *TestRunner) runTests(channel chan interface{}, imageName string, opts *config.StructureTestOptions) {
	args := &drivers.DriverConfig{
		Image:    imageName,
		Save:     opts.Save,
		Metadata: opts.Metadata,
		Runtime:  opts.Runtime,
		Platform: opts.Platform,
	}
	driverImpl := drivers.InitDriverImpl(opts.Driver)

	for _, testDefPath := range t.TestDefinitionPaths {
		tests, err := test.Parse(testDefPath, args, driverImpl)
		if err != nil {
			channel <- &unversioned.TestResult{
				Errors: []string{
					fmt.Sprintf("error parsing config file: %s", err),
				},
			}
		}
		tests.RunAll(channel, testDefPath)
	}

	close(channel)
}

// Run resolves the test image, executes all configured test definitions, and writes a JUnit report.
func (t *TestRunner) Run() error {
	imageName, err := t.resolveImageName(context.Background())
	if err != nil {
		return err
	}

	opts := t.getOptions(unversioned.Junit)
	channel := make(chan interface{}, 1)
	go t.runTests(channel, imageName, opts)

	testReportFile, err := os.Create(t.ReportFile)
	if err != nil {
		return err
	}
	defer testReportFile.Close()

	return test.ProcessResults(testReportFile, unversioned.Junit, opts.JunitSuiteName, channel)
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

// NewRunner creates a TestRunner with a Docker client for the given platform.
func NewRunner(platform string) (*TestRunner, error) {
	dc, err := docker.NewClient()
	if err != nil {
		return nil, err
	}
	return &TestRunner{
		DockerClient: dc,
		Platform:     platform,
	}, nil
}

// Close releases the Docker client resources.
func (t *TestRunner) Close() error {
	return t.DockerClient.Close()
}

// RunTestsForImage executes container structure tests against the given image source.
func (t *TestRunner) RunTestsForImage(imageSource string, testDefs []string, reportFile string) error {
	t.TestDefinitionPaths = testDefs
	t.Image = imageSource
	t.ReportFile = reportFile
	return t.Run()
}
