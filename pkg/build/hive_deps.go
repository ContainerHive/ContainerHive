package build

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/ContainerHive/ContainerHive/internal/buildkit/build_context"
	"github.com/ContainerHive/ContainerHive/internal/dependency"
	"github.com/ContainerHive/ContainerHive/internal/ocistore"
	"github.com/containerd/containerd/v2/core/content"
)

// namedContextPrefix replaces __hive__/ in the rewritten Dockerfile. It must be
// a valid Docker reference path component so that BuildKit's reference parser
// accepts it before named-context resolution kicks in.
const namedContextPrefix = "hive-dep/"

// HiveDeps holds the OCI layout stores and named context mappings needed to
// resolve inter-image dependencies via BuildKit named contexts instead of a
// registry.
type HiveDeps struct {
	OCIStores     map[string]content.Store
	NamedContexts map[string]string
	// Dockerfile is the path to a rewritten Dockerfile where __hive__/ has been
	// replaced with a valid Docker reference prefix for named context resolution.
	Dockerfile string
	cleanups   []func()
}

// Cleanup removes all temporary directories and the rewritten Dockerfile.
func (d *HiveDeps) Cleanup() {
	for _, fn := range d.cleanups {
		fn()
	}
}

// HiveDepsOpts configures how hive dependencies are resolved.
type HiveDepsOpts struct {
	DockerfilePath string
	DistPath       string
	PlatformStr    string
	// RegistryAddress is the registry address for CI fallback resolution.
	// When set and a local tar is not found, dependencies resolve to
	// docker-image:// references pointing at the registry manifest.
	RegistryAddress string
	// BuildID is appended to registry tags when resolving via registry.
	BuildID string
}

// ResolveHiveDeps scans a Dockerfile for __hive__/ references and resolves each
// to an OCI layout content store (local) or a docker-image:// registry
// reference (CI). It also creates a rewritten Dockerfile that uses a valid
// Docker reference prefix (hive-dep/) so BuildKit can parse it.
// Returns nil (not an error) when the Dockerfile has no hive dependencies.
func ResolveHiveDeps(opts HiveDepsOpts) (*HiveDeps, error) {
	refs, err := dependency.ScanDockerfileForHiveRefs(opts.DockerfilePath)
	if err != nil {
		return nil, fmt.Errorf("scanning Dockerfile for hive refs: %w", err)
	}

	if len(refs) == 0 {
		return nil, nil
	}

	d := &HiveDeps{
		OCIStores:     make(map[string]content.Store),
		NamedContexts: make(map[string]string),
	}

	// Rewrite __hive__/ to hive-dep/ so BuildKit's reference parser accepts it.
	rewritten, err := rewriteDockerfile(opts.DockerfilePath)
	if err != nil {
		return nil, fmt.Errorf("rewriting Dockerfile for named contexts: %w", err)
	}
	d.Dockerfile = rewritten
	d.cleanups = append(d.cleanups, func() { os.Remove(rewritten) })

	for _, ref := range refs {
		if err := resolveDepContext(d, ref.ImageName, ref.Tag, opts); err != nil {
			d.Cleanup()
			return nil, err
		}
	}

	return d, nil
}

