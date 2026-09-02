package docker

import (
	"github.com/docker/docker/api/types/registry"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
)

// registryAuthFor resolves credentials for imageRef from the Docker config
// (honouring DOCKER_CONFIG and credential helpers) and encodes them for the
// daemon's X-Registry-Auth header.
//
// The daemon never reads the client's config file, so a pull from a private
// registry fails with "no basic auth credentials" unless the client passes the
// credentials along - `ch login` alone is not enough. Returns an empty string
// when no credentials are configured, which the daemon treats as anonymous.
func registryAuthFor(imageRef string) (string, error) {
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return "", err
	}

	authenticator, err := authn.DefaultKeychain.Resolve(ref.Context().Registry)
	if err != nil {
		return "", err
	}

	cfg, err := authenticator.Authorization()
	if err != nil {
		return "", err
	}
	if cfg == nil || *cfg == (authn.AuthConfig{}) {
		return "", nil
	}

	return registry.EncodeAuthConfig(registry.AuthConfig{
		Username:      cfg.Username,
		Password:      cfg.Password,
		Auth:          cfg.Auth,
		IdentityToken: cfg.IdentityToken,
		RegistryToken: cfg.RegistryToken,
		ServerAddress: ref.Context().RegistryStr(),
	})
}
