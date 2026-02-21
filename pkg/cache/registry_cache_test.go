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
