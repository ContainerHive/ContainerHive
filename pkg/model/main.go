package model

// Versions maps version placeholder names to their resolved values.
type Versions map[string]string

// BuildArgs maps Docker build argument names to their values.
type BuildArgs map[string]string

// Tag defines a single image tag with its version overrides and build arguments.
type Tag struct {
	Name      string    `yaml:"name" json:"name" jsonschema:"Name of the tag"`
	Versions  Versions  `yaml:"versions" json:"versions,omitempty" jsonschema:"Versions to use for this tag"`
	BuildArgs BuildArgs `yaml:"build_args" json:"build_args,omitempty" jsonschema:"Build args to specify for this tag"`
}

// Image represents a fully resolved container image definition within a project.
type Image struct {
	Identifier          string
	Name                string
	RootDir             string
	RootFSDir           string
	TestConfigFilePath  string
	DefinitionFilePath  string
	BuildEntryPointPath string
	Versions            Versions
	BuildArgs           BuildArgs `yaml:"build_args"`
	Secrets             Secrets   `yaml:"secrets"`
	Tags                map[string]*Tag
	Variants            map[string]*ImageVariant
	DependsOn           []string
	Platforms           []string
	LatestAlias         *LatestAliasConfig
}

// ImageVariant represents an alternative build of an image with different configuration.
type ImageVariant struct {
	Name                string
	BuildEntryPointPath string
	RootDir             string
	RootFSDir           string
	TagSuffix           string `yaml:"tag_suffix"`
	TestConfigFilePath  string
	Versions            Versions
	BuildArgs           BuildArgs `yaml:"build_args"`
	Platforms           []string
}

// ContainerHiveProject represents a fully loaded project with its configuration and images.
type ContainerHiveProject struct {
	RootDir            string
	ConfigFilePath     string
	Config             HiveProjectConfig
	ImagesByIdentifier map[string]*Image
	ImagesByName       map[string][]*Image
}
