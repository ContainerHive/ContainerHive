package syft

import (
	"context"
	"fmt"

	"github.com/anchore/syft/syft"
	"github.com/anchore/syft/syft/format"
	"github.com/anchore/syft/syft/sbom"

	_ "modernc.org/sqlite" // required for rpmdb and other features
)

// SBOMImageTool generates and serializes software bills of materials using Syft.
type SBOMImageTool struct {
	encoders *format.EncoderCollection
}

// NewSBOMImageTool creates a new SBOMImageTool with default encoders.
func NewSBOMImageTool() (*SBOMImageTool, error) {
	defaultEncodersConfig := format.DefaultEncodersConfig()
	encoders, err := defaultEncodersConfig.Encoders()
	if err != nil {
		return nil, err
	}

	return &SBOMImageTool{
		encoders: format.NewEncoderCollection(encoders...),
	}, nil
}

// GenerateSBOM produces an SBOM from the given OCI tar archive.
func (s *SBOMImageTool) GenerateSBOM(ctx context.Context, tarPath string) (*sbom.SBOM, error) {
	src, err := syft.GetSource(ctx, tarPath, nil)
	if err != nil {
		return nil, err
	}

	return syft.CreateSBOM(ctx, src, nil)
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
