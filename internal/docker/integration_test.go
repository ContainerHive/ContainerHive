package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/moby/buildkit/client"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/timo-reymann/ContainerHive/internal/buildkit"
	"github.com/timo-reymann/ContainerHive/internal/buildkit/build_context"
	"github.com/timo-reymann/ContainerHive/internal/testutil"
)

func TestIntegrationDockerClient(t *testing.T) {
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

	// --- Docker-in-Docker container ---
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

	dockerClient, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { dockerClient.Close() })

	// --- Build a test image via BuildKit ---
	buildCtxDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(buildCtxDir, "Dockerfile"), []byte("FROM alpine:latest\nRUN echo hello\n"), 0644); err != nil {
		t.Fatal(err)
	}

	tarFile := filepath.Join(t.TempDir(), "image.tar")
	const testImageName = "integration-test:latest"

	err = bkClient.Build(ctx, &buildkit.BuildOpts{
		ImageName: testImageName,
		TarFile:   tarFile,
		BuildContext: &build_context.DockerfileBuildContext{
			Root: buildCtxDir,
		},
		Platform: platform,
	}, func(ch chan *client.SolveStatus) error {
		for range ch {
		}
		return nil
	})
	if err != nil {
		t.Fatal("buildkit build failed:", err)
	}

	// --- Subtests ---

	var loadedImageName string

	t.Run("LoadImageFromTar", func(t *testing.T) {
		name, err := dockerClient.LoadImageFromTar(ctx, tarFile)
		if err != nil {
			t.Fatal("LoadImageFromTar failed:", err)
		}
		// BuildKit may normalise the name to include the registry prefix
		// (e.g. "docker.io/library/integration-test:latest"), so check with Contains.
		if !strings.Contains(name, "integration-test:latest") {
			t.Fatalf("expected image name to contain %q, got %q", "integration-test:latest", name)
		}
		loadedImageName = name
		t.Logf("loaded image: %s", name)
	})

	t.Run("HasImage_exists", func(t *testing.T) {
		if loadedImageName == "" {
			t.Skip("LoadImageFromTar did not succeed")
		}
		if !dockerClient.HasImage(ctx, loadedImageName) {
			t.Fatalf("expected HasImage(%q) to return true", loadedImageName)
		}
	})

	t.Run("HasImage_not_exists", func(t *testing.T) {
		if dockerClient.HasImage(ctx, "nonexistent:tag") {
			t.Fatal("expected HasImage(\"nonexistent:tag\") to return false")
		}
	})

	t.Run("PullImage", func(t *testing.T) {
		ref, err := dockerClient.PullImage(ctx, "alpine:latest")
		if err != nil {
			t.Fatal("PullImage failed:", err)
		}
		if ref == "" {
			t.Fatal("expected non-empty image ref from PullImage")
		}
		t.Logf("pulled image ref: %s", ref)
	})
}
