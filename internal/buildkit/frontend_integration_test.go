package buildkit

// End-to-end tests for the custom BuildKit frontends (chisel, nix). Both
// tests run a real BuildKit daemon in a testcontainer and solve the
// production LLB graph of the backend, so they cover the frontend contract,
// the builder images, and the exported image config.

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ContainerHive/ContainerHive/internal/buildkit/build_context"
	"github.com/google/go-containerregistry/pkg/v1/layout"
)

// extractOCIArchive unpacks an OCI image archive tar into dir, so the
// layout reader can load it. The archive only holds regular files.
func extractOCIArchive(tarFile, dir string) error {
	f, err := os.Open(tarFile)
	if err != nil {
		return err
	}
	defer f.Close()

	tr := tar.NewReader(f)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		target := filepath.Join(dir, filepath.FromSlash(header.Name))
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		}
	}
}

// requireIntegration skips in short mode and without a usable Docker daemon.
func requireIntegration(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
}

// exportedImage loads the exported OCI tar into a go-containerregistry image.
func exportedImage(t *testing.T, tarFile string) (entrypoint []string, layers int, files map[string]byte) {
	t.Helper()

	layoutDir := t.TempDir()
	if err := extractOCIArchive(tarFile, layoutDir); err != nil {
		t.Fatalf("extracting exported OCI archive failed: %v", err)
	}
	exported, err := layout.FromPath(layoutDir)
	if err != nil {
		t.Fatalf("loading exported OCI layout failed: %v", err)
	}
	index, err := exported.ImageIndex()
	if err != nil {
		t.Fatalf("reading exported image index failed: %v", err)
	}
	manifest, err := index.IndexManifest()
	if err != nil {
		t.Fatalf("reading exported manifest failed: %v", err)
	}
	if len(manifest.Manifests) == 0 {
		t.Fatal("exported OCI archive contains no manifests")
	}
	img, err := index.Image(manifest.Manifests[0].Digest)
	if err != nil {
		t.Fatalf("loading exported image failed: %v", err)
	}
	cfg, err := img.ConfigFile()
	if err != nil {
		t.Fatalf("reading exported image config failed: %v", err)
	}
	imageLayers, err := img.Layers()
	if err != nil {
		t.Fatalf("reading exported image layers failed: %v", err)
	}

	files = map[string]byte{}
	for _, layer := range imageLayers {
		reader, err := layer.Uncompressed()
		if err != nil {
			t.Fatalf("opening layer failed: %v", err)
		}
		tr := tar.NewReader(reader)
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
		reader.Close()
	}

	return cfg.Config.Entrypoint, len(imageLayers), files
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
