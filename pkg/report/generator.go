package report

import (
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/timo-reymann/ContainerHive/internal/buildconfig_resolver"
	"github.com/timo-reymann/ContainerHive/pkg/model"
	"github.com/timo-reymann/ContainerHive/pkg/platform"
)

type Generator struct {
}

func NewGenerator() *Generator {
	return &Generator{}
}

func (g *Generator) Generate(project *model.ContainerHiveProject) (*ProjectReport, error) {
	return &ProjectReport{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Images:      scanProject(project),
	}, nil
}

func scanProject(project *model.ContainerHiveProject) []ImageReport {
	var images []ImageReport
	for imageName, modelImages := range project.ImagesByName {
		if len(modelImages) == 0 {
			continue
		}
		images = append(images, scanImage(project.RootDir, imageName, modelImages[0]))
	}
	slices.SortFunc(images, func(a, b ImageReport) int {
		return strings.Compare(a.Name, b.Name)
	})
	return images
}

func scanImage(projectRoot, imageName string, img *model.Image) ImageReport {
	distPath := filepath.Join(projectRoot, "dist")

	var tagReports []TagReport
	for _, tagDef := range img.Tags {
		var platforms []PlatformReport
		for _, plat := range img.Platforms {
			platDir := platform.Sanitize(plat)
			sbomPath := filepath.Join(distPath, imageName, tagDef.Name, platDir, "cyclonedx.json")
			var sbom []SBOMPackage
			if sbomData, err := parseSBOMFile(sbomPath); err == nil {
				sbom = sbomData
			}
			platforms = append(platforms, PlatformReport{
				Platform: plat,
				SBOM:     sbom,
			})
		}

		// ignore error explicitly as in report step build already succeeded
		resolvedTagArgs, _ := buildconfig_resolver.ForTag(img, tagDef)

		tagReports = append(tagReports, TagReport{
			Name:      tagDef.Name,
			Platforms: platforms,
			Versions:  resolvedTagArgs.Versions,
			BuildArgs: resolvedTagArgs.BuildArgs,
		})
	}

	var variantReports []VariantReport
	for _, variantDef := range img.Variants {
		var variantTagReports []TagReport
		variantPlatforms := variantDef.Platforms
		if len(variantPlatforms) == 0 {
			variantPlatforms = img.Platforms
		}
		for _, baseTag := range img.Tags {
			var platforms []PlatformReport
			for _, plat := range variantPlatforms {
				platDir := platform.Sanitize(plat)
				sbomPath := filepath.Join(distPath, imageName, baseTag.Name+variantDef.TagSuffix, platDir, "cyclonedx.json")
				var sbom []SBOMPackage
				if sbomData, err := parseSBOMFile(sbomPath); err == nil {
					sbom = sbomData
				}
				platforms = append(platforms, PlatformReport{
					Platform: plat,
					SBOM:     sbom,
				})
			}

			// ignore error explicitly as in report step build already succeeded
			resolvedVariantTagArgs, _ := buildconfig_resolver.ForTagVariant(img, variantDef, baseTag)

			variantTagReports = append(variantTagReports, TagReport{
				Name:      baseTag.Name + variantDef.TagSuffix,
				Platforms: platforms,
				BuildArgs: resolvedVariantTagArgs.BuildArgs,
				Versions:  resolvedVariantTagArgs.Versions,
			})
		}

		variantReports = append(variantReports, VariantReport{
			Name:      variantDef.Name,
			Report:    Report{Icon: variantDef.Report.Icon},
			TagSuffix: variantDef.TagSuffix,
			Platforms: variantDef.Platforms,
			Tags:      variantTagReports,
		})
	}

	return ImageReport{
		Name:        imageName,
		Description: img.Description,
		Platforms:   img.Platforms,
		Tags:        tagReports,
		Variants:    variantReports,
		Report: Report{
			Icon: img.Report.Icon,
		},
	}
}

func parseSBOMFile(path string) ([]SBOMPackage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var sbom struct {
		Components []struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"components"`
	}

	if err := json.Unmarshal(data, &sbom); err != nil {
		return nil, err
	}

	var packages []SBOMPackage
	for _, comp := range sbom.Components {
		if comp.Name == "" {
			continue
		}
		if comp.Version == "" || comp.Version == "-" || comp.Version == "UNKNOWN" {
			continue
		}
		packages = append(packages, SBOMPackage{
			Name:    comp.Name,
			Version: comp.Version,
		})
	}
	slices.SortFunc(packages, func(a, b SBOMPackage) int {
		return strings.Compare(a.Name, b.Name)
	})
	return packages, nil
}

func MergeBuildArgs(base, override map[string]string) map[string]string {
	if base == nil && override == nil {
		return nil
	}
	result := make(map[string]string)
	for k, v := range base {
		result[k] = v
	}
	for k, v := range override {
		result[k] = v
	}
	return result
}

func (g *Generator) GenerateJSON(report *ProjectReport) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

func (g *Generator) GenerateHTMLFromAssets(report *ProjectReport) ([]byte, error) {
	reportJSON, err := json.Marshal(report)
	if err != nil {
		return nil, err
	}

	html := string(embeddedHTML)
	html = ReplacePlaceholder(html, string(reportJSON))
	html = strings.ReplaceAll(html, "/*INJECT_GENERATED_AT*/", report.GeneratedAt)
	html = strings.ReplaceAll(html, "/*INJECT_REGISTRY*/", "")

	return []byte(html), nil
}

func ReplacePlaceholder(html, data string) string {
	for i := 0; i < len(html)-len("/*INJECT_JSON_DATA*/"); i++ {
		if html[i:i+len("/*INJECT_JSON_DATA*/")] == "/*INJECT_JSON_DATA*/" {
			return html[:i] + data + html[i+len("/*INJECT_JSON_DATA*/"):]
		}
	}
	return html
}
