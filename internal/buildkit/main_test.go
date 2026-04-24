package buildkit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildExports_NoRegistryRef(t *testing.T) {
	opts := &BuildOpts{
		ImageName: "my-image:latest",
		TarFile:   "/tmp/unused.tar",
	}

	exports := buildExports(opts)

	if len(exports) != 1 {
		t.Fatalf("expected 1 export entry, got %d", len(exports))
	}

	entry := exports[0]

	if entry.Type != "oci" {
		t.Errorf("expected type %q, got %q", "oci", entry.Type)
	}

	if got := entry.Attrs["name"]; got != opts.ImageName {
		t.Errorf("expected attrs[\"name\"] = %q, got %q", opts.ImageName, got)
	}

	if got := entry.Attrs["rewrite-timestamp"]; got != "true" {
		t.Errorf("expected attrs[\"rewrite-timestamp\"] = %q, got %q", "true", got)
	}

	if entry.Output == nil {
		t.Error("expected Output function to be non-nil, got nil")
	}
}

func TestBuildExports_WithRegistryRef(t *testing.T) {
	opts := &BuildOpts{
		ImageName:   "my-image:latest",
		TarFile:     "/tmp/unused.tar",
		RegistryRef: "registry.example.com/my-image:latest",
	}

	exports := buildExports(opts)

	if len(exports) != 2 {
		t.Fatalf("expected 2 export entries, got %d", len(exports))
	}

	imageEntry := exports[1]

	if imageEntry.Type != "image" {
		t.Errorf("expected second entry type %q, got %q", "image", imageEntry.Type)
	}

	if got := imageEntry.Attrs["name"]; got != opts.RegistryRef {
		t.Errorf("expected attrs[\"name\"] = %q, got %q", opts.RegistryRef, got)
	}

	if got := imageEntry.Attrs["push"]; got != "true" {
		t.Errorf("expected attrs[\"push\"] = %q, got %q", "true", got)
	}

	if got := imageEntry.Attrs["rewrite-timestamp"]; got != "true" {
		t.Errorf("expected attrs[\"rewrite-timestamp\"] = %q, got %q", "true", got)
	}

	if _, ok := imageEntry.Attrs["registry.insecure"]; ok {
		t.Error("expected no \"registry.insecure\" attr, but it was present")
	}

	if _, ok := imageEntry.Attrs["oci-mediatypes"]; ok {
		t.Error("expected no \"oci-mediatypes\" attr by default, but it was present")
	}
}

func TestBuildExports_WithDockerMediaTypes(t *testing.T) {
	opts := &BuildOpts{
		ImageName:        "my-image:latest",
		TarFile:          "/tmp/unused.tar",
		RegistryRef:      "docker.io/me/my-image:latest",
		DockerMediaTypes: true,
	}

	exports := buildExports(opts)

	if len(exports) != 2 {
		t.Fatalf("expected 2 export entries, got %d", len(exports))
	}

	imageEntry := exports[1]
	if got := imageEntry.Attrs["oci-mediatypes"]; got != "false" {
		t.Errorf("expected attrs[\"oci-mediatypes\"] = \"false\", got %q", got)
	}
}

func TestBuildExports_WithRegistryRefAndInsecure(t *testing.T) {
	opts := &BuildOpts{
		ImageName:        "my-image:latest",
		TarFile:          "/tmp/unused.tar",
		RegistryRef:      "localhost:5000/my-image:latest",
		RegistryInsecure: true,
	}

	exports := buildExports(opts)

	if len(exports) != 2 {
		t.Fatalf("expected 2 export entries, got %d", len(exports))
	}

	imageEntry := exports[1]

	if got := imageEntry.Attrs["registry.insecure"]; got != "true" {
		t.Errorf("expected attrs[\"registry.insecure\"] = %q, got %q", "true", got)
	}
}

func TestBuildExports_OCITarOutputCreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	tarPath := filepath.Join(tmpDir, "image.tar")

	opts := &BuildOpts{
		ImageName: "my-image:latest",
		TarFile:   tarPath,
	}

	exports := buildExports(opts)

	if len(exports) == 0 {
		t.Fatal("expected at least one export entry, got 0")
	}

	outputFn := exports[0].Output
	if outputFn == nil {
		t.Fatal("expected Output function to be non-nil, got nil")
	}

	wc, err := outputFn(nil)
	if err != nil {
		t.Fatalf("Output function returned unexpected error: %v", err)
	}
	if err := wc.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	if _, err := os.Stat(tarPath); os.IsNotExist(err) {
		t.Errorf("expected file to be created at %q, but it does not exist", tarPath)
	}
}
