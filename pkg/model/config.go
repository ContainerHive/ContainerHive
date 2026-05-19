package model

const DistDirName = "dist"

// HiveParentPlaceholder is the source-Dockerfile token that pkg/rendering
// substitutes with a concrete __hive__/<image>:<tag> reference at generate
// time. Lint and rendering both depend on it, so it lives here to avoid
// divergence between consumers.
const HiveParentPlaceholder = "__hive_parent__"

// Secrets represents a collection of named secrets
// This is a type alias to the secrets.Secrets type
type Secrets map[string]Secret

// Secret represents a named secret with its value configuration
// This is a simplified version for the model package
type Secret struct {
	SourceType string `yaml:"source,omitempty" json:"source,omitempty" jsonschema:"Source type of the secret (env, plain). If omitted, auto-detected from value."`
	Value      string `yaml:"value" json:"value" jsonschema:"Value of the secret (env var name or plain text)"`
}

// SecretValue represents how a secret value should be resolved
type SecretValue struct {
	SourceType string `yaml:"source,omitempty" json:"source,omitempty" jsonschema:"Source type of the secret (env, plain). If omitted, auto-detected from value."`
	Value      string `yaml:"value" json:"value" jsonschema:"Value of the secret (env var name or plain text)"`
}

// VariantConfig defines a variant in the image definition YAML file.
type VariantConfig struct {
	Name      string            `yaml:"name" json:"name" jsonschema:"Name of the variant"`
	TagSuffix string            `yaml:"tag_suffix" json:"tag_suffix" jsonschema:"Suffix to append to the tag name for this variant"`
	Versions  Versions          `yaml:"versions" json:"versions,omitempty" jsonschema:"Versions to use for this variant"`
	BuildArgs BuildArgs         `yaml:"build_args" json:"build_args,omitempty" jsonschema:"Build args to add for this variant"`
	Platforms []string          `yaml:"platforms,omitempty" json:"platforms,omitempty" jsonschema:"Target platforms for this variant (e.g. linux/amd64)"`
	Report    ReportConfig      `yaml:"report" json:"report,omitempty" jsonschema:"Report metadata"`
	Labels    map[string]string `yaml:"labels,omitempty" json:"labels,omitempty" jsonschema:"Custom OCI image labels applied to this variant. Overrides tag and image labels."`
}

// ReportConfig holds report-related metadata for an image or variant.
type ReportConfig struct {
	Icon *string `yaml:"icon" json:"icon" jsonschema:"Icon slug for devicon (e.g. go-original)"`
}

// LatestAliasConfig configures automatic latest-alias assignment for an image.
type LatestAliasConfig struct {
	Tag       string `yaml:"tag" json:"tag" jsonschema:"Alias tag name to assign to the highest semantic version (e.g. latest, stable),required"`
	OnMissing string `yaml:"on_missing,omitempty" json:"on_missing,omitempty" jsonschema:"Behaviour when no semantic tags are found: error (default), warning, silent"`
}

// ImageDefinitionConfig is the parsed content of an image definition YAML file.
type ImageDefinitionConfig struct {
	Description string             `yaml:"description" json:"description,omitempty" jsonschema:"Description of the image"`
	Tags        []*Tag             `yaml:"tags" json:"tags" jsonschema:"Tags to create for this image"`
	Variants    []VariantConfig    `yaml:"variants" json:"variants,omitempty" jsonschema:"Variants to create for this image"`
	Versions    Versions           `yaml:"versions" json:"versions,omitempty" jsonschema:"Versions to use for this image"`
	BuildArgs   BuildArgs          `yaml:"build_args" json:"build_args,omitempty" jsonschema:"Build args to add for this image"`
	Secrets     Secrets            `yaml:"secrets" json:"secrets,omitempty" jsonschema:"Secrets to resolve for this image"`
	DependsOn   []string           `yaml:"depends_on" json:"depends_on,omitempty" jsonschema:"Names of other images in this project that must be built before this image"`
	Platforms   []string           `yaml:"platforms,omitempty" json:"platforms,omitempty" jsonschema:"Target platforms for this image (e.g. linux/amd64)"`
	LatestAlias *LatestAliasConfig `yaml:"latest_alias,omitempty" json:"latest_alias,omitempty" jsonschema:"Configure an alias pointing to the highest semantic version tag"`
	Report      ReportConfig       `yaml:"report" json:"report,omitempty" jsonschema:"Report metadata"`
	Labels      map[string]string  `yaml:"labels,omitempty" json:"labels,omitempty" jsonschema:"Custom OCI image labels applied to this image. Standard auto-derived OCI keys override these."`
}

// BuildKitConfig holds the BuildKit daemon connection settings.
type BuildKitConfig struct {
	Address string `yaml:"address" json:"address" jsonschema:"BuildKit daemon address (e.g. tcp://127.0.0.1:8502)"`
}

