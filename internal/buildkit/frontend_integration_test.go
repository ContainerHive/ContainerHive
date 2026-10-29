package buildkit

// End-to-end tests for the custom BuildKit frontends (chisel, nix). Both
// tests run a real BuildKit daemon in a testcontainer and solve the
// production LLB graph of the backend, so they cover the frontend contract,
// the builder images, and the exported image config.

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ContainerHive/ContainerHive/internal/buildkit/build_context"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

// requireIntegration skips in short mode and without a usable Docker daemon.
func requireIntegration(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
}

// exportedImage reads the exported OCI archive in memory and returns the
// image entrypoint, the layer count, and the file type flags of all layers.
// Reading from memory avoids unpacking archive entry names to disk.
func exportedImage(t *testing.T, tarFile string) (entrypoint []string, layerCount int, files map[string]byte) {
	t.Helper()

	entries := readOCIArchive(t, tarFile)

	var index ocispecs.Index
	unmarshalOCIEntry(t, entries, "index.json", &index)
	if len(index.Manifests) == 0 {
		t.Fatal("exported OCI archive contains no manifests")
	}
	var manifest ocispecs.Manifest
	unmarshalOCIEntry(t, entries, ociBlobPath(index.Manifests[0].Digest.String()), &manifest)
	var config ocispecs.Image
	unmarshalOCIEntry(t, entries, ociBlobPath(manifest.Config.Digest.String()), &config)

	files = map[string]byte{}
	for _, layer := range manifest.Layers {
		data, ok := entries[ociBlobPath(layer.Digest.String())]
		if !ok {
			t.Fatalf("layer blob %s missing from the archive", layer.Digest)
		}
		tr := tar.NewReader(bytes.NewReader(decompressLayer(t, data, layer.MediaType)))
		for {
			header, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("reading layer entries failed: %v", err)
			}
			files[strings.TrimPrefix(header.Name, "./")] = header.Typeflag
		}
	}

	return config.Config.Entrypoint, len(manifest.Layers), files
}

// readOCIArchive reads all regular files of an OCI image archive into memory.
func readOCIArchive(t *testing.T, tarFile string) map[string][]byte {
	t.Helper()

	f, err := os.Open(tarFile)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	entries := map[string][]byte{}
	tr := tar.NewReader(f)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("reading OCI archive failed: %v", err)
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			t.Fatalf("reading OCI archive entry %q failed: %v", header.Name, err)
		}
		entries[header.Name] = data
	}
	return entries
}

// unmarshalOCIEntry decodes a JSON document from the in-memory archive.
func unmarshalOCIEntry(t *testing.T, entries map[string][]byte, name string, target any) {
	t.Helper()
	data, ok := entries[name]
	if !ok {
		t.Fatalf("OCI archive misses %q", name)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("decoding %q failed: %v", name, err)
	}
}

// ociBlobPath maps a digest to its blobs/ path inside an OCI archive.
func ociBlobPath(digest string) string {
	algorithm, encoded, ok := strings.Cut(digest, ":")
	if !ok {
		return "blobs/" + digest
	}
	return "blobs/" + algorithm + "/" + encoded
}

// decompressLayer returns the plain layer tar, decompressing gzip layers.
func decompressLayer(t *testing.T, data []byte, mediaType string) []byte {
	t.Helper()
	if !strings.Contains(mediaType, "gzip") {
		return data
	}
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decompressing layer failed: %v", err)
	}
	out, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading decompressed layer failed: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("closing layer reader failed: %v", err)
	}
	return out
}

func TestIntegrationChiselFrontend(t *testing.T) {
	requireIntegration(t)

	platform := "linux/" + runtime.GOARCH
	bkClient := startBuildKit(t)

	buildCtxDir := t.TempDir()
	// The cut spec drives the CLI side; the frontend solves the LLB graph.
	// Write one so the build context exists and OriginPath resolves.
	if err := os.WriteFile(filepath.Join(buildCtxDir, "chisel.yaml"), []byte(`release: ubuntu-24.04
slices:
  - base-files_release-info
`), 0644); err != nil {
		t.Fatal(err)
	}

	tarFile := filepath.Join(t.TempDir(), "image.tar")
	entrypoint := []string{"/bin/hive-e2e"}
	err := bkClient.Build(t.Context(), &BuildOpts{
		ImageName: "integration-chisel:latest",
		TarFile:   tarFile,
		Platform:  platform,
		BuildContext: &build_context.ChiselBuildContext{
			Root:       buildCtxDir,
			Release:    build_context.ChiselRelease{Name: "ubuntu-24.04"},
			Slices:     []string{"base-files_release-info"},
			Entrypoint: entrypoint,
		},
	}, drainStatus)
	if err != nil {
		t.Fatal("Build() returned error:", err)
	}

	gotEntrypoint, layers, files := exportedImage(t, tarFile)
	if len(gotEntrypoint) != 1 || gotEntrypoint[0] != entrypoint[0] {
		t.Errorf("exported entrypoint = %v, want %v", gotEntrypoint, entrypoint)
	}
	if layers == 0 {
		t.Fatal("exported image has no layers")
	}

	// base-files_release-info ships /etc/os-release as a symlink. The tar
	// flag proves the cut output copied verbatim instead of dereferencing it.
	flag, found := files["etc/os-release"]
	if !found {
		t.Fatalf("cut rootfs misses /etc/os-release (files: %d)", len(files))
	}
	if flag != tar.TypeSymlink {
		t.Errorf("/etc/os-release type = %c, want symlink", flag)
	}
	if _, found := files["etc/lsb-release"]; !found {
		t.Error("cut rootfs misses /etc/lsb-release")
	}
}

func TestIntegrationNixFrontend(t *testing.T) {
	requireIntegration(t)

	platform := "linux/" + runtime.GOARCH
	bkClient := startBuildKit(t)

	buildCtxDir := t.TempDir()
	// A flake without inputs whose default package is a plain rootfs path.
	// The build evaluates offline, so the test only needs the nix builder
	// image, not nixpkgs.
	flake := `{
  outputs = { self }: {
    packages.x86_64-linux.default = ./rootfs;
    packages.aarch64-linux.default = ./rootfs;
  };
}
`
	if err := os.MkdirAll(filepath.Join(buildCtxDir, "rootfs", "etc"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(buildCtxDir, "rootfs", "etc", "hive-e2e"), []byte("hive"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(buildCtxDir, "flake.nix"), []byte(flake), 0644); err != nil {
		t.Fatal(err)
	}

	tarFile := filepath.Join(t.TempDir(), "image.tar")
	err := bkClient.Build(t.Context(), &BuildOpts{
		ImageName: "integration-nix:latest",
		TarFile:   tarFile,
		Platform:  platform,
		BuildContext: &build_context.NixBuildContext{
			Root: buildCtxDir,
		},
	}, drainStatus)
	if err != nil {
		t.Fatal("Build() returned error:", err)
	}

	_, layers, files := exportedImage(t, tarFile)
	if layers == 0 {
		t.Fatal("exported image has no layers")
	}
	if _, found := files["etc/hive-e2e"]; !found {
		t.Errorf("flake output missing from image rootfs (files: %d)", len(files))
	}
}
