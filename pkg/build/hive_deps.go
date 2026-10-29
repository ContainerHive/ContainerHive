package build

import (
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"

	"github.com/ContainerHive/ContainerHive/internal/buildkit/build_context"
	"github.com/ContainerHive/ContainerHive/internal/dependency"
	"github.com/ContainerHive/ContainerHive/internal/ocistore"
	"github.com/ContainerHive/ContainerHive/pkg/model"
	"github.com/ContainerHive/ContainerHive/pkg/platform"
	"github.com/containerd/containerd/v2/core/content"
	"github.com/distribution/reference"
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
	// Project, when set, is used to reject a dependency that the project
	// does not build for PlatformStr, so a dependent fails with a clear error
	// instead of a missing named context or registry pull failure.
	Project *model.ContainerHiveProject
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
	if err := checkDepPlatform(opts, imageName, tag); err != nil {
		return err
	}

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

// checkDepPlatform returns an error when the project defines the dependency
// but does not configure it for opts.PlatformStr. Dependencies the project
// does not define are left to the regular tar/registry resolution.
func checkDepPlatform(opts HiveDepsOpts, imageName, tag string) error {
	if opts.Project == nil || opts.PlatformStr == "" {
		return nil
	}
	platforms, found := tagPlatforms(opts.Project, imageName, tag)
	if !found || slices.Contains(platforms, platform.Normalize(opts.PlatformStr)) {
		return nil
	}
	return fmt.Errorf("dependency %s:%s is not configured for platform %s (configured: %v), add the platform to its image.yml or remove it from the dependent image",
		imageName, tag, platform.Normalize(opts.PlatformStr), platforms)
}

// tagPlatforms returns the configured platforms of a base or variant tag.
// found is false when the project does not define the tag.
func tagPlatforms(project *model.ContainerHiveProject, imageName, tag string) (platforms []string, found bool) {
	for _, img := range project.ImagesByName[imageName] {
		for tagName := range img.Tags {
			if tagName == tag {
				return platform.Resolve(nil, project.Config.Platforms, img.Platforms, nil), true
			}
			for _, variant := range img.Variants {
				if tagName+variant.TagSuffix == tag {
					return platform.Resolve(nil, project.Config.Platforms, img.Platforms, variant.Platforms), true
				}
			}
		}
	}
	return nil, false
}

// ParentDeps couples the resolved parent image source with its cleanup
// handle. Deps is nil for plain image references.
type ParentDeps struct {
	Parent *build_context.ParentImage
	Deps   *HiveDeps
}

// ResolveParentImage resolves a backend parent image reference. A __hive__/
// reference resolves through the local build cache or the registry, like a
// Dockerfile dependency. Any other value is a plain image reference for the
// registry path.
func ResolveParentImage(parentRef string, opts HiveDepsOpts) (*ParentDeps, error) {
	if !strings.HasPrefix(parentRef, dependency.HivePrefix) {
		return &ParentDeps{
			Parent: &build_context.ParentImage{RegistryRef: parentRef},
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

	parent := &build_context.ParentImage{}
	for _, contextValue := range d.NamedContexts {
		if storeID, digest, found := parseOCILayoutRef(contextValue); found {
			named, err := reference.ParseNormalizedNamed(storeID)
			if err != nil {
				d.Cleanup()
				return nil, fmt.Errorf("normalizing parent store id %q: %w", storeID, err)
			}
			// The OCI source parser requires a host-qualified name: an
			// unqualified "store@sha256:..." puts the digest into the URL
			// authority and fails to parse.
			parent.OCIStoreID = storeID
			parent.OCIDigestRef = named.String() + "@" + digest
		}
		if strings.HasPrefix(contextValue, "docker-image://") {
			parent.RegistryRef = strings.TrimPrefix(contextValue, "docker-image://")
		}
	}
	return &ParentDeps{Parent: parent, Deps: d}, nil
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
