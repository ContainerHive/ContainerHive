package model

type Versions map[string]string

type BuildArgs map[string]string

type Tag struct {
	Name      string    `yaml:"name" json:"name" jsonschema:"Name of the tag"`
	Versions  Versions  `yaml:"versions" json:"versions,omitempty" jsonschema:"Versions to use for this tag"`
	BuildArgs BuildArgs `yaml:"build_args" json:"build_args,omitempty" jsonschema:"Build args to specify for this tag"`
}

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
}

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

type ContainerHiveProject struct {
	RootDir            string
	ConfigFilePath     string
	Config             HiveProjectConfig
	ImagesByIdentifier map[string]*Image
	ImagesByName       map[string][]*Image
}
