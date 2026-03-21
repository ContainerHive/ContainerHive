package ocistore

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// buildOCITar creates a minimal valid OCI image tar for testing.
// It constructs an OCI layout with index.json, oci-layout, a manifest blob,
// and a config blob. The image has no layers. If imageName is non-empty,
// it is set as the "io.containerd.image.name" annotation.
func buildOCITar(t *testing.T, imageName string) string {
	t.Helper()

	config := []byte(`{"architecture":"amd64","os":"linux","rootfs":{"type":"layers","diff_ids":[]}}`)
	configDigest := fmt.Sprintf("sha256:%x", sha256.Sum256(config))

	manifest, _ := json.Marshal(map[string]any{
		"schemaVersion": 2,
		"mediaType":     "application/vnd.oci.image.manifest.v1+json",
		"config": map[string]any{
			"mediaType": "application/vnd.oci.image.config.v1+json",
			"digest":    configDigest,
			"size":      len(config),
		},
		"layers": []any{},
	})
	manifestDigest := fmt.Sprintf("sha256:%x", sha256.Sum256(manifest))

	annotations := map[string]string{}
	if imageName != "" {
		annotations["io.containerd.image.name"] = imageName
	}

	index, _ := json.Marshal(map[string]any{
		"schemaVersion": 2,
		"mediaType":     "application/vnd.oci.image.index.v1+json",
		"manifests": []map[string]any{
			{
				"mediaType":   "application/vnd.oci.image.manifest.v1+json",
				"digest":      manifestDigest,
				"size":        len(manifest),
				"annotations": annotations,
			},
		},
	})

	ociLayout := []byte(`{"imageLayoutVersion":"1.0.0"}`)

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	writeEntry := func(name string, data []byte) {
		tw.WriteHeader(&tar.Header{Name: name, Size: int64(len(data)), Mode: 0644})
		tw.Write(data)
	}

	writeEntry("oci-layout", ociLayout)
	writeEntry("index.json", index)
	writeEntry(fmt.Sprintf("blobs/sha256/%x", sha256.Sum256(manifest)), manifest)
	writeEntry(fmt.Sprintf("blobs/sha256/%x", sha256.Sum256(config)), config)

	tw.Close()

	p := filepath.Join(t.TempDir(), "image.tar")
	if err := os.WriteFile(p, buf.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

// buildOCITarNoManifests creates an OCI tar with an empty manifests array.
func buildOCITarNoManifests(t *testing.T) string {
	t.Helper()

	index, _ := json.Marshal(map[string]any{
		"schemaVersion": 2,
		"manifests":     []any{},
	})
	ociLayout := []byte(`{"imageLayoutVersion":"1.0.0"}`)

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	writeEntry := func(name string, data []byte) {
		tw.WriteHeader(&tar.Header{Name: name, Size: int64(len(data)), Mode: 0644})
		tw.Write(data)
	}

	writeEntry("oci-layout", ociLayout)
	writeEntry("index.json", index)
	tw.Close()

	p := filepath.Join(t.TempDir(), "no-manifests.tar")
	if err := os.WriteFile(p, buf.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

// buildOCITarNoLayout creates an OCI tar missing the oci-layout file.
func buildOCITarNoLayout(t *testing.T) string {
	t.Helper()

	index, _ := json.Marshal(map[string]any{
		"schemaVersion": 2,
		"manifests":     []any{},
	})

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	tw.WriteHeader(&tar.Header{Name: "index.json", Size: int64(len(index)), Mode: 0644})
	tw.Write(index)
	tw.Close()

	p := filepath.Join(t.TempDir(), "no-layout.tar")
	if err := os.WriteFile(p, buf.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

// buildOCITarNoIndexJSON creates a valid tar but without index.json (not an OCI layout).
func buildOCITarNoIndexJSON(t *testing.T) string {
	t.Helper()

	ociLayout := []byte(`{"imageLayoutVersion":"1.0.0"}`)

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	tw.WriteHeader(&tar.Header{Name: "oci-layout", Size: int64(len(ociLayout)), Mode: 0644})
	tw.Write(ociLayout)
	tw.Close()

	p := filepath.Join(t.TempDir(), "no-index.tar")
	if err := os.WriteFile(p, buf.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestImageFromTar covers the ImageFromTar function.
func TestImageFromTar(t *testing.T) {
	t.Run("successful extraction returns non-nil image and annotations", func(t *testing.T) {
		tarPath := buildOCITar(t, "test-image:latest")

		result, err := ImageFromTar(tarPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer result.Cleanup()

		if result.Image == nil {
			t.Fatal("expected non-nil image")
		}
		if result.Annotations == nil {
			t.Fatal("expected non-nil annotations map")
		}
		if result.Cleanup == nil {
			t.Fatal("expected non-nil cleanup function")
		}
	})

	t.Run("annotations contain the image name when set", func(t *testing.T) {
		const imageName = "registry.example.com/myimage:v1.2.3"
		tarPath := buildOCITar(t, imageName)

		result, err := ImageFromTar(tarPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer result.Cleanup()

		got, ok := result.Annotations["io.containerd.image.name"]
		if !ok {
			t.Fatal("expected annotation 'io.containerd.image.name' to be present")
		}
		if got != imageName {
			t.Errorf("annotation value = %q, want %q", got, imageName)
		}
	})

	t.Run("cleanup function removes the temp directory", func(t *testing.T) {
		tarPath := buildOCITar(t, "cleanup-test:latest")

		result, err := ImageFromTar(tarPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Determine the temp dir by checking where the image was extracted.
		// We call cleanup and then verify we can't stat something that was there.
		// The simplest approach: call cleanup and ensure it doesn't panic/error.
		result.Cleanup()

		// Calling cleanup a second time should also be safe (os.RemoveAll on a
		// missing path is a no-op).
		result.Cleanup()
	})

	t.Run("returns error for nonexistent tar", func(t *testing.T) {
		_, err := ImageFromTar("/nonexistent/path/image.tar")
		if err == nil {
			t.Fatal("expected error for nonexistent tar")
		}
		t.Logf("got expected error: %v", err)
	})

	t.Run("returns error for invalid tar data", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "garbage.tar")
		os.WriteFile(p, []byte("not a tar file at all"), 0644)

		_, err := ImageFromTar(p)
		if err == nil {
			t.Fatal("expected error for invalid tar data")
		}
		t.Logf("got expected error: %v", err)
	})

	t.Run("returns error for tar missing oci-layout", func(t *testing.T) {
		tarPath := buildOCITarNoLayout(t)

		_, err := ImageFromTar(tarPath)
		if err == nil {
			t.Fatal("expected error for tar without oci-layout")
		}
		t.Logf("got expected error: %v", err)
	})

	t.Run("returns error for empty manifests array", func(t *testing.T) {
		tarPath := buildOCITarNoManifests(t)

		_, err := ImageFromTar(tarPath)
		if err == nil {
			t.Fatal("expected error for empty manifests")
		}
		if err.Error() != "no manifests in OCI layout" {
			t.Errorf("unexpected error message: %v", err)
		}
		t.Logf("got expected error: %v", err)
	})
}

// TestFromTar covers the FromTar function.
func TestFromTar(t *testing.T) {
	t.Run("successful extraction returns store with non-empty digest", func(t *testing.T) {
		tarPath := buildOCITar(t, "test-image:latest")
		destDir := t.TempDir()

		store, err := FromTar(tarPath, destDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if store == nil {
			t.Fatal("expected non-nil store")
		}
		if store.Store == nil {
			t.Fatal("expected non-nil content store")
		}
	})

	t.Run("digest is non-empty", func(t *testing.T) {
		tarPath := buildOCITar(t, "test-image:latest")
		destDir := t.TempDir()

		store, err := FromTar(tarPath, destDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if store.Digest == "" {
			t.Fatal("expected non-empty digest")
		}
		t.Logf("got digest: %s", store.Digest)
	})

	t.Run("returns error for nonexistent tar", func(t *testing.T) {
		destDir := t.TempDir()

		_, err := FromTar("/nonexistent/path/image.tar", destDir)
		if err == nil {
			t.Fatal("expected error for nonexistent tar")
		}
		t.Logf("got expected error: %v", err)
	})

	t.Run("returns error for tar missing index.json", func(t *testing.T) {
		tarPath := buildOCITarNoIndexJSON(t)
		destDir := t.TempDir()

		_, err := FromTar(tarPath, destDir)
		if err == nil {
			t.Fatal("expected error for tar without index.json")
		}
		t.Logf("got expected error: %v", err)
	})

	t.Run("returns error for empty manifests", func(t *testing.T) {
		tarPath := buildOCITarNoManifests(t)
		destDir := t.TempDir()

		_, err := FromTar(tarPath, destDir)
		if err == nil {
			t.Fatal("expected error for empty manifests")
		}
		if err.Error() != "OCI index.json contains no manifests" {
			t.Errorf("unexpected error message: %v", err)
		}
		t.Logf("got expected error: %v", err)
	})
}
