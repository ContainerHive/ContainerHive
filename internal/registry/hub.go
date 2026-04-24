package registry

import "strings"

// IsDockerHubAddress reports whether addr targets Docker Hub. Accepts the
// three canonical spellings (docker.io, index.docker.io, registry-1.docker.io)
// with or without a trailing repository path. Docker Hub's frontend rejects
// PUTs of pure OCI image indexes with an HTML 400; callers use this to decide
// whether to emit Docker-scheme media types instead.
func IsDockerHubAddress(addr string) bool {
	host := addr
	if i := strings.Index(addr, "/"); i >= 0 {
		host = addr[:i]
	}
	switch host {
	case "docker.io", "index.docker.io", "registry-1.docker.io":
		return true
	}
	return false
}

// resolveDockerMediaTypes applies the override-before-auto-detect rule shared
// by every Registry implementation: an explicit config value always wins;
// otherwise fall back to address-based Docker Hub detection.
func resolveDockerMediaTypes(address string, override *bool) bool {
	if override != nil {
		return *override
	}
	return IsDockerHubAddress(address)
}
