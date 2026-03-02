package build

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/containerd/containerd/v2/core/content"
	"github.com/timo-reymann/ContainerHive/internal/dependency"
	"github.com/timo-reymann/ContainerHive/internal/ocistore"
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

// ResolveHiveDeps scans a Dockerfile for __hive__/ references and resolves each
// to an OCI layout content store. It also creates a rewritten Dockerfile that
// uses a valid Docker reference prefix (hive-dep/) so BuildKit can parse it.
// Returns nil (not an error) when the Dockerfile has no hive dependencies.
func ResolveHiveDeps(dockerfilePath, distPath, platformStr string) (*HiveDeps, error) {
	refs, err := dependency.ScanDockerfileForHiveRefs(dockerfilePath)
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
	rewritten, err := rewriteDockerfile(dockerfilePath)
	if err != nil {
		return nil, fmt.Errorf("rewriting Dockerfile for named contexts: %w", err)
	}
	d.Dockerfile = rewritten
	d.cleanups = append(d.cleanups, func() { os.Remove(rewritten) })

	for _, ref := range refs {
		tarPath := TarFilePath(distPath, ref.ImageName, ref.Tag, platformStr)
		if _, err := os.Stat(tarPath); err != nil {
			d.Cleanup()
			return nil, fmt.Errorf("dependency %s:%s not built yet (expected %s): %w", ref.ImageName, ref.Tag, tarPath, err)
		}

		tmpDir, err := os.MkdirTemp("", fmt.Sprintf("hive-oci-%s-%s-*", ref.ImageName, ref.Tag))
		if err != nil {
			d.Cleanup()
			return nil, fmt.Errorf("creating temp dir for %s:%s: %w", ref.ImageName, ref.Tag, err)
		}
		d.cleanups = append(d.cleanups, func() { os.RemoveAll(tmpDir) })

		ols, err := ocistore.FromTar(tarPath, tmpDir)
		if err != nil {
			d.Cleanup()
			return nil, fmt.Errorf("loading OCI layout for %s:%s: %w", ref.ImageName, ref.Tag, err)
		}

		storeID := fmt.Sprintf("hive-%s-%s", ref.ImageName, ref.Tag)
		contextKey := fmt.Sprintf("context:%s%s:%s", namedContextPrefix, ref.ImageName, ref.Tag)
		contextValue := fmt.Sprintf("oci-layout:%s@%s", storeID, ols.Digest)

		d.OCIStores[storeID] = ols.Store
		d.NamedContexts[contextKey] = contextValue

		log.Printf("Resolved hive dep %s:%s -> %s", ref.ImageName, ref.Tag, contextValue)
	}

	return d, nil
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
