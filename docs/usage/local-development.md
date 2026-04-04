# Local development

The `ch dev` command group provides convenience helpers for running a local development environment. This removes the need to set up infrastructure manually when iterating on images.

## Prerequisites

A Docker-compatible daemon must be reachable. By default the Docker socket at `/var/run/docker.sock` is used. Set `$DOCKER_HOST` to point at a different daemon, including Podman:

```bash
export DOCKER_HOST=unix:///run/user/1000/podman/podman.sock
```

## BuildKit daemon

`ch dev buildkitd` manages a local [BuildKit](https://github.com/moby/buildkit) daemon container named `containerhive-buildkitd`.

### Start

```bash
ch dev buildkitd start
```

This will:

1. Pull the buildkitd image if not present locally.
2. Create and start a privileged container listening on TCP.
3. Wait until the daemon is ready to accept connections.
4. Print the `BUILDKIT_HOST` export line.

```
export BUILDKIT_HOST=tcp://127.0.0.1:8372
```

Evaluate this in your shell to make subsequent `ch build` invocations use the local daemon:

```bash
eval $(ch dev buildkitd start)
ch build
```

| Flag | Default | Description |
|:-----|:--------|:------------|
| `--image` | *(see below)* | Full `image:tag` to use for the buildkitd container |
| `--port` | `8372` | Host port to bind |
| `--timeout` | `1m` | How long to wait for the daemon to become ready |

### Image version

By default the image is resolved from your project's `template_options` in `hive.yml`, using the same `ci_buildkit_image` and `ci_buildkit_version` keys that control the version used in generated CI pipelines. This keeps your local daemon in sync with CI.

If no overrides are configured, the version is taken from the `github.com/moby/buildkit` dependency in `go.mod`.

```yaml
# hive.yml — pin the same version locally and in CI
template_options:
  ci_buildkit_version: v0.19.0
```

Use `--image` to override the full reference for a one-off run:

```bash
ch dev buildkitd start --image moby/buildkit:v0.19.0
```

### Status

```bash
ch dev buildkitd status
```

Shows the container state, image, bound port, and the `BUILDKIT_HOST` value to use.

### Logs

```bash
ch dev buildkitd logs
ch dev buildkitd logs --follow
```

| Flag | Description |
|:-----|:------------|
| `--follow`, `-f` | Stream log output until interrupted |

### Stop

```bash
ch dev buildkitd stop
ch dev buildkitd stop --remove
```

| Flag | Description |
|:-----|:------------|
| `--remove` | Also remove the container after stopping |

Without `--remove` the container is only stopped and can be started again with `ch dev buildkitd start`.
