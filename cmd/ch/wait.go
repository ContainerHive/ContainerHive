package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ContainerHive/ContainerHive/pkg/wait"
	"github.com/urfave/cli/v3"
)

func waitCmd() *cli.Command {
	return &cli.Command{
		Name:  "wait",
		Usage: "Wait for infrastructure dependencies to become available",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "buildkitd",
				Usage: "Wait for BuildKit daemon (uses $BUILDKIT_HOST, default tcp://127.0.0.1:8372)",
			},
			&cli.BoolFlag{
				Name:  "docker-socket",
				Usage: "Wait for Docker daemon (uses $DOCKER_HOST, default unix:///var/run/docker.sock)",
			},
			&cli.DurationFlag{
				Name:  "timeout",
				Usage: "Maximum time to wait",
				Value: 1 * time.Minute,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			var targets []wait.Target

			if cmd.Bool("buildkitd") {
				targets = append(targets, wait.BuildkitdTarget())
			}
			if cmd.Bool("docker-socket") {
				targets = append(targets, wait.DockerSocketTarget())
			}

			if len(targets) == 0 {
				return fmt.Errorf("at least one wait target is required (--buildkitd, --docker-socket)")
			}

			timeout := cmd.Duration("timeout")
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			slog.Info("Waiting for targets", "count", len(targets), "timeout", timeout)

			results := wait.WaitAll(ctx, targets)

			var errs []error
			for _, r := range results {
				if r.Err != nil {
					errs = append(errs, r.Err)
				} else {
					slog.Info("Target is ready", "name", r.Name)
				}
			}

			if len(errs) > 0 {
				for _, e := range errs {
					slog.Error("Target failed", "error", e)
				}
				return fmt.Errorf("%d target(s) failed to become ready", len(errs))
			}

			return nil
		},
	}
}
