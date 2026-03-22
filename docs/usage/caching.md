# Set up build caching

ContainerHive supports build layer caching through BuildKit, using either S3-compatible storage or a container registry as the cache backend. Caching is configured in `hive.yml` and applies to all image builds.

## S3 cache

S3-compatible storage works with AWS S3, MinIO, Garage, and other S3-compatible services.

```yaml
# hive.yml
cache:
  type: s3
  endpoint: http://garage:3900
  bucket: buildkit-cache
  region: garage
  access_key_id: <your-access-key>
  secret_access_key: <your-secret-key>
  use_path_style: true
```

| Field | Description |
|:------|:------------|
| `type` | Must be `s3` |
| `endpoint` | S3-compatible endpoint URL |
| `bucket` | Bucket name |
| `region` | S3 region |
| `access_key_id` | Access key |
| `secret_access_key` | Secret key |
| `use_path_style` | Use path-style addressing (required for MinIO, Garage, and most self-hosted S3) |

### When to use S3

- You run builds across multiple machines or CI runners and need shared caching
- You already have S3-compatible storage available (AWS, MinIO, Garage)
- You want cache persistence independent of your container registry

### Setting up MinIO for local development

Using the [pgsty/minio](https://github.com/pgsty/minio) image for a lightweight local setup:

```bash
docker run -d \
  --name minio \
  -p 9000:9000 \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=minioadmin \
  pgsty/minio server /data
```

```yaml
cache:
  type: s3
  endpoint: http://localhost:9000
  bucket: buildkit-cache
  region: us-east-1
  access_key_id: <created-via-minio-admin-ui>
  secret_access_key: <created-via-minio-admin-ui>
  use_path_style: true
```

## Registry cache

Uses a container registry to store cache layers alongside your images.

```yaml
# hive.yml
cache:
  type: registry
  ref: registry:5000/cache
  insecure: true
```

| Field | Description |
|:------|:------------|
| `type` | Must be `registry` |
| `ref` | Registry reference for cache storage (e.g. `registry:5000/cache`) |
| `insecure` | Allow HTTP connections to the registry |

### When to use registry cache

- You want a simpler setup without additional infrastructure
- Your container registry supports OCI image manifests
- You want cache stored in the same place as your images

## How caching works

BuildKit uses the cache configuration for both importing (reading) and exporting (writing) cache layers:

1. Before building, BuildKit checks the cache backend for existing layers that match the build context.
2. Matching layers are reused, skipping the corresponding build steps.
3. After building, all new layers are pushed to the cache backend.

Caching operates in `max` mode, meaning all intermediate layers are cached — not just the final image layers. This maximizes cache hits for multi-stage builds.

Cache errors are non-fatal. If the cache backend is unavailable, builds proceed without caching.

## No caching

If no `cache` section is defined in `hive.yml`, builds run without caching. This is fine for local development but not recommended for CI.
