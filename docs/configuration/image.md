# Define your images

Each image is defined by an `image.yml` file located at `images/<image-name>/image.yml`.

A JSON schema is published to [SchemaNest](https://schema-nest.timo-reymann.de/schemas/json-schema/containerhive-image/latest?tab=setup) and can be referenced directly at [schema-nest.timo-reymann.de/api/schema/json-schema/containerhive-image/latest](https://schema-nest.timo-reymann.de/api/schema/json-schema/containerhive-image/latest).

## Example

```yaml
tags:
  - name: 3.13.7
    versions:
      python: 3.13.7
    build_args:
      base_tag: bookworm

versions:
  uv: 0.8.22
  poetry: 2.2.1

build_args:
  foo: bar

secrets:
  api_key:
    source: env
    value: API_KEY

variants:
  - name: slim
    tag_suffix: -slim
    versions:
      python: 3.13.7-slim
    build_args:
      variant: slim
    platforms:
      - linux/amd64

depends_on:
  - base-image

platforms:
  - linux/amd64
  - linux/arm64
```

## Fields

### `tags`

List of tags to build for this image. Each tag can override versions and build args.

| Field | Type | Description |
|:------|:-----|:------------|
| `name` | string | Tag name |
| `versions` | map | Version overrides for this tag |
| `build_args` | map | Build arg overrides for this tag |
| `labels` | map | Custom OCI labels for this tag. Overrides image-level labels. See [`labels`](#labels) |

### `versions`

Default version variables available in Dockerfiles and templates.

### `build_args`

Default build arguments passed to the Dockerfile.

### `secrets`

Secrets made available during the build.

| Field | Type | Description |
|:------|:-----|:------------|
| `source` | string | `env` (from environment variable) or `plain` (literal value) |
| `value` | string | Environment variable name or literal value |

### `variants`

Variants allow building multiple flavors of the same image (e.g. slim, debug).

| Field | Type | Description |
|:------|:-----|:------------|
| `name` | string | Variant name |
| `tag_suffix` | string | Suffix appended to tag names |
| `versions` | map | Version overrides |
| `build_args` | map | Build arg overrides |
| `platforms` | list | Platform overrides |
| `labels` | map | Custom OCI labels for this variant. Overrides tag- and image-level labels. See [`labels`](#labels) |

Each variant can have its own `Dockerfile`, `test.yml.gotpl`, and `rootfs/` directory in a subdirectory named after the variant.

### `depends_on`

List of image names this image depends on. ContainerHive resolves dependencies automatically from `FROM` references in Dockerfiles, but explicit dependencies can be declared here.

### `platforms`

Override the project-level platform list for this specific image.

### `labels`

Custom OCI image labels applied to every tag and variant of this image. Values are merged with project-, tag-, and variant-level labels following the precedence chain documented in [Configure your project › `labels`](hive.md#labels).

```yaml
labels:
  com.acme.layer: image
  com.acme.image: python

tags:
  - name: 3.13.7
    labels:
      com.acme.layer: tag        # overrides image-level entry for this tag

variants:
  - name: slim
    tag_suffix: -slim
    labels:
      com.acme.layer: variant    # overrides image and tag for the variant build
```

Standard auto-derived OCI keys (`title`, `version`, `created`, etc.) always win over custom map entries with the same key.

### `latest_alias`

Configure an alias pointing to the highest semantic version tag. This allows you to automatically retag the highest semantic version as a configurable alias (e.g., `latest`, `stable`).

| Field | Type | Description |
|:------|:-----|:------------|
| `tag` | string | Required. The alias tag name (e.g., `latest`, `stable`) |
| `on_missing` | string | Optional. Behavior when no semantic tags are found: `error` (default), `warning`, or `silent` |

**Example:**

```yaml
latest_alias:
  tag: latest
  on_missing: warning
```

In this example, the highest semantic version tag will be retagged as `latest`. If no semantic tags are found, a warning will be logged instead of failing.

## Directory structure

Each image directory can contain:

```
images/<image-name>/
├── image.yml                  # Image definition (required)
├── Dockerfile[.gotpl]         # Build instructions (required)
├── test.yml[.gotpl]           # Container structure tests (optional)
├── rootfs/                    # Files to copy into the image (optional)
└── <variant-name>/            # Variant subdirectory (optional)
    ├── Dockerfile[.gotpl]
    ├── test.yml[.gotpl]
    └── rootfs/
```

The `.gotpl` extension is optional for Dockerfiles and test files. Plain files are copied as-is. Use `.gotpl` when you need Go template features (e.g. `resolve_base`, conditionals). See [Write your Dockerfiles](../usage/dockerfiles.md) for details.
