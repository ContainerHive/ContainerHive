package report

import (
	"context"
	_ "embed"
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/timo-reymann/ContainerHive/pkg/model"
)

//go:embed assets/index.html
var embeddedHTML []byte

type SourceType string

const (
	SourceTar      SourceType = "tar"
	SourceRegistry SourceType = "registry"
	SourceAuto     SourceType = "auto"
)

type Generator struct {
	source   SourceType
	distPath string
}

func NewGenerator(source SourceType, distPath string) *Generator {
	return &Generator{
		source:   source,
		distPath: distPath,
	}
}

func (g *Generator) Generate(ctx context.Context, project *model.ContainerHiveProject, registryAddr string) (*ProjectReport, error) {
	source, images, err := g.collectImages(ctx, project, registryAddr)
	if err != nil {
		return nil, err
	}

	return &ProjectReport{
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		Source:       source,
		RegistryAddr: registryAddr,
		Images:       images,
	}, nil
}

func (g *Generator) collectImages(ctx context.Context, project *model.ContainerHiveProject, registryAddr string) (string, []ImageReport, error) {
	switch g.source {
	case SourceTar:
		return g.fromTar(ctx, project)
	case SourceRegistry:
		return g.fromRegistry(ctx, project, registryAddr)
	case SourceAuto:
		if os.Getenv("CI") != "" {
			return g.fromRegistry(ctx, project, registryAddr)
		}
		return g.fromTar(ctx, project)
	default:
		return g.fromTar(ctx, project)
	}
}

func (g *Generator) fromTar(ctx context.Context, project *model.ContainerHiveProject) (string, []ImageReport, error) {
	scanner, err := NewTarScanner(g.distPath, project)
	if err != nil {
		return "tar", nil, err
	}

	images, err := scanner.Scan()
	if err != nil {
		return "tar", nil, err
	}

	return "tar", images, nil
}

func (g *Generator) fromRegistry(ctx context.Context, project *model.ContainerHiveProject, registryAddr string) (string, []ImageReport, error) {
	scanner := NewRegistryScanner(registryAddr, project)
	images, err := scanner.Scan(ctx)
	if err != nil {
		return "registry", nil, err
	}

	return "registry", images, nil
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
	html = replacePlaceholder(html, string(reportJSON))
	html = strings.ReplaceAll(html, "/*INJECT_GENERATED_AT*/", report.GeneratedAt)
	html = strings.ReplaceAll(html, "/*INJECT_SOURCE*/", report.Source)
	html = strings.ReplaceAll(html, "/*INJECT_REGISTRY*/", report.RegistryAddr)

	return []byte(html), nil
}

func replacePlaceholder(html, data string) string {
	for i := 0; i < len(html)-len("/*INJECT_JSON_DATA*/"); i++ {
		if html[i:i+len("/*INJECT_JSON_DATA*/")] == "/*INJECT_JSON_DATA*/" {
			return html[:i] + data + html[i+len("/*INJECT_JSON_DATA*/"):]
		}
	}
	return html
}
