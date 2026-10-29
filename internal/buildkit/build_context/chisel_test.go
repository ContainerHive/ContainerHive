package build_context

import (
	"context"
	"strings"
	"testing"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/solver/pb"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestChiselBuildContext_FrontendType(t *testing.T) {
	ctx := ChiselBuildContext{}
	if got := ctx.FrontendType(); got != "chisel.v0" {
		t.Errorf("FrontendType() = %q, want %q", got, "chisel.v0")
	}
}

func TestChiselBuildContext_FileName(t *testing.T) {
	ctx := ChiselBuildContext{}
	if got := ctx.FileName(); got != "chisel.yaml" {
		t.Errorf("FileName() = %q, want %q", got, "chisel.yaml")
	}
}

func TestChiselBuildContext_ToLocalMounts(t *testing.T) {
	dir := t.TempDir()
	mounts, err := (ChiselBuildContext{Root: dir}).ToLocalMounts()
	if err != nil {
		t.Fatalf("ToLocalMounts() failed: %v", err)
	}
	if len(mounts) != 1 {
		t.Fatalf("ToLocalMounts() = %d mounts, want 1", len(mounts))
	}
	if _, found := mounts[chiselContextKey]; !found {
		t.Errorf("ToLocalMounts() has no %q mount", chiselContextKey)
	}
}

func TestChiselBuildContext_ToLocalMountsInvalidRoot(t *testing.T) {
	ctx := ChiselBuildContext{Root: "/nonexistent/hive-chisel"}
	if _, err := ctx.ToLocalMounts(); err == nil {
		t.Fatal("ToLocalMounts() for a missing root returned nil error")
	}
}

func TestChiselBuildContext_Validate(t *testing.T) {
	tests := map[string]struct {
		ctx     ChiselBuildContext
		wantErr bool
	}{
		"branch release":   {ctx: ChiselBuildContext{Release: ChiselRelease{Name: "ubuntu-24.04"}, Slices: []string{"base-files_base"}}},
		"git release":      {ctx: ChiselBuildContext{Release: ChiselRelease{Git: "https://example.com/chisel-releases.git"}, Slices: []string{"base-files_base"}}},
		"missing release":  {ctx: ChiselBuildContext{Slices: []string{"base-files_base"}}, wantErr: true},
		"both forms":       {ctx: ChiselBuildContext{Release: ChiselRelease{Name: "ubuntu-24.04", Git: "https://example.com/r.git"}, Slices: []string{"base-files_base"}}, wantErr: true},
		"missing slices":   {ctx: ChiselBuildContext{Release: ChiselRelease{Name: "ubuntu-24.04"}}, wantErr: true},
		"empty slice name": {ctx: ChiselBuildContext{Release: ChiselRelease{Name: "ubuntu-24.04"}, Slices: []string{""}}, wantErr: true},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := tc.ctx.Validate()
			if tc.wantErr && err == nil {
				t.Error("Validate() = nil, want error")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("Validate() = %v, want nil", err)
			}
		})
	}
}

