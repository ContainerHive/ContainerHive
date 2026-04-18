package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/timo-reymann/ContainerHive/pkg/discovery"
	"github.com/timo-reymann/ContainerHive/pkg/logging"
	"github.com/timo-reymann/ContainerHive/pkg/model"
	"github.com/timo-reymann/ContainerHive/pkg/registry"
	"github.com/timo-reymann/ContainerHive/pkg/version"
	"github.com/urfave/cli/v3"
)

// discoverProject is a helper method that discovers the project and handles common validation
func discoverProject(ctx context.Context, cmd *cli.Command) (*model.ContainerHiveProject, error) {
	projectRoot := cmd.String("project")

	project, err := discovery.DiscoverProject(ctx, projectRoot)
	if err != nil {
		return nil, fmt.Errorf("discovery failed: %w", err)
	}

	return project, nil
}

// getDistPath returns the distribution path for the project
func getDistPath(cmd *cli.Command) string {
	projectRoot := cmd.String("project")
	return filepath.Join(projectRoot, "dist")
}

// setupRegistry creates, starts, and returns a registry instance with proper cleanup
func setupRegistry(ctx context.Context, distPath string, config *model.RegistryConfig) (*registry.Registry, error) {
	reg, err := registry.NewRegistry(filepath.Join(distPath, ".registry"), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry: %w", err)
	}
	if err := reg.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start registry: %w", err)
	}
	// Note: caller is responsible for deferring reg.Stop(ctx)
	slog.Info("Registry started", "local", reg.IsLocal(), "address", reg.Address())

	return reg, nil
}

func main() {
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
		},
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := app.Run(ctx, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
