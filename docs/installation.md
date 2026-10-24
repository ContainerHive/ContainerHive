# Install ContainerHive

ContainerHive supports a variety of platforms.

## Homebrew

```sh
brew tap ContainerHive/ContainerHive
brew install ContainerHive
```

## Containerized

If you prefer to use containerized workflows, use the provided OCI image.

```sh
docker run --rm -it -v $PWD:/workspace containerhive/containerhive
```

## Manual

Pick your platform (`linux-amd64`, `linux-arm64`, `darwin-amd64` or `darwin-arm64`) and run:

```bash
PLATFORM=linux-amd64  # or linux-arm64, darwin-amd64, darwin-arm64
VERSION=$(curl -Lso /dev/null -w %{url_effective} https://github.com/ContainerHive/ContainerHive/releases/latest | grep -o '[^/]*$')
curl -LO "https://github.com/ContainerHive/ContainerHive/releases/download/${VERSION}/${PLATFORM}.tar.zst" && \
tar --zstd -xf "${PLATFORM}.tar.zst" ch && \
chmod +x ch && \
sudo mv ch /usr/local/bin/ch
```

## Next steps

Continue with [Get started](./getting-started.md). To run ContainerHive from pre-commit, see the [pre-commit guide](usage/pre-commit.md).

## Where to find the latest release

### Binaries

Binaries for all platforms can be found on
the [latest release page](https://github.com/ContainerHive/ContainerHive/releases/latest).

### Docker

For the Docker image, check [Docker Hub](https://hub.docker.com/r/containerhive/containerhive).
