package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
)

// ComputeContentHash derives a stable, content-sensitive hash from a build
// unit's Dockerfile content, resolved build args, and target platform. It is
// used to scope the registry build cache so that any change to the build
// inputs invalidates the cache instead of silently reusing a stale entry
// under an unchanged name.tag.platform scope.
func ComputeContentHash(dockerfileContent []byte, buildArgs map[string]string, platformStr string) string {
	h := sha256.New()
	h.Write(dockerfileContent)

	keys := make([]string, 0, len(buildArgs))
	for k := range buildArgs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(h, "\x00%s=%s", k, buildArgs[k])
	}

	fmt.Fprintf(h, "\x00platform=%s", platformStr)

	return hex.EncodeToString(h.Sum(nil))[:16]
}
