package build

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/ContainerHive/ContainerHive/internal/buildkit"
	"github.com/ContainerHive/ContainerHive/internal/buildkit/build_context"
	"github.com/ContainerHive/ContainerHive/pkg/cache"
	"github.com/ContainerHive/ContainerHive/pkg/progress"
	"github.com/containerd/containerd/v2/core/content"
)

// Client wraps the internal BuildKit client.
type Client struct {
	inner *buildkit.Client
}

// BuildOpts contains all parameters for a single image build.
type BuildOpts struct {
	ImageName string
	Platform  string
	TarFile   string
	BuildArgs map[string]string
	Secrets   map[string][]byte
	Cache     cache.BuildkitCache

	// ContextDir is the build context root directory.
	ContextDir string
	// Dockerfile is the relative path to the Dockerfile within ContextDir.
	// Defaults to "Dockerfile" if empty.
	Dockerfile string

	// RegistryRef is the full image reference for direct registry push
	// (e.g. "localhost:8500/ubuntu:22.04.linux-amd64").
	RegistryRef string
	// RegistryInsecure allows pushing over HTTP.
	RegistryInsecure bool
	// DockerMediaTypes forces BuildKit's image exporter to emit Docker-scheme
	// media types (manifest, config, layers) rather than OCI. Required when
	// the target registry (e.g. Docker Hub) doesn't accept pure OCI.
	DockerMediaTypes bool

	// OCIStores maps store IDs to content stores for OCI layout named contexts.
	OCIStores map[string]content.Store
	// NamedContexts maps frontend attribute keys to OCI layout references.
	NamedContexts map[string]string

	// Labels is the OCI image label map applied to the built image. Each
	// entry becomes a `label:<key>` BuildKit frontend attribute.
	Labels map[string]string

	// ProgressConfig controls how build progress is displayed.
	// When zero-valued, AutoMode with DefaultColors is used.
	ProgressConfig progress.Config
}

// NewClient connects to a BuildKit daemon at the given endpoint.
// If endpoint is empty, it falls back to BUILDKIT_HOST environment variable.
func NewClient(ctx context.Context, endpoint string) (*Client, error) {
	if endpoint == "" {
		if envAddr := os.Getenv("BUILDKIT_HOST"); envAddr != "" {
			endpoint = envAddr
		}
	}

	if endpoint == "" {
		return nil, fmt.Errorf("no BuildKit endpoint provided and BUILDKIT_HOST not set")
	}

	c, err := buildkit.NewClient(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	return &Client{inner: c}, nil
}

// Close releases the BuildKit connection.
func (c *Client) Close() error {
	return c.inner.Close()
}

// Version returns the BuildKit daemon version string.
func (c *Client) Version(ctx context.Context) (string, error) {
	return c.inner.Version(ctx)
}

// Build builds a container image. Progress output is written to w unless
// opts.ProgressConfig.Writer is set, in which case that takes precedence.
func (c *Client) Build(ctx context.Context, opts *BuildOpts, w io.Writer) error {
	cfg := opts.ProgressConfig
	if cfg.Writer == nil {
		cfg.Writer = w
	}
	if cfg.Colors == (progress.Colors{}) {
		cfg.Colors = progress.DefaultColors()
	}
	statusHandler := progress.NewHandler(cfg)

	return c.inner.Build(ctx, &buildkit.BuildOpts{
		ImageName:        opts.ImageName,
		Platform:         opts.Platform,
		TarFile:          opts.TarFile,
		BuildArgs:        opts.BuildArgs,
		Secrets:          opts.Secrets,
		Labels:           opts.Labels,
		Cache:            opts.Cache,
		RegistryRef:      opts.RegistryRef,
		RegistryInsecure: opts.RegistryInsecure,
		DockerMediaTypes: opts.DockerMediaTypes,
		OCIStores:        opts.OCIStores,
		NamedContexts:    opts.NamedContexts,
		BuildContext: &build_context.DockerfileBuildContext{
			Root:       opts.ContextDir,
			Dockerfile: opts.Dockerfile,
		},
	}, statusHandler)
}
