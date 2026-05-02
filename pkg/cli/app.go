package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/ContainerHive/ContainerHive/pkg/discovery"
	"github.com/ContainerHive/ContainerHive/pkg/logging"
	"github.com/ContainerHive/ContainerHive/pkg/model"
	"github.com/ContainerHive/ContainerHive/pkg/registry"
	"github.com/ContainerHive/ContainerHive/pkg/rendering"
	"github.com/ContainerHive/ContainerHive/pkg/version"
	"github.com/urfave/cli/v3"
)

func discoverProject(ctx context.Context, cmd *cli.Command) (*model.ContainerHiveProject, error) {
	projectRoot := cmd.String("project")

	project, err := discovery.DiscoverProject(ctx, projectRoot)
	if err != nil {
		return nil, fmt.Errorf("discovery failed: %w", err)
	}

	return project, nil
}

func getDistPath(cmd *cli.Command) string {
	projectRoot := cmd.String("project")
	return filepath.Join(projectRoot, model.DistDirName)
}

func setupRegistry(ctx context.Context, distPath string, config *model.RegistryConfig) (*registry.Registry, error) {
	reg, err := registry.NewRegistry(filepath.Join(distPath, ".registry"), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry: %w", err)
	}
	if err := reg.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start registry: %w", err)
	}
	slog.Info("Registry started", "local", reg.IsLocal(), "address", reg.Address())

	return reg, nil
}

func generateProject(ctx context.Context, cmd *cli.Command) error {
	project, err := discoverProject(ctx, cmd)
	if err != nil {
		return err
	}

	distPath := getDistPath(cmd)
	if err := rendering.RenderProject(ctx, project, distPath); err != nil {
		return fmt.Errorf("rendering failed: %w", err)
	}

	slog.Info("Rendered images to dist/", "count", len(project.ImagesByIdentifier), "path", distPath)
	return nil
}

func NewApp() *cli.Command {
	app := &cli.Command{
		Name:                  "ch",
		Usage:                 "ContainerHive - Swarm it. Build it. Run it. Managing container base and library images has never been easier.",
		Version:               version.Get(),
		EnableShellCompletion: true,
		Suggest:               true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "project",
				Aliases: []string{"p"},
				Usage:   "Project root directory",
				Value:   ".",
			},
			&cli.StringFlag{
				Name:  "build-id",
				Usage: "Build ID to append to tags as +<id>",
			},
			&cli.StringFlag{
				Name:    "log-level",
				Usage:   "Log level (debug, info, warn, error)",
				Sources: cli.EnvVars("LOG_LEVEL"),
				Value:   "info",
			},
			&cli.BoolFlag{
				Name:    "generate",
				Aliases: []string{"g"},
				Usage:   "Run generate before the command",
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			logging.Setup(os.Stderr, cmd.String("log-level"))
			return ctx, nil
		},
		Commands: []*cli.Command{
			generateCmd(),
			buildCmd(),
			finalizeCmd(),
			testCmd(),
			sbomCmd(),
			verifyCmd(),
			templateCmd(),
			waitCmd(),
			loginCmd(),
			licenseCmd(),
			devCmd(),
			reportCmd(),
			mcpCmd(),
		},
	}

	return app
}

func Run() {
	app := NewApp()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := app.Run(ctx, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
