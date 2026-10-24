# Get started

This guide walks you through setting up your first ContainerHive project from scratch.

## 1. Install ContainerHive

Download the latest binary for your platform or use the Docker image. See [Install ContainerHive](./installation.md) for all options.

## 2. Set up BuildKit

ContainerHive uses [BuildKit](https://github.com/moby/buildkit) under the hood. Let `ch` start a local daemon and point it at it:

```bash
eval $(ch dev buildkitd start)
```

This exports `BUILDKIT_HOST` for your shell. Already have a BuildKit daemon? Set `BUILDKIT_HOST` or `buildkit.address` in `hive.yml` instead. See [Local development](./usage/local-development.md).

## 3. Create your project

Create a `hive.yml` at the root of your project:

```yaml
platforms:
  - linux/amd64
  - linux/arm64
```

For all configuration options, see [Configure your project](./configuration/hive.md).

## 4. Define an image

Create an image directory with a definition and Dockerfile:

```
images/my-app/
├── image.yml
└── Dockerfile
```

`image.yml`:

```yaml
tags:
  - name: 1.0.0
    versions:
      app: 1.0.0

versions:
  app: latest
```

`Dockerfile`:

```dockerfile
FROM ubuntu:24.04
ARG APP_VERSION
RUN echo "Building version ${APP_VERSION}"
```

Versions defined in `image.yml` are automatically passed as Docker build arguments with a `_VERSION` suffix (`app` becomes `APP_VERSION`). See [Write your Dockerfiles](./usage/dockerfiles.md) for details.

### Build on top of another image

Images reference each other with `FROM __hive__/<image>:<tag>`. Add a second image that builds on `my-app`:

```
images/my-app-debug/
├── image.yml
└── Dockerfile
```

```dockerfile
FROM __hive__/my-app:1.0.0
RUN apt-get update && apt-get install -y --no-install-recommends strace
```

The tag must be one of the parent's tags (`1.0.0` here). ContainerHive detects the dependency from the `FROM` line, builds `my-app` first and passes it to `my-app-debug` without a registry round trip. See [How dependency resolution works](./how-it-works/dependency-resolution.md).

## 5. Add tests (optional)

Create a `test.yml` (or `test.yml.gotpl` if you need template variables) next to the Dockerfile:

```yaml
schemaVersion: 2.0.0
commandTests:
  - name: "OS check"
    command: "cat"
    args: ["/etc/os-release"]
    expectedOutput: ["Ubuntu"]
```

See [Define your tests](./configuration/test.md) for all test types and template usage.

## 6. Generate, build, and test

```bash
# Discover project and render configurations
ch generate

# Build images
ch build

# Run tests
ch test
```

## 7. Generate SBOMs (optional)

```bash
ch sbom
```

Generates CycloneDX JSON SBOMs for all built images. See [How SBOM generation works](./how-it-works/sbom-generation.md).

## 8. Set up CI (optional)

Generate a CI pipeline for your provider:

```bash
ch template ci --provider github --output .github/workflows/build.yml
ch template ci --provider gitlab --output .gitlab-ci.yml
```

The generated pipeline has one stage per dependency depth, so `my-app` builds first and `my-app-debug` follows. See [Generate CI pipelines](./usage/ci-integration.md) for all options.

Something not working? See [Troubleshooting](./troubleshooting.md). Unfamiliar terms are explained in [Concepts](./concepts.md).

## Next steps

- [Write your Dockerfiles](./usage/dockerfiles.md) — versions, secrets, and inter-image dependencies
- [Define your images](./configuration/image.md) — tags, variants, secrets, and dependencies
- [Use the CLI](./usage/cli.md) — full command reference
- [Test your containers](./usage/testing.md) — run and filter tests
- [Generate CI pipelines](./usage/ci-integration.md) — GitLab CI and GitHub Actions
- [How dependency resolution works](./how-it-works/dependency-resolution.md) — build ordering
