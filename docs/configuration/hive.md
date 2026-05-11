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

| Key                   | Default                       | Description                                                          |
|:----------------------|:------------------------------|:---------------------------------------------------------------------|
| `ci_buildkit_image`   | `moby/buildkit`               | BuildKit container image                                             |
| `ci_buildkit_version` | *(matches go.mod dependency)* | BuildKit image tag                                                   |
| `ci_lint`             | `true`                        | Run hadolint linting in CI pipeline before builds                   |
| `ci_report`           | `true`                        | Generate and publish HTML/JSON report to GitHub Pages / GitLab Pages |

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
