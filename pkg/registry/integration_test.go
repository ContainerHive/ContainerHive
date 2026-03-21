package registry

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
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/timo-reymann/ContainerHive/internal/buildkit"
	"github.com/timo-reymann/ContainerHive/internal/buildkit/build_context"
	internalregistry "github.com/timo-reymann/ContainerHive/internal/registry"
	"github.com/timo-reymann/ContainerHive/internal/testutil"
	"github.com/timo-reymann/ContainerHive/pkg/build"
	"github.com/timo-reymann/ContainerHive/pkg/model"
	"github.com/timo-reymann/ContainerHive/pkg/platform"
)

func integrationDrainStatus(ch chan *client.SolveStatus) error {
	for range ch {
	}
	return nil
}

func TestIntegrationCreateAllManifests(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := t.Context()
	plat := "linux/" + runtime.GOARCH

	// --- Start BuildKit testcontainer ---
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

	// --- Start local zot registry ---
	zotReg := internalregistry.NewZotRegistry("")
	if err := zotReg.Start(ctx); err != nil {
		t.Fatal("failed to start zot registry:", err)
	}
	t.Cleanup(func() { zotReg.Stop(ctx) })

	reg := &Registry{inner: zotReg}

	// --- Build a test image using BuildKit ---
	imageName := "test-img"
	tagName := "latest"
	distPath := t.TempDir()

	// Create the directory structure that TarFilePath expects
	tarPath := build.TarFilePath(distPath, imageName, tagName, plat)
	if err := os.MkdirAll(filepath.Dir(tarPath), 0755); err != nil {
		t.Fatal(err)
	}

	buildCtxDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(buildCtxDir, "Dockerfile"), []byte("FROM alpine:latest\nRUN echo hello > /hello.txt\n"), 0644); err != nil {
		t.Fatal(err)
	}

	err = bkClient.Build(ctx, &buildkit.BuildOpts{
		ImageName: imageName + ":" + tagName,
		TarFile:   tarPath,
		BuildContext: &build_context.DockerfileBuildContext{
			Root: buildCtxDir,
		},
		Platform: plat,
	}, integrationDrainStatus)
	if err != nil {
		t.Fatal("buildkit build failed:", err)
	}

	// --- Push the OCI tar to the zot registry ---
	pushTag := build.PushTag(tagName, plat, "")
	if err := reg.Push(ctx, imageName, pushTag, tarPath); err != nil {
		t.Fatal("failed to push image to zot:", err)
	}

	// --- Create manifest lists via CreateAllManifests ---
	project := &model.ContainerHiveProject{
		Config: model.HiveProjectConfig{
			Platforms: []string{plat},
		},
		ImagesByIdentifier: map[string]*model.Image{
			"test-img": {
				Name:     imageName,
				Tags:     map[string]*model.Tag{tagName: {}},
				Variants: map[string]*model.ImageVariant{},
			},
		},
	}

	if err := reg.CreateAllManifests(project, nil, "", distPath); err != nil {
		t.Fatal("CreateAllManifests failed:", err)
	}

	// --- Verify the manifest list exists ---
	manifestRef := fmt.Sprintf("%s/%s:%s", reg.Address(), imageName, tagName)
	tag, err := name.NewTag(manifestRef, name.Insecure)
	if err != nil {
		t.Fatal(err)
	}

	desc, err := remote.Get(tag)
	if err != nil {
		t.Fatal("failed to fetch manifest from registry:", err)
	}

	// The manifest should be an index
	if !desc.MediaType.IsIndex() {
		t.Errorf("expected index media type, got %s", desc.MediaType)
	}
}

func TestIntegrationPushAndImageRef(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := t.Context()
	plat := "linux/" + runtime.GOARCH

	// --- Start BuildKit testcontainer ---
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

	// --- Start local zot registry ---
	zotReg := internalregistry.NewZotRegistry("")
	if err := zotReg.Start(ctx); err != nil {
		t.Fatal("failed to start zot registry:", err)
	}
	t.Cleanup(func() { zotReg.Stop(ctx) })

	reg := &Registry{inner: zotReg}

	// --- Build a test image using BuildKit ---
	imageName := "push-test"
	tagName := "v1.0"

	buildCtxDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(buildCtxDir, "Dockerfile"), []byte("FROM alpine:latest\nRUN echo test > /test.txt\n"), 0644); err != nil {
		t.Fatal(err)
	}

	tarFile := filepath.Join(t.TempDir(), "image.tar")
	err = bkClient.Build(ctx, &buildkit.BuildOpts{
		ImageName: imageName + ":" + tagName,
		TarFile:   tarFile,
		BuildContext: &build_context.DockerfileBuildContext{
			Root: buildCtxDir,
		},
		Platform: plat,
	}, integrationDrainStatus)
	if err != nil {
		t.Fatal("buildkit build failed:", err)
	}

	// --- Push via the registry wrapper ---
	pt := build.PushTag(tagName, plat, "")
	if err := reg.Push(ctx, imageName, pt, tarFile); err != nil {
		t.Fatal("Push failed:", err)
	}

	// --- Verify ImageRef format ---
	expectedRef := fmt.Sprintf("%s/%s:%s.%s", reg.Address(), imageName, tagName, platform.Sanitize(plat))
	gotRef := reg.ImageRef(imageName, tagName, plat, "")
	if gotRef != expectedRef {
		t.Errorf("ImageRef() = %q, want %q", gotRef, expectedRef)
	}

	// --- Verify the image is reachable in the registry ---
	refTag, err := name.NewTag(gotRef, name.Insecure)
	if err != nil {
		t.Fatal(err)
	}

	_, err = remote.Get(refTag)
	if err != nil {
		t.Fatal("failed to fetch pushed image from registry:", err)
	}
}
