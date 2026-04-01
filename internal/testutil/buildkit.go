package testutil

import "github.com/timo-reymann/ContainerHive/internal/buildkit"

// BuildKitImage returns the moby/buildkit Docker image tag matching the version in go.mod.
func BuildKitImage() string {
	return "moby/buildkit:" + buildkit.Version
}
