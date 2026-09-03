# Configure your project

The `hive.yml` file is the project-level configuration file. It must be placed at the root of your ContainerHive project.

A JSON schema is published to [SchemaNest](https://schema-nest.timo-reymann.de/schemas/json-schema/containerhive-project/latest?tab=setup) and can be referenced directly at [schema-nest.timo-reymann.de/api/schema/json-schema/containerhive-project/latest](https://schema-nest.timo-reymann.de/api/schema/json-schema/containerhive-project/latest).

## Example

```yaml
buildkit:
  address: tcp://127.0.0.1:8502

platforms:
  - linux/amd64
  - linux/arm64

cache:
  type: s3
  endpoint: http://garage:3900
  bucket: buildkit-cache
  region: garage
  access_key_id: <your-access-key>
  secret_access_key: <your-secret-key>
  use_path_style: true

registry:
  address: localhost:8500
```

## Fields

### `buildkit`

Configuration for the BuildKit daemon.

| Field | Type | Description |
|:------|:-----|:------------|
| `address` | string | BuildKit daemon address (e.g. `tcp://127.0.0.1:8502`) |

### `platforms`

List of target platforms for multi-architecture builds.

```yaml
platforms:
  - linux/amd64
  - linux/arm64
```

### `cache`

Build cache configuration. Supports S3-compatible storage or registry-based caching.

#### S3 cache

| Field | Type | Description |
|:------|:-----|:------------|
| `type` | string | Must be `s3` |
| `endpoint` | string | S3 endpoint URL |
| `bucket` | string | Bucket name |
| `region` | string | S3 region |
| `access_key_id` | string | Access key |
| `secret_access_key` | string | Secret key |
| `use_path_style` | boolean | Use path-style addressing |

#### Registry cache

| Field | Type | Description |
|:------|:-----|:------------|
| `type` | string | Must be `registry` |
| `ref` | string | Registry reference (e.g. `registry:5000/cache`) |
| `insecure` | boolean | Allow insecure connections |

### `registry`

Local OCI registry used for inter-image dependencies and multi-arch manifest creation.

| Field | Type | Description |
|:------|:-----|:------------|
| `address` | string | Registry address (e.g. `localhost:8500`) |

### `labels`

Project-level OCI image labels applied to every built image. All fields are optional.

```yaml
labels:
  vendor: Acme Corp
  authors: platform@acme.test
  url: https://github.com/acme/images/tree/main/%image%
  documentation: https://docs.acme.test/images/%image%/%tag%
  custom:
    com.acme.team: platform
```

| Field           | Type   | Description                                                                                         |
|:----------------|:-------|:----------------------------------------------------------------------------------------------------|
| `vendor`        | string | Sets `org.opencontainers.image.vendor`                                                              |
| `authors`       | string | Sets `org.opencontainers.image.authors`                                                             |
| `url`           | string | Sets `org.opencontainers.image.url`. Supports `%image%` and `%tag%` placeholders                    |
| `documentation` | string | Sets `org.opencontainers.image.documentation`. Supports `%image%` and `%tag%` placeholders          |
| `custom`        | map    | Arbitrary labels merged into every image. Standard auto-derived OCI keys override colliding entries |

#### Auto-derived labels

ContainerHive applies the following labels to every image without configuration:

| Label                                  | Value                                                 |
|:---------------------------------------|:------------------------------------------------------|
| `org.opencontainers.image.title`       | Image name                                            |
| `org.opencontainers.image.ref.name`    | Image name                                            |
| `org.opencontainers.image.version`     | Tag name (including any variant suffix)               |
| `org.opencontainers.image.created`     | Build time in RFC3339                                 |
| `org.opencontainers.image.description` | `description` field from `image.yml`, when set        |
| `org.opencontainers.image.revision`    | Current git commit, when the build runs in a git repo |
| `org.opencontainers.image.source`      | `origin` remote URL, when available                   |

#### Precedence

Custom labels can be declared at four scopes. They merge from least to most specific, so deeper scopes override shallower ones:

```
project labels < image labels < tag labels < variant labels
```

Standard auto-derived OCI keys (the table above plus the structured project fields like `vendor`) always win over a custom-map entry with the same key.

### `template_options`

Custom key-value variables available in CI and custom templates via the `option` function.

```yaml
template_options:
  ci_buildkit_image: registry.io/buildkit
  ci_buildkit_version: v1.4.0
  my_custom_var: some-value
```

All values must be strings. Keys prefixed with `ci_` have built-in defaults:

| Key                                | Default                       | Description                                                                |
|:-----------------------------------|:------------------------------|:---------------------------------------------------------------------------|
| `ci_buildkit_image`                | `moby/buildkit`               | BuildKit container image                                                   |
| `ci_buildkit_version`              | *(matches go.mod dependency)* | BuildKit image tag                                                         |
| `ci_lint`                          | `true`                        | Run hadolint linting in CI pipeline before builds                         |
| `ci_report`                        | `true`                        | Generate and publish HTML/JSON report to GitHub Pages / GitLab Pages       |
| `actions_checkout_version`         | `v6`                          | Version of `actions/checkout`                                              |
| `actions_upload_artifact_version`  | `v7`                          | Version of `actions/upload-artifact`                                       |
| `actions_download_artifact_version` | `v7`                         | Version of `actions/download-artifact`                                     |
| `actions_upload_pages_artifact_version` | `v3`                     | Version of `actions/upload-pages-artifact`                                 |
| `actions_deploy_pages_version`     | `v4`                          | Version of `actions/deploy-pages`                                          |
| `actions_junit_report_version`     | `v6`                          | Version of `mikepenz/action-junit-report`                                  |

User-provided values override built-in defaults.

### `lint`

Configuration for [`ch lint`](../usage/cli.md#lint), which runs [hadolint](https://github.com/hadolint/hadolint) against plain Dockerfiles in the project. Templated Dockerfiles (e.g. `Dockerfile.gotpl`) are skipped — hadolint cannot parse Go template syntax.

```yaml
lint:
  failure_threshold: warning
  ignored:
    - DL3008
  trusted_registries:
    - my-company.com:5000
  label_schema:
    com.acme.team: text
  strict_labels: true
```

| Field                | Type        | Description                                                                                              |
|:---------------------|:------------|:---------------------------------------------------------------------------------------------------------|
| `failure_threshold`  | string      | Lowest severity that causes a non-zero exit: `error`, `warning`, `info`, `style`, `ignore`. Default: `error` |
| `ignored`            | string list | Rule IDs to ignore (e.g. `DL3000`)                                                                       |
| `trusted_registries` | string list | Registries hadolint treats as trusted (suppresses `DL3026`)                                              |
| `label_schema`       | map         | Expected LABEL keys and their validation types (see [hadolint docs](https://github.com/hadolint/hadolint#configure)) |
| `strict_labels`      | bool        | Fail on labels missing from `label_schema`                                                               |

### `report`

Report generation settings.

```yaml
report:
  styleSheet: styles/custom.css
```

| Field        | Type   | Description                                                                 |
|:-------------|:-------|:----------------------------------------------------------------------------|
| `styleSheet` | string | Path to a CSS file to embed in the generated HTML report for custom styling |

#### Theme CSS variables

The report uses CSS custom properties for theming. Override any of these in your custom stylesheet to restyle the report. All variables are scoped to `[data-theme="light"]` or `[data-theme="dark"]`.

=== "Light — The Precision Architect"

    ```css
    [data-theme="light"], :root {
      /* Surfaces */
      --surface: #f7f9fb;
      --surface-container-lowest: #ffffff;
      --surface-container-low: #f2f4f6;
      --surface-container: #edf0f2;
      --surface-container-high: #e6e8ea;
      --surface-container-highest: #e0e3e5;

      /* Brand */
      --primary: #006591;
      --primary-container: #0ea5e9;
      --on-primary: #ffffff;
      --on-primary-container: #ffffff;

      /* Surfaces (content) */
      --on-surface: #191c1e;
      --on-surface-variant: #40484f;

      /* Chips / tags */
      --secondary-container: #b8dffe;
      --on-secondary-container: #dde2e8;

      /* Status */
      --tertiary: #3cddc7;
      --error: #ba1a1a;

      /* Search highlight */
      --highlight-bg: #fef08a;
      --highlight-color: var(--on-surface);
    }
    ```

    | Variable | Hex | Role |
    |:---------|:----|:-----|
    | `--surface` | `#f7f9fb` | Base canvas |
    | `--surface-container-lowest` | `#ffffff` | Cards, code blocks (pop layer) |
    | `--surface-container-low` | `#f2f4f6` | Sidebars, secondary navigation |
    | `--surface-container` | `#edf0f2` | Mid-tier grouping containers |
    | `--surface-container-high` | `#e6e8ea` | Hover states, active selection |
    | `--surface-container-highest` | `#e0e3e5` | Code blocks, high-contrast zones |
    | `--primary` | `#006591` | Primary brand, CTAs |
    | `--primary-container` | `#0ea5e9` | Gradient endpoint, highlights |
    | `--on-primary` | `#ffffff` | Text on primary backgrounds |
    | `--on-primary-container` | `#ffffff` | Text on primary-container |
    | `--on-surface` | `#191c1e` | All body text |
    | `--on-surface-variant` | `#40484f` | Metadata, secondary labels |
    | `--secondary-container` | `#b8dffe` | Chip backgrounds |
    | `--on-secondary-container` | `#dde2e8` | Chip text |
    | `--tertiary` | `#3cddc7` | Healthy/Running status |
    | `--error` | `#ba1a1a` | Vulnerability alerts, destructive actions |
    | `--highlight-bg` | `#fef08a` | Search highlight background |
    | `--highlight-color` | `var(--on-surface)` | Search highlight text |

=== "Dark — The Observability Monolith"

    ```css
    [data-theme="dark"] {
      /* Surfaces */
      --surface: #0b1326;
      --surface-container-lowest: #0d1520;
      --surface-container-low: #131b2e;
      --surface-container: #1a2236;
      --surface-container-high: #222c42;
      --surface-container-highest: #2d3449;
      --surface-variant: #1e2a40;
      --surface-bright: #2a3552;

      /* Brand */
      --primary: #7bd0ff;
      --primary-container: #4db8ff;
      --on-primary: #003751;
      --on-primary-container: #0086b5;

      /* Surfaces (content) */
      --on-surface: #e2e8f0;
      --on-surface-variant: #8899b0;

      /* Chips / tags */
      --secondary-container: #1e3a50;
      --on-secondary-container: #7bd0ff;

      /* Status */
      --tertiary: #3cddc7;
      --error: #ffb4ab;

      /* Search highlight */
      --highlight-bg: #854d0e;
      --highlight-color: #fef08a;
    }
    ```

    | Variable | Hex | Role |
    |:---------|:----|:-----|
    | `--surface` | `#0b1326` | Base canvas (bedrock) |
    | `--surface-container-lowest` | `#0d1520` | Recessed cards, terminal blocks |
    | `--surface-container-low` | `#131b2e` | Main workspace background |
    | `--surface-container` | `#1a2236` | Secondary navigation, panels |
    | `--surface-container-high` | `#222c42` | Interactive modules, hover states |
    | `--surface-container-highest` | `#2d3449` | Critical active data, selected |
    | `--surface-variant` | `#1e2a40` | Glassmorphism base (60% opacity) |
    | `--surface-bright` | `#2a3552` | Tertiary hover lift |
    | `--primary` | `#7bd0ff` | Primary brand, CTAs |
    | `--primary-container` | `#4db8ff` | Row indicator lines on hover |
    | `--on-primary` | `#003751` | Text on primary backgrounds |
    | `--on-primary-container` | `#0086b5` | CTA gradient endpoint |
    | `--on-surface` | `#e2e8f0` | All body text |
    | `--on-surface-variant` | `#8899b0` | Metadata, secondary labels |
    | `--secondary-container` | `#1e3a50` | Chip backgrounds |
    | `--on-secondary-container` | `#7bd0ff` | Chip text |
    | `--tertiary` | `#3cddc7` | Healthy/Running status, terminal text |
    | `--error` | `#ffb4ab` | Vulnerability alerts, destructive actions |
    | `--highlight-bg` | `#854d0e` | Search highlight background |
    | `--highlight-color` | `#fef08a` | Search highlight text |
