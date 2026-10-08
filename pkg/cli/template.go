package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"text/template/parse"

	"github.com/ContainerHive/ContainerHive/pkg/ci"
	"github.com/ContainerHive/ContainerHive/pkg/templating"
	"github.com/ContainerHive/ContainerHive/pkg/version"
	"github.com/urfave/cli/v3"
)

func templateCmd() *cli.Command {
	return &cli.Command{
		Name:  "template",
		Usage: "Generate files from templates",
		Commands: []*cli.Command{
			templateCICmd(),
			templateCustomCmd(),
		},
	}
}

func templateCICmd() *cli.Command {
	return &cli.Command{
		Name:  "ci",
		Usage: "Generate CI pipeline configuration",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "provider",
				Usage:    "CI provider (gitlab, github)",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "output",
				Usage: "Output file (default: stdout)",
			},
			&cli.StringFlag{
				Name:  "template-dir",
				Usage: "Custom template directory (overrides built-in templates)",
			},
			&cli.BoolFlag{
				Name:  "artifacts",
				Usage: "Upload/download build artifacts between jobs",
			},
			&cli.StringFlag{
				Name:  "version",
				Usage: "CH CLI version to use in CI templates (default: current CLI version)",
			},
			&cli.StringFlag{
				Name:  "image-name",
				Usage: "Container image name for the CH CLI (default: containerhive/containerhive)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			project, err := discoverProject(ctx, cmd)
			if err != nil {
				return err
			}

			ciCtx, err := ci.BuildCIContext(project, cmd.Bool("artifacts"))
			if err != nil {
				return fmt.Errorf("failed to build CI context: %w", err)
			}

			projectPath := cmd.String("project")
			if projectPath != "" && projectPath != "." {
				ciCtx.ProjectPath = projectPath
			}

			ciCtx.Command = buildCICommand(cmd)

			versionOverride := cmd.String("version")
			if versionOverride != "" {
				ciCtx.Version = versionOverride
			} else {
				ciCtx.Version = version.Get()
			}

			imageName := cmd.String("image-name")
			if imageName != "" {
				ciCtx.ImageName = imageName
			} else {
				ciCtx.ImageName = "containerhive/containerhive"
			}

			result, err := ci.Generate(cmd.String("provider"), ciCtx, cmd.String("template-dir"))
			if err != nil {
				return fmt.Errorf("failed to generate CI config: %w", err)
			}

			return writeOutput(cmd.String("output"), result)
		},
	}
}

func templateCustomCmd() *cli.Command {
	return &cli.Command{
		Name:  "custom",
		Usage: "Render a custom Go template with project context",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "template",
				Usage:    "Path to Go template file (.gotpl)",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "output",
				Usage: "Output file (default: stdout)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			project, err := discoverProject(ctx, cmd)
			if err != nil {
				return err
			}

			templatePath := cmd.String("template")
			content, err := os.ReadFile(templatePath)
			if err != nil {
				return fmt.Errorf("failed to read template: %w", err)
			}

			ciCtx, err := ci.BuildCIContext(project, false)
			if err != nil {
				return fmt.Errorf("failed to build CI context: %w", err)
			}

			output := cmd.String("output")
			perImage, err := outputPathIsPerImage(output, ciCtx.TemplateOptions)
			if err != nil {
				return err
			}

			if perImage {
				return writePerImageOutputs(output, templatePath, string(content), ciCtx)
			}

			result, err := templating.RenderStringWithOptions(templatePath, string(content), ciCtx, ciCtx.TemplateOptions)
			if err != nil {
				return fmt.Errorf("failed to render template: %w", err)
			}

			renderedOutput, err := renderOutputPath(output, ciCtx, ciCtx.TemplateOptions)
			if err != nil {
				return err
			}

			return writeOutput(renderedOutput, result)
		},
	}
}

// imageTemplateContext binds the current image as .Image alongside the full
// CI context, used when --output renders one file per image.
type imageTemplateContext struct {
	*ci.CIContext
	Image ci.CIImage
}

// outputPathIsPerImage reports whether the --output template references
// .Image, which switches template custom into one-file-per-image mode.
func outputPathIsPerImage(output string, options map[string]string) (bool, error) {
	tpl, err := template.New("output").Funcs(templating.FuncMapWithOptions(options)).Parse(output)
	if err != nil {
		return false, fmt.Errorf("invalid --output template: %w", err)
	}
	return nodeReferencesImage(tpl.Tree.Root), nil
}

