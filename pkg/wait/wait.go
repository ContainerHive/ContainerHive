package wait

import (
	"context"
	"fmt"
	"sync"
	"time"

	dockerClient "github.com/docker/docker/client"

	"github.com/ContainerHive/ContainerHive/pkg/build"
)

const pollInterval = 500 * time.Millisecond

// Target represents something to wait for.
type Target struct {
	Name    string
	CheckFn func(ctx context.Context) error
}

// BuildkitdTarget returns a target that waits for BuildKit daemon.
// It uses build.NewClient which respects $BUILDKIT_HOST.
func BuildkitdTarget() Target {
	return Target{
		Name: "buildkitd",
		CheckFn: func(ctx context.Context) error {
			c, err := build.NewClient(ctx, "")
			if err != nil {
				return err
			}
			defer c.Close()

			_, err = c.Version(ctx)
			return err
		},
	}
}

// DockerSocketTarget returns a target that waits for the Docker daemon.
// It uses the Docker client which respects $DOCKER_HOST.
func DockerSocketTarget() Target {
	return Target{
		Name: "docker-socket",
		CheckFn: func(ctx context.Context) error {
			c, err := dockerClient.NewClientWithOpts(dockerClient.FromEnv, dockerClient.WithAPIVersionNegotiation())
			if err != nil {
				return err
			}
			defer c.Close()

			_, err = c.Ping(ctx)
			return err
		},
	}
}

// TargetResult holds the outcome for a single target.
type TargetResult struct {
	Name string
	Err  error
}

// WaitAll waits for all targets in parallel, returning when all succeed or the context expires.
func WaitAll(ctx context.Context, targets []Target) []TargetResult {
	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		results = make([]TargetResult, len(targets))
	)

	for i, t := range targets {
		wg.Add(1)
		go func(idx int, target Target) {
			defer wg.Done()
			err := poll(ctx, target)
			mu.Lock()
			results[idx] = TargetResult{Name: target.Name, Err: err}
			mu.Unlock()
		}(i, t)
	}

	wg.Wait()
	return results
}

func poll(ctx context.Context, target Target) error {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	var lastErr error
	for {
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return fmt.Errorf("timed out waiting for %s: %w", target.Name, lastErr)
			}
			return fmt.Errorf("timed out waiting for %s", target.Name)
		case <-ticker.C:
			if err := target.CheckFn(ctx); err != nil {
				lastErr = err
				continue
			}
			return nil
		}
	}
}
