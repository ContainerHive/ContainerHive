# Build only changed images

Change detection works out which images a change affects, so a merge or pull request pipeline does not have to rebuild the whole project.

## How it works

Every changed file is checked against these rules, in order:

1. It matches a [`treat_as_global`](../configuration/hive.md#change_detection) pattern: every image is affected.
2. It matches an [`exclude`](../configuration/hive.md#change_detection) pattern: it is ignored.
3. It is inside an image directory (the directory holding `image.yml`, including variants): that image is changed.
4. Anything else is unmatched. By default an unmatched file affects every image. Pass `--unmatched ignore` to drop unmatched files.

A changed image is affected, and so is every image that depends on it, directly or transitively. Dependencies are the same as in [`ch graph`](cli.md#graph): `FROM __hive__/...`, `parent` references and `depends_on`.

For `base -> python -> pre-commit`, where `pre-commit` depends on `python`:

| Changed file | Affected images |
|:-------------|:----------------|
| `images/pre-commit/Dockerfile` | `pre-commit` |
| `images/python/image.yml` | `python`, `pre-commit` |
| `images/base/Dockerfile` | `base`, `python`, `pre-commit` |
| `README.md` | none (excluded by default) |
| `.pre-commit-config.yaml` | all (global by default) |

## Compute the affected images

Compare `HEAD` with a base revision. ContainerHive diffs against their merge base, the same diff a merge request shows:

```bash
ch changes --base origin/main
```

The command prints one affected image per line. Use `--format json` for the full result, including the changed images and the reason for a global rebuild.

Without git, pass the changed files yourself, relative to the project root. List both paths of a moved file, so the image it moved away from counts as changed too:

```bash
git diff --no-renames --name-only origin/main... | ch changes --files-from -
```

If the diff cannot be computed, for example because the clone is shallow and the base commit is missing, every image is affected. Fetch the full history, or at least the base commit, in CI.

## Limit a pipeline to the affected images

`--write` stores the result as `dist/changes.json`. Run it after `ch generate`, which clears `dist/`, and so does the global `--generate` flag. A stale change set therefore never limits a later build:

```bash
ch generate --build-id "$ID"
ch changes --base "$BASE" --write
ch build --build-id "$ID"
```

While `dist/changes.json` exists, `build`, `test`, `finalize` and `sbom` only process affected images. With explicit [filters](filtering.md), unaffected images are dropped from the filters.

With a [build ID](build-ids.md), a dependency that was not rebuilt has no `-build.<id>` tag. The build then pulls the dependency's production tag from the registry instead, so that tag must exist, which it does after a regular build of the default branch.

## Generated CI pipelines

[Generated pipelines](ci-integration.md#change-detection) use change detection by default. Pipelines on the default branch, tags and schedules still build every image.

| | GitHub Actions | GitLab CI |
|:--|:--|:--|
| Applies to | `pull_request` events | Push pipelines of branches other than the default branch |
| Compared against | The pull request base | The default branch |
| Unaffected images | Jobs are skipped | Jobs are not created (`rules:changes`) |
| Unmatched files | Rebuild every image | Ignored |

GitLab decides which jobs exist when it creates the pipeline, from static `rules:changes` paths. Those paths cannot express "any other file", so on GitLab unmatched files never trigger a build. Add such files to `treat_as_global` if they should. For the same reason the GitLab `generate` job runs `ch changes --strict`: if the diff cannot be computed, the pipeline fails instead of expecting jobs that GitLab did not create. GitLab `rules:changes:compare_to` with variables needs GitLab 17.2 or later.

GitLab accepts at most 50 paths in one `rules:changes` rule. An image whose directories, dependency directories and `treat_as_global` patterns add up to more gets no change rule: its jobs are always created, and `dist/changes.json` makes them skip the work when the image is not affected.

## Configure the rules

See [`change_detection`](../configuration/hive.md#change_detection) for the patterns and their defaults.
