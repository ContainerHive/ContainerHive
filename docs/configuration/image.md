# Define your images

Each image is defined by an `image.yml` file located at `images/<image-name>/image.yml`.

A JSON schema is available at [container-hive.timo-reymann.de/schemas/image.schema.json](https://container-hive.timo-reymann.de/schemas/image.schema.json).

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

Each variant can have its own `Dockerfile`, `test.yml.gotpl`, and `rootfs/` directory in a subdirectory named after the variant.

### `depends_on`

List of image names this image depends on. ContainerHive resolves dependencies automatically from `FROM` references in Dockerfiles, but explicit dependencies can be declared here.

### `platforms`

Override the project-level platform list for this specific image.

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
