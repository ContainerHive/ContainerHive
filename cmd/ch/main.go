package main

import (
	"context"
	"fmt"
	"os"

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

	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
