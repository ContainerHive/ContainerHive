package build_context

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/containerd/platforms"
	"github.com/distribution/reference"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/client/llb/sourceresolver"
	"github.com/moby/buildkit/exporter/containerimage/exptypes"
	gatewayClient "github.com/moby/buildkit/frontend/gateway/client"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/tonistiigi/fsutil"
)

// flakeFileName is the build entry point name inside the context. It mirrors
// model.FlakeFileName, which internal packages must not import.
const flakeFileName = "flake.nix"

// DefaultNixBuilderImage provides nix when the image config does not set one.
const DefaultNixBuilderImage = "nixos/nix:2.31.6"

// nixContextKey is the local mount name the flake source resolves under. It
// must match the key in ToLocalMounts.
const nixContextKey = "context"

// outMountPath is the internal mount the nix build writes the rootfs into.
const outMountPath = "/out"

// srcMountPath is the internal mount the flake source resolves under inside
// the builder image.
const srcMountPath = "/src"

// nixContextSharedKeyHint keeps the local flake context cacheable across
// solve sessions.
const nixContextSharedKeyHint = "hive-nix-flake"

// parentMountPath is the internal mount the parent image rootfs resolves
// under inside the builder image.
const parentMountPath = "/parent"

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
	Parent *NixParent
	// StoreCache keeps the nix store across builds in a persistent cache
	// mount.
	StoreCache bool
}

// NixParent describes the resolved parent image source for a nix build.
// A local build resolves through an OCI layout store; the registry fallback
// resolves as a plain image reference.
type NixParent struct {
	// OCIStoreID is the SolveOpt.OCIStores id of the local parent tar.
	// Empty with OCIDigestRef set means the registry path applies.
	OCIStoreID string
	// OCIDigestRef is the "<store-id>@<digest>" reference inside the OCI
	// layout store.
	OCIDigestRef string
	// RegistryRef is the plain image reference for the registry path.
	RegistryRef string
}

// RunBuild builds the LLB graph for the Nix build and solves it. The plugin
// options (platform, labels) are read from the gateway build options.
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

	build, err := d.resolveBuilder(ctx, client, plat)
	if err != nil {
		return nil, err
	}

	parentState, parentConfig, err := d.resolveParentState(ctx, client, plat)
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

	config, err := d.imageConfig(plat, opts, parentConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal image config: %w", err)
	}
	solved.AddMeta(exptypes.ExporterImageConfigKey, config)
	return solved, nil
}

// resolvedBuilder carries the resolved builder image state inputs.
type resolvedBuilder struct {
	digestRef string
	env       []string
}

