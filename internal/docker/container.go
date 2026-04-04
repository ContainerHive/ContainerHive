package docker

import (
	"context"
	"fmt"
	"io"

	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
)

// ContainerStatus holds the inspected state of a named container.
type ContainerStatus struct {
	State    string // e.g. "running", "exited", "not found"
	Image    string
	HostPort int // 0 when not bound or not found
}

// ContainerRun pulls the image if absent, creates a privileged container
// binding hostPort on the host to containerPort inside the container, then starts it.
// cmd overrides the container's default command (nil uses the image default).
// Returns an error if the container already exists and is running.
func (c *Client) ContainerRun(ctx context.Context, name, imageRef string, hostPort, containerPort int, cmd []string) error {
	// Pull image if not present locally.
	if _, _, err := c.docker.ImageInspectWithRaw(ctx, imageRef); err != nil {
		if !errdefs.IsNotFound(err) {
			return fmt.Errorf("inspect image %s: %w", imageRef, err)
		}
		rc, err := c.docker.ImagePull(ctx, imageRef, image.PullOptions{})
		if err != nil {
			return fmt.Errorf("pull image %s: %w", imageRef, err)
		}
		_, _ = io.Copy(io.Discard, rc)
		rc.Close()
	}

	// Check existing container.
	resp, err := c.docker.ContainerInspect(ctx, name)
	if err != nil && !errdefs.IsNotFound(err) {
		return fmt.Errorf("inspect container %s: %w", name, err)
	}
	if err == nil {
		if resp.State.Running {
			return fmt.Errorf("container %s is already running", name)
		}
		// Container exists but stopped — remove it so it can be recreated with the correct config.
		if err := c.docker.ContainerRemove(ctx, resp.ID, container.RemoveOptions{}); err != nil {
			return fmt.Errorf("remove stopped container %s: %w", name, err)
		}
	}

	// Create the container.
	internalPort := nat.Port(fmt.Sprintf("%d/tcp", containerPort))
	cfg := &container.Config{
		Image: imageRef,
		Cmd:   cmd,
		ExposedPorts: nat.PortSet{
			internalPort: struct{}{},
		},
	}
	hostCfg := &container.HostConfig{
		Privileged: true,
		PortBindings: nat.PortMap{
			internalPort: []nat.PortBinding{
				{HostPort: fmt.Sprintf("%d", hostPort)},
			},
		},
	}
	created, err := c.docker.ContainerCreate(ctx, cfg, hostCfg, nil, nil, name)
	if err != nil {
		return fmt.Errorf("create container %s: %w", name, err)
	}
	return c.docker.ContainerStart(ctx, created.ID, container.StartOptions{})
}

// ContainerStop stops the named container. It is a no-op if the container is
// already stopped or does not exist. When remove is true the container is also removed.
func (c *Client) ContainerStop(ctx context.Context, name string, remove bool) error {
	_, err := c.docker.ContainerInspect(ctx, name)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("inspect container %s: %w", name, err)
	}
	if err := c.docker.ContainerStop(ctx, name, container.StopOptions{}); err != nil {
		return fmt.Errorf("stop container %s: %w", name, err)
	}
	if remove {
		if err := c.docker.ContainerRemove(ctx, name, container.RemoveOptions{}); err != nil {
			return fmt.Errorf("remove container %s: %w", name, err)
		}
	}
	return nil
}

// ContainerInspect returns the status of the named container.
// When the container does not exist, State is "not found" and no error is returned.
func (c *Client) ContainerInspect(ctx context.Context, name string) (ContainerStatus, error) {
	resp, err := c.docker.ContainerInspect(ctx, name)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return ContainerStatus{State: "not found"}, nil
		}
		return ContainerStatus{}, fmt.Errorf("inspect container %s: %w", name, err)
	}

	hostPort := 0
	for _, bindings := range resp.NetworkSettings.Ports {
		if len(bindings) > 0 {
			_, err := fmt.Sscanf(bindings[0].HostPort, "%d", &hostPort)
			if err != nil {
				hostPort = 0
			}
			break
		}
	}

	return ContainerStatus{
		State:    resp.State.Status,
		Image:    resp.Config.Image,
		HostPort: hostPort,
	}, nil
}

// ContainerLogs streams stdout and stderr of the named container to w.
// When follow is true the stream stays open until ctx is cancelled or the container exits.
func (c *Client) ContainerLogs(ctx context.Context, name string, w io.Writer, follow bool) error {
	if _, err := c.docker.ContainerInspect(ctx, name); err != nil {
		if errdefs.IsNotFound(err) {
			return fmt.Errorf("container %s not found", name)
		}
		return fmt.Errorf("inspect container %s: %w", name, err)
	}

	rc, err := c.docker.ContainerLogs(ctx, name, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
	})
	if err != nil {
		return fmt.Errorf("logs container %s: %w", name, err)
	}
	defer rc.Close()
	_, err = stdcopy.StdCopy(w, w, rc)
	return err
}
