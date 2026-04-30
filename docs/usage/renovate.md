# Renovate integration

ContainerHive ships a [Renovate](https://docs.renovatebot.com/) preset that enables automated version bump PRs for the `versions:` entries in your `image.yml` files. It uses the [regex manager](https://docs.renovatebot.com/modules/manager/regex/) together with inline comments to identify which values to track and how.

## Setup

Add the preset to your `renovate.json5`:

```json5
{
  "$schema": "https://docs.renovatebot.com/renovate-schema.json",
  "extends": [
    // your other presets ...
    "github>ContainerHive/ContainerHive//renovate/image-versions.json5"
  ]
}
```

## Annotating versions

Add a `# renovate:` comment to any version entry you want Renovate to track:

```yaml
versions:
  python: "3.13.7"    # renovate: datasource=pypi depName=python versioning=pep440
  zig: "0.14.1"       # renovate: datasource=github-releases depName=ziglang/zig
  nodejs: "24"        # renovate: datasource=node depName=node versioning=node
```

Annotations work at all three version levels — image, tag, and variant:

```yaml
versions:
  uv: "0.8.22" # renovate: datasource=pypi depName=uv versioning=pep440

tags:
  - name: "3.13.7"
    versions:
      python: "3.13.7" # renovate: datasource=pypi depName=python versioning=pep440

variants:
  - name: slim
    tag_suffix: -slim
    versions:
      python: "3.13.7-slim" # renovate: datasource=docker depName=python versioning=docker
```

## Comment fields

| Field | Required | Description |
|:------|:---------|:------------|
| `datasource=` | yes | Renovate datasource — e.g. `pypi`, `docker`, `github-releases`, `go`, `node` |
| `depName=` | yes | Package or image name as the datasource knows it |
| `versioning=` | no | Versioning scheme (default: `semver`) — e.g. `pep440`, `docker`, `node` |
| `registryUrl=` | no | Custom registry URL for private registries |

## Examples by datasource

=== "PyPI"

    ```yaml
    versions:
      python: "3.13.7"  # renovate: datasource=pypi depName=python versioning=pep440
      uv: "0.8.22"      # renovate: datasource=pypi depName=uv versioning=pep440
    ```

=== "Docker Hub"

    ```yaml
    versions:
      ubuntu: "24.04" # renovate: datasource=docker depName=ubuntu versioning=docker
    ```

=== "GitHub Releases"

    ```yaml
    versions:
      zig: "0.14.1"   # renovate: datasource=github-releases depName=ziglang/zig
      kubectl: "1.32.0" # renovate: datasource=github-releases depName=kubernetes/kubernetes versioning=semver
    ```

=== "Go"

    ```yaml
    versions:
      golang: "1.23.0" # renovate: datasource=go depName=go versioning=semver
    ```

=== "Node.js"

    ```yaml
    versions:
      nodejs: "22" # renovate: datasource=node depName=node versioning=node
    ```

=== "Private registry"

    ```yaml
    versions:
      myapp: "1.2.3" # renovate: datasource=docker depName=myapp registryUrl=https://registry.example.com
    ```
