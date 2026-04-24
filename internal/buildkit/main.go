package buildkit

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/containerd/containerd/v2/core/content"
	"github.com/docker/cli/cli/config"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth/authprovider"
	"github.com/moby/buildkit/session/secrets/secretsprovider"
	"github.com/timo-reymann/ContainerHive/internal/buildkit/build_context"
	"github.com/timo-reymann/ContainerHive/internal/utils"
	"github.com/timo-reymann/ContainerHive/pkg/cache"
	"golang.org/x/sync/errgroup"
)

type Client struct {
	buildkit *client.Client
}

type BuildOpts struct {
	ImageName    string
	Platform     string
	TarFile      string
	BuildArgs    map[string]string
	Secrets      map[string][]byte
	Labels       map[string]string
	Cache        cache.BuildkitCache
	BuildContext build_context.BuildContext

	// RegistryRef is the full image reference to push to (e.g. "localhost:8500/ubuntu:22.04.linux-amd64").
	// When set, BuildKit pushes directly to the registry via the image exporter.
	RegistryRef string
	// RegistryInsecure allows pushing over HTTP (for local registries).
	RegistryInsecure bool
	// DockerMediaTypes forces the image exporter to emit Docker-scheme media
	// types instead of OCI. Set when the target registry rejects pure OCI
	// (e.g. Docker Hub).
	DockerMediaTypes bool

	// OCIStores maps store IDs to content stores for OCI layout named contexts.
	// Used to resolve inter-image dependencies without a registry.
	OCIStores map[string]content.Store
	// NamedContexts maps frontend attribute keys (e.g. "context:__hive__/ubuntu:22.04")
	// to OCI layout references (e.g. "oci-layout:hive-ubuntu-22.04@sha256:...").
	NamedContexts map[string]string
}

func NewClient(ctx context.Context, endpoint string) (*Client, error) {
	buildkit, err := client.New(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	return &Client{buildkit}, nil
}

func (c *Client) Close() error {
	return c.buildkit.Close()
}

func (c *Client) Version(ctx context.Context) (string, error) {
	info, err := c.buildkit.Info(ctx)
	if err != nil {
		return "", err
	}
	return info.BuildkitVersion.Version, nil
}

func (c *Client) Build(ctx context.Context, opts *BuildOpts, statusUpdateHandler func(chan *client.SolveStatus) error) error {
	var buildCache []client.CacheOptionsEntry
	if opts.Cache != nil {
		cacheOpt := opts.Cache
		attributes := cacheOpt.ToAttributes()
		_, hasExplicitIgnoreErr := attributes["ignore-errors"]
		if !hasExplicitIgnoreErr {
			attributes["ignore-errors"] = "true"
		}
		buildCache = []client.CacheOptionsEntry{
			{
				Type:  cacheOpt.Name(),
				Attrs: attributes,
			},
		}
	}

	localMounts, err := opts.BuildContext.ToLocalMounts()
	if err != nil {
		return errors.Join(errors.New("failed to mount build context"), err)
	}

	frontendAttrs := map[string]string{
		"filename":                    filepath.Base(opts.BuildContext.FileName()),
		"build-arg:SOURCE_DATE_EPOCH": "1770336000",
		"platform":                    opts.Platform,
		// this will be done using syft explicitly
		// as this should not rely on a upstream image
		// "attest:sbom":                 "",
	}

	utils.MergeMapWithPrefix("label:", frontendAttrs, opts.Labels)
	utils.MergeMapWithPrefix("build-arg:", frontendAttrs, opts.BuildArgs)
	utils.MergeMap(frontendAttrs, opts.NamedContexts)

	dockerConfig := config.LoadDefaultConfigFile(os.Stderr)
	solveOpts := client.SolveOpt{
		Session: []session.Attachable{
			authprovider.NewDockerAuthProvider(authprovider.DockerAuthProviderConfig{
				AuthConfigProvider: authprovider.LoadAuthConfig(dockerConfig),
			}),
			secretsprovider.FromMap(opts.Secrets),
		},
		CacheExports:  buildCache,
		CacheImports:  buildCache,
		Exports:       buildExports(opts),
		LocalMounts:   localMounts,
		OCIStores:     opts.OCIStores,
		Frontend:      opts.BuildContext.FrontendType(),
		FrontendAttrs: frontendAttrs,
	}

	statusUpdates := make(chan *client.SolveStatus)
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return statusUpdateHandler(statusUpdates)
	})

	eg.Go(func() error {
		_, err = c.buildkit.Build(ctx, solveOpts, "ContainerHive", opts.BuildContext.RunBuild, statusUpdates)
		return err
	})

	return eg.Wait()
}

// buildExports returns the BuildKit export entries. It always includes the OCI
// tar exporter. When RegistryRef is set, it adds an image exporter that pushes
// directly to the registry.
func buildExports(opts *BuildOpts) []client.ExportEntry {
	exports := []client.ExportEntry{
		{
			Type: "oci",
			Attrs: map[string]string{
				"name":              opts.ImageName,
				"rewrite-timestamp": "true",
			},
			Output: func(_ map[string]string) (io.WriteCloser, error) {
				return os.Create(opts.TarFile)
			},
		},
	}

	if opts.RegistryRef != "" {
		attrs := map[string]string{
			"name":              opts.RegistryRef,
			"push":              "true",
			"rewrite-timestamp": "true",
		}
		if opts.RegistryInsecure {
			attrs["registry.insecure"] = "true"
		}
		if opts.DockerMediaTypes {
			attrs["oci-mediatypes"] = "false"
		}
		exports = append(exports, client.ExportEntry{
			Type:  "image",
			Attrs: attrs,
		})
	}

	return exports
}
