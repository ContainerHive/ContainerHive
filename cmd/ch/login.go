package main

import (
	"context"
	"errors"
	"log"
	"os"

	"github.com/timo-reymann/ContainerHive/pkg/login"
	"github.com/urfave/cli/v3"
)

func loginCmd() *cli.Command {
	return &cli.Command{
		Name:      "login",
		Usage:     "Log in to a registry",
		ArgsUsage: "REGISTRY",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "username",
				Aliases: []string{"u"},
				Usage:   "Username",
			},
			&cli.StringFlag{
				Name:    "password",
				Aliases: []string{"p"},
				Usage:   "Password",
			},
			&cli.BoolFlag{
				Name:  "password-stdin",
				Usage: "Take the password from stdin",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			args := cmd.Args()
			if args.Len() != 1 {
				return errors.New("expected exactly one argument: REGISTRY")
			}

			opts := login.Options{
				ServerAddress: args.First(),
				Username:      cmd.String("username"),
				Password:      cmd.String("password"),
				ConfigDir:     os.Getenv("DOCKER_CONFIG"),
			}

			if cmd.Bool("password-stdin") {
				opts.PasswordStdin = os.Stdin
			}

			configPath, err := login.Login(opts)
			if err != nil {
				return err
			}

			log.Printf("logged in via %s", configPath)
			return nil
		},
	}
}
