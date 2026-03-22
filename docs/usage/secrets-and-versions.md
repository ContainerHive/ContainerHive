# Manage versions and secrets

ContainerHive provides a structured way to manage versions and secrets across your image definitions, with a clear merge hierarchy and multiple secret backends.

## Versions

Versions are key-value pairs that flow through your entire build pipeline — from `image.yml` through Dockerfiles and test definitions.

### Defining versions

Versions can be set at three levels, with each level overriding the previous:

**1. Image level** (defaults for all tags):

```yaml
# image.yml
versions:
  python: "3.13"
  uv: "0.8.22"
```

**2. Tag level** (overrides per tag):

```yaml
tags:
  - name: 3.13.7
    versions:
      python: "3.13.7"    # overrides image-level "3.13"
                           # uv remains "0.8.22"
  - name: 3.12.9
    versions:
      python: "3.12.9"
```

**3. Variant level** (overrides per variant):

```yaml
variants:
  - name: slim
    tag_suffix: -slim
    versions:
      python: "3.13.7-slim"   # overrides tag-level value
```

### Using versions in Dockerfiles

Versions are automatically converted to Docker build arguments with an uppercase `_VERSION` suffix. Hyphens are replaced with underscores.

| Version key | Docker ARG |
|:------------|:-----------|
| `python` | `PYTHON_VERSION` |
| `dotnet-sdk-channel` | `DOTNET_SDK_CHANNEL_VERSION` |
| `nodejs` | `NODEJS_VERSION` |

```dockerfile
ARG PYTHON_VERSION
FROM python:${PYTHON_VERSION}
```

### Using versions in templates

In `.gotpl` files (Dockerfiles or tests), versions are also available as template variables:

```gotpl
FROM python:{{ .Versions.python }}
```

### Using versions in test definitions

In `test.yml.gotpl`:

```yaml
schemaVersion: 2.0.0
commandTests:
  - name: "Python version"
    command: "python"
    args: ["--version"]
    expectedOutput: ["Python {{ .Versions.python }}"]
```

## Build arguments

Build arguments work the same as versions but without the `_VERSION` suffix transformation.

```yaml
# image.yml
build_args:
  foo: bar

tags:
  - name: latest
    build_args:
      foo: baz     # overrides image-level value for this tag
```

```dockerfile
ARG foo
RUN echo "${foo}"
```

In `.gotpl` templates: `{{ .BuildArgs.foo }}`.

## Secrets

Secrets are mounted into the build process via BuildKit's secret mount mechanism. They are never baked into image layers.

### Defining secrets

```yaml
# image.yml
secrets:
  my_credentials:
    source: env
    value: API_KEY
```

### Secret sources

| Source | Description | Example |
|:-------|:------------|:--------|
| `env` | Read from an environment variable | `value: API_KEY` or `value: ${API_KEY}` |
| `plain` | Use the value as-is | `value: my-secret-token` |
| `vault` | Fetch from HashiCorp Vault | `value: vault://secret/data/app#token` |

### Using secrets in Dockerfiles

Mount secrets with BuildKit's `--mount=type=secret` syntax:

```dockerfile
RUN --mount=type=secret,id=my_credentials \
    cat /run/secrets/my_credentials
```

The `id` must match the key name from your `image.yml` secrets definition.

### Environment variable resolution

The `env` source supports both `$VAR` and `${VAR}` syntax. The value is resolved at build time from the environment where `ch build` runs.

### Vault integration

For Vault secrets, set `VAULT_ADDR` to your Vault instance URL and provide authentication via `VAULT_TOKEN` or `~/.vault-token`:

```yaml
secrets:
  db_password:
    source: vault
    value: vault://secret/data/myapp#db_password
```

## Complete example

```yaml
# image.yml
versions:
  dotnet-sdk-channel: "8.0"
  nodejs: "24"

build_args:
  extra_packages: "curl git"

secrets:
  registry_token:
    source: env
    value: REGISTRY_TOKEN

tags:
  - name: 8.0.100
    versions:
      dotnet-sdk-channel: "8.0.1xx"
  - name: 8.0.200
    versions:
      dotnet-sdk-channel: "8.0.2xx"

variants:
  - name: node
    tag_suffix: -node
    versions:
      nodejs: "24"
```

```dockerfile
FROM ubuntu:24.04

ARG DOTNET_SDK_CHANNEL_VERSION
ARG extra_packages
RUN apt-get update && apt-get install -y ${extra_packages}
RUN install-dotnet --channel "${DOTNET_SDK_CHANNEL_VERSION}"

RUN --mount=type=secret,id=registry_token \
    configure-registry /run/secrets/registry_token
```
