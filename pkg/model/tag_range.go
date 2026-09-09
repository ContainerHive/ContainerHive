package model

import (
	"errors"
	"fmt"
	"strconv"

	"gopkg.in/yaml.v3"
)

// TagRange generates concrete tags from an external version source at
// discovery time. Exactly one of Source or Generator must be set.
type TagRange struct {
	TagName   string            `yaml:"tag_name" json:"tag_name" jsonschema:"Go template for the generated tag name. Data: .major .minor .patch .full .raw .prerelease .extra.<key>,required"`
	Source    *SourceConfig     `yaml:"source,omitempty" json:"source,omitempty" jsonschema:"External version source. Mutually exclusive with generator."`
	Generator string            `yaml:"generator,omitempty" json:"generator,omitempty" jsonschema:"Named generator instead of an explicit source (e.g. semver-matrix). Mutually exclusive with source."`
	Params    map[string]string `yaml:"params,omitempty" json:"params,omitempty" jsonschema:"Parameters for the named generator"`
	Select    *SelectConfig     `yaml:"select,omitempty" json:"select,omitempty" jsonschema:"How many versions to keep per semantic version level. Required when source is set. Omitted levels default to latest."`
	Filter    *FilterConfig     `yaml:"filter,omitempty" json:"filter,omitempty" jsonschema:"Filters applied to fetched versions before selection"`
	MaxTags   int               `yaml:"max_tags,omitempty" json:"max_tags,omitempty" jsonschema:"Hard cap on generated tags for this range. Exceeding it is an error. Defaults to 50."`
	Versions  Versions          `yaml:"versions,omitempty" json:"versions,omitempty" jsonschema:"Versions for generated tags. Values are Go templates rendered per resolved version."`
	BuildArgs BuildArgs         `yaml:"build_args,omitempty" json:"build_args,omitempty" jsonschema:"Build args for generated tags. Values are Go templates rendered per resolved version."`
	Labels    map[string]string `yaml:"labels,omitempty" json:"labels,omitempty" jsonschema:"Custom OCI labels for generated tags. Values are Go templates rendered per resolved version."`
}

// SourceConfig describes an external version source. Following the
// CacheConfig precedent, this is a flat struct with a Type discriminator;
// fields that do not belong to the chosen type are rejected by the source's
// Validate method, not by the JSON schema.
type SourceConfig struct {
	Type string `yaml:"type" json:"type" jsonschema:"Source type (json, registry, dockerhub, github),required"`

	// type: json
	URL       string            `yaml:"url,omitempty" json:"url,omitempty" jsonschema:"HTTP(S) URL returning JSON (type: json)"`
	Transform string            `yaml:"transform,omitempty" json:"transform,omitempty" jsonschema:"JSONata expression mapping the document to a list of {version, ...extras} objects (type: json)"`
	Headers   map[string]string `yaml:"headers,omitempty" json:"headers,omitempty" jsonschema:"Extra request headers. Values support $VAR / ${VAR} environment expansion (type: json)"`

	// type: registry / dockerhub
	Image string `yaml:"image,omitempty" json:"image,omitempty" jsonschema:"Repository whose tags are listed, e.g. library/node or ghcr.io/owner/image (type: registry, dockerhub)"`

	// type: github
	Repo string `yaml:"repo,omitempty" json:"repo,omitempty" jsonschema:"GitHub repository as owner/name (type: github)"`
	Kind string `yaml:"kind,omitempty" json:"kind,omitempty" jsonschema:"What to list: releases (default) or tags (type: github)"`

	// common
	TokenEnv string `yaml:"token_env,omitempty" json:"token_env,omitempty" jsonschema:"Name of the environment variable holding an auth token. Never put the token itself here."`
	TTL      string `yaml:"ttl,omitempty" json:"ttl,omitempty" jsonschema:"Cache TTL override for this source, as a Go duration (e.g. 6h)"`
}

