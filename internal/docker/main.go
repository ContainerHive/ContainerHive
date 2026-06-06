package docker

import (
	"context"
	"errors"
	"io"

	"github.com/ContainerHive/ContainerHive/internal/ocistore"
	dockerClient "github.com/docker/docker/client"
	"github.com/docker/docker/api/types/image"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
)

// Client wraps the Docker client library for image operations.
type Client struct {
	docker *dockerClient.Client
}

// Close releases the underlying Docker client connection.
func (c *Client) Close() error {
	return c.docker.Close()
}

// NewClient creates a new Docker client configured from environment variables.
func NewClient() (*Client, error) {
	docker, err := dockerClient.NewClientWithOpts(dockerClient.FromEnv, dockerClient.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return &Client{
		docker,
	}, nil
}

// HasImage checks whether the Docker daemon already has the given image.
func (c *Client) HasImage(ctx context.Context, imageRef string) bool {
	_, _, err := c.docker.ImageInspectWithRaw(ctx, imageRef)
	return err == nil
}

// PullImage pulls an image from a remote registry into the local Docker daemon.
func (c *Client) PullImage(ctx context.Context, imageRef string) (string, error) {
	rc, err := c.docker.ImagePull(ctx, imageRef, image.PullOptions{})
	if err != nil {
		return "", errors.Join(errors.New("failed to pull image"), err)
	}
	defer rc.Close()
	if _, err := io.Copy(io.Discard, rc); err != nil {
		return "", errors.Join(errors.New("failed to complete image pull"), err)
	}
	return imageRef, nil
}

// LoadImageFromTar loads an OCI image from a tar archive into the local Docker daemon.
func (c *Client) LoadImageFromTar(ctx context.Context, tarPath string) (string, error) {
	ociImage, err := ocistore.ImageFromTar(tarPath)
	if err != nil {
		return "", err
	}
	defer ociImage.Cleanup()

	imageName := ociImage.Annotations["io.containerd.image.name"]
	if imageName == "" {
		return "", errors.New("no image name annotation in OCI index")
	}

	tag, err := name.NewTag(imageName)
	if err != nil {
		return "", errors.Join(errors.New("invalid image name"), err)
	}

	if _, err := daemon.Write(tag, ociImage.Image, daemon.WithContext(ctx)); err != nil {
		return "", errors.Join(errors.New("failed to load image into Docker"), err)
	}

	return imageName, nil
}
