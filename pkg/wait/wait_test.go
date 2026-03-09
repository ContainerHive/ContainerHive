package wait

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/timo-reymann/ContainerHive/internal/testutil"
)

func TestBuildkitdTarget(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := t.Context()

	buildkitC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        testutil.BuildKitImage(),
			ExposedPorts: []string{"1234/tcp"},
			Cmd:          []string{"--addr", "tcp://0.0.0.0:1234"},
			HostConfigModifier: func(hc *container.HostConfig) {
				hc.Privileged = true
			},
			WaitingFor: wait.ForListeningPort("1234/tcp"),
		},
		Started: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { buildkitC.Terminate(ctx) })

	host, err := buildkitC.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := buildkitC.MappedPort(ctx, "1234/tcp")
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("BUILDKIT_HOST", fmt.Sprintf("tcp://%s:%s", host, port.Port()))

	target := BuildkitdTarget()

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	results := WaitAll(timeoutCtx, []Target{target})
	if results[0].Err != nil {
		t.Fatalf("expected buildkitd to be ready, got: %v", results[0].Err)
	}
}

func TestDockerSocketTarget(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := t.Context()

	dindC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "docker:dind",
			ExposedPorts: []string{"2375/tcp"},
			HostConfigModifier: func(hc *container.HostConfig) {
				hc.Privileged = true
			},
			Env: map[string]string{
				"DOCKER_TLS_CERTDIR": "",
			},
			WaitingFor: wait.ForHTTP("/_ping").WithPort("2375/tcp"),
		},
		Started: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { dindC.Terminate(ctx) })

	host, err := dindC.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := dindC.MappedPort(ctx, "2375/tcp")
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("DOCKER_HOST", fmt.Sprintf("tcp://%s:%s", host, port.Port()))

	target := DockerSocketTarget()

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	results := WaitAll(timeoutCtx, []Target{target})
	if results[0].Err != nil {
		t.Fatalf("expected docker socket to be ready, got: %v", results[0].Err)
	}
}

func TestWaitAll_Timeout(t *testing.T) {
	target := Target{
		Name: "never-ready",
		CheckFn: func(ctx context.Context) error {
			return fmt.Errorf("not ready")
		},
	}

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Second)
	defer cancel()

	results := WaitAll(ctx, []Target{target})
	if results[0].Err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestWaitAll_Parallel(t *testing.T) {
	ready1 := make(chan struct{})
	ready2 := make(chan struct{})

	target1 := Target{
		Name: "target-1",
		CheckFn: func(ctx context.Context) error {
			select {
			case <-ready1:
				return nil
			default:
				return fmt.Errorf("not ready")
			}
		},
	}
	target2 := Target{
		Name: "target-2",
		CheckFn: func(ctx context.Context) error {
			select {
			case <-ready2:
				return nil
			default:
				return fmt.Errorf("not ready")
			}
		},
	}

	// Make both ready after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		close(ready1)
		close(ready2)
	}()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	results := WaitAll(ctx, []Target{target1, target2})
	for _, r := range results {
		if r.Err != nil {
			t.Fatalf("expected %s to be ready, got: %v", r.Name, r.Err)
		}
	}
}
