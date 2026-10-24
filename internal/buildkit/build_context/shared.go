package build_context

// Shared helpers for the entrypoint-less build backends (nix, chisel).
// Both run one exec op inside a digest-pinned source image and export the
// produced rootfs as the image filesystem, with the runtime image config
// assembled from the config block and the parent image.

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
	gatewayClient "github.com/moby/buildkit/frontend/gateway/client"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

// runConfig carries the runtime configuration an entrypoint-less backend

// Internal mount paths shared by the entrypoint-less backends.
const (
	// srcMountPath is the internal mount the build entrypoint source
	// resolves under inside the builder image.
	srcMountPath = "/src"
	// outMountPath is the internal mount the build writes the rootfs into.
	outMountPath = "/out"
	// parentMountPath is the internal mount the parent image rootfs
	// resolves under inside the builder image.
	parentMountPath = "/parent"
)

// ParentImage describes the resolved parent image source for a nix or
// chisel build. A local build resolves through an OCI layout store; the
// registry fallback resolves as a plain image reference.
type ParentImage struct {
	// OCIStoreID is the SolveOpt.OCIStores id of the local parent tar.
	// Empty with OCIDigestRef set means the registry path applies.
	OCIStoreID string
	// OCIDigestRef is the "<store-id>@<digest>" reference inside the OCI
	// layout store.
	OCIDigestRef string
	// RegistryRef is the plain image reference for the registry path.
	RegistryRef string
}

// runConfig carries the runtime configuration an entrypoint-less backend
// applies to the exported image config.
type runConfig struct {
	Entrypoint []string
	Cmd        []string
	Env        map[string]string
	User       string
	WorkingDir string
}

// resolvedBuilder carries the resolved builder image state inputs.
type resolvedBuilder struct {
	digestRef string
	env       []string
}