// resolveBuilder resolves the builder image config upfront and pins the
// builder image by digest, mirroring what the dockerfile frontend does for a
// `FROM` instruction. A plain image source does not carry its config env
// into executor steps, so the resolution happens client-side.
func (d NixBuildContext) resolveBuilder(ctx context.Context, client gatewayClient.Client, plat v1.Platform) (*resolvedBuilder, error) {
	ref := d.BuilderImage
	if ref == "" {
		ref = DefaultNixBuilderImage
	}
	named, err := canonicalImageRef(ref)
	if err != nil {
		return nil, fmt.Errorf("invalid builder image reference %q: %w", ref, err)
	}
	_, imageDigest, config, err := client.ResolveImageConfig(ctx, named, sourceresolver.Opt{
		ImageOpt: &sourceresolver.ResolveImageOpt{Platform: &plat},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to resolve builder image %s: %w", named, err)
	}

	var img v1.Image
	if err := json.Unmarshal(config, &img); err != nil {
		return nil, fmt.Errorf("failed to parse builder image config for %s: %w", named, err)
	}
	return &resolvedBuilder{
		digestRef: pinImageRef(named, imageDigest.String()),
		env:       img.Config.Env,
	}, nil
}

// resolveParentState resolves the parent image into an LLB state. A local
// build resolves the parent through the OCI layout store attached to the
// solve session; the registry path resolves as a plain image reference. The
// state applies the parent image config, matching BuildKit named contexts.
func (d NixBuildContext) resolveParentState(ctx context.Context, client gatewayClient.Client, plat v1.Platform) (*llb.State, *v1.ImageConfig, error) {
	if d.Parent == nil {
		return nil, nil, nil
	}

	if d.Parent.OCIStoreID != "" {
		sessionID := client.BuildOpts().SessionID
		_, _, config, err := client.ResolveImageConfig(ctx, d.Parent.OCIDigestRef, sourceresolver.Opt{
			OCILayoutOpt: &sourceresolver.ResolveOCILayoutOpt{
				Platform: &plat,
				Store: sourceresolver.ResolveImageConfigOptStore{
					SessionID: sessionID,
					StoreID:   d.Parent.OCIStoreID,
				},
			},
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to resolve parent image %s: %w", d.Parent.OCIDigestRef, err)
		}
		st := llb.OCILayout(d.Parent.OCIDigestRef,
			llb.OCIStore(sessionID, d.Parent.OCIStoreID),
			llb.Platform(plat),
		)
		var img v1.Image
		if err := json.Unmarshal(config, &img); err != nil {
			return nil, nil, fmt.Errorf("failed to parse parent image config for %s: %w", d.Parent.OCIDigestRef, err)
		}
		st, err = st.WithImageConfig(config)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to apply parent image config for %s: %w", d.Parent.OCIDigestRef, err)
		}
		return &st, &img.Config, nil
	}

	named, err := canonicalImageRef(d.Parent.RegistryRef)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid nix parent image reference %q: %w", d.Parent.RegistryRef, err)
	}
	_, imageDigest, config, err := client.ResolveImageConfig(ctx, named, sourceresolver.Opt{
		ImageOpt: &sourceresolver.ResolveImageOpt{Platform: &plat},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve parent image %s: %w", named, err)
	}
	var img v1.Image
	if err := json.Unmarshal(config, &img); err != nil {
		return nil, nil, fmt.Errorf("failed to parse parent image config for %s: %w", named, err)
	}
	digestRef := pinImageRef(named, imageDigest.String())
	st := llb.Image(digestRef)
	st, err = st.WithImageConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to apply parent image config for %s: %w", digestRef, err)
	}
	return &st, &img.Config, nil
}

// canonicalImageRef returns the fully qualified image reference. ResolveImage
// config on the daemon side does not normalize shorthands: a blob like
// "nixos/nix" reaches the resolver as registry host "nixos" and fails DNS.
func canonicalImageRef(ref string) (string, error) {
	named, err := reference.ParseNormalizedNamed(ref)
	if err != nil {
		return "", err
	}
	return reference.TagNameOnly(named).String(), nil
}

// pinImageRef appends the resolved digest to a canonical image reference. A
// reference that already carries a digest is returned unchanged: appending a
// second digest yields a reference llb.Image cannot parse.
func pinImageRef(canonical, imageDigest string) string {
	named, err := reference.ParseNormalizedNamed(canonical)
	if err != nil {
		return canonical + "@" + imageDigest
	}
	if _, ok := named.(reference.Digested); ok {
		return canonical
	}
	return canonical + "@" + imageDigest
}

