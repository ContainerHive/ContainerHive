package model

// Versions maps version placeholder names to their resolved values.
type Versions map[string]string

// BuildArgs maps Docker build argument names to their values.
type BuildArgs map[string]string

// ReportModel holds report-related metadata for an image or variant.
type ReportModel struct {
	Icon string
}

// Tag defines a single image tag with its version overrides and build arguments.
type Tag struct {
	Name      string            `yaml:"name" json:"name" jsonschema:"Name of the tag"`
	Versions  Versions          `yaml:"versions" json:"versions,omitempty" jsonschema:"Versions to use for this tag"`
	BuildArgs BuildArgs         `yaml:"build_args" json:"build_args,omitempty" jsonschema:"Build args to specify for this tag"`
	Labels    map[string]string `yaml:"labels,omitempty" json:"labels,omitempty" jsonschema:"Custom OCI image labels applied to this tag. Overrides image-level labels."`

	// IsPrerelease marks a tag whose upstream version is a prerelease. Set by
	// the tag_ranges resolver for generated tags; static tags default to
	// false. Prerelease tags are excluded from latest_alias and alias
	// resolution by default. Not part of the YAML/JSON schema surface.
	IsPrerelease bool `yaml:"-" json:"-"`

	// GeneratedFrom identifies the tag_range that produced this tag (a
	// label including the range's index, unique even when two ranges share
	// a tag_name template). Empty for tags declared statically in
	// image.yml. Not part of the YAML/JSON schema surface.
	GeneratedFrom string `yaml:"-" json:"-"`
}

// Image represents a fully resolved container image definition within a project.
type Image struct {
	Identifier          string
	Name                string
	Description         string
	RootDir             string
	RootFSDir           string
	TestConfigFilePath  string
	DefinitionFilePath  string
	BuildEntryPointPath string
	ReadmePath          string
	Versions            Versions
	BuildArgs           BuildArgs `yaml:"build_args"`
	Secrets             Secrets   `yaml:"secrets"`
	Tags                map[string]*Tag
	TagRanges           []*TagRange
	Variants            map[string]*ImageVariant
	DependsOn           []string
	Platforms           []string
	LatestAlias         *LatestAliasConfig
	Report              ReportModel
	Labels              map[string]string
}

// ImageVariant represents an alternative build of an image with different configuration.
type ImageVariant struct {
	Name                string
	BuildEntryPointPath string
	ReadmePath          string
	RootDir             string
	RootFSDir           string
	TagSuffix           string `yaml:"tag_suffix"`
	TestConfigFilePath  string
	Versions            Versions
	BuildArgs           BuildArgs `yaml:"build_args"`
	Platforms           []string
	Report              ReportModel
	Labels              map[string]string
}

// ContainerHiveProject represents a fully loaded project with its configuration and images.
type ContainerHiveProject struct {
	RootDir            string
	ConfigFilePath     string
	Config             HiveProjectConfig
	ImagesByIdentifier map[string]*Image
	ImagesByName       map[string][]*Image
}
