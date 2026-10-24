package build_context

// ChiselBuildContext implements BuildContext for chisel cut specs. RunBuild
// prepares a cached toolchain stage (chisel release binary plus git and
// ca-certificates) on top of the builder image and cuts the rootfs with
// `chisel cut`. The produced rootfs becomes the image filesystem on a
// scratch base. The chisel release checkout and the downloaded packages
// persist in a BuildKit cache mount across builds.

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/exporter/containerimage/exptypes"
	gatewayClient "github.com/moby/buildkit/frontend/gateway/client"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/tonistiigi/fsutil"
)

// chiselFileName is the build entry point name inside the context. It
// mirrors model.ChiselFileName, which internal packages must not import.
const chiselFileName = "chisel.yaml"

// DefaultChiselBuilderImage provides the base for the build stages. The
// build installs git, ca-certificates and the pinned chisel release on top,
// so the image only needs a shell and standard tools.
const DefaultChiselBuilderImage = "ubuntu:24.04"

// DefaultChiselVersion pins the chisel tool fetched for the build.
const DefaultChiselVersion = "v1.5.1"

// chiselReleaseURLFormat downloads a chisel release tarball from GitHub.
// The version appears twice: once in the release tag, once in the asset
// file name.
const chiselReleaseURLFormat = "https://github.com/canonical/chisel/releases/download/%s/chisel_%s_linux_%s.tar.gz"

// CanonicalChiselReleasesRepo carries the official chisel-releases service
// database. A branch name release clones this repository at the branch.
const CanonicalChiselReleasesRepo = "https://github.com/canonical/chisel-releases"

// chiselContextKey is the local mount name the cut spec resolves under. It
// must match the key in ToLocalMounts.
const chiselContextKey = "context"

// chiselContextSharedKeyHint keeps the local context cacheable across solve
// sessions.
const chiselContextSharedKeyHint = "hive-chisel-context"

// chiselCachePath is the internal mount for the persistent cache. It holds
// the git checkouts of the release repositories and the chisel package
// cache (chisel/ below the mount root via XDG_CACHE_HOME).
const chiselCachePath = "/chisel-cache"

// chiselCacheKey names the persistent cache mount.
const chiselCacheKey = "hive-chisel-cache"

// chiselReleaseCheckoutDirName is the internal directory for the git
// release checkouts inside the cache mount.
const chiselReleaseCheckoutDirName = "releases-git"

// chiselRootfsPath is the internal path of the freshly cut rootfs before
// the assembly into the out mount.
const chiselRootfsPath = "/tmp/hive-rootfs"

// ChiselRelease mirrors the release source forms of the chisel.yaml cut
// spec. Name and Git are mutually exclusive.
type ChiselRelease struct {
	// Name is the chisel-releases branch name, for example "ubuntu-24.04".
	Name string
	// Git is a git repository URL with a chisel release checkout.
	Git string
	// GitRef is the revision or branch to check out. Empty checks out the
	// default branch.
	GitRef string
}

// ChiselBuildContext implements BuildContext for chisel cut spec builds.
type ChiselBuildContext struct {
	// Root is the build context directory containing the cut spec.
	Root string
	// Release is the resolved release source from the cut spec.
	Release ChiselRelease
	// Slices lists the package slices to cut.
	Slices []string
	// Ignore allows unstable or unmaintained releases.
	Ignore []string
	// BuilderImage provides the base for the build stages. Empty uses
	// DefaultChiselBuilderImage.
	BuilderImage string
	// ChiselVersion pins the chisel tool version. Empty uses
	// DefaultChiselVersion.
	ChiselVersion string
	// ExtraArgs appends to the chisel cut invocation.
	ExtraArgs []string
	// Parent mounts a parent image rootfs into the build. Nil builds on an
	// empty base.
	Parent *ParentImage
	// Cache keeps the release checkout and the downloaded packages in a
	// persistent cache mount.
	Cache bool
	// Entrypoint, Cmd, Env, User and WorkingDir end up in the exported
	// image config.
	Entrypoint []string
	Cmd        []string
	Env        map[string]string
	User       string
	WorkingDir string
}

