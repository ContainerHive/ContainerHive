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
	"github.com/timo-reymann/ContainerHive/internal/file_resolver"
	"github.com/timo-reymann/ContainerHive/internal/file_resolver/templating"
	"github.com/timo-reymann/ContainerHive/pkg/model"
	"github.com/timo-reymann/ContainerHive/pkg/platform"
)

func renderReadmeContent(readmePath string, imageName string, versions model.Versions, buildArgs model.BuildArgs) string {
	if readmePath == "" {
		return ""
	}

	tplCtx := &templating.TemplateContext{
		Versions:  versions,
		BuildArgs: buildArgs,
		ImageName: imageName,
	}
	rendered, err := file_resolver.ReadAndRenderFile(tplCtx, readmePath)
	if err != nil {
		return ""
	}
	return string(rendered)
}

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
	distPath := filepath.Join(projectRoot, model.DistDirName)

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

		variantReadme := renderReadmeContent(variantDef.ReadmePath, imageName, variantDef.Versions, variantDef.BuildArgs)

		variantReports = append(variantReports, VariantReport{
			Name:      variantDef.Name,
			Readme:    variantReadme,
			Report:    Report{Icon: variantDef.Report.Icon},
			TagSuffix: variantDef.TagSuffix,
			Platforms: variantDef.Platforms,
			Tags:      variantTagReports,
		})
	}

	readmeContent := renderReadmeContent(img.ReadmePath, imageName, img.Versions, img.BuildArgs)

	return ImageReport{
		Name:        imageName,
		Description: img.Description,
		Readme:      readmeContent,
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

func (g *Generator) GenerateJSON(report *ProjectReport) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

func (g *Generator) GenerateHTMLFromAssets(report *ProjectReport) ([]byte, error) {
	reportJSON, err := json.Marshal(report)
	if err != nil {
		return nil, err
	}

	html := string(embeddedHTML)
	html = replaceFirstPlaceholder(html, "/*INJECT_JSON_DATA*/", string(reportJSON))
	html = strings.ReplaceAll(html, "/*INJECT_GENERATED_AT*/", report.GeneratedAt)
	html = strings.ReplaceAll(html, "/*INJECT_REGISTRY*/", "")

	return []byte(html), nil
}

func replaceFirstPlaceholder(html, placeholder, data string) string {
	if idx := strings.Index(html, placeholder); idx >= 0 {
		return html[:idx] + data + html[idx+len(placeholder):]
	}
	return html
}
