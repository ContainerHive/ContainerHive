package discovery

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ContainerHive/ContainerHive/internal/file_resolver"
	"github.com/ContainerHive/ContainerHive/pkg/model"
)

var (
	dockerfileConfigFileNames = file_resolver.GetFileCandidates("Dockerfile")
	flakeConfigFileNames      = file_resolver.GetFileCandidates(model.FlakeFileName)
	chiselConfigFileNames     = file_resolver.GetFileCandidates(model.ChiselFileName)
)

// buildEntrypoint couples the resolved build entrypoint path with its backend.
type buildEntrypoint struct {
	Path    string
	Backend string
}

// resolveBuildEntrypoint picks the build entrypoint and backend for a build
// root. An explicit backend wins. Auto-detection prefers the Nix flake when
// one exists and falls back to the Dockerfile. A chisel cut spec is only
// auto-detected when it is the sole entrypoint: a cut spec next to another
// entrypoint is ambiguous and fails.
func resolveBuildEntrypoint(backend, root string) (*buildEntrypoint, error) {
	dockerfilePath, dockerfileErr := file_resolver.ResolveFirstExistingFile(root, dockerfileConfigFileNames...)
	dockerfileMissing := dockerfileErr != nil
	flakePath, flakeErr := file_resolver.ResolveFirstExistingFile(root, flakeConfigFileNames...)
	flakeMissing := flakeErr != nil
	chiselPath, chiselErr := file_resolver.ResolveFirstExistingFile(root, chiselConfigFileNames...)
	chiselMissing := chiselErr != nil

	if backend != "" && backend != model.BuildBackendDockerfile && backend != model.BuildBackendNix && backend != model.BuildBackendChisel {
		return nil, fmt.Errorf("unknown build backend %q (supported: %q, %q, %q)", backend, model.BuildBackendDockerfile, model.BuildBackendNix, model.BuildBackendChisel)
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
	case model.BuildBackendChisel:
		if chiselMissing {
			return nil, errors.Join(fmt.Errorf("build backend %q requires %s in %s", backend, model.ChiselFileName, root), chiselErr)
		}
		return &buildEntrypoint{Path: chiselPath, Backend: model.BuildBackendChisel}, nil
	default:
		if !chiselMissing && (!flakeMissing || !dockerfileMissing) {
			return nil, fmt.Errorf("ambiguous build entrypoint in %s: %s needs an explicit build backend in image.yml (found %s and %s)",
				root, model.BuildBackendChisel, model.ChiselFileName, otherEntrypointNames(flakeMissing, dockerfileMissing, flakePath, dockerfilePath))
		}
		switch {
		case chiselMissing && flakeMissing && dockerfileMissing:
			return nil, fmt.Errorf("no %s, Dockerfile, or %s found in %s (%w)", model.FlakeFileName, model.ChiselFileName, root, flakeErr)
		case flakeMissing && dockerfileMissing:
			return &buildEntrypoint{Path: chiselPath, Backend: model.BuildBackendChisel}, nil
		case flakeMissing:
			return &buildEntrypoint{Path: dockerfilePath, Backend: model.BuildBackendDockerfile}, nil
		default:
			return &buildEntrypoint{Path: flakePath, Backend: model.BuildBackendNix}, nil
		}
	}
}

// otherEntrypointNames lists the file names of the non-chisel entrypoints
// found next to a chisel cut spec, for the ambiguity error message.
func otherEntrypointNames(flakeMissing, dockerfileMissing bool, flakePath, dockerfilePath string) string {
	names := make([]string, 0, 2)
	if !flakeMissing {
		names = append(names, filepath.Base(flakePath))
	}
	if !dockerfileMissing {
		names = append(names, filepath.Base(dockerfilePath))
	}
	return strings.Join(names, " and ")
}
