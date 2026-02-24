package registry

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func buildOCITar(t *testing.T) string {
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

	index, _ := json.Marshal(map[string]any{
		"schemaVersion": 2,
		"mediaType":     "application/vnd.oci.image.index.v1+json",
		"manifests": []map[string]any{
			{
				"mediaType": "application/vnd.oci.image.manifest.v1+json",
				"digest":    manifestDigest,
				"size":      len(manifest),
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

func TestZotRegistry(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping zot integration test")
	}

	t.Run("starts and responds to health check", func(t *testing.T) {
		reg := NewZotRegistry("")
		if err := reg.Start(t.Context()); err != nil {
			t.Fatalf("failed to start zot: %v", err)
		}
		t.Cleanup(func() { reg.Stop(t.Context()) })

		addr := reg.Address()
		if addr == "" {
			t.Fatal("expected non-empty address")
		}

		resp, err := http.Get(fmt.Sprintf("http://%s/v2/", addr))
		if err != nil {
			t.Fatalf("failed to reach zot: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
	})

	t.Run("push image and verify via catalog", func(t *testing.T) {
		reg := NewZotRegistry("")
		if err := reg.Start(t.Context()); err != nil {
			t.Fatalf("failed to start zot: %v", err)
		}
		t.Cleanup(func() { reg.Stop(t.Context()) })

		tarPath := buildOCITar(t)

		if err := reg.Push(t.Context(), "ubuntu", "22.04", tarPath); err != nil {
			t.Fatalf("push failed: %v", err)
		}

		resp, err := http.Get(fmt.Sprintf("http://%s/v2/_catalog", reg.Address()))
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
			if repo == "ubuntu" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected 'ubuntu' in catalog, got %v", catalog.Repositories)
		}
	})

	t.Run("is local", func(t *testing.T) {
		reg := NewZotRegistry("")
		if !reg.IsLocal() {
			t.Error("expected IsLocal() to be true")
		}
	})
}
