# Write your Dockerfiles

ContainerHive renders Dockerfiles as part of `ch generate`. This page covers how versions, build arguments, secrets, and inter-image dependencies work in your Dockerfiles.

## Plain Dockerfiles vs templates

ContainerHive supports both plain Dockerfiles and Go template Dockerfiles:

- **`Dockerfile`** — copied as-is. Versions and build args are passed as Docker `ARG`s.
- **`Dockerfile.gotpl`** — rendered as a Go template before building. Has access to template variables and functions in addition to Docker `ARG`s.

The `.gotpl` extension is optional. Use it only when you need Go template features like `resolve_base` or conditional logic. For most cases, plain Dockerfiles with `ARG` are sufficient.

!!! info

    The same applies to test files: `test.yml` is copied as-is, `test.yml.gotpl` is rendered as a template.

## Using versions

Versions defined in `image.yml` are automatically converted to Docker build arguments with an uppercase `_VERSION` suffix. Hyphens in the key name are replaced with underscores.

Given this `image.yml`:

```yaml
versions:
  dotnet-sdk-channel: "8.0"
  nodejs: "24"
```

The following build arguments are available in your Dockerfile:

```dockerfile
ARG DOTNET_SDK_CHANNEL_VERSION
ARG NODEJS_VERSION

RUN install-dotnet --channel "${DOTNET_SDK_CHANNEL_VERSION}"
RUN setup-node "${NODEJS_VERSION}"
```

Tag-level versions override image-level versions:

```yaml
versions:
  app: latest

tags:
  - name: 1.0.0
    versions:
      app: 1.0.0   # overrides "latest" for this tag
```

## Using build arguments

Build arguments from `image.yml` are passed directly to Docker without any name transformation:

```yaml
build_args:
  foo: bar
```

```dockerfile
ARG foo
RUN echo "${foo}"
```

## Using secrets

Secrets defined in `image.yml` are mounted via BuildKit's secret mount mechanism:

```yaml
secrets:
  my_credentials:
    source: env       # read from environment variable
    value: API_KEY    # name of the env var
```

Access them in your Dockerfile with `--mount=type=secret`:

```dockerfile
RUN --mount=type=secret,id=my_credentials \
    cat /run/secrets/my_credentials
```

### Secret sources

| Source | Description |
|:-------|:------------|
| `env` | Resolves `value` as an environment variable name (supports `$VAR` and `${VAR}` syntax) |
| `plain` | Uses `value` as the literal secret content |
| `vault` | Resolves `value` as a Vault path (`vault://<path>#<field>`) |

## Inter-image dependencies

When your image depends on another image in the same project, use the `__hive__/` prefix in `FROM`:

```dockerfile
FROM __hive__/ubuntu:22.04
RUN apt-get update && apt-get install -y curl
```

ContainerHive resolves these references to local OCI layout contexts during the build, so inter-image dependencies don't need to be pushed to an external registry.

### Variant parent references

Variants that build on top of their base image can use `__hive_parent__`:

```dockerfile
FROM __hive_parent__
ARG NODEJS_VERSION
RUN setup-node "${NODEJS_VERSION}"
```

This is automatically replaced with the correct `__hive__/<image>:<tag>` reference for the current tag during `ch generate`.

## Go template Dockerfiles

When using the `.gotpl` extension, you get access to the full template context:

| Variable | Description |
|:---------|:------------|
| `{{ .Versions.key }}` | Version values from image/tag/variant |
| `{{ .BuildArgs.key }}` | Build arguments from image/tag/variant |
| `{{ .ImageName }}` | The image name |

### Template functions

All [Sprig](http://masterminds.github.io/sprig/) functions are available, plus:

| Function | Description |
|:---------|:------------|
| `resolve_base "name" "tag"` | Produces `__hive__/name:tag` — use this instead of hardcoding `__hive__/` references |

### Example

```gotpl
FROM {{ resolve_base "ubuntu" .Versions.ubuntu_tag }}
RUN echo "Building {{ .ImageName }}"
```

## Complete example

`image.yml`:

```yaml
tags:
  - name: 8.0.100
    versions:
      dotnet-sdk-channel: 8.0.1xx
  - name: 8.0.200
    versions:
      dotnet-sdk-channel: 8.0.2xx

variants:
  - name: node
    tag_suffix: -node
    versions:
      nodejs: "24"

secrets:
  foo:
    value: bar

depends_on:
  - ubuntu

platforms:
  - linux/amd64
```

`Dockerfile`:

```dockerfile
FROM __hive__/ubuntu:22.04
ARG DOTNET_SDK_CHANNEL_VERSION
RUN curl -L https://dot.net/v1/dotnet-install.sh -o /usr/bin/install-dotnet \
    && chmod +x /usr/bin/install-dotnet \
    && install-dotnet --channel "${DOTNET_SDK_CHANNEL_VERSION}" --install-dir /usr/share/dotnet

RUN --mount=type=secret,id=foo \
    cat /run/secrets/foo
```

`node/Dockerfile` (variant):

```dockerfile
FROM __hive_parent__
ARG NODEJS_VERSION
RUN apt-get update \
    && apt-get install --no-install-recommends -y nodejs=${NODEJS_VERSION}
```
