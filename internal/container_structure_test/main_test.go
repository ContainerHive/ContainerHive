package container_structure_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/moby/buildkit/client"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/timo-reymann/ContainerHive/internal/buildkit"
	"github.com/timo-reymann/ContainerHive/internal/buildkit/build_context"
	"github.com/timo-reymann/ContainerHive/internal/docker"
	"github.com/timo-reymann/ContainerHive/internal/testutil"
)

func drainStatus(ch chan *client.SolveStatus) error {
	for range ch {
	}
	return nil
}

func TestIntegrationContainerStructureTest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := t.Context()
	platform := "linux/" + runtime.GOARCH

	// --- Shared network ---
	net, err := network.New(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { net.Remove(ctx) })

	// --- BuildKit container ---
	buildkitC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        testutil.BuildKitImage(),
			ExposedPorts: []string{"1234/tcp"},
			Cmd:          []string{"--addr", "tcp://0.0.0.0:1234"},
			HostConfigModifier: func(hc *container.HostConfig) {
				hc.Privileged = true
			},
			Networks:   []string{net.Name},
			WaitingFor: wait.ForListeningPort("1234/tcp"),
		},
		Started: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { buildkitC.Terminate(ctx) })

	buildkitHost, err := buildkitC.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	buildkitPort, err := buildkitC.MappedPort(ctx, "1234/tcp")
	if err != nil {
		t.Fatal(err)
	}

	bkClient, err := buildkit.NewClient(ctx, fmt.Sprintf("tcp://%s:%s", buildkitHost, buildkitPort.Port()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { bkClient.Close() })

	// --- Docker-in-Docker container for container-structure-test ---
	dindC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "docker:dind",
			ExposedPorts: []string{"2375/tcp"},
			HostConfigModifier: func(hc *container.HostConfig) {
				hc.Privileged = true
			},
			Env: map[string]string{
				"DOCKER_TLS_CERTDIR": "",
			},
			Networks:   []string{net.Name},
			WaitingFor: wait.ForHTTP("/_ping").WithPort("2375/tcp"),
		},
		Started: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { dindC.Terminate(ctx) })

	dindHost, err := dindC.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	dindPort, err := dindC.MappedPort(ctx, "2375/tcp")
	if err != nil {
		t.Fatal(err)
	}

	// Point Docker client at the DinD container
	t.Setenv("DOCKER_HOST", fmt.Sprintf("tcp://%s:%s", dindHost, dindPort.Port()))
	t.Setenv("DOCKER_TLS_VERIFY", "")

	dockerClient, err := docker.NewClient()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { dockerClient.Close() })

	// --- Build a test image using BuildKit ---
	buildCtxDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(buildCtxDir, "Dockerfile"), []byte("FROM alpine:latest\nRUN echo hello > /hello.txt\n"), 0644); err != nil {
		t.Fatal(err)
	}

	tarFile := filepath.Join(t.TempDir(), "image.tar")
	err = bkClient.Build(ctx, &buildkit.BuildOpts{
		ImageName: "cst-test:latest",
		TarFile:   tarFile,
		BuildContext: &build_context.DockerfileBuildContext{
			Root: buildCtxDir,
		},
		Platform: platform,
	}, drainStatus)
	if err != nil {
		t.Fatal("buildkit build failed:", err)
	}

	// --- Write test definition ---
	testDefPath := filepath.Join(t.TempDir(), "test.yml")
	testDef := `schemaVersion: 2.0.0

fileExistenceTests:
  - name: 'hello.txt exists'
    path: '/hello.txt'
    shouldExist: true
`
	if err := os.WriteFile(testDefPath, []byte(testDef), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("passing_tests", func(t *testing.T) {
		reportFile := filepath.Join(t.TempDir(), "junit.xml")
		runner := TestRunner{
			TestDefinitionPaths: []string{testDefPath},
			Image:               tarFile,
			Platform:            platform,
			ReportFile:          reportFile,
			DockerClient:        dockerClient,
		}

		err := runner.Run(t.Context())
		if err != nil {
			t.Fatal("container-structure-test run failed:", err)
		}

		info, err := os.Stat(reportFile)
		if err != nil {
			t.Fatal("expected junit report to exist:", err)
		}
		if info.Size() == 0 {
			t.Fatal("expected junit report to be non-empty")
		}
	})

	t.Run("docker_image_without_tar", func(t *testing.T) {
		reportFile := filepath.Join(t.TempDir(), "junit-docker.xml")
		runner := TestRunner{
			TestDefinitionPaths: []string{testDefPath},
			Image:               "cst-test:latest",
			Platform:            platform,
			ReportFile:          reportFile,
			DockerClient:        dockerClient,
		}

		err := runner.Run(t.Context())
		if err != nil {
			t.Fatal("container-structure-test with docker image name failed:", err)
		}

		info, err := os.Stat(reportFile)
		if err != nil {
			t.Fatal("expected junit report to exist:", err)
		}
		if info.Size() == 0 {
			t.Fatal("expected junit report to be non-empty")
		}
	})

	t.Run("failing_tests", func(t *testing.T) {
		failDefPath := filepath.Join(t.TempDir(), "fail-test.yml")
		failDef := `schemaVersion: 2.0.0

fileExistenceTests:
  - name: 'nonexistent file'
    path: '/does-not-exist.txt'
    shouldExist: true
`
		if err := os.WriteFile(failDefPath, []byte(failDef), 0644); err != nil {
			t.Fatal(err)
		}

		reportFile := filepath.Join(t.TempDir(), "junit-fail.xml")
		runner := TestRunner{
			TestDefinitionPaths: []string{failDefPath},
			Image:               tarFile,
			Platform:            platform,
			ReportFile:          reportFile,
			DockerClient:        dockerClient,
		}

		err := runner.Run(t.Context())
		if err == nil {
			t.Fatal("expected container-structure-test to report failure for missing file")
		}

		info, statErr := os.Stat(reportFile)
		if statErr != nil {
			t.Fatal("expected junit report to exist even for failures:", statErr)
		}
		if info.Size() == 0 {
			t.Fatal("expected junit report to be non-empty even for failures")
		}
	})
}