// Validate reports whether the build context carries a complete cut spec.
func (d ChiselBuildContext) Validate() error {
	if d.Release.Name == "" && d.Release.Git == "" {
		return fmt.Errorf("chisel build needs a release: set release in %s", chiselFileName)
	}
	if d.Release.Name != "" && d.Release.Git != "" {
		return fmt.Errorf("chisel release must set either a branch name or git, not both")
	}
	if len(d.Slices) == 0 {
		return fmt.Errorf("chisel build needs slices: list the package slices to cut in %s", chiselFileName)
	}
	for _, slice := range d.Slices {
		if slice == "" {
			return fmt.Errorf("chisel slices must not contain empty entries")
		}
	}
	return nil
}

// RunBuild builds the LLB graph for the chisel cut and solves it. The plugin
// options (platform, labels) are read from the gateway build options.
func (d ChiselBuildContext) RunBuild(ctx context.Context, client gatewayClient.Client) (*gatewayClient.Result, error) {
	opts := client.BuildOpts().Opts

	plat, err := parsePlatform(opts["platform"])
	if err != nil {
		return nil, err
	}

	toolchainArch, err := chiselToolchainArch(plat)
	if err != nil {
		return nil, err
	}
	packageArch, err := chiselPackageArch(plat)
	if err != nil {
		return nil, err
	}

	build, err := resolveBuilderImage(ctx, client, plat, d.BuilderImage, DefaultChiselBuilderImage)
	if err != nil {
		return nil, err
	}

	toolchain := d.toolchainState(plat, toolchainArch, build.digestRef, build.env)

	parentState, parentConfig, err := resolveParentState(ctx, client, plat, d.Parent)
	if err != nil {
		return nil, err
	}

	cutState := d.buildGraph(packageArch, toolchain, parentState, parentState != nil)
	def, err := cutState.Marshal(ctx, llb.Platform(plat))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal chisel build: %w", err)
	}

	solved, err := client.Solve(ctx, gatewayClient.SolveRequest{Definition: def.ToPB()})
	if err != nil {
		return nil, fmt.Errorf("chisel cut failed: %w", err)
	}

	config, err := exportImageConfig(plat, opts, parentConfig, runConfig{
		Entrypoint: d.Entrypoint,
		Cmd:        d.Cmd,
		Env:        d.Env,
		User:       d.User,
		WorkingDir: d.WorkingDir,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal image config: %w", err)
	}
	solved.AddMeta(exptypes.ExporterImageConfigKey, config)
	return solved, nil
}

// toolchainState assembles the cached toolchain stage: it installs git and
// ca-certificates into the builder image and extracts the pinned chisel
// release tarball into /usr/local/bin. BuildKit caches the stage on the
// builder image digest and the tarball content, so later builds skip the
// installs and the download.
func (d ChiselBuildContext) toolchainState(plat ocispecs.Platform, toolchainArch, digestRef string, env []string) llb.State {
	http := llb.HTTP(fmt.Sprintf(chiselReleaseURLFormat, d.version(), d.version(), toolchainArch),
		llb.Filename("chisel.tar.gz"),
		llb.Platform(plat),
	)
	script := "apt-get update" +
		" && apt-get install -y --no-install-recommends ca-certificates git" +
		" && tar -xzf /tmp/chisel/chisel.tar.gz -C /usr/local/bin/"
	return builderStateFor(digestRef, env).Run(
		llb.Args([]string{"/bin/sh", "-euc", script}),
		llb.AddMount("/tmp/chisel", http),
	).Root()
}

// version returns the pinned chisel tool version with a v prefix.
func (d ChiselBuildContext) version() string {
	version := strings.TrimSpace(d.ChiselVersion)
	if version == "" {
		version = DefaultChiselVersion
	}
	return "v" + strings.TrimPrefix(version, "v")
}