func TestChiselToolchainArch(t *testing.T) {
	tests := map[string]struct {
		plat    ocispecs.Platform
		want    string
		wantErr bool
	}{
		"amd64":           {plat: ocispecs.Platform{OS: "linux", Architecture: "amd64"}, want: "amd64"},
		"arm64":           {plat: ocispecs.Platform{OS: "linux", Architecture: "arm64"}, want: "arm64"},
		"arm v7":          {plat: ocispecs.Platform{OS: "linux", Architecture: "arm", Variant: "v7"}, want: "arm"},
		"ppc64le":         {plat: ocispecs.Platform{OS: "linux", Architecture: "ppc64le"}, want: "ppc64le"},
		"riscv64":         {plat: ocispecs.Platform{OS: "linux", Architecture: "riscv64"}, want: "riscv64"},
		"s390x":           {plat: ocispecs.Platform{OS: "linux", Architecture: "s390x"}, want: "s390x"},
		"386 unsupported": {plat: ocispecs.Platform{OS: "linux", Architecture: "386"}, wantErr: true},
		"windows":         {plat: ocispecs.Platform{OS: "windows", Architecture: "amd64"}, wantErr: true},
		"arm v6 unknown":  {plat: ocispecs.Platform{OS: "linux", Architecture: "arm", Variant: "v6"}, wantErr: true},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := chiselToolchainArch(tc.plat)
			if tc.wantErr {
				if err == nil {
					t.Errorf("chiselToolchainArch() = %q, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("chiselToolchainArch() error: %v", err)
			}
			if got != tc.want {
				t.Errorf("chiselToolchainArch() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestChiselPackageArch(t *testing.T) {
	tests := map[string]struct {
		plat    ocispecs.Platform
		want    string
		wantErr bool
	}{
		"amd64":          {plat: ocispecs.Platform{OS: "linux", Architecture: "amd64"}, want: "amd64"},
		"arm64":          {plat: ocispecs.Platform{OS: "linux", Architecture: "arm64"}, want: "arm64"},
		"arm v7":         {plat: ocispecs.Platform{OS: "linux", Architecture: "arm", Variant: "v7"}, want: "armhf"},
		"386":            {plat: ocispecs.Platform{OS: "linux", Architecture: "386"}, want: "i386"},
		"ppc64le":        {plat: ocispecs.Platform{OS: "linux", Architecture: "ppc64le"}, want: "ppc64el"},
		"riscv64":        {plat: ocispecs.Platform{OS: "linux", Architecture: "riscv64"}, want: "riscv64"},
		"s390x":          {plat: ocispecs.Platform{OS: "linux", Architecture: "s390x"}, want: "s390x"},
		"darwin":         {plat: ocispecs.Platform{OS: "darwin", Architecture: "arm64"}, wantErr: true},
		"arm v6 unknown": {plat: ocispecs.Platform{OS: "linux", Architecture: "arm", Variant: "v6"}, wantErr: true},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := chiselPackageArch(tc.plat)
			if tc.wantErr {
				if err == nil {
					t.Errorf("chiselPackageArch() = %q, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("chiselPackageArch() error: %v", err)
			}
			if got != tc.want {
				t.Errorf("chiselPackageArch() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestChiselBuildContext_version(t *testing.T) {
	tests := map[string]string{
		"":       DefaultChiselVersion,
		"v1.5.1": "v1.5.1",
		"1.5.1":  "v1.5.1",
	}
	for in, want := range tests {
		t.Run("input '"+in+"'", func(t *testing.T) {
			if got := (ChiselBuildContext{ChiselVersion: in}).version(); got != want {
				t.Errorf("version() = %q, want %q", got, want)
			}
		})
	}
}

func TestChiselBuildContext_cutScript(t *testing.T) {
	binChisel := true
	_ = binChisel

	tests := []struct {
		name     string
		ctx      ChiselBuildContext
		expected []string
		missing  []string
	}{
		{
			name: "branch release with cache",
			ctx: ChiselBuildContext{
				Release: ChiselRelease{Name: "ubuntu-24.04"},
				Slices:  []string{"base-files_base", "libc6_libs"},
				Cache:   true,
			},
			expected: []string{
				"export XDG_CACHE_HOME=" + chiselCachePath,
				"git clone '" + CanonicalChiselReleasesRepo + "' \"$release_dir\"",
				"git -C \"$release_dir\" fetch --force --prune --tags origin",
				"git -C \"$release_dir\" checkout --force --detach 'origin/ubuntu-24.04'",
				"mkdir -p " + chiselRootfsPath,
				"chisel cut --release \"$release_dir\" --root " + chiselRootfsPath,
				"--arch 'amd64'",
				"'base-files_base' 'libc6_libs'",
				"cp -a " + chiselRootfsPath + "/. " + outMountPath + "/",
			},
		},
		{
			name: "git release with ref tracks a tag or commit",
			ctx: ChiselBuildContext{
				Release: ChiselRelease{Git: "https://example.com/chisel-releases.git", GitRef: "1.0"},
				Slices:  []string{"hello_bins"},
			},
			expected: []string{
				"git clone 'https://example.com/chisel-releases.git' \"$release_dir\"",
				"checkout --force --detach 'origin/1.0'",
				"checkout --force --detach '1.0'",
			},
		},
		{
			name: "cache disabled clones without fetch",
			ctx: ChiselBuildContext{
				Release: ChiselRelease{Name: "ubuntu-24.04"},
				Slices:  []string{"base-files_base"},
				Cache:   false,
			},
			missing: []string{
				"fetch --force",
				"XDG_CACHE_HOME",
				"releases-git",
			},
			expected: []string{"git clone '" + CanonicalChiselReleasesRepo + "' \"$release_dir\""},
		},
		{
			name: "ignore flags are repeated and quoted",
			ctx: ChiselBuildContext{
				Release: ChiselRelease{Name: "ubuntu-20.04"},
				Slices:  []string{"base-files_base"},
				Ignore:  []string{"unmaintained", "unstable"},
				Cache:   true,
			},
			expected: []string{"--ignore 'unmaintained' --ignore 'unstable'"},
		},
		{
			name: "extra args are quoted",
			ctx: ChiselBuildContext{
				Release:   ChiselRelease{Name: "ubuntu-24.04"},
				Slices:    []string{"base-files_base"},
				ExtraArgs: []string{"--option", "extra value", "it's;$HOME"},
				Cache:     true,
			},
			expected: []string{` '--option' 'extra value' 'it'\''s;$HOME'`},
		},
		{
			name: "custom slices overlay",
			ctx: ChiselBuildContext{
				Release: ChiselRelease{Name: "ubuntu-24.04"},
				Slices:  []string{"base-files_base"},
			},
			expected: []string{
				"if [ -d /src/slices ]; then",
				"cp -a \"$release_dir\" /tmp/hive-release",
				"cp -a /src/slices/. /tmp/hive-release/slices/",
				"release_dir=/tmp/hive-release",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.ctx.cutScript("amd64", false)
			for _, want := range tc.expected {
				if !strings.Contains(got, want) {
					t.Errorf("cutScript() misses %q:\n%s", want, got)
				}
			}
			for _, gone := range tc.missing {
				if strings.Contains(got, gone) {
					t.Errorf("cutScript() must not contain %q:\n%s", gone, got)
				}
			}
		})
	}
}

func TestChiselBuildContext_cutScriptParentOverlay(t *testing.T) {
	ctx := ChiselBuildContext{Release: ChiselRelease{Name: "ubuntu-24.04"}, Slices: []string{"base-files_base"}}
	got := ctx.cutScript("amd64", true)
	if want := parentOverlayScript(parentMountPath, chiselRootfsPath, outMountPath, false); !strings.Contains(got, want) {
		t.Errorf("cutScript() does not use the shared parent overlay script:\n%s", got)
	}
}

func TestChiselBuildContext_cutScriptRootfsOverlay(t *testing.T) {
	ctx := ChiselBuildContext{Release: ChiselRelease{Name: "ubuntu-24.04"}, Slices: []string{"base-files_base"}}
	got := ctx.cutScript("amd64", false)
	if !strings.Contains(got, "cp -a /src/rootfs/. "+outMountPath+"/") {
		t.Errorf("cutScript() misses the static rootfs overlay:\n%s", got)
	}
}

func TestChiselBuildContext_buildGraphMarshal(t *testing.T) {
	ctx := ChiselBuildContext{Release: ChiselRelease{Name: "ubuntu-24.04"}, Slices: []string{"base-files_base"}}
	state := ctx.buildGraph(
		"amd64",
		llb.Image("example.com/tool@sha256:b75cca259b5e7c1f8a5a36ff9d0a29c10bb67a4b2e96e08caaffb7b77a99c352"),
		nil, false,
	)
	if _, err := state.Marshal(context.Background()); err != nil {
		t.Fatalf("buildGraph state marshal failed: %v", err)
	}
}

func TestChiselBuildContext_buildGraphParentMount(t *testing.T) {
	ctx := ChiselBuildContext{Release: ChiselRelease{Name: "ubuntu-24.04"}, Slices: []string{"base-files_base"}}
	parent := llb.Image("example.com/base:1.0")
	state := ctx.buildGraph("amd64", llb.Image("example.com/tool@sha256:b75cca259b5e7c1f8a5a36ff9d0a29c10bb67a4b2e96e08caaffb7b77a99c352"), &parent, true)
	if !hasMount(t, state, parentMountPath) {
		t.Errorf("buildGraph has no %s mount; the parent rootfs would not reach the build", parentMountPath)
	}
}

func TestChiselBuildContext_buildGraphCacheMount(t *testing.T) {
	ctx := ChiselBuildContext{Release: ChiselRelease{Name: "ubuntu-24.04"}, Slices: []string{"base-files_base"}, Cache: true}
	state := ctx.buildGraph("amd64", llb.Image("example.com/tool@sha256:b75cca259b5e7c1f8a5a36ff9d0a29c10bb67a4b2e96e08caaffb7b77a99c352"), nil, false)
	mount, found := findMount(t, state, chiselCachePath)
	if !found {
		t.Fatalf("buildGraph has no %s mount; the chisel cache would not persist", chiselCachePath)
	}
	if mount.CacheOpt == nil {
		t.Fatal("cache mount has no cache options")
	}
	// Locked sharing serializes the git checkout: concurrent builds of the
	// same release must not fetch or check out in one worktree.
	if mount.CacheOpt.Sharing != pb.CacheSharingOpt_LOCKED {
		t.Errorf("cache sharing = %v, want LOCKED", mount.CacheOpt.Sharing)
	}
	if mount.CacheOpt.ID != chiselCacheKey {
		t.Errorf("cache ID = %q, want %q", mount.CacheOpt.ID, chiselCacheKey)
	}

	ctx.Cache = false
	state = ctx.buildGraph("amd64", llb.Image("example.com/tool@sha256:b75cca259b5e7c1f8a5a36ff9d0a29c10bb67a4b2e96e08caaffb7b77a99c352"), nil, false)
	if hasMount(t, state, chiselCachePath) {
		t.Errorf("buildGraph mounts %s with cache disabled", chiselCachePath)
	}
}

func TestChiselBuildContext_checkoutDirHashesRepoAndRef(t *testing.T) {
	canonical := ChiselBuildContext{Release: ChiselRelease{Name: "ubuntu-24.04"}}
	fork := ChiselBuildContext{Release: ChiselRelease{Git: "https://example.com/fork.git"}}
	otherBranch := ChiselBuildContext{Release: ChiselRelease{Name: "ubuntu-22.04"}}
	if canonical.checkoutDir() == fork.checkoutDir() {
		t.Error("different release repositories must not share a checkout dir")
	}
	if canonical.checkoutDir() == otherBranch.checkoutDir() {
		t.Error("different refs must not share a checkout dir")
	}
}

func TestChiselBuildContext_releaseForms(t *testing.T) {
	branch := ChiselBuildContext{Release: ChiselRelease{Name: "ubuntu-24.04"}}
	if got := branch.releaseRepo(); got != CanonicalChiselReleasesRepo {
		t.Errorf("releaseRepo() = %q, want the canonical repo", got)
	}
	if got := branch.releaseRef(); got != "ubuntu-24.04" {
		t.Errorf("releaseRef() = %q, want the branch name", got)
	}

	fork := ChiselBuildContext{Release: ChiselRelease{Git: "https://example.com/fork.git", GitRef: "custom"}}
	if got := fork.releaseRepo(); got != "https://example.com/fork.git" {
		t.Errorf("releaseRepo() = %q, want the configured git", got)
	}
	if got := fork.releaseRef(); got != "custom" {
		t.Errorf("releaseRef() = %q, want the configured git ref", got)
	}
}

func TestChiselBuildContext_cutScriptPreservesSymlinks(t *testing.T) {
	ctx := ChiselBuildContext{Release: ChiselRelease{Name: "ubuntu-24.04"}, Slices: []string{"base-files_base"}}

	// Chisel slice definitions can create symlinks. Dereferencing them would
	// resolve absolute links against the builder filesystem and flatten
	// merged-usr layouts.
	for name, hasParent := range map[string]bool{"no parent": false, "parent overlay": true} {
		t.Run(name, func(t *testing.T) {
			got := ctx.cutScript("amd64", hasParent)
			if strings.Contains(got, "cp -RL") {
				t.Errorf("cutScript() dereferences symlinks:\n%s", got)
			}
			if !strings.Contains(got, "cp -a ") {
				t.Errorf("cutScript() misses a verbatim copy:\n%s", got)
			}
		})
	}
}

func TestParentOverlayScriptRootfsDereferenceMode(t *testing.T) {
	deref := parentOverlayScript(parentMountPath, chiselRootfsPath, outMountPath, true)
	if !strings.Contains(deref, "cp -RL --preserve=links") {
		t.Errorf("dereference mode must resolve rootfs symlinks:\n%s", deref)
	}

	noDeref := parentOverlayScript(parentMountPath, chiselRootfsPath, outMountPath, false)
	if strings.Contains(noDeref, "cp -RL") {
		t.Errorf("no-dereference mode must keep rootfs symlinks:\n%s", noDeref)
	}
	// The parent always keeps its symlinks and the merge handling stays.
	if !strings.Contains(noDeref, `cp -a "$entry" /out/`) {
		t.Error("no-dereference mode lost the parent entry copy")
	}
	if !strings.Contains(noDeref, "readlink") {
		t.Error("no-dereference mode lost the parent symlink merge handling")
	}
}

func TestChiselBuildContext_needsChiselDownload(t *testing.T) {
	tests := []struct {
		name   string
		pinned string
		labels map[string]string
		want   bool
	}{
		{"no label downloads", "", nil, true},
		{"no label with pin downloads", "v1.5.1", nil, true},
		{"shipped chisel without pin is used", "", map[string]string{chiselVersionLabel: "1.5.1"}, false},
		{"shipped chisel matching pin is used", "v1.5.1", map[string]string{chiselVersionLabel: "1.5.1"}, false},
		{"pin without v prefix matches", "1.5.1", map[string]string{chiselVersionLabel: "v1.5.1"}, false},
		{"shipped chisel differing from pin downloads", "v1.4.2", map[string]string{chiselVersionLabel: "1.5.1"}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := (ChiselBuildContext{ChiselVersion: tc.pinned}).needsChiselDownload(tc.labels)
			if got != tc.want {
				t.Errorf("needsChiselDownload() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestToolchainScript(t *testing.T) {
	without := toolchainScript(false)
	with := toolchainScript(true)

	for _, want := range []string{"command -v git", "/etc/ssl/certs/ca-certificates.crt", "apt-get install"} {
		if !strings.Contains(without, want) {
			t.Errorf("script without download missing %q:\n%s", want, without)
		}
	}
	if strings.Contains(without, "tar -xzf") {
		t.Errorf("script without download must not extract chisel:\n%s", without)
	}
	if !strings.Contains(with, "tar -xzf /tmp/chisel/chisel.tar.gz") {
		t.Errorf("script with download must extract chisel:\n%s", with)
	}
	// apt-get only runs inside the missing-tools guard.
	if !strings.Contains(without, "if [ -n \"$missing\" ]; then\n  apt-get update") {
		t.Errorf("apt-get must be guarded by the missing check:\n%s", without)
	}
}

func TestChiselBuildContext_toolchainStateMarshal(t *testing.T) {
	plat := ocispecs.Platform{OS: "linux", Architecture: "amd64"}
	for name, labels := range map[string]map[string]string{
		"download":  nil,
		"preloaded": {chiselVersionLabel: "1.5.1"},
	} {
		t.Run(name, func(t *testing.T) {
			state := (ChiselBuildContext{}).toolchainState(plat, "amd64", &resolvedBuilder{
				digestRef: "docker.io/library/ubuntu@sha256:b75cca259b5e7c1f8a5a36ff9d0a29c10bb67a4b2e96e08caaffb7b77a99c352",
				labels:    labels,
			})
			if _, err := state.Marshal(context.Background(), llb.Platform(plat)); err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
		})
	}
}
