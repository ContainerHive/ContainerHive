package registry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/timo-reymann/ContainerHive/pkg/model"
)

func TestRemoteRegistry(t *testing.T) {
	t.Run("returns configured address", func(t *testing.T) {
		reg := NewRemoteRegistry("docker.io/myorg")
		if reg.Address() != "docker.io/myorg" {
			t.Errorf("expected docker.io/myorg, got %s", reg.Address())
		}
	})

	t.Run("is not local", func(t *testing.T) {
		reg := NewRemoteRegistry("docker.io/myorg")
		if reg.IsLocal() {
			t.Error("expected IsLocal() to be false")
		}
	})

	t.Run("start and stop are no-ops", func(t *testing.T) {
		reg := NewRemoteRegistry("docker.io/myorg")
		if err := reg.Start(t.Context()); err != nil {
			t.Fatalf("unexpected error from Start: %v", err)
		}
		if err := reg.Stop(t.Context()); err != nil {
			t.Fatalf("unexpected error from Stop: %v", err)
		}
	})
}

func TestRemoteRegistry_Push(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping remote push integration test")
	}

	t.Run("pushes OCI tar to remote registry", func(t *testing.T) {
		zot := NewZotRegistry("")
		if err := zot.Start(t.Context()); err != nil {
			t.Fatalf("failed to start zot: %v", err)
		}
		t.Cleanup(func() { zot.Stop(t.Context()) })

		reg := NewRemoteRegistry(zot.Address())
		tarPath := buildOCITar(t)

		if err := reg.Push(t.Context(), "myapp", "v1.0.0", tarPath); err != nil {
			t.Fatalf("push failed: %v", err)
		}

		resp, err := http.Get(fmt.Sprintf("http://%s/v2/_catalog", zot.Address()))
		if err != nil {
			t.Fatalf("catalog request failed: %v", err)
		}
		defer resp.Body.Close()

		var catalog struct {
			Repositories []string `json:"repositories"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&catalog); err != nil {
			t.Fatalf("failed to decode catalog: %v", err)
		}

		found := false
		for _, repo := range catalog.Repositories {
			if repo == "myapp" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected 'myapp' in catalog, got %v", catalog.Repositories)
		}
	})

	t.Run("fails with invalid tar path", func(t *testing.T) {
		reg := NewRemoteRegistry("localhost:9999")
		err := reg.Push(t.Context(), "myapp", "v1.0.0", "/nonexistent/image.tar")
		if err == nil {
			t.Error("expected error for nonexistent tar path")
		}
	})
}

func TestNewRegistry_CI(t *testing.T) {
	t.Run("uses hive.yml registry address in CI", func(t *testing.T) {
		t.Setenv("CI", "true")
		t.Setenv("CONTAINER_HIVE_REGISTRY", "")
		reg, err := NewRegistry("", &model.RegistryConfig{Address: "registry.example.com"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if reg.IsLocal() {
			t.Error("expected remote registry in CI mode")
		}
		if reg.Address() != "registry.example.com" {
			t.Errorf("expected registry.example.com, got %s", reg.Address())
		}
	})

	t.Run("CONTAINER_HIVE_REGISTRY env var takes precedence over hive.yml", func(t *testing.T) {
		t.Setenv("CI", "true")
		t.Setenv("CONTAINER_HIVE_REGISTRY", "ghcr.io/myorg")
		reg, err := NewRegistry("", &model.RegistryConfig{Address: "registry.example.com"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if reg.Address() != "ghcr.io/myorg" {
			t.Errorf("expected ghcr.io/myorg, got %s", reg.Address())
		}
	})

	t.Run("errors when CI is set but no registry configured", func(t *testing.T) {
		t.Setenv("CI", "true")
		t.Setenv("CONTAINER_HIVE_REGISTRY", "")
		_, err := NewRegistry("", nil)
		if err == nil {
			t.Error("expected error when no registry is configured in CI")
		}
	})

	t.Run("returns zot registry when CI is not set", func(t *testing.T) {
		t.Setenv("CI", "")
		reg, err := NewRegistry("", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reg.IsLocal() {
			t.Error("expected local (zot) registry when CI not set")
		}
	})
}
