package registry

import (
	"context"
	"fmt"
	"os"

	"github.com/timo-reymann/ContainerHive/pkg/model"
)

// Registry manages an OCI registry for staging local base images.
type Registry interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Address() string
	Push(ctx context.Context, imageName, tag, ociTarPath string) error
	IsLocal() bool
	UseDockerMediaTypes() bool
}

// NewRegistry creates a Registry based on the environment.
// In CI (CI env var set), it returns a remote registry passthrough.
// The registry address is resolved with the following precedence:
//  1. CONTAINER_HIVE_REGISTRY env var
//  2. registry.address from hive.yml (passed via registryConfig)
//
// Otherwise, it returns an embedded zot registry for local builds.
// The dataDir parameter sets persistent storage for the local registry;
// if empty, a temporary directory is used.
func NewRegistry(dataDir string, registryConfig *model.RegistryConfig) (Registry, error) {
	if ci := os.Getenv("CI"); ci != "" {
		remoteAddr := os.Getenv("CONTAINER_HIVE_REGISTRY")
		if remoteAddr == "" && registryConfig != nil && registryConfig.Address != "" {
			remoteAddr = registryConfig.Address
		}
		if remoteAddr == "" {
			return nil, fmt.Errorf("CI detected but no registry configured: set CONTAINER_HIVE_REGISTRY env var or registry.address in hive.yml")
		}
		var override *bool
		if registryConfig != nil {
			override = registryConfig.DockerMediaTypes
		}
		return NewRemoteRegistryWithMediaTypes(remoteAddr, override), nil
	}
	return NewZotRegistry(dataDir), nil
}
