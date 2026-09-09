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

### `tag_ranges`

Generates tags automatically from an external version source (a JSON feed, an OCI
registry, or GitHub releases), instead of listing every tag by hand. Resolution happens
at discovery time (`ch generate`, `ch build`, ...) and requires network access unless a
warm cache is available.

```yaml
tag_ranges:
  - tag_name: "{{.major}}.{{.minor}}"
    source:
      type: json
      url: "https://nodejs.org/dist/index.json"
      transform: |
        $[lts != false].{
          "version": $substring(version, 1),
          "npm": npm,
          "openssl": openssl
        }
    select:
      major: { last: 3 }
      minor: { last: 3 }
      patch: latest
    filter:
      exclude_prerelease: true
      max_major_increment: 5
    versions:
      nodejs: "{{.full}}"
      npm: "{{.extra.npm}}"
      openssl: "{{.extra.openssl}}"
```

For the Node.js LTS feed, this generates tags like `24.11, 24.10, 24.9, 22.22, 22.21,
20.20, 20.19` — the 3 highest majors, the 3 highest minors within each, and the latest
patch of each minor — each with its own `NODEJS_VERSION`, `NPM_VERSION` and
`OPENSSL_VERSION` build args.

| Field | Type | Description |
|:------|:-----|:------------|
| `tag_name` | string | Required. Go template for the generated tag name. Available data: `.major`, `.minor`, `.patch`, `.full` (normalized version), `.raw` (exactly as the source reported it), `.prerelease`, `.extra.<key>` (source-specific metadata). |
| `source` | object | External version source. Mutually exclusive with `generator`. See below. |
| `generator` | string | Named generator to use instead of an explicit `source` (currently only `semver-matrix`). Mutually exclusive with `source`. |
| `params` | map | Parameters for the named generator. |
| `select` | object | How many versions to keep per semantic version level. **Required** when `source` is set. See below. |
| `filter` | object | Filters applied to fetched versions before selection. See below. |
| `max_tags` | integer | Hard cap on generated tags for this range. Exceeding it is an error. Defaults to 50. |
| `versions` | map | Versions for generated tags. Values are Go templates rendered per resolved version (same syntax as `tag_name`). |
| `build_args` | map | Build args for generated tags, rendered the same way. |
| `labels` | map | Custom OCI labels for generated tags, rendered the same way. |

#### `source`

| Field | Type | Description |
|:------|:-----|:------------|
| `type` | string | Required. `json`, `registry`, `dockerhub` (alias of `registry`), or `github`. |
| `url` | string | HTTP(S) URL returning JSON (`type: json`). |
| `transform` | string | JSONata expression mapping the document to a list of `{version, ...extras}` objects (`type: json`). ContainerHive uses the JSONata **1.5.4** subset — an expression copied from a newer JSONata Exerciser may not compile. |
| `headers` | map | Extra request headers for `type: json`. Values support `$VAR` / `${VAR}` environment expansion, so a token never has to be inlined. |
| `image` | string | Repository whose tags are listed, e.g. `library/node` or `ghcr.io/owner/image` (`type: registry`, `dockerhub`). Uses the same registry credentials as `ch login`. |
| `repo` | string | GitHub repository as `owner/name` (`type: github`). |
| `kind` | string | What to list for `type: github`: `releases` (default, drafts excluded) or `tags`. |
| `token_env` | string | Name of the environment variable holding an auth token. Never put the token itself here. Falls back to `GITHUB_TOKEN` / `GH_TOKEN` for `type: github`. Anonymous GitHub API access is rate-limited to 60 requests/hour. |
| `ttl` | string | Cache TTL override for this source, as a Go duration (e.g. `6h`). |

#### `select`

Selects how many versions survive at each semantic version level, applied hierarchically:
majors first, then minors within each kept major, then patches within each kept minor.
Each level accepts:

```yaml
select:
  major: latest        # keep the single highest value (equivalent to { last: 1 })
  minor: all           # keep every distinct value
  patch: 2             # keep the 2 highest distinct values (equivalent to { last: 2 })
  # patch: { last: 2 } # same as above, mapping form
```

An **omitted level defaults to `latest`**, never `all` — a config that forgets a level
can't silently explode the tag count.

#### `filter`

Filters are applied, in this fixed order, before selection:

| Field | Type | Description |
|:------|:-----|:------------|
| `exclude_prerelease` | boolean | Drop versions with a semantic version prerelease component. Defaults to `true`. |
| `exclude_suffixes` | list | Drop versions whose raw string contains any of these substrings — for **non**-semver flavours like `-alpine`. `exclude_prerelease` already covers `-rc`/`-beta`/`-alpha`. |
| `min_version` / `max_version` | string | Drop versions outside this inclusive range. |
| `max_major_increment` | integer | Sanity guard, not a window: drops a major (and everything above it) only if it's separated from the rest of the set by a gap larger than this. `[18, 20, 22, 24]` with `5` keeps everything (gaps of 2); adding `99` drops it (a gap of 75). `0` disables the guard. |

#### `generator: semver-matrix`

A shorthand for the common "last N majors, last M minors" shape, expanding to an
equivalent `source`/`select`/`filter`:

```yaml
tag_ranges:
  - generator: semver-matrix
    params:
      source: dockerhub
      image: library/node
      majors: "3"
      minors: "3"
    tag_name: "{{.major}}.{{.minor}}"
    versions:
      nodejs: "{{.full}}"
```

`params`: `source` (required — any `source.type`), `image` / `repo` / `url` /
`transform` (matching the chosen source type), `majors` / `minors` / `patches` (each
defaulting to `1`). Explicit `select` or `filter` on the range override the generator's.

#### Collision rules

- A **static** tag in `tags` always wins over a generated tag of the same name — this is
  the pin/escape hatch.
- Two different `tag_ranges` producing the same tag name is a **hard error**: there's no
  defensible winner.
- Within one range, if the `tag_name` template is coarser than the `select` granularity
  (e.g. `"{{.major}}.{{.minor}}"` with `patch: { last: 2 }`), the **highest version wins**
  the collision by design — the template is what sets the granularity.
- A generated tag colliding with another tag's variant-suffixed form (`tag +
  variant.tag_suffix`) is a hard error.

#### Known limitations

- `tag_ranges` feed into `versions`, which become build args, which are part of the
  BuildKit cache key — so an upstream release can change what a tag builds to without a
  new commit. The same commit can produce different images over time.
- A tag-level `build_args` entry (generated or static) can currently be silently
  overwritten by an image-level `build_args` entry of the same key — see the
  `secrets-and-versions` docs for the merge precedence. This is a pre-existing issue,
  tracked separately, not specific to `tag_ranges`.

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
