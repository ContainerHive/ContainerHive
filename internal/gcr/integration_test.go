package gcr

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ContainerHive/ContainerHive/internal/buildkit"
	"github.com/ContainerHive/ContainerHive/internal/buildkit/build_context"
	"github.com/ContainerHive/ContainerHive/internal/ocistore"
	"github.com/ContainerHive/ContainerHive/internal/testutil"
	"github.com/docker/docker/api/types/container"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/moby/buildkit/client"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func drainStatus(ch chan *client.SolveStatus) error {
	for range ch {
	}
	return nil
}

// startBuildKitAndRegistry starts a BuildKit testcontainer and a registry:2
// testcontainer. It returns the BuildKit client, the registry host:port, and
// registers cleanup via t.Cleanup.
func startBuildKitAndRegistry(t *testing.T) (*buildkit.Client, string) {
	t.Helper()
	ctx := t.Context()

	// --- BuildKit container ---
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

	// --- Registry container ---
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

	regHost, err := registryC.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	regPort, err := registryC.MappedPort(ctx, "5000/tcp")
	if err != nil {
		t.Fatal(err)
	}

	registryAddr := fmt.Sprintf("%s:%s", regHost, regPort.Port())
	return bkClient, registryAddr
}

// buildTestImage builds a simple alpine-based image via BuildKit and returns
// the path to the OCI tar.
func buildTestImage(t *testing.T, bkClient *buildkit.Client) string {
	t.Helper()
	ctx := t.Context()
	platform := "linux/" + runtime.GOARCH

	buildCtxDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(buildCtxDir, "Dockerfile"), []byte("FROM alpine:latest\nRUN echo hello > /hello.txt\n"), 0644); err != nil {
		t.Fatal(err)
	}

	tarFile := filepath.Join(t.TempDir(), "image.tar")
	err := bkClient.Build(ctx, &buildkit.BuildOpts{
		ImageName: "gcr-integration-test:latest",
		TarFile:   tarFile,
		BuildContext: &build_context.DockerfileBuildContext{
			Root: buildCtxDir,
		},
		Platform: platform,
	}, drainStatus)
	if err != nil {
		t.Fatal("buildkit build failed:", err)
	}
	return tarFile
}

// pushImageToRegistry loads an OCI tar and pushes it to the given registry ref.
// Returns the loaded v1.Image for further use.
func pushImageToRegistry(t *testing.T, tarPath, ref string) v1.Image {
	t.Helper()

	ociImage, err := ocistore.ImageFromTar(tarPath)
	if err != nil {
		t.Fatal("failed to load image from tar:", err)
	}
	t.Cleanup(ociImage.Cleanup)

	tag, err := name.NewTag(ref, name.Insecure)
	if err != nil {
		t.Fatal("failed to parse tag:", err)
	}

	if err := remote.Write(tag, ociImage.Image, remote.WithTransport(http.DefaultTransport)); err != nil {
		t.Fatal("failed to push image to registry:", err)
	}

	return ociImage.Image
}

