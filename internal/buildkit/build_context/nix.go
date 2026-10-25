package build_context

import (
	"context"
	"fmt"
	"strings"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/exporter/containerimage/exptypes"
	gatewayClient "github.com/moby/buildkit/frontend/gateway/client"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/tonistiigi/fsutil"
)

// flakeFileName is the build entry point name inside the context. It mirrors
// model.FlakeFileName, which internal packages must not import.
const flakeFileName = "flake.nix"

// DefaultNixBuilderImage provides nix when the image config does not set one.
const DefaultNixBuilderImage = "ghcr.io/containerhive/helper-images/nix:2.35"

// nixContextKey is the local mount name the flake source resolves under. It
// must match the key in ToLocalMounts.
const nixContextKey = "context"

// nixContextSharedKeyHint keeps the local flake context cacheable across
// solve sessions.
const nixContextSharedKeyHint = "hive-nix-flake"

// nixStoreCachePath is the internal mount for the persistent nix store
// closure cache.
const nixStoreCachePath = "/nix-cache"

// nixStoreCacheKey names the persistent cache mount the chroot store uses.
const nixStoreCacheKey = "hive-nix-store"

// NixBuildContext implements BuildContext for Nix flake builds. RunBuild
// executes `nix build` inside a builder image and uses the produced rootfs
// output as the image filesystem on a scratch base.
type NixBuildContext struct {
	// Root is the build context directory containing the flake.
	Root string
	// BuilderImage provides the nix binary for the exec op. Empty uses
	// DefaultNixBuilderImage.
	BuilderImage string
	// ExtraArgs appends to the nix build invocation.
	ExtraArgs []string
	// BuilderEnv maps environment variables applied to the nix build op.
	BuilderEnv map[string]string
	// Entrypoint, Cmd, Env, User and WorkingDir end up in the exported
	// image config.
	Entrypoint []string
	Cmd        []string
	Env        map[string]string
	User       string
	WorkingDir string
	// Parent mounts a parent image rootfs into the build. Nil builds on an
	// empty base.
	Parent *ParentImage
	// StoreCache keeps the nix store across builds in a persistent cache
	// mount.
	StoreCache bool
}

// resolvedBuilder carries the resolved builder image state inputs. It lives
// in shared.go for the entrypoint-less backends.

