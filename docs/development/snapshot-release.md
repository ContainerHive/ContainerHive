# Snapshot builds and internal releases

`hack/release-snapshot.sh` builds a ContainerHive image from any git ref or pull request and pushes it to a registry of
your choice. The Go binaries compile inside a Docker container, the build artifacts land on the host, and the image is
built for `linux/amd64` and `linux/arm64`.

This is the fastest way to consume a fix from a pull request, and it works without any maintainer involvement — for
example when a critical fix has to ship while the maintainer is unavailable.

## Prerequisites

- `bash` 3 or newer
- `git`
- `docker` with a reachable daemon
- `docker buildx` (the script creates its own builder if none exists)
- Registry credentials are handled by the caller: `docker login`, a credential helper, or a company authorization tool.
  The script performs no authorization checks itself.

On a plain Linux host, cross-arch emulation may be missing. If the `arm64` build fails, run:

```shell
docker run --privileged --rm tonistiigi/binfmt --install all
```

## Usage

```shell
./hack/release-snapshot.sh [--ref <git-ref>] [--pr <id>] \
    --target <registry.tld/image> [--tag <tag>] [--no-push]
```

| Flag | Meaning |
| --- | --- |
| `--pr <id>` | Build from `refs/pull/<id>/head`. Fetched into a detached worktree, so your checkout stays clean. |
| `--ref <git-ref>` | Build from any git ref: `HEAD`, a commit hash, a branch, or a tag. |
| `--target` | Target image without tag, for example `registry.tld/containerhive`. |
| `--tag` | Tag for the image. Defaults to `pr-<id>` for `--pr` and `rev-<short-sha>` otherwise. Sanitized to a docker-safe tag. |
| `--no-push` | Build a single-arch image for the host platform and load it into the local Docker store. |

With no flags at all the script asks interactively: source, target, and tag — using
[bash-tui-toolkit](https://github.com/timo-reymann/bash-tui-toolkit).

## Examples

Publish a pull request build for testing:

```shell
./hack/release-snapshot.sh --pr 42 --target registry.tld/containerhive
# -> registry.tld/containerhive:pr-42 (+ :sha-<short>) for linux/amd64 and linux/arm64
```

Build the current HEAD without pushing:

```shell
./hack/release-snapshot.sh --no-push --target registry.tld/containerhive
```

Release an urgent internal fix with an explicit version:

```shell
./hack/release-snapshot.sh --pr 42 --target registry.tld/internal/containerhive --tag 1.4.2-internal.3
```

## Where the artifacts go

Build artifacts land in `dist-snapshot/<tag>/` (configure the parent folder with `DIST_SNAPSHOT_DIR`). The folder holds
the `dist/*.tar.zst` archives plus the assembled Docker build context. The path is gitignored.

## Urgent fix flow (maintainer unavailable)

1. A contributor opens the hotfix pull request.
2. Someone runs `./hack/release-snapshot.sh --pr <id> --target <your-registry/image>`.
3. Test the resulting `pr-<id>` image in your pipeline.
4. The fix ships from the pull request ref; upstream merges follow later through the normal review flow.

No write or maintainer access is needed at any step.
