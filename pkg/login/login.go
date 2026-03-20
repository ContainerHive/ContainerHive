package login

import (
	"errors"
	"io"
	"strings"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/types"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
)

// Options holds the parameters for a registry login.
type Options struct {
	ServerAddress string
	Username      string
	Password      string
	PasswordStdin io.Reader
	// ConfigDir overrides the Docker config directory. If empty, uses DOCKER_CONFIG env or default.
	ConfigDir string
}

// Login authenticates against a container registry and stores the credentials
// in the Docker config file, respecting any configured credential helpers.
func Login(opts Options) (configPath string, err error) {
	reg, err := name.NewRegistry(opts.ServerAddress)
	if err != nil {
		return "", err
	}
	serverAddress := reg.Name()

	password := opts.Password
	if opts.PasswordStdin != nil {
		contents, err := io.ReadAll(opts.PasswordStdin)
		if err != nil {
			return "", err
		}
		password = strings.TrimSuffix(string(contents), "\n")
		password = strings.TrimSuffix(password, "\r")
	}

	if opts.Username == "" && password == "" {
		return "", errors.New("username and password required")
	}

	cf, err := config.Load(opts.ConfigDir)
	if err != nil {
		return "", err
	}

	creds := cf.GetCredentialsStore(serverAddress)
	if serverAddress == name.DefaultRegistry {
		serverAddress = authn.DefaultAuthKey
	}

	if err := creds.Store(types.AuthConfig{
		ServerAddress: serverAddress,
		Username:      opts.Username,
		Password:      password,
	}); err != nil {
		return "", err
	}

	if err := cf.Save(); err != nil {
		return "", err
	}

	return cf.Filename, nil
}