// resolveBuilderImage resolves a builder image config upfront and pins the
// builder image by digest, mirroring what the dockerfile frontend does for a
// `FROM` instruction. A plain image source does not carry its config env
// into executor steps, so the resolution happens client-side. An empty
// builderRef falls back to defaultRef.
func resolveBuilderImage(ctx context.Context, client gatewayClient.Client, plat ocispecs.Platform, builderRef, defaultRef string) (*resolvedBuilder, error) {
	ref := builderRef
	if ref == "" {
		ref = defaultRef
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

	var img ocispecs.Image
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
func resolveParentState(ctx context.Context, client gatewayClient.Client, plat ocispecs.Platform, parent *ParentImage) (*llb.State, *ocispecs.ImageConfig, error) {
	if parent == nil {
		return nil, nil, nil
	}

	if parent.OCIStoreID != "" {
		sessionID := client.BuildOpts().SessionID
		_, _, config, err := client.ResolveImageConfig(ctx, parent.OCIDigestRef, sourceresolver.Opt{
			OCILayoutOpt: &sourceresolver.ResolveOCILayoutOpt{
				Platform: &plat,
				Store: sourceresolver.ResolveImageConfigOptStore{
					SessionID: sessionID,
					StoreID:   parent.OCIStoreID,
				},
			},
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to resolve parent image %s: %w", parent.OCIDigestRef, err)
		}
		st := llb.OCILayout(parent.OCIDigestRef,
			llb.OCIStore(sessionID, parent.OCIStoreID),
			llb.Platform(plat),
		)
		var img ocispecs.Image
		if err := json.Unmarshal(config, &img); err != nil {
			return nil, nil, fmt.Errorf("failed to parse parent image config for %s: %w", parent.OCIDigestRef, err)
		}
		st, err = st.WithImageConfig(config)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to apply parent image config for %s: %w", parent.OCIDigestRef, err)
		}
		return &st, &img.Config, nil
	}

	named, err := canonicalImageRef(parent.RegistryRef)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid parent image reference %q: %w", parent.RegistryRef, err)
	}
	_, imageDigest, config, err := client.ResolveImageConfig(ctx, named, sourceresolver.Opt{
		ImageOpt: &sourceresolver.ResolveImageOpt{Platform: &plat},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve parent image %s: %w", named, err)
	}
	var img ocispecs.Image
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

// exportImageConfig marshals the image config metadata for the export. The
// value must be an OCI image document (architecture, os and config), which
// the containerimage exporter parses from the result metadata. Frontend
// label options build the exported label map, matching Dockerfile builds. A
// parent config backs the unset fields: the parent entrypoint, cmd, user and
// workingdir flow in, and the parent env entries merge with the config
// block on top.
func exportImageConfig(plat ocispecs.Platform, opts map[string]string, parentConfig *ocispecs.ImageConfig, run runConfig) ([]byte, error) {
	env := mergedEnv(parentConfig, run.Env)
	labels := labelMap(opts)

	entrypoint := run.Entrypoint
	cmd := run.Cmd
	user := run.User
	workingDir := run.WorkingDir
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

	config := &ocispecs.ImageConfig{
		User:       user,
		Env:        env,
		Entrypoint: entrypoint,
		Cmd:        cmd,
		WorkingDir: workingDir,
		Labels:     labels,
	}

	return json.Marshal(ocispecs.Image{
		Platform: ocispecs.Platform{
			Architecture: plat.Architecture,
			OS:           plat.OS,
			Variant:      plat.Variant,
		},
		Config: *config,
	})
}

// mergedEnv merges the parent environment entries with the config block.
// The config block wins per key.
func mergedEnv(parentConfig *ocispecs.ImageConfig, env map[string]string) []string {
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
	for key, value := range env {
		merged[key] = value
	}
	if len(merged) == 0 {
		return nil
	}
	mergedEnv := make([]string, 0, len(merged))
	for _, key := range sortedKeys(merged) {
		mergedEnv = append(mergedEnv, key+"="+merged[key])
	}
	return mergedEnv
}

func labelMap(opts map[string]string) map[string]string {
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

// parentOverlayScript returns the shell statements that lay the parent rootfs
// into out and overlay the build output (rootfs) on top of it.
//
// The parent keeps its symlinks (cp -a): dereferencing them would flatten
// merged /usr layouts, and an absolute link resolves against the builder
// filesystem instead of the parent. The parent nix store moves into
// out/nix/store when it has one, so a non nix parent works too. The rest of
// the parent nix tree stays out: a carried over store database and dangling
// gcroot symlinks corrupt the destination store, and the child closure copy
// brings the shared store paths anyway.
//
// The build output copy depends on the backend: nix dereferences its store
// symlinks (dereferenceRootfs), because the store paths must exist in the
// image under their absolute paths. A chisel cut keeps its symlinks, so its
// output copies verbatim. In both modes a top level directory of the output
// that meets a parent symlink, like bin over a merged /usr layout, merges
// into the link target instead of failing on the type mismatch.
func parentOverlayScript(parent, rootfs, out string, dereferenceRootfs bool) string {
	rootfsCopy := "cp -a"
	if dereferenceRootfs {
		rootfsCopy = "cp -RL --preserve=links"
	}
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
    CP_RULE "$entry/." "$dest/"
    continue
  fi
  CP_RULE "$entry" OUT/
done
`
	script = strings.ReplaceAll(script, "CP_RULE", rootfsCopy)
	return strings.NewReplacer("PARENT", parent, "ROOTFS", rootfs, "OUT", out).Replace(script)
}

func shellQuote(arg string) string {
	return "'" + strings.ReplaceAll(arg, "'", `'\''`) + "'"
}

// parsePlatform parses a Docker target platform name. An empty name is the
// daemon default.
func parsePlatform(name string) (ocispecs.Platform, error) {
	if name == "" {
		return platforms.DefaultSpec(), nil
	}
	plat, err := platforms.Parse(name)
	if err != nil {
		return ocispecs.Platform{}, fmt.Errorf("invalid platform %q for build: %w", name, err)
	}
	return plat, nil
}
