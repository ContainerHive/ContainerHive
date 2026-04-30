package cli

import (
	"context"

	"github.com/ContainerHive/ContainerHive/pkg/mcp"
	"github.com/urfave/cli/v3"
)

func mcpCmd() *cli.Command {
	return &cli.Command{
		Name:  "mcp",
		Usage: "Start the ContainerHive MCP server",
		Description: `Start an MCP (Model Context Protocol) server that provides tools for managing ContainerHive images.

This server can be used by AI assistants to:
- List all images in the project
- Get details about specific images
- Get build dependencies
- Get JSON schemas for configuration
- Add new images
- Add variants to existing images`,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			projectRoot := cmd.String("project")
			return mcp.RunMCPServer(projectRoot)
		},
	}
}
