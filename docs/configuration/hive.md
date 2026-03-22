# Configure your project

The `hive.yml` file is the project-level configuration file. It must be placed at the root of your ContainerHive project.

A JSON schema is available at [container-hive.timo-reymann.de/schemas/project.schema.json](https://container-hive.timo-reymann.de/schemas/project.schema.json).

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
