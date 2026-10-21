package build_context

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/moby/buildkit/client/llb"
	pb "github.com/moby/buildkit/solver/pb"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
	"google.golang.org/protobuf/proto"
)

func TestNixBuildContext_FrontendType(t *testing.T) {
	nc := NixBuildContext{}
	if got := nc.FrontendType(); got != "nix.v0" {
		t.Errorf("FrontendType() = %q, want %q", got, "nix.v0")
	}
}

func TestNixBuildContext_FileName(t *testing.T) {
	nc := NixBuildContext{}
	if got := nc.FileName(); got != "flake.nix" {
		t.Errorf("FileName() = %q, want %q", got, "flake.nix")
	}
}

func TestNixBuildContext_ToLocalMounts(t *testing.T) {
	dir := t.TempDir()
	nc := NixBuildContext{Root: dir}

	mounts, err := nc.ToLocalMounts()
	if err != nil {
		t.Fatalf("ToLocalMounts failed: %v", err)
	}
	if _, ok := mounts["context"]; !ok {
		t.Fatalf("ToLocalMounts missing %q mount, got keys: %v", "context", mounts)
	}
}

func TestNixBuildContext_buildScript(t *testing.T) {
	nc := NixBuildContext{}
	got := nc.buildScript("x86_64-linux", false)
	want := "lockmode=''; [ -f /src/flake.lock ] && lockmode='--no-update-lock-file'\nnix --extra-experimental-features 'nix-command flakes' --option sandbox false --option filter-syscalls false build $lockmode --show-trace -L --out-link /tmp/hive-rootfs .#packages.x86_64-linux.default\ncp -RL --preserve=links /tmp/hive-rootfs/. /out/\nnix --extra-experimental-features 'nix-command flakes' copy --no-check-sigs --to /out /tmp/hive-rootfs"
	if got != want {
		t.Errorf("buildScript() =\n%q\nwant\n%q", got, want)
	}

	got = nc.buildScript("aarch64-linux", false)
	if !strings.Contains(got, ".#packages.aarch64-linux.default") {
		t.Errorf("buildScript() = %q, want the mapped system attr", got)
	}

	parentNC := NixBuildContext{Parent: &NixParent{RegistryRef: "example.com/base:1.0"}}
	got = parentNC.buildScript("x86_64-linux", true)
	if want := "if [ -d /parent/nix/store ]; then\n  mkdir -p /out/nix/store\n  cp -a /parent/nix/store/. /out/nix/store/\nfi\n"; !strings.Contains(got, want) {
		t.Errorf("buildScript() = %q, want the guarded parent store copy %q", got, want)
	}
	if want := "  cp -a \"$entry\" /out/\ndone\n"; !strings.Contains(got, want) {
		t.Errorf("buildScript() = %q, want the symlink preserving parent tree copy", got)
	}
	if strings.Contains(got, "|| true; done") {
		t.Errorf("buildScript() = %q, parent copy errors must not be swallowed", got)
	}

	cacheNC := NixBuildContext{StoreCache: true}
	got = cacheNC.buildScript("x86_64-linux", false)
	if want := "install='.#packages.x86_64-linux.default'"; !strings.Contains(got, want) {
		t.Errorf("buildScript() = %q, want the installable variable %q", got, want)
	}
	if want := "nix --extra-experimental-features 'nix-command flakes' copy --no-check-sigs --all --from local?root=/nix-cache || true\nnix"; !strings.Contains(got, want) {
		t.Errorf("buildScript() = %q, want the store import before the build", got)
	}
	if want := "\n[ -z \"$drv\" ] || nix --extra-experimental-features 'nix-command flakes' copy --no-check-sigs --to local?root=/nix-cache \"$drv\" || true"; !strings.Contains(got, want) {
		t.Errorf("buildScript() = %q, want the guarded store export after the copy", got)
	}
	if want := "drv=\"$(nix --extra-experimental-features 'nix-command flakes' eval --raw $lockmode \"$install.drvPath\")\" || drv=''\n"; !strings.Contains(got, want) {
		t.Errorf("buildScript() = %q, want the best effort eval %q", got, want)
	}

	argsNC := NixBuildContext{StoreCache: true, ExtraArgs: []string{"--impure", "it's"}}
	got = argsNC.buildScript("x86_64-linux", false)
	if want := "eval --raw $lockmode \"$install.drvPath\" '--impure' 'it'\\''s')\" || drv=''\n"; !strings.Contains(got, want) {
		t.Errorf("buildScript() = %q, want the eval to carry the quoted extra args %q", got, want)
	}
}

