# Use build IDs for merge requests

The `--build-id` flag appends a unique identifier to all image tags, allowing multiple builds to coexist in the same registry without overwriting each other. This is essential for merge request (MR) / pull request (PR) workflows where you want to build, test, and preview images before merging.

## How it works

When you pass `--build-id`, all tags get a `.buildID` suffix:

| Tag | Build ID | Result |
|:----|:---------|:-------|
| `1.0.0` | `mr-42` | `1.0.0.mr-42` |
| `1.0.0` | `build-1234` | `1.0.0.build-1234` |

This applies to:

- Platform-specific push tags (e.g. `1.0.0.linux-amd64.mr-42`)
- Multi-arch manifest tags created by `finalize` (e.g. `1.0.0.mr-42`)
- Semantic version aliases created by `finalize` (e.g. `1.mr-42`, `1.0.mr-42`)

## Recommended MR/PR workflow

### 1. Build with a build ID

Use the MR/PR number or CI job ID as the build ID:

```bash
# GitLab CI
ch --build-id "mr-${CI_MERGE_REQUEST_IID}" build

# GitHub Actions
ch --build-id "pr-${GITHUB_EVENT_NUMBER}" build
```

### 2. Finalize with the same build ID

```bash
ch --build-id "mr-${CI_MERGE_REQUEST_IID}" finalize
```

This creates manifests like `my-image:1.0.0.mr-42` and aliases like `my-image:1.mr-42`.

### 3. Test with the same build ID

```bash
ch --build-id "mr-${CI_MERGE_REQUEST_IID}" test
```

### 4. Merge and build without build ID

On the main branch, build without `--build-id` to produce the final tags:

```bash
ch build
ch finalize
```

This creates the clean `my-image:1.0.0` tag and aliases `my-image:1.0`, `my-image:1`.

## Why use build IDs

Without build IDs, concurrent MR builds would overwrite each other's tags in the registry. Build IDs solve this by namespacing tags per MR:

- **MR 42** pushes `1.0.0.mr-42` — safe to test and preview
- **MR 43** pushes `1.0.0.mr-43` — doesn't conflict with MR 42
- **Main branch** pushes `1.0.0` — the final, clean tag after merge

## Flag placement

`--build-id` is a global flag, placed before the subcommand:

```bash
ch --build-id "mr-42" build
ch --build-id "mr-42" finalize
ch --build-id "mr-42" test
ch --build-id "mr-42" sbom
```

Use the same build ID across all commands in a single pipeline run to keep tags consistent.
