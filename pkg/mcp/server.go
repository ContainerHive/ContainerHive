package mcp

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/timo-reymann/ContainerHive/pkg/version"
)

var (
	listImagesSchema      = json.RawMessage(`{"type": "object", "properties": {}}`)
	getImageInputSchema   = json.RawMessage(`{"type": "object", "properties": {"name": {"type": "string", "description": "name of the image to get"}}, "required": ["name"]}`)
	getDependenciesSchema = json.RawMessage(`{"type": "object", "properties": {"name": {"type": "string", "description": "name of the image"}, "direction": {"type": "string", "description": "direction: forward (dependencies) or reverse (dependents)", "enum": ["forward", "reverse"]}}, "required": ["name", "direction"]}`)
	getImageFileSchema    = json.RawMessage(`{"type": "object", "properties": {}}`)
	getHiveFileSchema     = json.RawMessage(`{"type": "object", "properties": {}}`)
	addImageSchema        = json.RawMessage(`{"type": "object", "properties": {"name": {"type": "string", "description": "name of the image to create"}, "description": {"type": "string", "description": "description of the image"}, "base_tag": {"type": "string", "description": "base docker tag (e.g., ubuntu:22.04)"}, "dockerfile_content": {"type": "string", "description": "optional custom Dockerfile content"}}, "required": ["name", "description", "base_tag"]}`)
	addImageVariantSchema = json.RawMessage(`{"type": "object", "properties": {"image_name": {"type": "string", "description": "name of the image to add variant to"}, "variant_name": {"type": "string", "description": "name of the variant"}, "tag_suffix": {"type": "string", "description": "suffix to append to tags (e.g., -slim)"}, "versions": {"type": "object", "description": "version overrides for this variant"}, "build_args": {"type": "object", "description": "build args for this variant"}}, "required": ["image_name", "variant_name", "tag_suffix"]}`)
)

func RunMCPServer(projectRoot string) error {
	s := mcp.NewServer(&mcp.Implementation{
		Name:    "containerhive",
		Version: version.Get(),
	}, nil)

	handlers := newHandlers(projectRoot)

	s.AddTool(&mcp.Tool{
		Name:        "list_images",
		Description: "List all images in the ContainerHive project with their tags, variants, versions, and platforms",
		InputSchema: listImagesSchema,
	}, handlers.handleListImages)

	s.AddTool(&mcp.Tool{
		Name:        "get_image",
		Description: "Get full configuration details for a specific image (excluding secrets)",
		InputSchema: getImageInputSchema,
	}, handlers.handleGetImage)

	s.AddTool(&mcp.Tool{
		Name:        "get_dependencies",
		Description: "Get build dependencies for an image (forward) or images that depend on it (reverse)",
		InputSchema: getDependenciesSchema,
	}, handlers.handleGetDependencies)

	s.AddTool(&mcp.Tool{
		Name:        "get_image_schema",
		Description: "Get the JSON schema for image.yml configuration files",
		InputSchema: getImageFileSchema,
	}, handlers.handleGetImageSchema)

	s.AddTool(&mcp.Tool{
		Name:        "get_hive_schema",
		Description: "Get the JSON schema for hive.yml configuration files",
		InputSchema: getHiveFileSchema,
	}, handlers.handleGetHiveSchema)

	s.AddTool(&mcp.Tool{
		Name:        "add_image",
		Description: "Create a new image directory with a stub Dockerfile and image.yml configuration",
		InputSchema: addImageSchema,
	}, handlers.handleAddImage)

	s.AddTool(&mcp.Tool{
		Name:        "add_image_variant",
		Description: "Add a new variant to an existing image with a stub Dockerfile in the variant subdirectory",
		InputSchema: addImageVariantSchema,
	}, handlers.handleAddImageVariant)

	resourceHandler := &imageResourceHandler{projectRoot: projectRoot}

	s.AddResource(&mcp.Resource{
		URI:         "image://schema",
		MIMEType:    "application/json",
		Description: "JSON schema for image.yml configuration",
	}, resourceHandler.ReadResource)

	s.AddResource(&mcp.Resource{
		URI:         "project://schema",
		MIMEType:    "application/json",
		Description: "JSON schema for hive.yml configuration",
	}, resourceHandler.ReadResource)

	s.AddResource(&mcp.Resource{
		URI:         "project://config",
		MIMEType:    "application/x-yaml",
		Description: "The project's hive.yml configuration",
	}, resourceHandler.ReadResource)

	s.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "image",
		URITemplate: "image://{name}",
		MIMEType:    "application/x-yaml",
		Description: "Image configuration files",
	}, resourceHandler.ReadResource)

	return s.Run(context.Background(), &mcp.StdioTransport{})
}
