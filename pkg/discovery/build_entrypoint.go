package discovery

import (
	"errors"
	"fmt"

	"github.com/ContainerHive/ContainerHive/internal/file_resolver"
	"github.com/ContainerHive/ContainerHive/pkg/model"
)

var (
	dockerfileConfigFileNames = file_resolver.GetFileCandidates("Dockerfile")
	flakeConfigFileNames      = file_resolver.GetFileCandidates(model.FlakeFileName)
)

// buildEntrypoint couples the resolved build entrypoint path with its backend.
type buildEntrypoint struct {
	Path    string
	Backend string
}

// resolveBuildEntrypoint picks the build entrypoint and backend for a build
// root. An explicit backend wins. Auto-detection prefers the Nix flake when
// one exists and falls back to the Dockerfile.
func resolveBuildEntrypoint(backend, root string) (*buildEntrypoint, error) {
	dockerfilePath, dockerfileErr := file_resolver.ResolveFirstExistingFile(root, dockerfileConfigFileNames...)
	dockerfileMissing := dockerfileErr != nil
	flakePath, flakeErr := file_resolver.ResolveFirstExistingFile(root, flakeConfigFileNames...)
	flakeMissing := flakeErr != nil

	if backend != "" && backend != model.BuildBackendDockerfile && backend != model.BuildBackendNix {
		return nil, fmt.Errorf("unknown build backend %q (supported: %q, %q)", backend, model.BuildBackendDockerfile, model.BuildBackendNix)
	}

	switch backend {
	case model.BuildBackendDockerfile:
		if dockerfileMissing {
			return nil, errors.Join(fmt.Errorf("build backend %q requires a Dockerfile in %s", backend, root), dockerfileErr)
		}
		return &buildEntrypoint{Path: dockerfilePath, Backend: model.BuildBackendDockerfile}, nil
	case model.BuildBackendNix:
		if flakeMissing {
			return nil, errors.Join(fmt.Errorf("build backend %q requires %s in %s", backend, model.FlakeFileName, root), flakeErr)
		}
		return &buildEntrypoint{Path: flakePath, Backend: model.BuildBackendNix}, nil
	default:
		switch {
		case flakeMissing && dockerfileMissing:
			return nil, fmt.Errorf("no %s or Dockerfile found in %s (%w)", model.FlakeFileName, root, flakeErr)
		case flakeMissing:
			return &buildEntrypoint{Path: dockerfilePath, Backend: model.BuildBackendDockerfile}, nil
		default:
			return &buildEntrypoint{Path: flakePath, Backend: model.BuildBackendNix}, nil
		}
	}
}
