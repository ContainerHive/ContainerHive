package build

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/util/progress/progressui"
	"github.com/timo-reymann/ContainerHive/internal/buildkit"
	"github.com/timo-reymann/ContainerHive/internal/buildkit/build_context"
	"github.com/timo-reymann/ContainerHive/pkg/cache"
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

// Build builds a container image. Progress output is written to w.
func (c *Client) Build(ctx context.Context, opts *BuildOpts, w io.Writer) error {
	statusHandler := newProgressHandler(w)

	return c.inner.Build(ctx, &buildkit.BuildOpts{
		ImageName: opts.ImageName,
		Platform:  opts.Platform,
		TarFile:   opts.TarFile,
		BuildArgs: opts.BuildArgs,
		Secrets:   opts.Secrets,
		Cache:     opts.Cache,
		BuildContext: &build_context.DockerfileBuildContext{
			Root:       opts.ContextDir,
			Dockerfile: opts.Dockerfile,
		},
	}, statusHandler)
}

func newProgressHandler(w io.Writer) func(chan *client.SolveStatus) error {
	return func(ch chan *client.SolveStatus) error {
		if w == nil {
			w = os.Stdout
		}
		d, err := progressui.NewDisplay(w, progressui.TtyMode)
		if err != nil {
			d, _ = progressui.NewDisplay(w, progressui.PlainMode)
		}
		_, err = d.UpdateFrom(context.TODO(), ch)
		return err
	}
}