// SelectConfig picks how many versions survive at each semantic version
// level. Levels are applied hierarchically: major first, then minor within
// each kept major, then patch within each kept minor.
type SelectConfig struct {
	Major *LevelSelect `yaml:"major,omitempty" json:"major,omitempty" jsonschema:"Selection at the major level"`
	Minor *LevelSelect `yaml:"minor,omitempty" json:"minor,omitempty" jsonschema:"Selection at the minor level"`
	Patch *LevelSelect `yaml:"patch,omitempty" json:"patch,omitempty" jsonschema:"Selection at the patch level"`
}

// LevelSelect selects values at one semantic version level. YAML accepts
// four forms: the string "latest" (Last: 1), the string "all" (All: true),
// a bare positive integer (Last: N), or a mapping {last: N} / {all: true}.
type LevelSelect struct {
	Last int  `yaml:"last,omitempty" json:"last,omitempty" jsonschema:"Keep the N highest distinct values at this level"`
	All  bool `yaml:"all,omitempty" json:"all,omitempty" jsonschema:"Keep every distinct value at this level"`
}

// UnmarshalYAML implements custom decoding for the four accepted forms.
// KnownFields(true) on the outer decoder does not propagate into a custom
// unmarshaler, so unknown mapping keys are rejected by hand here to keep
// strictness parity with the rest of image.yml.
func (l *LevelSelect) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		return l.unmarshalScalar(node)
	case yaml.MappingNode:
		return l.unmarshalMapping(node)
	default:
		return fmt.Errorf("invalid selection: expected a scalar or mapping, got %v", node.Kind)
	}
}

func (l *LevelSelect) unmarshalScalar(node *yaml.Node) error {
	switch node.Value {
	case "latest":
		l.Last = 1
		return nil
	case "all":
		l.All = true
		return nil
	}
	n, err := strconv.Atoi(node.Value)
	if err != nil || n < 1 {
		return fmt.Errorf(`invalid selection %q: expected "latest", "all", a positive integer, or a mapping with last/all`, node.Value)
	}
	l.Last = n
	return nil
}

func (l *LevelSelect) unmarshalMapping(node *yaml.Node) error {
	for i := 0; i+1 < len(node.Content); i += 2 {
		key, val := node.Content[i].Value, node.Content[i+1]
		switch key {
		case "last":
			n, err := strconv.Atoi(val.Value)
			if err != nil || n < 1 {
				return fmt.Errorf("select last must be a positive integer, got %q", val.Value)
			}
			l.Last = n
		case "all":
			b, err := strconv.ParseBool(val.Value)
			if err != nil {
				return fmt.Errorf("select all must be a boolean, got %q", val.Value)
			}
			l.All = b
		default:
			return fmt.Errorf("unknown selection field %q: expected last or all", key)
		}
	}
	if l.Last > 0 && l.All {
		return errors.New("select last and all are mutually exclusive")
	}
	return nil
}

// FilterConfig narrows fetched versions before selection. Filters apply in
// this fixed order: ExcludePrerelease, ExcludeSuffixes, MinVersion/
// MaxVersion, MaxMajorIncrement.
type FilterConfig struct {
	ExcludePrerelease *bool    `yaml:"exclude_prerelease,omitempty" json:"exclude_prerelease,omitempty" jsonschema:"Drop versions with a semantic version prerelease component. Defaults to true."`
	ExcludeSuffixes   []string `yaml:"exclude_suffixes,omitempty" json:"exclude_suffixes,omitempty" jsonschema:"Drop versions whose raw string contains any of these substrings (e.g. -alpine for non-semver flavours)"`
	MinVersion        string   `yaml:"min_version,omitempty" json:"min_version,omitempty" jsonschema:"Drop versions lower than this (inclusive)"`
	MaxVersion        string   `yaml:"max_version,omitempty" json:"max_version,omitempty" jsonschema:"Drop versions higher than this (inclusive)"`
	MaxMajorIncrement int      `yaml:"max_major_increment,omitempty" json:"max_major_increment,omitempty" jsonschema:"Sanity guard: drop majors separated from the rest of the set by a version gap larger than this. 0 disables."`
}