// buildGraph assembles the LLB state for the chisel cut. The cut exec op
// runs on the toolchain state and returns the out mount as the image
// rootfs. A parent rootfs mounts at /parent and the cut output overlays it.
func (d ChiselBuildContext) buildGraph(packageArch string, toolchain llb.State, parentState *llb.State, hasParent bool) llb.State {
	script := d.cutScript(packageArch, hasParent)

	runOpts := []llb.RunOption{
		llb.Args([]string{"/bin/sh", "-euc", script}),
		llb.Dir(srcMountPath),
		// SharedKeyHint makes the local context cacheable across solve
		// sessions: without it, the exec op cache key embeds the one-shot
		// session id and every build re-runs the whole cut.
		// ExcludePatterns keeps build outputs out of the cache key: builds
		// write the OCI image tar into the tag directory, which is also the
		// context root.
		llb.AddMount(srcMountPath, llb.Local(chiselContextKey,
			llb.SharedKeyHint(chiselContextSharedKeyHint),
			llb.ExcludePatterns([]string{"*/image.tar"}),
		)),
		llb.AddMount(outMountPath, llb.Scratch()),
	}
	if hasParent {
		runOpts = append(runOpts, llb.AddMount(parentMountPath, *parentState))
	}
	if d.Cache {
		// Locked, not shared: concurrent builds of the same release would
		// otherwise run git fetch and checkout in one worktree and hit ref
		// locks, or read it mid-checkout. The lock serializes chisel cuts on
		// one daemon.
		runOpts = append(runOpts, llb.AddMount(chiselCachePath, llb.Scratch(),
			llb.AsPersistentCacheDir(chiselCacheKey, llb.CacheMountLocked)))
	}
	return toolchain.Run(runOpts...).GetMount(outMountPath)
}

// cutScript assembles the shell script the cut exec op runs. It checks out
// the release, overlays custom slice definitions, cuts the rootfs, and
// assembles the out mount: parent rootfs first, cut output second, static
// rootfs files last. The release git checkout and the package downloads
// persist in the cache mount, so unchanged releases skip the checkout and
// the deb downloads.
func (d ChiselBuildContext) cutScript(packageArch string, hasParent bool) string {
	b := &strings.Builder{}

	if d.Cache {
		b.WriteString("export XDG_CACHE_HOME=" + chiselCachePath + "\n")
		b.WriteString("export GIT_TERMINAL_PROMPT=0\n")
	}

	repo, ref, dirName := d.releaseRepo(), d.releaseRef(), d.checkoutDir()
	// GIT checkout of the release: cached builds keep the checkout and fetch
	// incremental updates, so an unchanged release skips the fetch.
	if d.Cache {
		b.WriteString("release_dir=" + chiselCachePath + "/" + chiselReleaseCheckoutDirName + "/" + dirName + "\n")
		b.WriteString("if [ ! -d \"$release_dir/.git\" ]; then\n")
		b.WriteString("  git clone " + shellQuote(repo) + " \"$release_dir\"\n")
		b.WriteString("fi\n")
		b.WriteString("git -C \"$release_dir\" fetch --force --prune --tags origin\n")
	} else {
		b.WriteString("release_dir=/tmp/hive-release-src\n")
		b.WriteString("git clone " + shellQuote(repo) + " \"$release_dir\"\n")
	}
	if ref == "" {
		b.WriteString("git -C \"$release_dir\" checkout --force --detach origin/HEAD\n")
	} else {
		// A branch ref resolves against the fetched remote tracking branch;
		// tags and commits resolve directly. Git creates no local branches.
		b.WriteString("git -C \"$release_dir\" checkout --force --detach " + shellQuote("origin/"+ref) +
			" || git -C \"$release_dir\" checkout --force --detach " + shellQuote(ref) + "\n")
	}

	// Custom slice definitions overlay the release, so chisel resolves them
	// like release definitions.
	b.WriteString("if [ -d " + srcMountPath + "/slices ]; then\n")
	b.WriteString("  cp -a \"$release_dir\" /tmp/hive-release\n")
	b.WriteString("  cp -a " + srcMountPath + "/slices/. /tmp/hive-release/slices/\n")
	b.WriteString("  release_dir=/tmp/hive-release\n")
	b.WriteString("fi\n")

	b.WriteString("chisel cut --release \"$release_dir\" --root " + chiselRootfsPath)
	b.WriteString(" --arch " + shellQuote(packageArch))
	for _, cond := range d.Ignore {
		b.WriteString(" --ignore " + shellQuote(cond))
	}
	d.writeExtraArgs(b)
	for _, slice := range d.Slices {
		b.WriteString(" " + shellQuote(slice))
	}
	b.WriteString("\n")

	if hasParent {
		b.WriteString(parentOverlayScript(parentMountPath, chiselRootfsPath, outMountPath, false))
	} else {
		// cp -a keeps the symlinks chisel creates. Dereferencing them would
		// resolve absolute links against the builder filesystem and change
		// the image layout.
		b.WriteString("cp -a " + chiselRootfsPath + "/. " + outMountPath + "/\n")
	}
	// Static rootfs files overlay the cut output last.
	b.WriteString("if [ -d " + srcMountPath + "/rootfs ]; then cp -a " + srcMountPath + "/rootfs/. " + outMountPath + "/ ; fi\n")
	return b.String()
}

