package build

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/timo-reymann/ContainerHive/internal/testutil"
)

func startBuildKitContainer(t *testing.T) *Client {
	t.Helper()
	ctx := t.Context()

	buildkitC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        testutil.BuildKitImage(),
			ExposedPorts: []string{"1234/tcp"},
			Cmd:          []string{"--addr", "tcp://0.0.0.0:1234"},
			HostConfigModifier: func(hc *container.HostConfig) {
				hc.Privileged = true
			},
			WaitingFor: wait.ForListeningPort("1234/tcp"),
		},
		Started: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { buildkitC.Terminate(ctx) })

	host, err := buildkitC.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := buildkitC.MappedPort(ctx, "1234/tcp")
	if err != nil {
		t.Fatal(err)
	}

	client, err := NewClient(ctx, fmt.Sprintf("tcp://%s:%s", host, port.Port()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })

	return client
}

func TestIntegrationBuild(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := t.Context()
	platform := "linux/" + runtime.GOARCH
	client := startBuildKitContainer(t)

	// Verify version
	version, err := client.Version(ctx)
	if err != nil {
		t.Fatal("Version() returned error:", err)
	}
	if version == "" {
		t.Fatal("expected non-empty version string")
	}
	t.Logf("BuildKit version: %s", version)

	// Write a simple Dockerfile
	buildCtxDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(buildCtxDir, "Dockerfile"), []byte("FROM alpine:latest\nRUN echo hello > /hello.txt\n"), 0644); err != nil {
		t.Fatal(err)
	}

	tarFile := filepath.Join(t.TempDir(), "image.tar")
	err = client.Build(ctx, &BuildOpts{
		ImageName:  "integration-test:latest",
		TarFile:    tarFile,
		Platform:   platform,
		ContextDir: buildCtxDir,
	}, os.Stdout)
	if err != nil {
		t.Fatal("Build() returned error:", err)
	}

	info, err := os.Stat(tarFile)
	if err != nil {
		t.Fatal("expected OCI tar file to exist:", err)
	}
	if info.Size() == 0 {
		t.Fatal("expected OCI tar file to be non-empty")
	}
	t.Logf("OCI tar size: %d bytes", info.Size())
}

func TestIntegrationBuildWithRegistryPush(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := t.Context()
	platform := "linux/" + runtime.GOARCH
	client := startBuildKitContainer(t)

	// Start a local HTTP registry
	registryC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "registry:2",
			ExposedPorts: []string{"5000/tcp"},
			WaitingFor:   wait.ForHTTP("/v2/").WithPort("5000/tcp"),
		},
		Started: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { registryC.Terminate(ctx) })

	registryHost, err := registryC.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	registryPort, err := registryC.MappedPort(ctx, "5000/tcp")
	if err != nil {
		t.Fatal(err)
	}
	registryAddr := fmt.Sprintf("%s:%s", registryHost, registryPort.Port())
	registryRef := fmt.Sprintf("%s/integration-test:latest", registryAddr)

	// Write a simple Dockerfile
	buildCtxDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(buildCtxDir, "Dockerfile"), []byte("FROM alpine:latest\nRUN echo hello > /hello.txt\n"), 0644); err != nil {
		t.Fatal(err)
	}

	tarFile := filepath.Join(t.TempDir(), "image.tar")
	err = client.Build(ctx, &BuildOpts{
		ImageName:        "integration-test:latest",
		TarFile:          tarFile,
		Platform:         platform,
		ContextDir:       buildCtxDir,
		RegistryRef:      registryRef,
		RegistryInsecure: true,
	}, os.Stdout)
	if err != nil {
		t.Fatal("Build() with registry push returned error:", err)
	}

	// Verify OCI tar was created
	info, err := os.Stat(tarFile)
	if err != nil {
		t.Fatal("expected OCI tar file to exist:", err)
	}
	if info.Size() == 0 {
		t.Fatal("expected OCI tar file to be non-empty")
	}
	t.Logf("OCI tar size: %d bytes", info.Size())

	// Verify the image was pushed to the registry
	ref, err := name.ParseReference(registryRef, name.Insecure)
	if err != nil {
		t.Fatal("failed to parse registry reference:", err)
	}
	_, err = remote.Image(ref)
	if err != nil {
		t.Fatal("failed to fetch image from registry:", err)
	}
	t.Logf("Image successfully pushed and verified at %s", registryRef)
}