// RunBuild builds the LLB graph for the Nix flake build and solves it. The
// plugin options (platform, labels) are read from the gateway build options.
func (d NixBuildContext) RunBuild(ctx context.Context, client gatewayClient.Client) (*gatewayClient.Result, error) {
	opts := client.BuildOpts().Opts

	plat, err := parsePlatform(opts["platform"])
	if err != nil {
		return nil, err
	}

	system, err := nixSystemName(plat)
	if err != nil {
		return nil, err
	}

	build, err := resolveBuilderImage(ctx, client, plat, d.BuilderImage, DefaultNixBuilderImage)
	if err != nil {
		return nil, err
	}

	parentState, parentConfig, err := resolveParentState(ctx, client, plat, d.Parent)
	if err != nil {
		return nil, err
	}

	buildState := d.buildGraph(plat, system, build.env, build.digestRef, parentState, parentState != nil)
	def, err := buildState.Marshal(ctx, llb.Platform(plat))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal nix build: %w", err)
	}

	solved, err := client.Solve(ctx, gatewayClient.SolveRequest{Definition: def.ToPB()})
	if err != nil {
		return nil, fmt.Errorf("nix build failed: %w", err)
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

// buildGraph assembles the LLB state for the Nix flake build. The exec op
// runs `nix build` in the builder image; the getMount of the out mount
// becomes the image rootfs. A parent rootfs mounts at /parent and the flake
// output overlays it. StoreCache keeps the nix store in a persistent cache
// mount across builds.
func (d NixBuildContext) buildGraph(plat ocispecs.Platform, system string, builderEnv []string, digestRef string, parentState *llb.State, hasParent bool) llb.State {
	script := d.buildScript(system, hasParent)

	runState := builderStateFor(digestRef, builderEnv)
	for _, key := range sortedKeys(d.BuilderEnv) {
		runState = runState.With(llb.AddEnv(key, d.BuilderEnv[key]))
	}

	runOpts := []llb.RunOption{
		llb.Args([]string{"/bin/sh", "-euc", script}),
		llb.Dir(srcMountPath),
		// SharedKeyHint makes the local context cacheable across solve
		// sessions: without it, the exec op cache key embeds the one-shot
		// session id and every build re-runs the whole nix build.
		// ExcludePatterns keeps build outputs out of the cache key: builds
		// write the OCI image tar into the tag directory, which is also the
		// context root. The tar is a build product, never a flake input, so
		// an unfiltered tar invalidates the exec op key on every run.
		llb.AddMount(srcMountPath, llb.Local(nixContextKey,
			llb.SharedKeyHint(nixContextSharedKeyHint),
			llb.ExcludePatterns([]string{"*/image.tar"}),
		)),
		llb.AddMount(outMountPath, llb.Scratch()),
	}
	if hasParent {
		runOpts = append(runOpts, llb.AddMount(parentMountPath, *parentState))
	}
	if d.StoreCache {
		runOpts = append(runOpts, llb.AddMount(nixStoreCachePath, llb.Scratch(),
			llb.AsPersistentCacheDir(nixStoreCacheKey, llb.CacheMountShared)))
	}
	run := runState.Run(runOpts...)
	return run.GetMount(outMountPath)
}

// builderStateFor, exportImageConfig, parentOverlayScript, shellQuote and
// the small shared helpers live in shared.go for the entrypoint-less
// backends.

// buildScript assembles the shell script the builder exec op runs. nix stores
// the flake result as a symlink, so the script dereferences it and copies the
// rootfs directory into the out mount. With a parent, the script copies the
// parent rootfs into the out mount first, so the flake output overlays it.
// With StoreCache, the script warms the default store from a persistent
// cache mount before the build and saves the build closure after it.
func (d NixBuildContext) buildScript(system string, hasParent bool) string {
	b := &strings.Builder{}
	// Respect a committed flake.lock and fail when it no longer matches the
	// flake inputs. Without a lockfile the build resolves inputs freely, so
	// the lock stays optional.
	b.WriteString("lockmode=''; [ -f /src/flake.lock ] && lockmode='--no-update-lock-file'\n")
	gnix := func() { b.WriteString("nix --extra-experimental-features 'nix-command flakes'") }

	if d.StoreCache {
		// Store caching runs the build against the default store: a chroot
		// store cannot build with sandbox=false, because builders resolve
		// store paths against the image store root. The cache mount only
		// carries the closure: import its content before the build and save
		// the new closure after it. Both copies are best effort.
		b.WriteString("install='.#packages." + system + ".default'\n")
		// The eval is best effort and takes the same lock mode and extra args as
		// the build, so flakes needing e.g. --impure evaluate. A failed eval
		// leaves drv empty and skips the export instead of aborting under set -e.
		b.WriteString("drv=\"$(nix --extra-experimental-features 'nix-command flakes' eval --raw $lockmode \"$install.drvPath\"")
		d.writeExtraArgs(b)
		b.WriteString(")\" || drv=''\n")
		b.WriteString("nix --extra-experimental-features 'nix-command flakes' copy --no-check-sigs --all --from local?root=/nix-cache || true\n")
	}
	b.WriteString("nix --extra-experimental-features 'nix-command flakes' --option sandbox false")
	// Syscall filtering breaks under emulation (qemu syscall translation), a
	// known limitation. The build exec op already runs without the Nix chroot
	// sandbox, so the filter is the remaining hardening to drop.
	b.WriteString(" --option filter-syscalls false")
	b.WriteString(" build $lockmode --show-trace -L --out-link /tmp/hive-rootfs .")
	b.WriteString(d.installableAttr(system))
	d.writeExtraArgs(b)
	b.WriteString("\n")
	// Rootfs copy plus the full store closure: dereferenced symlinks point
	// into the nix store, and dynamically linked binaries need their
	// interpreter (ld.so) at the exact store path. Copy the closure of the
	// build output into the image under the same absolute paths so the
	// references resolve inside the container. A scheme-less destination is a
	// chroot store: nix unpacks the closure below /out/nix/store. Statements
	// stand on separate lines so set -e aborts on any failure; the best effort
	// cache export is the only one allowed to fail.
	if hasParent {
		b.WriteString(parentOverlayScript(parentMountPath, "/tmp/hive-rootfs", "/out", true))
	} else {
		b.WriteString("cp -RL --preserve=links /tmp/hive-rootfs/. /out/\n")
	}
	gnix()
	b.WriteString(" copy --no-check-sigs --to /out /tmp/hive-rootfs")
	if d.StoreCache {
		b.WriteString("\n[ -z \"$drv\" ] || ")
		gnix()
		b.WriteString(" copy --no-check-sigs --to local?root=" + nixStoreCachePath + " \"$drv\" || true")
	}
	return b.String()
}

// writeExtraArgs appends the shell quoted extra args, each preceded by a space.
func (d NixBuildContext) writeExtraArgs(b *strings.Builder) {
	for _, arg := range d.ExtraArgs {
		b.WriteString(" ")
		b.WriteString(shellQuote(arg))
	}
}

// installableAttr returns the flake attribute expression the build runs on,
// prefixed with # for combination with the flake path. The installable is a
// convention, never user configured: packages.<system>.default pinning the
// mapped target system, so the result does not depend on the system of the
// builder image.
func (d NixBuildContext) installableAttr(system string) string {
	return fmt.Sprintf("#packages.%s.default", system)
}

// nixSystemName maps a Docker target platform to a Nix system string. Only
// Linux platforms known to the Nix system set are supported. Clear failures
// matter: an unmapped platform otherwise builds for the builder system and
// would reach the registry under the wrong architecture.
func nixSystemName(plat ocispecs.Platform) (string, error) {
	if plat.OS != "linux" {
		return "", fmt.Errorf("the nix build backend only builds linux images, got platform %s/%s", plat.OS, plat.Architecture)
	}
	switch plat.Architecture {
	case "amd64":
		return "x86_64-linux", nil
	case "arm64":
		return "aarch64-linux", nil
	case "386":
		return "i686-linux", nil
	case "ppc64le":
		return "powerpc64le-linux", nil
	case "riscv64":
		return "riscv64-linux", nil
	case "arm":
		switch plat.Variant {
		case "", "v7", "v7a":
			return "armv7l-linux", nil
		case "v6", "v6l":
			return "armv6l-linux", nil
		}
	}
	return "", fmt.Errorf("no nix system mapping for platform linux/%s (variant %q); remove it from the image platforms", plat.Architecture, plat.Variant)
}

// imageConfig marshals the image config metadata for the export. It
// delegates to the shared exportImageConfig of the entrypoint-less backends.
func (d NixBuildContext) imageConfig(plat ocispecs.Platform, opts map[string]string, parentConfig *ocispecs.ImageConfig) ([]byte, error) {
	return exportImageConfig(plat, opts, parentConfig, runConfig{
		Entrypoint: d.Entrypoint,
		Cmd:        d.Cmd,
		Env:        d.Env,
		User:       d.User,
		WorkingDir: d.WorkingDir,
	})
}

// ToLocalMounts exposes the flake source directory as the local context.
func (d NixBuildContext) ToLocalMounts() (map[string]fsutil.FS, error) {
	ctxFs, err := fsutil.NewFS(d.Root)
	if err != nil {
		return nil, fmt.Errorf("invalid nix build context path %q: %w", d.Root, err)
	}
	return map[string]fsutil.FS{
		nixContextKey: ctxFs,
	}, nil
}

// FileName returns the build entry point name inside the context.
func (d NixBuildContext) FileName() string {
	return flakeFileName
}

// FrontendType returns the frontend identifier for nix flake builds.
func (d NixBuildContext) FrontendType() string {
	return "nix.v0"
}