func TestPinImageRef(t *testing.T) {
	const dgst = "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	tests := []struct {
		name      string
		canonical string
		want      string
	}{
		{"tagged", "docker.io/library/ubuntu:22.04", "docker.io/library/ubuntu:22.04@" + dgst},
		{"untagged", "docker.io/library/ubuntu", "docker.io/library/ubuntu@" + dgst},
		{"digest pinned", "docker.io/library/ubuntu@" + dgst, "docker.io/library/ubuntu@" + dgst},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pinImageRef(tt.canonical, dgst); got != tt.want {
				t.Errorf("pinImageRef(%q) = %q, want %q", tt.canonical, got, tt.want)
			}
		})
	}
}

func TestNixBuildContext_resolveParentStateInvalidRef(t *testing.T) {
	nc := NixBuildContext{Parent: &NixParent{RegistryRef: "Not A Ref"}}
	_, _, err := nc.resolveParentState(context.Background(), nil, ocispecs.Platform{OS: "linux", Architecture: "amd64"})
	if err == nil {
		t.Fatal("resolveParentState() error = nil, want an invalid parent reference error")
	}
	if want := `invalid nix parent image reference "Not A Ref"`; !strings.Contains(err.Error(), want) {
		t.Errorf("resolveParentState() error = %q, want it to contain %q", err, want)
	}
	if strings.Contains(err.Error(), "builder") {
		t.Errorf("resolveParentState() error = %q, must not mention the builder image", err)
	}
}

func TestNixBuildContext_buildScriptQuotesExtraArgs(t *testing.T) {
	nc := NixBuildContext{ExtraArgs: []string{"--option", "extra substituters", "it's;$HOME"}}
	got := nc.buildScript("x86_64-linux", false)
	want := ".#packages.x86_64-linux.default '--option' 'extra substituters' 'it'\\''s;$HOME'\ncp"
	if !strings.Contains(got, want) {
		t.Errorf("buildScript() = %q, want it to contain %q", got, want)
	}
}

