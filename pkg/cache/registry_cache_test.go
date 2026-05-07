package cache

import "testing"

func TestRegistryCacheAttributes(t *testing.T) {
	cache := &RegistryCache{
		CacheRef: "registry.example.com/my-cache:latest",
	}

	attrs := cache.ToAttributes()

	expectedAttrs := map[string]string{
		"mode":           "max",
		"ref":            "registry.example.com/my-cache:latest",
		"image-manifest": "true",
		"oci-mediatypes": "true",
	}

	for key, want := range expectedAttrs {
		if got := attrs[key]; got != want {
			t.Errorf("attribute %q = %q, want %q", key, got, want)
		}
	}

	if _, ok := attrs["registry.insecure"]; ok {
		t.Error("expected registry.insecure to be absent when Insecure is false")
	}

	if cache.Name() != "registry" {
		t.Errorf("Name() = %q, want %q", cache.Name(), "registry")
	}
}

func TestRegistryCache_WithScope(t *testing.T) {
	cache := &RegistryCache{
		CacheRef: "registry.example.com/my-cache:latest",
	}

	scoped := cache.WithScope("ubuntu.22.04.linux/amd64")
	reg, ok := scoped.(*RegistryCache)
	if !ok {
		t.Fatalf("expected *RegistryCache, got %T", scoped)
	}

	if reg.CacheRef != "registry.example.com/my-cache:latest.ubuntu.22.04.linux_amd64" {
		t.Errorf("CacheRef = %q, want %q", reg.CacheRef, "registry.example.com/my-cache:latest.ubuntu.22.04.linux_amd64")
	}
	if reg.Insecure {
		t.Error("expected Insecure to remain false")
	}
}

func TestRegistryCache_WithScope_Insecure(t *testing.T) {
	cache := &RegistryCache{
		CacheRef: "localhost:5000/my-cache",
		Insecure: true,
	}

	scoped := cache.WithScope("python.3.11-slim.linux/arm64")
	reg, ok := scoped.(*RegistryCache)
	if !ok {
		t.Fatalf("expected *RegistryCache, got %T", scoped)
	}

	if reg.CacheRef != "localhost:5000/my-cache.python.3.11-slim.linux_arm64" {
		t.Errorf("CacheRef = %q, want %q", reg.CacheRef, "localhost:5000/my-cache.python.3.11-slim.linux_arm64")
	}
	if !reg.Insecure {
		t.Error("expected Insecure to remain true")
	}
}

func TestRegistryCache_WithScope_PreservesOriginal(t *testing.T) {
	cache := &RegistryCache{
		CacheRef: "registry.example.com/my-cache:latest",
	}

	_ = cache.WithScope("some.scope")

	if cache.CacheRef != "registry.example.com/my-cache:latest" {
		t.Error("WithScope should not mutate the original cache")
	}
}

func TestRegistryCacheAttributes_Insecure(t *testing.T) {
	cache := &RegistryCache{
		CacheRef: "localhost:5000/my-cache",
		Insecure: true,
	}

	attrs := cache.ToAttributes()

	if got := attrs["registry.insecure"]; got != "true" {
		t.Errorf("registry.insecure = %q, want %q", got, "true")
	}
}
