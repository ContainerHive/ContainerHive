package registry

import (
	"testing"
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
	t.Run("returns remote registry when CI is set", func(t *testing.T) {
		t.Setenv("CI", "true")
		t.Setenv("CONTAINER_HIVE_REGISTRY", "")
		reg := NewRegistry("")
		if reg.IsLocal() {
			t.Error("expected remote registry in CI mode")
		}
		if reg.Address() != "docker.io" {
			t.Errorf("expected docker.io default, got %s", reg.Address())
		}
	})

	t.Run("uses CONTAINER_HIVE_REGISTRY when set", func(t *testing.T) {
		t.Setenv("CI", "true")
		t.Setenv("CONTAINER_HIVE_REGISTRY", "ghcr.io/myorg")
		reg := NewRegistry("")
		if reg.Address() != "ghcr.io/myorg" {
			t.Errorf("expected ghcr.io/myorg, got %s", reg.Address())
		}
	})

	t.Run("returns zot registry when CI is not set", func(t *testing.T) {
		t.Setenv("CI", "")
		reg := NewRegistry("")
		if !reg.IsLocal() {
			t.Error("expected local (zot) registry when CI not set")
		}
	})
}
