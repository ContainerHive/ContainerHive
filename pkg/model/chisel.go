package model

import (
	"bytes"
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

// ChiselReleaseRef identifies the chisel release source of a cut spec. YAML
// accepts two forms: a plain branch name of the canonical chisel-releases
// repository (for example "ubuntu-24.04"), or a mapping with a git
// repository URL and an optional revision for pinned or custom releases.
type ChiselReleaseRef struct {
	// Name is the chisel-releases branch name, for example "ubuntu-24.04".
	// The build clones the branch of the canonical chisel-releases
	// repository, so the release tracks the branch head.
	Name string `json:"name,omitempty"`
	// Git is a git repository URL that carries a chisel release checkout. It
	// can point at a fork or a custom release.
	Git string `json:"git,omitempty"`
	// GitRef is the revision or branch to check out in the Git repository.
	// Empty checks out the default branch.
	GitRef string `json:"ref,omitempty"`
}

// UnmarshalYAML implements custom decoding for the two accepted forms.
// KnownFields(true) on the outer decoder does not propagate into a custom
// unmarshaler, so unknown mapping keys are rejected by hand here to keep
// strictness parity with the rest of the specs.
func (r *ChiselReleaseRef) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		if node.Tag == "!!null" {
			return errors.New("chisel release is required: use a branch name such as ubuntu-24.04, or a mapping with git and ref")
		}
		r.Name = node.Value
		return nil
	case yaml.MappingNode:
		return r.unmarshalMapping(node)
	default:
		return fmt.Errorf("invalid chisel release: expected a branch name or a mapping with git and ref, got %v", node.Kind)
	}
}

func (r *ChiselReleaseRef) unmarshalMapping(node *yaml.Node) error {
	for i := 0; i+1 < len(node.Content); i += 2 {
		key, val := node.Content[i].Value, node.Content[i+1]
		switch key {
		case "git":
			if err := val.Decode(&r.Git); err != nil {
				return fmt.Errorf("chisel release git must be a string, got %v", val.Tag)
			}
		case "ref":
			if err := val.Decode(&r.GitRef); err != nil {
				return fmt.Errorf("chisel release ref must be a string, got %v", val.Tag)
			}
		default:
			return fmt.Errorf("unknown chisel release field %q: expected git or ref", key)
		}
	}
	return nil
}

// IsGit reports whether the release resolves from a git repository checkout.
func (r ChiselReleaseRef) IsGit() bool {
	return r.Git != ""
}

// Validate reports whether the release reference is well formed.
func (r ChiselReleaseRef) Validate() error {
	if r.Name != "" && r.Git != "" {
		return errors.New("chisel release must set either a branch name or git, not both")
	}
	if r.Name == "" && r.Git == "" {
		return errors.New("chisel release is required: use a branch name such as ubuntu-24.04, or a mapping with git and ref")
	}
	return nil
}

// ChiselCutSpec is the parsed content of a chisel.yaml build entrypoint. It
// selects the package slices the build cuts and the release they cut from.
//
// It is ContainerHive's own cut spec format. It is not the chisel-releases
// release manifest format, even though the file names collide: the release
// manifest lives inside a chisel-releases checkout that `chisel cut`
// consumes, while this file drives the ContainerHive build.
type ChiselCutSpec struct {
	// Release is the chisel release source the slices cut from.
	Release ChiselReleaseRef `yaml:"release" json:"release" jsonschema:"Chisel release the slices cut from: a chisel-releases branch name (e.g. ubuntu-24.04) or a mapping with git and ref,required"`
	// Slices lists the package slices to cut, in chisel notation
	// (<package>_<slice>), for example base-files_base.
	Slices []string `yaml:"slices" json:"slices" jsonschema:"Package slices to cut, in chisel notation <package>_<slice>,required"`
	// Ignore allows chisel to cut unstable or unmaintained releases.
	Ignore []string `yaml:"ignore,omitempty" json:"ignore,omitempty" jsonschema:"Release conditions to ignore (unstable, unmaintained)"`
}

// Validate reports whether the cut spec is complete and well formed.
func (s ChiselCutSpec) Validate() error {
	if err := s.Release.Validate(); err != nil {
		return err
	}
	if len(s.Slices) == 0 {
		return errors.New("chisel slices are required: list the package slices to cut (<package>_<slice>)")
	}
	for _, slice := range s.Slices {
		if slice == "" {
			return errors.New("chisel slices must not contain empty entries")
		}
	}
	for _, cond := range s.Ignore {
		switch cond {
		case "unstable", "unmaintained":
		default:
			return fmt.Errorf("unknown chisel ignore condition %q: supported values are unstable and unmaintained", cond)
		}
	}
	return nil
}

// ParseChiselCutSpec parses and validates the content of a chisel.yaml cut
// spec. Unknown fields fail the parse, like the image.yml decoder.
func ParseChiselCutSpec(data []byte) (*ChiselCutSpec, error) {
	var spec ChiselCutSpec
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&spec); err != nil {
		return nil, fmt.Errorf("invalid chisel.yaml cut spec: %w", err)
	}
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	return &spec, nil
}
