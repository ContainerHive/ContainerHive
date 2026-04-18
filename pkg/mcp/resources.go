package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func getImageResource(projectRoot, imageName string) (string, error) {
	imageYAMLPath := filepath.Join(projectRoot, "images", imageName, "image.yml")
	data, err := os.ReadFile(imageYAMLPath)
	if err != nil {
		return "", fmt.Errorf("failed to read image.yml: %w", err)
	}
	return string(data), nil
}

func getProjectConfig(projectRoot string) (string, error) {
	configPath := filepath.Join(projectRoot, "hive.yml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read hive.yml: %w", err)
	}
	return string(data), nil
}

func getImageSchema() string {
	return imageSchemaJSON
}

func getHiveSchema() string {
	return hiveSchemaJSON
}

type imageResourceHandler struct {
	projectRoot string
}

func (h *imageResourceHandler) ReadResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	uri := req.Params.URI

	if uri == "image://schema" {
		schema := getImageSchema()
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      uri,
					Text:     schema,
					MIMEType: "application/json",
				},
			},
		}, nil
	}

	if uri == "project://schema" {
		schema := getHiveSchema()
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      uri,
					Text:     schema,
					MIMEType: "application/json",
				},
			},
		}, nil
	}

	if uri == "project://config" {
		config, err := getProjectConfig(h.projectRoot)
		if err != nil {
			return nil, err
		}
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      uri,
					Text:     config,
					MIMEType: "application/x-yaml",
				},
			},
		}, nil
	}

	imageName := uri[len("image://"):]
	imageYAML, err := getImageResource(h.projectRoot, imageName)
	if err != nil {
		return nil, mcp.ResourceNotFoundError(uri)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      uri,
				Text:     imageYAML,
				MIMEType: "application/x-yaml",
			},
		},
	}, nil
}