// resolveDepContext resolves one hive dependency to a named context. A local
// tar becomes an OCI layout content store; without a tar, a configured
// registry provides the fallback reference.
func resolveDepContext(d *HiveDeps, imageName, tag string, opts HiveDepsOpts) error {
	contextKey := fmt.Sprintf("context:%s%s:%s", namedContextPrefix, imageName, tag)

	tarPath := TarFilePath(opts.DistPath, imageName, tag, opts.PlatformStr)
	if _, err := os.Stat(tarPath); err != nil {
		// Fall back to registry reference in CI
		if opts.RegistryAddress == "" {
			return fmt.Errorf("dependency %s:%s not built yet (expected %s) and no registry configured: %w", imageName, tag, tarPath, err)
		}

		registryTag := WithBuildID(tag, opts.BuildID)
		contextValue := fmt.Sprintf("docker-image://%s/%s:%s", opts.RegistryAddress, imageName, registryTag)
		d.NamedContexts[contextKey] = contextValue
		slog.Info("Resolved hive dep via registry", "image", imageName, "tag", tag, "ref", contextValue)
		return nil
	}

	tmpDir, err := os.MkdirTemp("", fmt.Sprintf("hive-oci-%s-%s-*", imageName, tag))
	if err != nil {
		return fmt.Errorf("creating temp dir for %s:%s: %w", imageName, tag, err)
	}
	d.cleanups = append(d.cleanups, func() { os.RemoveAll(tmpDir) })

	ols, err := ocistore.FromTar(tarPath, tmpDir)
	if err != nil {
		return fmt.Errorf("loading OCI layout for %s:%s: %w", imageName, tag, err)
	}

	storeID := fmt.Sprintf("hive-%s-%s", imageName, tag)
	contextValue := fmt.Sprintf("oci-layout:%s@%s", storeID, ols.Digest)

	d.OCIStores[storeID] = ols.Store
	d.NamedContexts[contextKey] = contextValue

	slog.Info("Resolved hive dep", "image", imageName, "tag", tag, "ref", contextValue)
	return nil
}

// NixParentDeps couples the resolved nix parent source with its cleanup
// handle. Deps is nil for plain image references.
type NixParentDeps struct {
	Parent *build_context.NixParent
	Deps   *HiveDeps
}

// ResolveNixParent resolves the nix parent image reference. A __hive__/
// reference resolves through the local build cache or the registry, like a
// Dockerfile dependency. Any other value is a plain image reference for the
// registry path.
func ResolveNixParent(parentRef string, opts HiveDepsOpts) (*NixParentDeps, error) {
	if !strings.HasPrefix(parentRef, dependency.HivePrefix) {
		return &NixParentDeps{
			Parent: &build_context.NixParent{RegistryRef: parentRef},
		}, nil
	}

	name, tag := splitHiveRef(strings.TrimPrefix(parentRef, dependency.HivePrefix))
	if name == "" || tag == "" {
		return nil, fmt.Errorf("invalid hive parent reference %q, want %q<name>:<tag>", parentRef, dependency.HivePrefix)
	}

	d := &HiveDeps{
		OCIStores:     make(map[string]content.Store),
		NamedContexts: make(map[string]string),
	}
	if err := resolveDepContext(d, name, tag, opts); err != nil {
		d.Cleanup()
		return nil, err
	}

	parent := &build_context.NixParent{}
	for _, contextValue := range d.NamedContexts {
		if storeID, digest, found := parseOCILayoutRef(contextValue); found {
			parent.OCIStoreID = storeID
			parent.OCIDigestRef = fmt.Sprintf("%s@%s", storeID, digest)
		}
		if strings.HasPrefix(contextValue, "docker-image://") {
			parent.RegistryRef = strings.TrimPrefix(contextValue, "docker-image://")
		}
	}
	return &NixParentDeps{Parent: parent, Deps: d}, nil
}

// splitHiveRef splits an image name and tag at the last colon.
func splitHiveRef(ref string) (string, string) {
	idx := strings.LastIndex(ref, ":")
	if idx < 0 {
		return ref, ""
	}
	return ref[:idx], ref[idx+1:]
}

// parseOCILayoutRef reads the store id and digest from an "oci-layout:<id>@<digest>"
// named context value.
func parseOCILayoutRef(value string) (string, string, bool) {
	rest, found := strings.CutPrefix(value, "oci-layout:")
	if !found {
		return "", "", false
	}
	id, digest, found := strings.Cut(rest, "@")
	if !found || id == "" || digest == "" {
		return "", "", false
	}
	return id, digest, true
}

// rewriteDockerfile creates a copy of the Dockerfile with __hive__/ replaced by
// the namedContextPrefix. The rewritten file is placed next to the original.
func rewriteDockerfile(dockerfilePath string) (string, error) {
	data, err := os.ReadFile(dockerfilePath)
	if err != nil {
		return "", err
	}

	replaced := strings.ReplaceAll(string(data), dependency.HivePrefix, namedContextPrefix)
	target := dockerfilePath + ".hive"
	if err := os.WriteFile(target, []byte(replaced), 0644); err != nil {
		return "", err
	}
	return target, nil
}
