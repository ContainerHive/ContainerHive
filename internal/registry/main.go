package registry

import (
	"context"
	"os"
)

// Registry manages an OCI registry for staging local base images.
type Registry interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Address() string
	Push(ctx context.Context, imageName, tag, ociTarPath string) error
	IsLocal() bool
}

// NewRegistry creates a Registry based on the environment.
// In CI (CI env var set), it returns a remote registry passthrough.
// Otherwise, it returns an embedded zot registry for local builds.
// The dataDir parameter sets persistent storage for the local registry;
// if empty, a temporary directory is used.
func NewRegistry(dataDir string) Registry {
	if ci := os.Getenv("CI"); ci != "" {
		remoteAddr := os.Getenv("CONTAINER_HIVE_REGISTRY")
		if remoteAddr == "" {
			// TODO Use actual config value from tbd global configuration file
			remoteAddr = "docker.io"
		}
		return NewRemoteRegistry(remoteAddr)
	}
	return NewZotRegistry(dataDir)
}