// CacheConfig configures the build cache backend (S3 or registry).
type CacheConfig struct {
	// Type discriminates the cache backend: "s3" or "registry"
	Type string `yaml:"type" json:"type" jsonschema:"Cache type (s3, registry),required"`

	// S3 fields (type: s3)
	Endpoint        string `yaml:"endpoint,omitempty" json:"endpoint,omitempty" jsonschema:"S3 endpoint URL"`
	Bucket          string `yaml:"bucket,omitempty" json:"bucket,omitempty" jsonschema:"S3 bucket name"`
	Region          string `yaml:"region,omitempty" json:"region,omitempty" jsonschema:"S3 region"`
	AccessKeyId     string `yaml:"access_key_id,omitempty" json:"access_key_id,omitempty" jsonschema:"S3 access key ID"`
	SecretAccessKey string `yaml:"secret_access_key,omitempty" json:"secret_access_key,omitempty" jsonschema:"S3 secret access key"`
	UsePathStyle    bool   `yaml:"use_path_style,omitempty" json:"use_path_style,omitempty" jsonschema:"Use path-style S3 URLs"`

	// Registry fields (type: registry)
	Ref      string `yaml:"ref,omitempty" json:"ref,omitempty" jsonschema:"Registry cache ref (e.g. registry:5000/cache)"`
	Insecure bool   `yaml:"insecure,omitempty" json:"insecure,omitempty" jsonschema:"Allow insecure registry connections"`
}

// RegistryConfig holds the container registry connection settings.
type RegistryConfig struct {
	Address string `yaml:"address" json:"address" jsonschema:"Container registry address"`
	// DockerMediaTypes forces Docker-scheme media types for image manifests
	// and the multi-arch manifest list instead of OCI. Unset (nil) enables
	// auto-detect: Docker Hub addresses (docker.io / index.docker.io /
	// registry-1.docker.io) default to Docker-scheme; everything else stays
	// OCI. Set explicitly for Docker Hub mirrors or proxies that need the
	// same handling.
	DockerMediaTypes *bool `yaml:"docker_media_types,omitempty" json:"docker_media_types,omitempty" jsonschema:"Force Docker-scheme media types. Omit for auto-detect (Docker Hub auto-enables)."`
}

// LabelsConfig holds project-level OCI image label values applied to every
// built image. Url and Documentation support %image% and %tag% placeholders.
// Custom carries arbitrary user-defined labels; keys colliding with the
// standard fields above or with auto-derived OCI labels are ignored.
type LabelsConfig struct {
	Vendor        string            `yaml:"vendor,omitempty" json:"vendor,omitempty" jsonschema:"Vendor name applied as org.opencontainers.image.vendor"`
	Authors       string            `yaml:"authors,omitempty" json:"authors,omitempty" jsonschema:"Authors applied as org.opencontainers.image.authors"`
	Url           string            `yaml:"url,omitempty" json:"url,omitempty" jsonschema:"URL applied as org.opencontainers.image.url. Supports %image% and %tag% placeholders."`
	Documentation string            `yaml:"documentation,omitempty" json:"documentation,omitempty" jsonschema:"Documentation URL applied as org.opencontainers.image.documentation. Supports %image% and %tag% placeholders."`
	Custom        map[string]string `yaml:"custom,omitempty" json:"custom,omitempty" jsonschema:"Arbitrary custom labels applied to every built image. Standard OCI keys are reserved and override these."`
}

// LintConfig configures Dockerfile linting via hadolint. Only plain Dockerfiles
// are linted; Dockerfiles with a templating extension (e.g. Dockerfile.gotpl)
// are skipped because hadolint cannot parse Go-template syntax.
type LintConfig struct {
	Ignored           []string          `yaml:"ignored,omitempty" json:"ignored,omitempty" jsonschema:"Rule IDs to ignore (e.g. DL3000)"`
	TrustedRegistries []string          `yaml:"trusted_registries,omitempty" json:"trusted_registries,omitempty" jsonschema:"Registries hadolint treats as trusted (suppresses DL3026)"`
	LabelSchema       map[string]string `yaml:"label_schema,omitempty" json:"label_schema,omitempty" jsonschema:"Expected LABEL keys and their validation types"`
	StrictLabels      *bool             `yaml:"strict_labels,omitempty" json:"strict_labels,omitempty" jsonschema:"Fail on labels missing from label_schema"`
	FailureThreshold  string            `yaml:"failure_threshold,omitempty" json:"failure_threshold,omitempty" jsonschema:"Lowest severity that causes a non-zero exit (error, warning, info, style, ignore). Defaults to error."`
}

// HiveProjectConfig is the top-level project configuration from hive.yml.
type HiveProjectConfig struct {
	BuildKit        *BuildKitConfig   `yaml:"buildkit,omitempty" json:"buildkit,omitempty" jsonschema:"BuildKit daemon configuration"`
	Cache           *CacheConfig      `yaml:"cache,omitempty" json:"cache,omitempty" jsonschema:"Build cache configuration"`
	Registry        *RegistryConfig   `yaml:"registry,omitempty" json:"registry,omitempty" jsonschema:"Container registry configuration"`
	Platforms       []string          `yaml:"platforms,omitempty" json:"platforms,omitempty" jsonschema:"Default target platforms for all images (e.g. linux/amd64)"`
	TemplateOptions map[string]string `yaml:"template_options,omitempty" json:"template_options,omitempty" jsonschema:"Custom template variables available via the option function in CI and custom templates"`
	Labels          *LabelsConfig     `yaml:"labels,omitempty" json:"labels,omitempty" jsonschema:"Project-level OCI image labels applied to every built image"`
	Lint            *LintConfig       `yaml:"lint,omitempty" json:"lint,omitempty" jsonschema:"Dockerfile linting configuration (hadolint)"`
}
