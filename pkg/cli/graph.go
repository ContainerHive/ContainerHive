package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/ContainerHive/ContainerHive/pkg/deps"
	"github.com/urfave/cli/v3"
)

const (
	graphFormatMermaid = "mermaid"
	graphFormatDot     = "dot"
)

func graphCmd() *cli.Command {
	return &cli.Command{
		Name:  "graph",
		Usage: "Print the dependency graph of all images and variants",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "format",
				Usage: "Output format: " + graphFormatMermaid + " or " + graphFormatDot,
				Value: graphFormatMermaid,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			format := cmd.String("format")
			if err := validateGraphFormat(format); err != nil {
				return err
			}

			project, err := discoverProject(ctx, cmd)
			if err != nil {
				return err
			}

			return writeGraph(cmd.Root().Writer, deps.ProjectGraph(project), format)
		},
	}
}

func validateGraphFormat(format string) error {
	switch format {
	case graphFormatMermaid, graphFormatDot:
		return nil
	}
	return fmt.Errorf("unknown graph format %q, want %s or %s", format, graphFormatMermaid, graphFormatDot)
}

func writeGraph(w io.Writer, graph *deps.Graph, format string) error {
	if err := validateGraphFormat(format); err != nil {
		return err
	}
	if format == graphFormatDot {
		_, err := io.WriteString(w, graph.Dot())
		return err
	}
	_, err := io.WriteString(w, graph.Mermaid())
	return err
}
