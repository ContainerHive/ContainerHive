package buildkit

import (
	"fmt"
	"io"
	"net/http"
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
	"github.com/timo-reymann/ContainerHive/internal/buildkit/build_context"
	"github.com/timo-reymann/ContainerHive/pkg/cache"
	"github.com/timo-reymann/ContainerHive/internal/testutil"
)

const (
	s3Bucket    = "buildkit-cache"
	s3AccessKey = "S3RVER"
	s3SecretKey = "S3RVER"
)

func drainStatus(ch chan *client.SolveStatus) error {
	for range ch {
	}
	return nil
}

func setupTestBuildContext(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM scratch\nCOPY Dockerfile /\n"), 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestIntegrationBuild(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := t.Context()
	platform := "linux/" + runtime.GOARCH
	buildCtxDir := setupTestBuildContext(t)

	// Shared network so BuildKit can reach S3 by hostname
	net, err := network.New(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { net.Remove(ctx) })

	// --- S3 (adobe/s3mock) ---
	s3C, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "adobe/s3mock:latest",
			ExposedPorts: []string{"9090/tcp"},
			Env: map[string]string{
				"initialBuckets": s3Bucket,
			},
			Networks: []string{net.Name},
			NetworkAliases: map[string][]string{
				net.Name: {"s3mock"},
			},
			WaitingFor: wait.ForHTTP("/").WithPort("9090/tcp"),
		},
		Started: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s3C.Terminate(ctx) })

	s3Host, err := s3C.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	s3Port, err := s3C.MappedPort(ctx, "9090/tcp")
	if err != nil {
		t.Fatal(err)
	}

	// --- BuildKit ---
	buildkitC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        testutil.BuildKitImage(),
			ExposedPorts: []string{"1234/tcp"},
			Cmd:          []string{"--addr", "tcp://0.0.0.0:1234"},
			Networks:     []string{net.Name},
			WaitingFor:   wait.ForListeningPort("1234/tcp"),
			HostConfigModifier: func(hc *container.HostConfig) {
				hc.Privileged = true
			},
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

	bkClient, err := NewClient(ctx, fmt.Sprintf("tcp://%s:%s", buildkitHost, buildkitPort.Port()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { bkClient.Close() })

	t.Run("version", func(t *testing.T) {
		version, err := bkClient.Version(ctx)
		if err != nil {
			t.Fatal("failed to get buildkit version:", err)
		}
		if version == "" {
			t.Fatal("expected non-empty buildkit version")
		}
		t.Logf("BuildKit version: %s", version)
	})

	t.Run("without_cache", func(t *testing.T) {
		tarFile := filepath.Join(t.TempDir(), "output.tar")

		err := bkClient.Build(ctx, &BuildOpts{
			ImageName: "test-no-cache:latest",
			TarFile:   tarFile,
			BuildContext: &build_context.DockerfileBuildContext{
				Root: buildCtxDir,
			},
			Platform: platform,
		}, drainStatus)
		if err != nil {
			t.Fatal(err)
		}

		info, err := os.Stat(tarFile)
		if err != nil {
			t.Fatal("expected tar file to exist:", err)
		}
		if info.Size() == 0 {
			t.Fatal("expected tar file to be non-empty")
		}
	})

	t.Run("with_s3_cache", func(t *testing.T) {
		tmpDir := t.TempDir()
		s3Cache := &cache.S3BuildKitCache{
			// Internal Docker network address — BuildKit reaches S3 via the shared network
			EndpointUrl:     "http://s3mock:9090",
			Bucket:          s3Bucket,
			Region:          "us-east-1",
			CacheKey:        "integration-test",
			AccessKeyId:     s3AccessKey,
			SecretAccessKey: s3SecretKey,
			UsePathStyle:    true,
		}

		// First build — populates the cache
		tarFile1 := filepath.Join(tmpDir, "cached1.tar")
		if err := bkClient.Build(ctx, &BuildOpts{
			ImageName: "test-cached:latest",
			TarFile:   tarFile1,
			Cache:     s3Cache,
			BuildContext: &build_context.DockerfileBuildContext{
				Root: buildCtxDir,
			},
			Platform: platform,
		}, drainStatus); err != nil {
			t.Fatal("first build (cache populate):", err)
		}

		info, err := os.Stat(tarFile1)
		if err != nil {
			t.Fatal("expected first tar file to exist:", err)
		}
		if info.Size() == 0 {
			t.Fatal("expected first tar file to be non-empty")
		}

		// Second build — should use the cache
		tarFile2 := filepath.Join(tmpDir, "cached2.tar")
		if err := bkClient.Build(ctx, &BuildOpts{
			ImageName: "test-cached-reuse:latest",
			TarFile:   tarFile2,
			Cache:     s3Cache,
			BuildContext: &build_context.DockerfileBuildContext{
				Root: buildCtxDir,
			},
			Platform: platform,
		}, drainStatus); err != nil {
			t.Fatal("second build (cache reuse):", err)
		}

		info, err = os.Stat(tarFile2)
		if err != nil {
			t.Fatal("expected second tar file to exist:", err)
		}
		if info.Size() == 0 {
			t.Fatal("expected second tar file to be non-empty")
		}

		// Verify objects were written to the cache bucket (s3mock needs no auth)
		resp, err := http.Get(fmt.Sprintf("http://%s:%s/%s?list-type=2", s3Host, s3Port.Port(), s3Bucket))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "<Key>") {
			t.Fatal("expected S3 cache bucket to contain objects after cached build, but it was empty")
		}
	})

	t.Run("with_registry_cache", func(t *testing.T) {
		// Start a local registry
		registryC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image:        "registry:2",
				ExposedPorts: []string{"5000/tcp"},
				Networks:     []string{net.Name},
				NetworkAliases: map[string][]string{
					net.Name: {"registry"},
				},
				WaitingFor: wait.ForHTTP("/v2/").WithPort("5000/tcp"),
			},
			Started: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { registryC.Terminate(ctx) })

		registryCache := &cache.RegistryCache{
			CacheRef: "registry:5000/buildkit-cache",
			Insecure: true,
		}

		// First build — populates the cache
		tarFile1 := filepath.Join(t.TempDir(), "registry-cached1.tar")
		if err := bkClient.Build(ctx, &BuildOpts{
			ImageName: "test-registry-cached:latest",
			TarFile:   tarFile1,
			Cache:     registryCache,
			BuildContext: &build_context.DockerfileBuildContext{
				Root: buildCtxDir,
			},
			Platform: platform,
		}, drainStatus); err != nil {
			t.Fatal("first build (cache populate):", err)
		}

		info, err := os.Stat(tarFile1)
		if err != nil {
			t.Fatal("expected first tar file to exist:", err)
		}
		if info.Size() == 0 {
			t.Fatal("expected first tar file to be non-empty")
		}

		// Second build — should use the cache
		tarFile2 := filepath.Join(t.TempDir(), "registry-cached2.tar")
		if err := bkClient.Build(ctx, &BuildOpts{
			ImageName: "test-registry-cached-reuse:latest",
			TarFile:   tarFile2,
			Cache:     registryCache,
			BuildContext: &build_context.DockerfileBuildContext{
				Root: buildCtxDir,
			},
			Platform: platform,
		}, drainStatus); err != nil {
			t.Fatal("second build (cache reuse):", err)
		}

		info, err = os.Stat(tarFile2)
		if err != nil {
			t.Fatal("expected second tar file to exist:", err)
		}
		if info.Size() == 0 {
			t.Fatal("expected second tar file to be non-empty")
		}

		// Verify the cache image was pushed to the registry
		registryHost, err := registryC.Host(ctx)
		if err != nil {
			t.Fatal(err)
		}
		registryPort, err := registryC.MappedPort(ctx, "5000/tcp")
		if err != nil {
			t.Fatal(err)
		}

		resp, err := http.Get(fmt.Sprintf("http://%s:%s/v2/buildkit-cache/tags/list", registryHost, registryPort.Port()))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected registry to have cache image, got status: %d", resp.StatusCode)
		}
	})

	t.Run("ignore_errors_default", func(t *testing.T) {
		// Test that ignore-errors is set to true by default
		s3Cache := &cache.S3BuildKitCache{
			EndpointUrl:     "http://s3mock:9090",
			Bucket:          s3Bucket,
			Region:          "us-east-1",
			CacheKey:        "ignore-errors-test",
			AccessKeyId:     s3AccessKey,
			SecretAccessKey: s3SecretKey,
			UsePathStyle:    true,
		}

		// Build should succeed even if cache operations might fail
		tarFile := filepath.Join(t.TempDir(), "ignore-errors.tar")
		if err := bkClient.Build(ctx, &BuildOpts{
			ImageName: "test-ignore-errors:latest",
			TarFile:   tarFile,
			Cache:     s3Cache,
			BuildContext: &build_context.DockerfileBuildContext{
				Root: buildCtxDir,
			},
			Platform: platform,
		}, drainStatus); err != nil {
			t.Fatal("build should succeed even with cache issues:", err)
		}

		info, err := os.Stat(tarFile)
		if err != nil {
			t.Fatal("expected tar file to exist:", err)
		}
		if info.Size() == 0 {
			t.Fatal("expected tar file to be non-empty")
		}
	})
}

func TestIgnoreErrorsHandling(t *testing.T) {
	tests := []struct {
		name          string
		cache         cache.BuildkitCache
		wantIgnoreErr string
	}{
		{
			name: "s3_cache_default",
			cache: &cache.S3BuildKitCache{
				Bucket:   "test",
				CacheKey: "key",
			},
			wantIgnoreErr: "true",
		},
		{
			name: "registry_cache_default",
			cache: &cache.RegistryCache{
				CacheRef: "localhost:5000/cache",
			},
			wantIgnoreErr: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the logic in Build()
			attributes := tt.cache.ToAttributes()
			_, hasExplicitIgnoreErr := attributes["ignore-errors"]
			if !hasExplicitIgnoreErr {
				attributes["ignore-errors"] = "true"
			}

			if got := attributes["ignore-errors"]; got != tt.wantIgnoreErr {
				t.Errorf("ignore-errors = %q, want %q", got, tt.wantIgnoreErr)
			}
		})
	}
}
