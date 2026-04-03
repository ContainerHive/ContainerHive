package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/timo-reymann/ContainerHive/pkg/devenv"
	"github.com/urfave/cli/v3"
)

func devCmd() *cli.Command {
	return &cli.Command{
		Name:  "dev",
		Usage: "Local development environment helpers",
		Commands: []*cli.Command{
			buildkitdCmd(),
		},
	}
}

func buildkitdCmd() *cli.Command {
	return &cli.Command{
		Name:  "buildkitd",
		Usage: "Manage a local BuildKit daemon container",
		Commands: []*cli.Command{
			buildkitdStartCmd(),
			buildkitdStopCmd(),
			buildkitdStatusCmd(),
			buildkitdLogsCmd(),
		},
	}
}

func buildkitdStartCmd() *cli.Command {
	return &cli.Command{
		Name:  "start",
		Usage: "Pull image if needed, then create and start the buildkitd container",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "image",
				Usage: "BuildKit image to use (image:tag). Defaults to the version configured in hive.yml template_options or the bundled version.",
			},
			&cli.IntFlag{
				Name:  "port",
				Value: devenv.BuildkitdDefaultPort,
				Usage: "Host port to bind",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			imageRef := cmd.String("image")
			if imageRef == "" {
				project, err := discoverProject(ctx, cmd)
				if err != nil {
					return err
				}
				imageRef = devenv.ResolveImage(project.Config.TemplateOptions)
			}

			b, err := devenv.NewBuildkitd()
			if err != nil {
				return err
			}
			defer b.Close()

			hostPort := cmd.Int("port")
			if err := b.Start(ctx, imageRef, hostPort); err != nil {
				return err
			}

			fmt.Printf("export BUILDKIT_HOST=tcp://localhost:%d\n", hostPort)
			return nil
		},
	}
}

func buildkitdStopCmd() *cli.Command {
	return &cli.Command{
		Name:  "stop",
		Usage: "Stop the buildkitd container",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "remove",
				Usage: "Also remove the container after stopping",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			b, err := devenv.NewBuildkitd()
			if err != nil {
				return err
			}
			defer b.Close()
			return b.Stop(ctx, cmd.Bool("remove"))
		},
	}
}

func buildkitdStatusCmd() *cli.Command {
	return &cli.Command{
		Name:  "status",
		Usage: "Show the status of the buildkitd container",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			b, err := devenv.NewBuildkitd()
			if err != nil {
				return err
			}
			defer b.Close()

			status, err := b.Status(ctx)
			if err != nil {
				return err
			}

			log.Printf("state:         %s", status.State)
			if status.Image != "" {
				log.Printf("image:         %s", status.Image)
			}
			if status.HostPort > 0 {
				log.Printf("port:          %d", status.HostPort)
				log.Printf("BUILDKIT_HOST: tcp://localhost:%d", status.HostPort)
			}
			return nil
		},
	}
}

func buildkitdLogsCmd() *cli.Command {
	return &cli.Command{
		Name:  "logs",
		Usage: "Show logs from the buildkitd container",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "follow",
				Aliases: []string{"f"},
				Usage:   "Follow log output",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			b, err := devenv.NewBuildkitd()
			if err != nil {
				return err
			}
			defer b.Close()
			return b.Logs(ctx, os.Stdout, cmd.Bool("follow"))
		},
	}
}