// writeExtraArgs appends the shell quoted extra args, each preceded by a space.
func (d ChiselBuildContext) writeExtraArgs(b *strings.Builder) {
	for _, arg := range d.ExtraArgs {
		b.WriteString(" ")
		b.WriteString(shellQuote(arg))
	}
}

// releaseRepo returns the git repository URL the release checkouts from: a
// user provided git URL, or the canonical chisel-releases repository for a
// branch name release.
func (d ChiselBuildContext) releaseRepo() string {
	if d.Release.Git != "" {
		return d.Release.Git
	}
	return CanonicalChiselReleasesRepo
}

// releaseRef returns the git revision or branch the build checks out: the
// configured git ref, or the chisel-releases branch name.
func (d ChiselBuildContext) releaseRef() string {
	if d.Release.Git != "" {
		return d.Release.GitRef
	}
	return d.Release.Name
}

// checkoutDir returns the cache mount directory name of the release
// checkout. The name hashes the repository URL and the ref: different
// repositories or refs never share a checkout, so concurrent builds of
// different releases do not race on one git worktree.
func (d ChiselBuildContext) checkoutDir() string {
	digest := sha1.Sum([]byte(d.releaseRepo() + "@" + d.releaseRef()))
	return hex.EncodeToString(digest[:])[:12]
}

// chiselToolchainArch maps a Docker target platform to the chisel release
// asset architecture. The cut exec op runs on the target platform, so the
// toolchain follows it. Chisel publishes no linux/386 tool binary.
func chiselToolchainArch(plat ocispecs.Platform) (string, error) {
	if plat.OS != "linux" {
		return "", fmt.Errorf("the chisel build backend only builds linux images, got platform %s/%s", plat.OS, plat.Architecture)
	}
	switch plat.Architecture {
	case "amd64", "arm64", "ppc64le", "riscv64", "s390x":
		return plat.Architecture, nil
	case "arm":
		switch plat.Variant {
		case "", "v7", "v7a":
			return "arm", nil
		}
	case "386":
		return "", fmt.Errorf("chisel publishes no linux/386 tool binary; remove linux/386 from the image platforms")
	}
	return "", fmt.Errorf("no chisel tool mapping for platform linux/%s (variant %q); remove it from the image platforms", plat.Architecture, plat.Variant)
}

// chiselPackageArch maps a Docker target platform to the Debian package
// architecture of the package slices. Unmapped platforms fail: a wrong
// architecture would reach the registry silently.
func chiselPackageArch(plat ocispecs.Platform) (string, error) {
	if plat.OS != "linux" {
		return "", fmt.Errorf("the chisel build backend only builds linux images, got platform %s/%s", plat.OS, plat.Architecture)
	}
	switch plat.Architecture {
	case "amd64":
		return "amd64", nil
	case "arm64":
		return "arm64", nil
	case "ppc64le":
		return "ppc64el", nil
	case "riscv64":
		return "riscv64", nil
	case "s390x":
		return "s390x", nil
	case "386":
		return "i386", nil
	case "arm":
		switch plat.Variant {
		case "", "v7", "v7a":
			return "armhf", nil
		}
	}
	return "", fmt.Errorf("no chisel package architecture mapping for platform linux/%s (variant %q); remove it from the image platforms", plat.Architecture, plat.Variant)
}

// ToLocalMounts exposes the cut spec directory as the local context.
func (d ChiselBuildContext) ToLocalMounts() (map[string]fsutil.FS, error) {
	ctxFs, err := fsutil.NewFS(d.Root)
	if err != nil {
		return nil, fmt.Errorf("invalid chisel build context path %q: %w", d.Root, err)
	}
	return map[string]fsutil.FS{
		chiselContextKey: ctxFs,
	}, nil
}

// FileName returns the build entry point name inside the context.
func (d ChiselBuildContext) FileName() string {
	return chiselFileName
}

// FrontendType returns the frontend identifier for chisel cut builds.
func (d ChiselBuildContext) FrontendType() string {
	return "chisel.v0"
}
