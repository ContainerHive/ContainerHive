# Filter images and tags

Most ContainerHive commands accept `image:tag` filters as positional arguments to target specific images or tags instead of processing the entire project.

## Syntax

```bash
ch <command> [image[:tag] ...]
```

| Pattern | Matches |
|:--------|:--------|
| (none) | All images and tags |
| `ubuntu` | All tags of the `ubuntu` image |
| `ubuntu:24.04` | Only tag `24.04` of image `ubuntu` |
| `ubuntu:24.04` `alpine:3.20` | Both specified image:tag combinations |

## Commands that support filtering

- `ch build` — build only matching images
- `ch test` — test only matching images
- `ch sbom` — generate SBOMs for matching images
- `ch finalize` — create manifests for matching images

## Examples

Build a single image:

```bash
ch build ubuntu
```

Build a specific tag:

```bash
ch build ubuntu:24.04
```

Test multiple images:

```bash
ch test ubuntu:24.04 python:3.13.7
```

Generate SBOMs for one image:

```bash
ch sbom dotnet
```

## Variant tags

Variant tags include the tag suffix defined in `image.yml`. To filter for a variant, specify the full tag including the suffix:

```bash
ch build dotnet:8.0.200-node
```

## Matching rules

- If no filters are provided, all images and tags are processed.
- Image name matching is exact — no wildcards or glob patterns.
- Tag matching is exact — no wildcards or glob patterns.
- Specifying only an image name matches all tags of that image.
- Multiple filters are combined with OR logic — an image:tag matches if it matches any filter.
