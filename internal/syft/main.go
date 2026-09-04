package syft

import (
	"context"
	"fmt"

	"github.com/anchore/syft/syft"
	"github.com/anchore/syft/syft/cataloging"
	"github.com/anchore/syft/syft/format"
	"github.com/anchore/syft/syft/sbom"

	_ "modernc.org/sqlite" // required for rpmdb and other features
)

// SBOMImageTool generates and serializes software bills of materials using Syft.
type SBOMImageTool struct {
	encoders     *format.EncoderCollection
	generateCPEs bool
}

// Option configures an SBOMImageTool.
type Option func(*SBOMImageTool)

// WithGenerateCPEs controls whether CPEs are generated for each component.
// CPEs are a meaningful contributor to SBOM size on images with many
// packages; some vulnerability scanners rely on them for OS/binary package
// matching where a PURL alone isn't enough, so this defaults to true and is
// opt-out rather than opt-in.
func WithGenerateCPEs(generate bool) Option {
	return func(t *SBOMImageTool) {
		t.generateCPEs = generate
	}
}

// NewSBOMImageTool creates a new SBOMImageTool with default encoders.
func NewSBOMImageTool(opts ...Option) (*SBOMImageTool, error) {
	defaultEncodersConfig := format.DefaultEncodersConfig()
	encoders, err := defaultEncodersConfig.Encoders()
	if err != nil {
		return nil, err
	}

	tool := &SBOMImageTool{
		encoders:     format.NewEncoderCollection(encoders...),
		generateCPEs: true,
	}
	for _, opt := range opts {
		opt(tool)
	}
	return tool, nil
}

// GenerateSBOM produces an SBOM from the given OCI tar archive.
//
// File cataloging is disabled: syft would otherwise emit one CycloneDX
// component per file in the image (path + digest, no package information),
// which dwarfs the actual package/library components in both count and
// size and isn't consumed by our report generator or by GitLab's
// dependency/container scanning.
func (s *SBOMImageTool) GenerateSBOM(ctx context.Context, tarPath string) (*sbom.SBOM, error) {
	src, err := syft.GetSource(ctx, tarPath, nil)
	if err != nil {
		return nil, err
	}

	cfg := syft.DefaultCreateSBOMConfig().
		WithoutFiles().
		WithDataGenerationConfig(cataloging.DataGenerationConfig{GenerateCPEs: s.generateCPEs})
	return cfg.Create(ctx, src)
}

// SerializeSBOM encodes an SBOM into the specified output format (e.g. "spdx-json").
func (s *SBOMImageTool) SerializeSBOM(sbom *sbom.SBOM, outputFormat string) ([]byte, error) {
	encoder := s.encoders.GetByString(outputFormat)
	if encoder == nil {
		return nil, fmt.Errorf("unsupported output format: %s", outputFormat)
	}
	return format.Encode(*sbom, encoder)
}

// Generate produces an SBOM from the given OCI tar file and serializes it
// in the requested format (e.g. "spdx-json").
func (s *SBOMImageTool) Generate(ctx context.Context, tarFile, outputFormat string) ([]byte, error) {
	result, err := s.GenerateSBOM(ctx, tarFile)
	if err != nil {
		return nil, err
	}
	return s.SerializeSBOM(result, outputFormat)
}
