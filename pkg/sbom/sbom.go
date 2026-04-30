package sbom

import (
	"context"

	"github.com/ContainerHive/ContainerHive/internal/syft"
)

// Generator wraps the internal SBOM tool, providing a simpler API that
// combines generation and serialization into a single call.
type Generator struct {
	tool *syft.SBOMImageTool
}

// NewGenerator creates a new SBOM generator.
func NewGenerator() (*Generator, error) {
	tool, err := syft.NewSBOMImageTool()
	if err != nil {
		return nil, err
	}
	return &Generator{tool: tool}, nil
}

// Generate produces an SBOM from the given OCI tar file and serializes it
// in the requested format (e.g. "spdx-json").
func (g *Generator) Generate(ctx context.Context, tarFile, outputFormat string) ([]byte, error) {
	return g.tool.Generate(ctx, tarFile, outputFormat)
}
