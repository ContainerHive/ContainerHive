# Environment variables

| Variable | Used by | Description |
|:---------|:--------|:------------|
| `BUILDKIT_HOST` | `build`, `test`, `sbom`, `wait` | BuildKit daemon address. Ignored when `buildkit.address` is set in [`hive.yml`](hive.md#buildkit). |
| `DOCKER_HOST` | `dev buildkitd`, `wait` | Docker-compatible daemon to use (default `unix:///var/run/docker.sock`). |
| `LOG_LEVEL` | all | Log level: `debug`, `info`, `warn`, `error` (default `info`). Same as `--log-level`. |
| `CI_NODE_TOTAL` | `build`, `test`, `sbom` | Total number of shards. Same as `--max-shards`. |
| `CI_NODE_INDEX` | `build`, `test`, `sbom` | 1-based index of this shard. Same as `--current-shard`. |
| `CONTAINER_HIVE_CACHE_DIR` | `generate` | Directory for the [`tag_ranges`](image.md#tag_ranges) version cache. Always wins over `cache_dir` in `hive.yml`. |
| `CH_REFRESH_VERSIONS` | all | Refetch every external version source. Same as `--refresh-versions`. |

Flags always take precedence over their environment variables.
