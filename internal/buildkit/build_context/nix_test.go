package build_context

import (
	"context"
	"encoding/json"
	"os"
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
	want := "lockmode=''; [ -f /src/flake.lock ] && lockmode='--no-update-lock-file'\nnix --extra-experimental-features 'nix-command flakes' --option sandbox false --option filter-syscalls false build $lockmode --show-trace -L --out-link /tmp/hive-rootfs .#packages.x86_64-linux.default && cp -RL --preserve=links /tmp/hive-rootfs/. /out/ && nix --extra-experimental-features 'nix-command flakes' copy --no-check-sigs --to /out /tmp/hive-rootfs"
	if got != want {
		t.Errorf("buildScript() =\n%q\nwant\n%q", got, want)
	}

	got = nc.buildScript("aarch64-linux", false)
	if !strings.Contains(got, ".#packages.aarch64-linux.default") {
		t.Errorf("buildScript() = %q, want the mapped system attr", got)
	}

	parentNC := NixBuildContext{Parent: &NixParent{RegistryRef: "example.com/base:1.0"}}
	got = parentNC.buildScript("x86_64-linux", true)
	if want := "mkdir -p /out/nix/store && cp -RL --preserve=links /parent/nix/store/. /out/nix/store/"; !strings.Contains(got, want) {
		t.Errorf("buildScript() = %q, want the parent store copy %q", got, want)
	}
	if want := `do [ "$entry" != /parent/nix ] && [ -e "$entry" ] && cp -RL --preserve=links "$entry" /out/ || true; done && cp -RL --preserve=links /tmp/hive-rootfs/. /out/`; !strings.Contains(got, want) {
		t.Errorf("buildScript() = %q, want the parent tree copy without nix metadata", got)
	}

	cacheNC := NixBuildContext{StoreCache: true}
	got = cacheNC.buildScript("x86_64-linux", false)
	if want := "install='.#packages.x86_64-linux.default'"; !strings.Contains(got, want) {
		t.Errorf("buildScript() = %q, want the installable variable %q", got, want)
	}
	if want := "nix --extra-experimental-features 'nix-command flakes' copy --no-check-sigs --all --from local?root=/nix-cache || true\nnix"; !strings.Contains(got, want) {
		t.Errorf("buildScript() = %q, want the store import before the build", got)
	}
	if want := " && nix --extra-experimental-features 'nix-command flakes' copy --no-check-sigs --to local?root=/nix-cache \"$drv\" || true"; !strings.Contains(got, want) {
		t.Errorf("buildScript() = %q, want the store export after the copy", got)
	}
}

func TestNixBuildContext_buildScriptQuotesExtraArgs(t *testing.T) {
	nc := NixBuildContext{ExtraArgs: []string{"--option", "extra substituters", "it's;$HOME"}}
	got := nc.buildScript("x86_64-linux")
	want := ".#packages.x86_64-linux.default '--option' 'extra substituters' 'it'\\''s;$HOME' && cp"
	if !strings.Contains(got, want) {
		t.Errorf("buildScript() = %q, want it to contain %q", got, want)
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