// renderOutputPath renders the --output value as a Go template.
func renderOutputPath(output string, data any, options map[string]string) (string, error) {
	rendered, err := templating.RenderStringWithOptions("output", output, data, options)
	if err != nil {
		return "", fmt.Errorf("failed to render --output: %w", err)
	}
	return string(rendered), nil
}

// writePerImageOutputs renders the template once per image, with .Image
// bound in both the output path and the template content.
func writePerImageOutputs(output, templatePath, content string, ciCtx *ci.CIContext) error {
	if len(ciCtx.Images) == 0 {
		return fmt.Errorf("--output references .Image but the project has no images")
	}

	for _, img := range ciCtx.Images {
		data := &imageTemplateContext{CIContext: ciCtx, Image: img}

		outPath, err := renderOutputPath(output, data, ciCtx.TemplateOptions)
		if err != nil {
			return err
		}

		result, err := templating.RenderStringWithOptions(templatePath, content, data, ciCtx.TemplateOptions)
		if err != nil {
			return fmt.Errorf("failed to render template for image %s: %w", img.Name, err)
		}

		if err := writeOutput(outPath, result); err != nil {
			return fmt.Errorf("failed to write %s: %w", outPath, err)
		}
	}
	return nil
}

// nodeReferencesImage reports whether the parse tree references .Image or
// $.Image inside an action (literal text is ignored).
func nodeReferencesImage(n parse.Node) bool {
	switch t := n.(type) {
	case *parse.ListNode:
		if t == nil {
			return false
		}
		for _, child := range t.Nodes {
			if nodeReferencesImage(child) {
				return true
			}
		}
	case *parse.ActionNode:
		return nodeReferencesImage(t.Pipe)
	case *parse.PipeNode:
		for _, decl := range t.Decl {
			if nodeReferencesImage(decl) {
				return true
			}
		}
		for _, cmd := range t.Cmds {
			if nodeReferencesImage(cmd) {
				return true
			}
		}
	case *parse.CommandNode:
		for _, arg := range t.Args {
			if nodeReferencesImage(arg) {
				return true
			}
		}
	case *parse.IfNode:
		return nodeReferencesImage(&t.BranchNode)
	case *parse.RangeNode:
		return nodeReferencesImage(&t.BranchNode)
	case *parse.WithNode:
		return nodeReferencesImage(&t.BranchNode)
	case *parse.BranchNode:
		return nodeReferencesImage(t.Pipe) || nodeReferencesImage(t.List) || nodeReferencesImage(t.ElseList)
	case *parse.TemplateNode:
		return nodeReferencesImage(t.Pipe)
	case *parse.ChainNode:
		return matchesImageIdent(t.Field) || nodeReferencesImage(t.Node)
	case *parse.FieldNode:
		return matchesImageIdent(t.Ident)
	case *parse.VariableNode:
		return matchesImageIdent(t.Ident)
	}
	return false
}

// matchesImageIdent reports whether an identifier chain starts at the root
// and then reads the Image field ($.Image, $.Image.Name, .Image, .Image.Name).
func matchesImageIdent(ident []string) bool {
	if len(ident) > 0 && ident[0] == "$" {
		ident = ident[1:]
	}
	return len(ident) > 0 && ident[0] == "Image"
}

func buildCICommand(cmd *cli.Command) string {
	parts := []string{"ch"}
	if project := cmd.String("project"); project != "" && project != "." {
		parts = append(parts, "--project", project)
	}
	parts = append(parts, "template", "ci", "--provider", cmd.String("provider"))
	if cmd.Bool("artifacts") {
		parts = append(parts, "--artifacts")
	}
	if dir := cmd.String("template-dir"); dir != "" {
		parts = append(parts, "--template-dir", dir)
	}
	if output := cmd.String("output"); output != "" {
		parts = append(parts, "--output", output)
	}
	if imageName := cmd.String("image-name"); imageName != "" {
		parts = append(parts, "--image-name", imageName)
	}
	return strings.Join(parts, " ")
}

func writeOutput(outputPath string, data []byte) error {
	if outputPath == "" {
		_, err := os.Stdout.Write(data)
		return err
	}
	if dir := filepath.Dir(outputPath); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory %s: %w", dir, err)
		}
	}
	return os.WriteFile(outputPath, data, 0644)
}
