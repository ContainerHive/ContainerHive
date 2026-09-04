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
// in the requested format (e.g. "spdx-json"). For CycloneDX output, syft's
// per-component "properties" bookkeeping is stripped to reduce file size;
// see stripCycloneDXProperties for why this is safe to drop.
func (g *Generator) Generate(ctx context.Context, tarFile, outputFormat string) ([]byte, error) {
	data, err := g.tool.Generate(ctx, tarFile, outputFormat)
	if err != nil {
		return nil, err
	}

	if isCycloneDXFormat(outputFormat) {
		return stripCycloneDXProperties(data)
	}
	return data, nil
}
