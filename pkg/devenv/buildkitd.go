package devenv

import (
	"context"
	"io"

	"github.com/timo-reymann/ContainerHive/internal/buildkit"
	"github.com/timo-reymann/ContainerHive/internal/docker"
)

const (
	// BuildkitdContainerName is the well-known name used for the local buildkitd container.
	BuildkitdContainerName = "containerhive-buildkitd"
	// BuildkitdDefaultPort is the host port the buildkitd container binds to.
	BuildkitdDefaultPort = 8372
	// buildkitdContainerPort is the port buildkitd listens on inside the container.
	buildkitdContainerPort = 8372
)

// ResolveImage returns the full "image:tag" for the buildkitd container resolved from
// template_options (ci_buildkit_image, ci_buildkit_version), falling back to project defaults.
func ResolveImage(templateOpts map[string]string) string {
	return buildkit.ImageRef(templateOpts)
}

// Buildkitd manages the lifecycle of the local buildkitd container.
type Buildkitd struct {
	docker *docker.Client
}

// NewBuildkitd creates a Buildkitd manager using Docker settings from the environment.
func NewBuildkitd() (*Buildkitd, error) {
	c, err := docker.NewClient()
	if err != nil {
		return nil, err
	}
	return &Buildkitd{docker: c}, nil
}

// Close releases the underlying Docker client.
func (b *Buildkitd) Close() error {
	return b.docker.Close()
}

// Start pulls the image if absent, creates and starts the buildkitd container.
// Returns an error if the container is already running.
func (b *Buildkitd) Start(ctx context.Context, imageRef string, hostPort int) error {
	return b.docker.ContainerRun(ctx, BuildkitdContainerName, imageRef, hostPort, buildkitdContainerPort)
}

// Stop stops the buildkitd container. It is a no-op if the container is not running or does not exist.
// When remove is true the container is also removed after stopping.
func (b *Buildkitd) Stop(ctx context.Context, remove bool) error {
	return b.docker.ContainerStop(ctx, BuildkitdContainerName, remove)
}

// Status returns the current state of the buildkitd container.
func (b *Buildkitd) Status(ctx context.Context) (docker.ContainerStatus, error) {
	return b.docker.ContainerInspect(ctx, BuildkitdContainerName)
}

// Logs streams stdout and stderr of the buildkitd container to w.
// When follow is true the stream stays open until ctx is cancelled or the container exits.
func (b *Buildkitd) Logs(ctx context.Context, w io.Writer, follow bool) error {
	return b.docker.ContainerLogs(ctx, BuildkitdContainerName, w, follow)
}
