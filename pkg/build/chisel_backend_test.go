package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ContainerHive/ContainerHive/internal/buildkit/build_context"
	"github.com/ContainerHive/ContainerHive/pkg/model"
)

func TestEntrypointFileNameChisel(t *testing.T) {
	if got := entrypointFileName(model.BuildBackendChisel); got != model.ChiselFileName {
		t.Errorf("entrypointFileName(chisel) = %q, want %q", got, model.ChiselFileName)
	}
}

func TestHashSourceChiselSlices(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, model.ChiselFileName)
	if err := os.WriteFile(path, []byte("spec"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := hashSource(path)
	if err != nil {
		t.Fatalf("hashSource failed: %v", err)
	}
	if string(got) != "spec" {
		t.Errorf("hashSource() without slices/ = %q, want %q", got, "spec")
	}

	slicesDir := filepath.Join(dir, "slices")
	if err := os.MkdirAll(slicesDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(slicesDir, "hello.yaml"), []byte("custom"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err = hashSource(path)
	if err != nil {
		t.Fatalf("hashSource failed: %v", err)
	}
	if string(got) != "speccustom" {
		t.Errorf("hashSource() with slices/ = %q, want %q", got, "speccustom")
	}
}

func TestHashSourceDockerfileIgnoresSlicesDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Dockerfile")
	if err := os.WriteFile(path, []byte("entry"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "slices"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "slices", "hello.yaml"), []byte("custom"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := hashSource(path)
	if err != nil {
		t.Fatalf("hashSource failed: %v", err)
	}
	if string(got) != "entry" {
		t.Errorf("hashSource() = %q, want %q", got, "entry")
	}
}

func TestChiselConfigFor(t *testing.T) {
	imageDef := &model.Image{Name: "hello"}

	if got := chiselConfigFor(imageDef, nil); got.BuilderImage != "" {
		t.Errorf("chiselConfigFor without config = %+v, want empty", got)
	}
	if !chiselConfigFor(imageDef, nil).CacheEnabled() {
		t.Error("chisel cache must default to enabled")
	}

	variantDef := &model.ImageVariant{Chisel: &model.ChiselConfig{BuilderImage: "debian:12"}}
	if got := chiselConfigFor(imageDef, variantDef); got.BuilderImage != "debian:12" {
		t.Errorf("chiselConfigFor = %+v, want the variant config", got)
	}

	imageDef.Chisel = model.ChiselConfig{BuilderImage: "ubuntu:24.04"}
	if got := chiselConfigFor(imageDef, nil); got.BuilderImage != "ubuntu:24.04" {
		t.Errorf("chiselConfigFor = %+v, want the image config", got)
	}
}

func TestChiselParentForBuild(t *testing.T) {
	imageDef := &model.Image{Name: "hello"}

	t.Run("plain refs pass through", func(t *testing.T) {
		config, err := chiselParentForBuild(model.ChiselConfig{Parent: "ubuntu:22.04"}, imageDef, "")
		if err != nil {
			t.Fatalf("chiselParentForBuild failed: %v", err)
		}
		if config.Parent != "ubuntu:22.04" {
			t.Errorf("parent = %q", config.Parent)
		}
	})

	t.Run("placeholder expands in a variant", func(t *testing.T) {
		config, err := chiselParentForBuild(model.ChiselConfig{Parent: model.HiveParentPlaceholder}, imageDef, "1.0")
		if err != nil {
			t.Fatalf("chiselParentForBuild failed: %v", err)
		}
		if config.Parent != "__hive__/hello:1.0" {
			t.Errorf("parent = %q, want __hive__/hello:1.0", config.Parent)
		}
	})

	t.Run("placeholder outside a variant fails", func(t *testing.T) {
		_, err := chiselParentForBuild(model.ChiselConfig{Parent: model.HiveParentPlaceholder}, imageDef, "")
		if err == nil || !strings.Contains(err.Error(), "placeholder") {
			t.Errorf("expected a placeholder error, got: %v", err)
		}
	})
}

func TestChiselBuildContextAssembly(t *testing.T) {
	dir := t.TempDir()
	const cutSpec = `
release:
  git: https://example.com/chisel-releases.git
  ref: "1.0"
slices:
  - base-files_base
ignore: [unmaintained]
`
	if err := os.WriteFile(filepath.Join(dir, model.ChiselFileName), []byte(cutSpec), 0o600); err != nil {
		t.Fatal(err)
	}
	spec, err := parseChiselCutSpec(dir)
	if err != nil {
		t.Fatalf("parseChiselCutSpec failed: %v", err)
	}

	chisel := model.ChiselConfig{BuilderImage: "debian:12", ChiselVersion: "v1.5.1",
		ExtraArgs: []string{"--trace"}, Cache: boolRef(false)}
	run := &model.ImageRunConfig{Entrypoint: []string{"/bin/hello"}, User: "1000"}

	parent := &build_context.ParentImage{RegistryRef: "ubuntu:22.04"}
	ctx, err := chiselBuildContext(dir, spec, chisel, run, parent)
	if err != nil {
		t.Fatalf("chiselBuildContext failed: %v", err)
	}

	if ctx.BuilderImage != "debian:12" {
		t.Errorf("builder image = %q", ctx.BuilderImage)
	}
	if ctx.Release.Git != "https://example.com/chisel-releases.git" || ctx.Release.GitRef != "1.0" {
		t.Errorf("release = %+v", ctx.Release)
	}
	if len(ctx.Slices) != 1 || ctx.Slices[0] != "base-files_base" {
		t.Errorf("slices = %v", ctx.Slices)
	}
	if len(ctx.Ignore) != 1 || ctx.Ignore[0] != "unmaintained" {
		t.Errorf("ignore = %v", ctx.Ignore)
	}
	if ctx.Cache != false {
		t.Error("cache flag = true, want false")
	}
	if ctx.Parent != parent {
		t.Error("parent ref missing")
	}
	if len(ctx.Entrypoint) != 1 || ctx.Entrypoint[0] != "/bin/hello" || ctx.User != "1000" {
		t.Error("run config not applied")
	}
	if ctx.FrontendType() != "chisel.v0" {
		t.Errorf("frontend type = %q", ctx.FrontendType())
	}
}

func TestChiselBuildContextAssemblyDefaults(t *testing.T) {
	dir := t.TempDir()
	const cutSpec = "release: ubuntu-24.04\nslices:\n  - base-files_base\n"
	if err := os.WriteFile(filepath.Join(dir, model.ChiselFileName), []byte(cutSpec), 0o600); err != nil {
		t.Fatal(err)
	}
	spec, err := parseChiselCutSpec(dir)
	if err != nil {
		t.Fatalf("parseChiselCutSpec failed: %v", err)
	}

	ctx, err := chiselBuildContext(dir, spec, model.ChiselConfig{}, nil, nil)
	if err != nil {
		t.Fatalf("chiselBuildContext failed: %v", err)
	}
	if ctx.Release.Name != "ubuntu-24.04" {
		t.Errorf("release = %+v", ctx.Release)
	}
	if ctx.Parent != nil {
		t.Error("no parent expected")
	}
	if ctx.Cache != true {
		t.Error("cache must default on")
	}
}

func TestParseChiselCutSpecErrors(t *testing.T) {
	dir := t.TempDir()
	if _, err := parseChiselCutSpec(dir); err == nil {
		t.Error("expected a read error without chisel.yaml")
	}
	if err := os.WriteFile(filepath.Join(dir, model.ChiselFileName), []byte("bad"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := parseChiselCutSpec(dir); err == nil {
		t.Error("expected a parse error")
	}
}

func boolRef(b bool) *bool { return &b }