func TestIntegrationCreateManifestList(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	bkClient, registryAddr := startBuildKitAndRegistry(t)
	tarPath := buildTestImage(t, bkClient)
	plat := "linux/" + runtime.GOARCH

	// Push the platform image first
	platformRef := fmt.Sprintf("%s/test-img:latest.%s", registryAddr, runtime.GOARCH)
	img := pushImageToRegistry(t, tarPath, platformRef)

	cases := []struct {
		name             string
		tag              string
		dockerMediaTypes bool
		wantMediaType    types.MediaType
	}{
		{"oci (default)", "latest-oci", false, types.OCIImageIndex},
		{"docker manifest list", "latest-docker", true, types.DockerManifestList},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			targetRef := fmt.Sprintf("%s/test-img:%s", registryAddr, tc.tag)
			if err := CreateManifestList(targetRef, []PlatformImage{{Image: img, Platform: plat}}, tc.dockerMediaTypes); err != nil {
				t.Fatal("CreateManifestList failed:", err)
			}

			tag, err := name.NewTag(targetRef, name.Insecure)
			if err != nil {
				t.Fatal(err)
			}

			idx, err := remote.Index(tag, remote.WithTransport(http.DefaultTransport))
			if err != nil {
				t.Fatal("failed to fetch manifest list from registry:", err)
			}

			idxManifest, err := idx.IndexManifest()
			if err != nil {
				t.Fatal("failed to get index manifest:", err)
			}

			if len(idxManifest.Manifests) != 1 {
				t.Fatalf("expected 1 manifest in index, got %d", len(idxManifest.Manifests))
			}

			manifest := idxManifest.Manifests[0]
			if manifest.Platform == nil {
				t.Fatal("expected platform to be set on manifest descriptor")
			}
			if manifest.Platform.Architecture != runtime.GOARCH {
				t.Errorf("expected architecture %s, got %s", runtime.GOARCH, manifest.Platform.Architecture)
			}
			if manifest.Platform.OS != "linux" {
				t.Errorf("expected OS linux, got %s", manifest.Platform.OS)
			}

			mt, err := idx.MediaType()
			if err != nil {
				t.Fatal("failed to get index media type:", err)
			}
			if mt != tc.wantMediaType {
				t.Errorf("expected index media type %q, got %q", tc.wantMediaType, mt)
			}
		})
	}
}

func TestIntegrationCreateManifestListFromRefs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	bkClient, registryAddr := startBuildKitAndRegistry(t)
	tarPath := buildTestImage(t, bkClient)
	plat := "linux/" + runtime.GOARCH

	// Push the platform image first
	pushedRef := fmt.Sprintf("%s/test-img:latest.%s", registryAddr, runtime.GOARCH)
	pushImageToRegistry(t, tarPath, pushedRef)

	cases := []struct {
		name             string
		tag              string
		dockerMediaTypes bool
		wantMediaType    types.MediaType
	}{
		{"oci (default)", "fromrefs-oci", false, types.OCIImageIndex},
		{"docker manifest list", "fromrefs-docker", true, types.DockerManifestList},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			targetRef := fmt.Sprintf("%s/test-img:%s", registryAddr, tc.tag)
			if err := CreateManifestListFromRefs(targetRef, []PlatformRef{{Ref: pushedRef, Platform: plat}}, tc.dockerMediaTypes); err != nil {
				t.Fatal("CreateManifestListFromRefs failed:", err)
			}

			tag, err := name.NewTag(targetRef, name.Insecure)
			if err != nil {
				t.Fatal(err)
			}

			idx, err := remote.Index(tag, remote.WithTransport(http.DefaultTransport))
			if err != nil {
				t.Fatal("failed to fetch manifest list from registry:", err)
			}

			idxManifest, err := idx.IndexManifest()
			if err != nil {
				t.Fatal("failed to get index manifest:", err)
			}

			if len(idxManifest.Manifests) != 1 {
				t.Fatalf("expected 1 manifest in index, got %d", len(idxManifest.Manifests))
			}

			manifest := idxManifest.Manifests[0]
			if manifest.Platform == nil {
				t.Fatal("expected platform to be set on manifest descriptor")
			}
			if manifest.Platform.Architecture != runtime.GOARCH {
				t.Errorf("expected architecture %s, got %s", runtime.GOARCH, manifest.Platform.Architecture)
			}
			if manifest.Platform.OS != "linux" {
				t.Errorf("expected OS linux, got %s", manifest.Platform.OS)
			}

			mt, err := idx.MediaType()
			if err != nil {
				t.Fatal("failed to get index media type:", err)
			}
			if mt != tc.wantMediaType {
				t.Errorf("expected index media type %q, got %q", tc.wantMediaType, mt)
			}
		})
	}
}
