package docker

import (
	"context"
	"errors"

	"github.com/ContainerHive/ContainerHive/internal/ocistore"
	dockerClient "github.com/docker/docker/client"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/remote"
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
func (c *Client) PullImage(_ context.Context, imageRef string) (string, error) {
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return "", errors.Join(errors.New("invalid image reference"), err)
	}

	img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return "", errors.Join(errors.New("failed to pull image from registry"), err)
	}

	tag, err := name.NewTag(imageRef)
	if err != nil {
		return "", errors.Join(errors.New("invalid image tag"), err)
	}

	if _, err := daemon.Write(tag, img); err != nil {
		return "", errors.Join(errors.New("failed to load pulled image into Docker"), err)
	}

	return imageRef, nil
}

// LoadImageFromTar loads an OCI image from a tar archive into the local Docker daemon.
func (c *Client) LoadImageFromTar(_ context.Context, tarPath string) (string, error) {
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

	if _, err := daemon.Write(tag, ociImage.Image); err != nil {
		return "", errors.Join(errors.New("failed to load image into Docker"), err)
	}

	return imageName, nil
}
