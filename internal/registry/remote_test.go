package registry

import (
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
