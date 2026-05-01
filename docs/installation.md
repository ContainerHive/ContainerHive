# Install ContainerHive

ContainerHive supports a variety of platforms.

## Homebrew

```sh
brew tap ContainerHive/homebrew-ContainerHive
brew install containerhive
```

## Containerized

If you prefer to use containerized workflows, use the provided OCI image.

```sh
docker run --rm -it -v $PWD:/workspace containerhive/containerhive
```

## Pre-commit

ContainerHive provides [pre-commit](https://pre-commit.com) hooks for generating CI pipelines, verifying project structure, and running arbitrary ContainerHive commands.

```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/ContainerHive/pre-commit
    rev: v0.13.2
    hooks:
      - id: template
        args: [ci, --provider, github, --output, .github/workflows/main.yml]
```

See the [pre-commit usage guide](usage/pre-commit.md) for all available hooks and configuration options.

## Manual

### Linux (64-bit)

```bash
curl -LO https://github.com/ContainerHive/ContainerHive/releases/download/$(curl -Lso /dev/null -w %{url_effective} https://github.com/ContainerHive/ContainerHive/releases/latest | grep -o '[^/]*$')/linux-amd64.tar.zst && \
tar --zstd -xf linux-amd64.tar.zst ch && \
chmod +x ch && \
sudo mv ch /usr/local/bin/ch
```

### Linux (ARM 64-bit)

```bash
curl -LO https://github.com/ContainerHive/ContainerHive/releases/download/$(curl -Lso /dev/null -w %{url_effective} https://github.com/ContainerHive/ContainerHive/releases/latest | grep -o '[^/]*$')/linux-arm64.tar.zst && \
tar --zstd -xf linux-arm64.tar.zst ch && \
chmod +x ch && \
sudo mv ch /usr/local/bin/ch
```

### Darwin (Intel)

```bash
curl -LO https://github.com/ContainerHive/ContainerHive/releases/download/$(curl -Lso /dev/null -w %{url_effective} https://github.com/ContainerHive/ContainerHive/releases/latest | grep -o '[^/]*$')/darwin-amd64.tar.zst && \
tar --zstd -xf darwin-amd64.tar.zst ch && \
chmod +x ch && \
sudo mv ch /usr/local/bin/ch
```

### Darwin (Apple Silicon)

```bash
curl -LO https://github.com/ContainerHive/ContainerHive/releases/download/$(curl -Lso /dev/null -w %{url_effective} https://github.com/ContainerHive/ContainerHive/releases/latest | grep -o '[^/]*$')/darwin-arm64.tar.zst && \
tar --zstd -xf darwin-arm64.tar.zst ch && \
chmod +x ch && \
sudo mv ch /usr/local/bin/ch
```

## Where to find the latest release

### Binaries

Binaries for all platforms can be found on
the [latest release page](https://github.com/ContainerHive/ContainerHive/releases/latest).

### Docker

For the Docker image, check [Docker Hub](https://hub.docker.com/r/containerhive/containerhive).
