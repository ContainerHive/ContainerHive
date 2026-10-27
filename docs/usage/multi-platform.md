# Build multi-platform images

ContainerHive builds images for multiple architectures in a single workflow and creates multi-arch manifests so consumers pull the right image automatically.

## Configuring platforms

Platforms are configured at three levels, with the most specific taking precedence:

### Project-level (default for all images)

```yaml
# hive.yml
platforms:
  - linux/amd64
  - linux/arm64
```

### Image-level (override for a specific image)

```yaml
# image.yml
platforms:
  - linux/amd64
```

### Variant-level (override for a specific variant)

```yaml
# image.yml
variants:
  - name: slim
    tag_suffix: -slim
    platforms:
      - linux/amd64
```

The resolution order is: **variant > image > project**. If a variant defines platforms, those are used. Otherwise the image-level platforms are used, falling back to the project-level default.

### Selecting platforms with `--platform`

`ch build`, `ch test` and `ch sbom` accept `--platform` to work on a subset of platforms, for example one platform per CI job. The flag selects within the resolved list, it does not replace it. A tag or variant whose resolved platforms do not include the requested platform is skipped.

With the `slim` variant above restricted to `linux/amd64`, `ch build --platform linux/arm64` builds the arm64 base tags but skips `-slim`. Each skipped tag is logged. If no tag selected by the filters (and shard) supports the platform, the command logs a warning, does nothing and exits successfully.

Requesting a platform that no project, image or variant configures (for example a typo such as `arm` for `linux/arm/v7`) fails with an error.

An image that depends on another image (`FROM __hive__/base:1.0` or a `__hive__/` parent) can only be built for platforms that dependency configures. Building it for any other platform fails with an error naming the dependency and the platform. Add the platform to the dependency, or remove it from the dependent image.

## How multi-platform builds work

When you run `ch build`, ContainerHive builds each image once per platform. The output is a platform-specific OCI tar file:

```
dist/my-image/1.0.0/
├── linux-amd64/
│   └── image.tar
└── linux-arm64/
    └── image.tar
```

Each platform build runs independently through BuildKit, allowing parallel execution in CI.

## Creating multi-arch manifests with `finalize`

After building all platform-specific images, `ch finalize` creates OCI image indexes (multi-arch manifests) that reference all platform variants under a single tag:

```bash
ch finalize
```

This does two things:

1. **Creates manifests** — For each image and tag, creates an OCI image index referencing all platform-specific images.
2. **Creates semantic version aliases** — Generates shorter version tags that point to the full version manifest.

### Semantic version aliases

If your tags follow semantic versioning, `finalize` automatically creates alias tags pointing to the highest matching version:

| Tag | Generated aliases |
|:----|:-----------------|
| `1.2.3` | `1.2`, `1` |
| `v2.0.1` | `v2.0`, `v2` |
| `1.2.3-alpine` | `1.2-alpine`, `1-alpine` |
| `8.0.200` | `8.0`, `8` |

When multiple tags compete for the same alias, the highest version wins. For example, if you build tags `1.2.3` and `1.3.0`, the alias `1` points to `1.3.0`.

### Supported version formats

- `1.2.3` — standard semver
- `v1.2.3` — with prefix (preserved in aliases)
- `1.2.3.4` — four-component versions
- `1.2.3-alpine` — with suffix (preserved in aliases)

### Filtering

You can finalize specific images or tags:

```bash
ch finalize my-image:1.0.0
```

## Typical workflow

```bash
ch generate          # Render project configurations
ch build             # Build all platform-specific images
ch finalize          # Create multi-arch manifests and version aliases
ch test              # Run container structure tests
ch sbom              # Generate SBOMs
```

## CI pipeline integration

Generated CI pipelines (via `ch template ci`) automatically handle multi-platform builds by creating separate build jobs per platform and a manifest job that runs `finalize` after all builds complete. See [Generate CI pipelines](./ci-integration.md).