// buildGraph assembles the LLB state for the Nix flake build. The exec op
// runs `nix build` in the builder image; the getMount of the out mount
// becomes the image rootfs. A parent rootfs mounts at /parent and the flake
// output overlays it. StoreCache keeps the nix store in a persistent cache
// mount across builds.
func (d NixBuildContext) buildGraph(plat v1.Platform, system string, builderEnv []string, digestRef string, parentState *llb.State, hasParent bool) llb.State {
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

// builderStateFor builds the llb image state for a digest pinned ref and
// applies the image config env.
func builderStateFor(digestRef string, env []string) llb.State {
	state := llb.Image(digestRef)
	for _, envEntry := range env {
		key, value, found := strings.Cut(envEntry, "=")
		if !found {
			continue
		}
		state = state.With(llb.AddEnv(key, value))
	}
	return state
}

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
		b.WriteString(parentOverlayScript(parentMountPath, "/tmp/hive-rootfs", "/out"))
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
func nixSystemName(plat v1.Platform) (string, error) {
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

// imageConfig marshals the image config metadata for the export. The value
// must be an OCI image document (architecture, os and config), which the
// containerimage exporter parses from the result metadata. Frontend label
// options build the exported label map, matching Dockerfile builds. A parent
// config backs the unset fields: the parent entrypoint, cmd, user and
// workingdir flow in, and the parent env entries merge with the config
// block on top.
func (d NixBuildContext) imageConfig(plat v1.Platform, opts map[string]string, parentConfig *v1.ImageConfig) ([]byte, error) {
	env := d.envSlice(parentConfig)
	labels := d.labelMap(opts)

	entrypoint := d.Entrypoint
	cmd := d.Cmd
	user := d.User
	workingDir := d.WorkingDir
	if parentConfig != nil {
		if entrypoint == nil {
			entrypoint = parentConfig.Entrypoint
		}
		if cmd == nil {
			cmd = parentConfig.Cmd
		}
		if user == "" {
			user = parentConfig.User
		}
		if workingDir == "" {
			workingDir = parentConfig.WorkingDir
		}
	}

	config := &v1.ImageConfig{
		User:       user,
		Env:        env,
		Entrypoint: entrypoint,
		Cmd:        cmd,
		WorkingDir: workingDir,
		Labels:     labels,
	}

	return json.Marshal(v1.Image{
		Platform: v1.Platform{
			Architecture: plat.Architecture,
			OS:           plat.OS,
			Variant:      plat.Variant,
		},
		Config: *config,
	})
}

// envSlice merges the parent environment entries with the config block.
// The config block wins per key.
func (d NixBuildContext) envSlice(parentConfig *v1.ImageConfig) []string {
	merged := map[string]string{}
	if parentConfig != nil {
		for _, entry := range parentConfig.Env {
			key, value, found := strings.Cut(entry, "=")
			if !found {
				continue
			}
			merged[key] = value
		}
	}
	for key, value := range d.Env {
		merged[key] = value
	}
	if len(merged) == 0 {
		return nil
	}
	env := make([]string, 0, len(merged))
	for _, key := range sortedKeys(merged) {
		env = append(env, key+"="+merged[key])
	}
	return env
}

func (d NixBuildContext) labelMap(opts map[string]string) map[string]string {
	if len(opts) == 0 {
		return nil
	}
	labels := map[string]string{}
	for key, value := range opts {
		if name, found := strings.CutPrefix(key, "label:"); found {
			labels[name] = value
		}
	}
	if len(labels) == 0 {
		return nil
	}
	return labels
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func parsePlatform(name string) (v1.Platform, error) {
	if name == "" {
		return platforms.DefaultSpec(), nil
	}
	plat, err := platforms.Parse(name)
	if err != nil {
		return v1.Platform{}, fmt.Errorf("invalid platform %q for nix build: %w", name, err)
	}
	return plat, nil
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

// shellQuote wraps arg in single quotes so the build script passes it to nix
// as one literal word.
func shellQuote(arg string) string {
	return "'" + strings.ReplaceAll(arg, "'", `'\''`) + "'"
}

// parentOverlayScript returns the shell statements that lay the parent rootfs
// into out and overlay the flake output (rootfs) on top of it.
//
// The parent keeps its symlinks (cp -a): dereferencing them would flatten
// merged /usr layouts, and an absolute link resolves against the builder
// filesystem instead of the parent. The parent nix store moves into
// out/nix/store when it has one, so a non nix parent works too. The rest of
// the parent nix tree stays out: a carried over store database and dangling
// gcroot symlinks corrupt the destination store, and the child closure copy
// brings the shared store paths anyway.
//
// The flake output dereferences its store symlinks. A top level directory of
// it that meets a parent symlink, like bin over a merged /usr layout, merges
// into the link target instead of failing on the type mismatch.
func parentOverlayScript(parent, rootfs, out string) string {
	script := `if [ -d PARENT/nix/store ]; then
  mkdir -p OUT/nix/store
  cp -a PARENT/nix/store/. OUT/nix/store/
fi
for entry in PARENT/* PARENT/.[!.]* PARENT/..?*; do
  [ "$entry" = PARENT/nix ] && continue
  [ -e "$entry" ] || [ -L "$entry" ] || continue
  cp -a "$entry" OUT/
done
for entry in ROOTFS/* ROOTFS/.[!.]* ROOTFS/..?*; do
  [ -e "$entry" ] || [ -L "$entry" ] || continue
  name="${entry##*/}"
  if [ -d "$entry" ] && [ -L "OUT/$name" ]; then
    link="$(readlink "OUT/$name")"
    case "$link" in
      /*) dest="OUT$link" ;;
      *) dest="OUT/$link" ;;
    esac
    mkdir -p "$dest"
    cp -RL --preserve=links "$entry/." "$dest/"
    continue
  fi
  cp -RL --preserve=links "$entry" OUT/
done
`
	return strings.NewReplacer("PARENT", parent, "ROOTFS", rootfs, "OUT", out).Replace(script)
}
