package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/urfave/cli/v3"
)

func main() {
	app := &cli.Command{
		Name:  "ch",
		Usage: "ContainerHive - declarative container image management",
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
		},
		Commands: []*cli.Command{
			generateCmd(),
			buildCmd(),
			finalizeCmd(),
			testCmd(),
			sbomCmd(),
			verifyCmd(),
		},
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := app.Run(ctx, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
