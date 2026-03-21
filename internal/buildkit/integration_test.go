package buildkit

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/moby/buildkit/client"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/timo-reymann/ContainerHive/internal/buildkit/build_context"
	"github.com/timo-reymann/ContainerHive/internal/testutil"
)

func drainStatus(ch chan *client.SolveStatus) error {
	for range ch {
	}
	return nil
}

func startBuildKit(t *testing.T, networks ...string) *Client {
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
			Networks:   networks,
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

	bkClient, err := NewClient(ctx, fmt.Sprintf("tcp://%s:%s", host, port.Port()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { bkClient.Close() })

	return bkClient
}

func TestIntegrationVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := t.Context()
	bkClient := startBuildKit(t)

	version, err := bkClient.Version(ctx)
	if err != nil {
		t.Fatal("Version() returned error:", err)
	}
	if version == "" {
		t.Fatal("expected non-empty version string")
	}
	t.Logf("BuildKit version: %s", version)
}

func TestIntegrationBuild(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := t.Context()
	platform := "linux/" + runtime.GOARCH
	bkClient := startBuildKit(t)

	// Write a simple Dockerfile
	buildCtxDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(buildCtxDir, "Dockerfile"), []byte("FROM alpine:latest\nRUN echo hello > /hello.txt\n"), 0644); err != nil {
		t.Fatal(err)
	}

	tarFile := filepath.Join(t.TempDir(), "image.tar")
	err := bkClient.Build(ctx, &BuildOpts{
		ImageName: "integration-test:latest",
		TarFile:   tarFile,
		Platform:  platform,
		BuildContext: &build_context.DockerfileBuildContext{
			Root: buildCtxDir,
		},
	}, drainStatus)
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

	// Create a shared network so BuildKit can reach the registry by hostname
	net, err := network.New(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { net.Remove(ctx) })

	bkClient := startBuildKit(t, net.Name)

	// Start a local HTTP registry with a network alias
	const registryAlias = "testregistry"
	registryC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "registry:2",
			ExposedPorts: []string{"5000/tcp"},
			Networks:     []string{net.Name},
			NetworkAliases: map[string][]string{
				net.Name: {registryAlias},
			},
			WaitingFor: wait.ForHTTP("/v2/").WithPort("5000/tcp"),
		},
		Started: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { registryC.Terminate(ctx) })

	// Internal ref: used by BuildKit (container-to-container via Docker network)
	internalRef := fmt.Sprintf("%s:5000/integration-test:latest", registryAlias)

	// External ref: used by the test to verify the push (host-mapped port)
	registryHost, err := registryC.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	registryPort, err := registryC.MappedPort(ctx, "5000/tcp")
	if err != nil {
		t.Fatal(err)
	}
	externalRef := fmt.Sprintf("%s:%s/integration-test:latest", registryHost, registryPort.Port())

	// Write a simple Dockerfile
	buildCtxDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(buildCtxDir, "Dockerfile"), []byte("FROM alpine:latest\nRUN echo hello > /hello.txt\n"), 0644); err != nil {
		t.Fatal(err)
	}

	tarFile := filepath.Join(t.TempDir(), "image.tar")
	err = bkClient.Build(ctx, &BuildOpts{
		ImageName:        "integration-test:latest",
		TarFile:          tarFile,
		Platform:         platform,
		RegistryRef:      internalRef,
		RegistryInsecure: true,
		BuildContext: &build_context.DockerfileBuildContext{
			Root: buildCtxDir,
		},
	}, drainStatus)
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
	ref, err := name.ParseReference(externalRef, name.Insecure)
	if err != nil {
		t.Fatal("failed to parse registry reference:", err)
	}
	_, err = remote.Image(ref)
	if err != nil {
		t.Fatal("failed to fetch image from registry:", err)
	}
	t.Logf("Image successfully pushed and verified at %s", externalRef)
}
