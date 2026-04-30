# Install ContainerHive

ContainerHive supports a variety of platforms.

## Containerized

If you prefer to use containerized workflows, use the provided OCI image.

```sh
docker run --rm -it -v $PWD:/workspace timoreymann/containerhive
```

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

For the Docker image, check [Docker Hub](https://hub.docker.com/r/timoreymann/containerhive).
