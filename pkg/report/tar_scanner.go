package report

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/timo-reymann/ContainerHive/pkg/model"
	"github.com/timo-reymann/ContainerHive/pkg/platform"
)

type TarScanner struct {
	distPath string
	project  *model.ContainerHiveProject
}

func NewTarScanner(distPath string, project *model.ContainerHiveProject) (*TarScanner, error) {
	return &TarScanner{
		distPath: distPath,
		project:  project,
	}, nil
}

func (s *TarScanner) Scan() ([]ImageReport, error) {
	entries, err := os.ReadDir(s.distPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("dist directory not found: %s", s.distPath)
		}
		return nil, fmt.Errorf("failed to read dist directory: %w", err)
	}

	var images []ImageReport

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		imageName := entry.Name()
		imageReport, err := s.scanImage(imageName)
		if err != nil {
			return nil, fmt.Errorf("failed to scan image %s: %w", imageName, err)
		}

		if len(imageReport.Tags) > 0 || len(imageReport.Variants) > 0 {
			images = append(images, imageReport)
		}
	}

	return images, nil
}

func (s *TarScanner) scanImage(imageName string) (ImageReport, error) {
	imageDir := filepath.Join(s.distPath, imageName)

	entries, err := os.ReadDir(imageDir)
	if err != nil {
		return ImageReport{Name: imageName}, fmt.Errorf("failed to read image directory: %w", err)
	}

	var baseTags []TagReport
	variantMap := make(map[string][]TagReport)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		tagName := entry.Name()

		tagReport, err := s.scanTag(imageName, tagName)
		if err != nil {
			return ImageReport{Name: imageName}, fmt.Errorf("failed to scan tag %s: %w", tagName, err)
		}

		if tagReport.Name == "" {
			continue
		}

		_, variantSuffix := s.classifyTag(tagName)
		if variantSuffix == "" {
			baseTags = append(baseTags, tagReport)
		} else {
			variantMap[variantSuffix] = append(variantMap[variantSuffix], tagReport)
		}
	}

	var variants []VariantReport
	for suffix, tags := range variantMap {
		if len(tags) > 0 {
			variants = append(variants, VariantReport{
				Name:      strings.TrimPrefix(suffix, "-"),
				TagSuffix: suffix,
				Tags:      tags,
			})
		}
	}

	modelImages := s.project.ImagesByName[imageName]
	var versions map[string]string

	if len(modelImages) > 0 {
		versions = modelImages[0].Versions
	}

	return ImageReport{
		Name:     imageName,
		Versions: versions,
		Tags:     baseTags,
		Variants: variants,
	}, nil
}

func (s *TarScanner) classifyTag(tagName string) (string, string) {
	knownSuffixes := []string{"-node", "-slim", "-full", "-dev", "-prod"}

	for _, suffix := range knownSuffixes {
		if strings.HasSuffix(tagName, suffix) {
			baseTag := strings.TrimSuffix(tagName, suffix)
			return baseTag, suffix
		}
	}

	return tagName, ""
}

func (s *TarScanner) scanTag(imageName, tagName string) (TagReport, error) {
	tagDir := filepath.Join(s.distPath, imageName, tagName)

	platforms := s.getPlatformsForTag(imageName, tagName)

	var platformReports []PlatformReport
	for _, plat := range platforms {
		platformDir := filepath.Join(tagDir, platform.Sanitize(plat))
		platformReport, err := s.scanPlatform(platformDir, plat)
		if err != nil {
			continue
		}
		if platformReport.Platform != "" {
			platformReports = append(platformReports, platformReport)
		}
	}

	tagReport := TagReport{
		Name:      tagName,
		Platforms: platformReports,
	}

	return tagReport, nil
}

func (s *TarScanner) getPlatformsForTag(imageName, tagName string) []string {
	modelImages := s.project.ImagesByName[imageName]
	if len(modelImages) == 0 {
		return platform.DefaultPlatforms
	}

	img := modelImages[0]
	basePlatforms := img.Platforms

	if len(basePlatforms) == 0 {
		basePlatforms = platform.DefaultPlatforms
	}

	variantSuffix := ""
	if _, suffix := s.classifyTag(tagName); suffix != "" {
		variantSuffix = suffix
	}

	if variantSuffix != "" && img.Variants != nil {
		for _, variant := range img.Variants {
			if variant.TagSuffix == variantSuffix {
				if len(variant.Platforms) == 0 {
					return basePlatforms
				}
				return variant.Platforms
			}
		}
	}

	return basePlatforms
}

func (s *TarScanner) scanPlatform(platformDir, platform string) (PlatformReport, error) {
	tarPath := filepath.Join(platformDir, "image.tar")
	sbomPath := filepath.Join(platformDir, "cyclonedx.json")

	var size int64
	var hasSBOM bool
	var sbom []SBOMPackage

	if stat, err := os.Stat(tarPath); err == nil {
		size = stat.Size()
	}

	if sbomData, err := os.ReadFile(sbomPath); err == nil {
		hasSBOM = true
		sbom = s.parseSBOM(sbomData)
	}

	return PlatformReport{
		Platform: platform,
		Size:     size,
		HasSBOM:  hasSBOM,
		SBOM:     sbom,
	}, nil
}

func (s *TarScanner) parseSBOM(data []byte) []SBOMPackage {
	var sbom struct {
		Components []struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"components"`
	}

	if err := json.Unmarshal(data, &sbom); err != nil {
		return nil
	}

	var packages []SBOMPackage
	for _, comp := range sbom.Components {
		if comp.Name != "" {
			packages = append(packages, SBOMPackage{
				Name:    comp.Name,
				Version: comp.Version,
			})
		}
	}

	return packages
}

func (s *TarScanner) Source() string {
	return "tar"
}