func TestParentOverlayScript(t *testing.T) {
	if out, err := exec.Command("cp", "--version").Output(); err != nil || !strings.Contains(string(out), "GNU") {
		t.Skip("the overlay script needs GNU cp, like the nix builder image")
	}

	base := t.TempDir()
	parent := filepath.Join(base, "parent")
	rootfs := filepath.Join(base, "rootfs")
	out := filepath.Join(base, "out")
	for _, dir := range []string{
		filepath.Join(parent, "usr", "bin"),
		filepath.Join(parent, "etc"),
		filepath.Join(rootfs, "bin"),
		out,
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{
		filepath.Join(parent, "usr", "bin", "sh"): "parent-sh",
		filepath.Join(rootfs, "bin", "tool"):      "flake-tool",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// A merged /usr link, plus an absolute link that dangles on the builder.
	if err := os.Symlink("usr/bin", filepath.Join(parent, "bin")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/etc/alternatives/awk", filepath.Join(parent, "etc", "awk")); err != nil {
		t.Fatal(err)
	}

	script := "set -eu\n" + parentOverlayScript(parent, rootfs, out)
	if combined, err := exec.Command("sh", "-c", script).CombinedOutput(); err != nil {
		t.Fatalf("script failed: %v\n%s", err, combined)
	}

	if target, err := os.Readlink(filepath.Join(out, "bin")); err != nil || target != "usr/bin" {
		t.Errorf("bin = %q (err %v), want the parent symlink kept", target, err)
	}
	if target, err := os.Readlink(filepath.Join(out, "etc", "awk")); err != nil || target != "/etc/alternatives/awk" {
		t.Errorf("etc/awk = %q (err %v), want the absolute symlink kept as is", target, err)
	}
	for path, want := range map[string]string{
		filepath.Join(out, "usr", "bin", "sh"):   "parent-sh",
		filepath.Join(out, "usr", "bin", "tool"): "flake-tool",
	} {
		got, err := os.ReadFile(path)
		if err != nil || string(got) != want {
			t.Errorf("%s = %q (err %v), want %q", path, got, err, want)
		}
	}
}

func TestNixSystemName(t *testing.T) {
	cases := []struct {
		plat  ocispecs.Platform
		want  string
		error bool
	}{
		{plat: ocispecs.Platform{OS: "linux", Architecture: "amd64"}, want: "x86_64-linux"},
		{plat: ocispecs.Platform{OS: "linux", Architecture: "arm64"}, want: "aarch64-linux"},
		{plat: ocispecs.Platform{OS: "linux", Architecture: "386"}, want: "i686-linux"},
		{plat: ocispecs.Platform{OS: "linux", Architecture: "riscv64"}, want: "riscv64-linux"},
		{plat: ocispecs.Platform{OS: "linux", Architecture: "ppc64le"}, want: "powerpc64le-linux"},
		{plat: ocispecs.Platform{OS: "linux", Architecture: "arm"}, want: "armv7l-linux"},
		{plat: ocispecs.Platform{OS: "linux", Architecture: "arm", Variant: "v7a"}, want: "armv7l-linux"},
		{plat: ocispecs.Platform{OS: "linux", Architecture: "arm", Variant: "v6"}, want: "armv6l-linux"},
		{plat: ocispecs.Platform{OS: "darwin", Architecture: "amd64"}, error: true},
		{plat: ocispecs.Platform{OS: "linux", Architecture: "s390x"}, error: true},
	}
	for _, tc := range cases {
		got, err := nixSystemName(tc.plat)
		if tc.error {
			if err == nil {
				t.Errorf("nixSystemName(%v) = %q, want an error", tc.plat, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("nixSystemName(%v) failed: %v", tc.plat, err)
		}
		if got != tc.want {
			t.Errorf("nixSystemName(%v) = %q, want %q", tc.plat, got, tc.want)
		}
	}
}

func TestNixBuildContext_imageConfig(t *testing.T) {
	nc := NixBuildContext{
		Entrypoint: []string{"/bin/sh"},
		Cmd:        []string{"-c"},
		Env:        map[string]string{"B": "2", "A": "1"},
		User:       "1000",
		WorkingDir: "/workspace",
	}

	data, err := nc.imageConfig(ocispecs.Platform{
		OS:           "linux",
		Architecture: "arm64",
	}, map[string]string{
		"label:org.opencontainers.image.title": "hive",
		"platform":                             "linux/arm64",
	})
	if err != nil {
		t.Fatalf("imageConfig failed: %v", err)
	}

	var img ocispecs.Image
	if err := json.Unmarshal(data, &img); err != nil {
		t.Fatalf("imageConfig produce invalid JSON: %v", err)
	}

	if img.Architecture != "arm64" || img.OS != "linux" {
		t.Errorf("platform in config = %s/%s, want linux/arm64", img.Architecture, img.OS)
	}
	if want := []string{"A=1", "B=2"}; len(img.Config.Env) != len(want) || img.Config.Env[0] != want[0] || img.Config.Env[1] != want[1] {
		t.Errorf("Env = %v, want %v (sorted)", img.Config.Env, want)
	}
	if len(img.Config.Entrypoint) != 1 || img.Config.Entrypoint[0] != "/bin/sh" {
		t.Errorf("Entrypoint = %v, want [/bin/sh]", img.Config.Entrypoint)
	}
	if img.Config.User != "1000" || img.Config.WorkingDir != "/workspace" {
		t.Errorf("User/WorkingDir = %q/%q, want 1000//workspace", img.Config.User, img.Config.WorkingDir)
	}
	if img.Config.Labels["org.opencontainers.image.title"] != "hive" {
		t.Errorf("Labels = %v, want the title label", img.Config.Labels)
	}
}

func TestNixBuildContext_buildGraphMarshal(t *testing.T) {
	dir := t.TempDir()
	flake := filepath.Join(dir, "flake.nix")
	if err := os.WriteFile(flake, []byte("{ }\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	nc := NixBuildContext{Root: dir}
	state := nc.buildGraph(
		ocispecs.Platform{OS: "linux", Architecture: "amd64"},
		"x86_64-linux",
		[]string{"PATH=/nix/var/nix/profiles/default/bin:/usr/bin"},
		"nixos/nix@sha256:b75cca259b5e7c1f8a5a36ff9d0a29c10bb67a4b2e96e08caaffb7b77a99c352",
		nil, false,
	)

	if _, err := state.Marshal(context.Background()); err != nil {
		t.Fatalf("buildGraph state marshal failed: %v", err)
	}
}

func TestNixBuildContext_buildGraphExcludesBuildOutputs(t *testing.T) {
	dir := t.TempDir()
	flake := filepath.Join(dir, "flake.nix")
	if err := os.WriteFile(flake, []byte("{ }\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	nc := NixBuildContext{Root: dir}
	state := nc.buildGraph(
		ocispecs.Platform{OS: "linux", Architecture: "amd64"},
		"x86_64-linux",
		[]string{"PATH=/nix/var/nix/profiles/default/bin:/usr/bin"},
		"nixos/nix@sha256:b75cca259b5e7c1f8a5a36ff9d0a29c10bb67a4b2e96e08caaffb7b77a99c352",
		nil, false,
	)
	def, err := state.Marshal(context.Background())
	if err != nil {
		t.Fatalf("buildGraph state marshal failed: %v", err)
	}

	// Builds write the OCI image tar into the tag directory, which is also
	// the context root. The mount must exclude it, so the tar does not enter
	// the exec op cache key and invalidate the cache on every build.
	excluded := false
	for _, dt := range def.ToPB().Def {
		var op pb.Op
		if err := proto.Unmarshal(dt, &op); err != nil {
			t.Fatalf("failed to unmarshal llb op: %v", err)
		}
		source := op.GetSource()
		if source == nil || !strings.HasPrefix(source.Identifier, "local://") {
			continue
		}
		for _, value := range source.Attrs {
			if strings.Contains(value, "*/image.tar") {
				excluded = true
			}
		}
	}
	if !excluded {
		t.Error("the flake context mount does not exclude */image.tar; build outputs would invalidate the cache key on every build")
	}
}

func TestNixBuildContext_buildGraphParentMount(t *testing.T) {
	dir := t.TempDir()
	flake := filepath.Join(dir, "flake.nix")
	if err := os.WriteFile(flake, []byte("{ }\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	parent := llb.Image("example.com/base:1.0")
	nc := NixBuildContext{Root: dir}
	state := nc.buildGraph(
		ocispecs.Platform{OS: "linux", Architecture: "amd64"},
		"x86_64-linux",
		nil,
		"nixos/nix@sha256:b75cca259b5e7c1f8a5a36ff9d0a29c10bb67a4b2e96e08caaffb7b77a99c352",
		&parent, true,
	)
	if !hasMount(t, state, parentMountPath) {
		t.Errorf("buildGraph has no %s mount; the parent rootfs would not reach the build", parentMountPath)
	}
}

func TestNixBuildContext_buildGraphStoreCacheMount(t *testing.T) {
	dir := t.TempDir()
	flake := filepath.Join(dir, "flake.nix")
	if err := os.WriteFile(flake, []byte("{ }\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	nc := NixBuildContext{Root: dir, StoreCache: true}
	state := nc.buildGraph(
		ocispecs.Platform{OS: "linux", Architecture: "amd64"},
		"x86_64-linux",
		nil,
		"nixos/nix@sha256:b75cca259b5e7c1f8a5a36ff9d0a29c10bb67a4b2e96e08caaffb7b77a99c352",
		nil, false,
	)
	mount, found := findMount(t, state, nixStoreCachePath)
	if !found {
		t.Fatalf("buildGraph has no %s mount; the nix store cache would not persist", nixStoreCachePath)
	}
	if mount.MountType != pb.MountType_CACHE {
		t.Errorf("mount %s type = %d, want a cache mount (%d)", nixStoreCachePath, mount.MountType, pb.MountType_CACHE)
	}
	if mount.CacheOpt == nil || mount.CacheOpt.ID != nixStoreCacheKey {
		t.Errorf("mount %s cache id = %v, want %q", nixStoreCachePath, mount.CacheOpt, nixStoreCacheKey)
	}
	if mount.CacheOpt != nil && mount.CacheOpt.Sharing != pb.CacheSharingOpt_SHARED {
		t.Errorf("mount %s sharing = %d, want shared (%d)", nixStoreCachePath, mount.CacheOpt.Sharing, pb.CacheSharingOpt_SHARED)
	}
}

// hasMount reports whether the marshalled exec graph mounts dest.
func hasMount(t *testing.T, state llb.State, dest string) bool {
	_, found := findMount(t, state, dest)
	return found
}

// findMount returns the marshalled mount for dest.
func findMount(t *testing.T, state llb.State, dest string) (*pb.Mount, bool) {
	t.Helper()
	def, err := state.Marshal(context.Background())
	if err != nil {
		t.Fatalf("state marshal failed: %v", err)
	}
	for _, dt := range def.ToPB().Def {
		var op pb.Op
		if err := proto.Unmarshal(dt, &op); err != nil {
			t.Fatalf("failed to unmarshal llb op: %v", err)
		}
		exec := op.GetExec()
		if exec == nil {
			continue
		}
		for _, mount := range exec.Mounts {
			if mount.Dest == dest {
				return mount, true
			}
		}
	}
	return nil, false
}
