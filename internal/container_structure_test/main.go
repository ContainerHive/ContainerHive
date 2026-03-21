package container_structure_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/container-structure-test/cmd/container-structure-test/app/cmd/test"
	"github.com/GoogleContainerTools/container-structure-test/pkg/config"
	"github.com/GoogleContainerTools/container-structure-test/pkg/drivers"
	"github.com/GoogleContainerTools/container-structure-test/pkg/types/unversioned"
	"github.com/timo-reymann/ContainerHive/internal/docker"
)

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
