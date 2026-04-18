package mcp

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type handlers struct {
	projectRoot string
}

func newHandlers(projectRoot string) *handlers {
	return &handlers{projectRoot: projectRoot}
}

func (h *handlers) handleListImages(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	images, err := listImages(ctx, h.projectRoot)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(ListImagesOutput{Images: images})
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(data)},
		},
	}, nil
}

func (h *handlers) handleGetImage(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input GetImageInput
	if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
		return nil, err
	}

	image, err := getImage(ctx, h.projectRoot, input.Name)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(GetImageOutput{Image: image})
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(data)},
		},
	}, nil
}

func (h *handlers) handleGetDependencies(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input GetDependenciesInput
	if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
		return nil, err
	}

	images, err := getDependencies(ctx, h.projectRoot, input.Name, input.Direction)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(GetDependenciesOutput{Images: images})
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(data)},
		},
	}, nil
}

func (h *handlers) handleGetImageSchema(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: getImageSchema()},
		},
	}, nil
}

func (h *handlers) handleGetHiveSchema(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: getHiveSchema()},
		},
	}, nil
}

func (h *handlers) handleAddImage(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input AddImageInput
	if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
		return nil, err
	}

	err := addImage(ctx, h.projectRoot, input.Name, input.Description, input.BaseTag, input.DockerfileContent)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(AddImageOutput{Message: "Image created successfully"})
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(data)},
		},
	}, nil
}

func (h *handlers) handleAddImageVariant(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input AddImageVariantInput
	if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
		return nil, err
	}

	err := addImageVariant(ctx, h.projectRoot, input.ImageName, input.VariantName, input.TagSuffix, input.Versions, input.BuildArgs)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(AddImageVariantOutput{Message: "Variant added successfully"})
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(data)},
		},
	}, nil
}

func (h *handlers) handleSearchDocumentation(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input SearchDocumentationInput
	if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
		return nil, err
	}

	results, err := searchDocumentation(ctx, input.Query, input.Limit)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(SearchDocumentationOutput{Results: results})
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(data)},
		},
	}, nil
}

func (h *handlers) handleGetDocumentation(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input GetDocumentationInput
	if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
		return nil, err
	}

	result, err := getDocumentation(ctx, input.Path)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(data)},
		},
	}, nil
}
