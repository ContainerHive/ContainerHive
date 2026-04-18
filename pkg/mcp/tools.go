package mcp

type ListImagesInput struct {
}

type ListImagesOutput struct {
	Images []imageInfo `json:"images"`
}

type GetImageInput struct {
	Name string `json:"name" jsonschema:"name of the image to get"`
}

type GetImageOutput struct {
	Image *imageDetail `json:"image"`
}

type GetDependenciesInput struct {
	Name      string `json:"name" jsonschema:"name of the image"`
	Direction string `json:"direction" jsonschema:"direction: forward (dependencies) or reverse (dependents)"`
}

type GetDependenciesOutput struct {
	Images []string `json:"images"`
}

type AddImageInput struct {
	Name              string `json:"name" jsonschema:"name of the image to create"`
	Description       string `json:"description" jsonschema:"description of the image"`
	BaseTag           string `json:"base_tag" jsonschema:"base docker tag (e.g., ubuntu:22.04)"`
	DockerfileContent string `json:"dockerfile_content,omitempty" jsonschema:"optional custom Dockerfile content"`
}

type AddImageOutput struct {
	Message string `json:"message"`
}

type AddImageVariantInput struct {
	ImageName   string            `json:"image_name" jsonschema:"name of the image to add variant to"`
	VariantName string            `json:"variant_name" jsonschema:"name of the variant"`
	TagSuffix   string            `json:"tag_suffix" jsonschema:"suffix to append to tags (e.g., -slim)"`
	Versions    map[string]string `json:"versions,omitempty" jsonschema:"version overrides for this variant"`
	BuildArgs   map[string]string `json:"build_args,omitempty" jsonschema:"build args for this variant"`
}

type AddImageVariantOutput struct {
	Message string `json:"message"`
}
