package cli

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/ContainerHive/ContainerHive/pkg/login"
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

			username := cmd.String("username")
			password := cmd.String("password")

			if cmd.Bool("password-stdin") {
				data, err := io.ReadAll(os.Stdin)
				if err != nil {
					return err
				}
				password = strings.TrimRight(string(data), "\r\n")
			}

			if username == "" && password == "" {
				slog.Warn("Skipping login: no credentials provided", "registry", args.First())
				return nil
			}

			configPath, err := login.Login(login.Options{
				ServerAddress: args.First(),
				Username:      username,
				Password:      password,
				ConfigDir:     os.Getenv("DOCKER_CONFIG"),
			})
			if err != nil {
				return err
			}

			slog.Info("Logged in", "config", configPath)
			return nil
		},
	}
}
