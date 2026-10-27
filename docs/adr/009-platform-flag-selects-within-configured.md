---
id: 009
status: accepted
date: 2026-10-27
---

# `--platform` selects within the configured platforms

## Context and Problem Statement

`ch build`, `ch test` and `ch sbom` accept `--platform`. Until now the flag replaced the platforms that
`hive.yml` and `image.yml` configure for every image, tag and variant. A CI job ran one platform per job and
passed that platform with `--platform`. The job then built a platform the project never configured, without a
warning. A typo such as `--platform arm` for `linux/arm/v7` built an image under the wrong platform instead of
failing. Also `resolvePlatforms` stripped the platform to its last path component for the CI context, so
`linux/arm/v7` became `v7` and the jobs passed a value `platform.Normalize` did not map back to the configured
platform. The question: what does `--platform` mean?

## Decision Drivers

* A CI job must fail clearly when nothing matches, not build an unintended platform
* One platform definition per project: the configured lists in `hive.yml` and `image.yml` stay the single
  source of truth
* A typo in `--platform` must fail instead of silently building something
* Tags and variants that support a platform must stay supported when only a subset of platforms is requested
* The change must keep the generated GitLab and GitHub pipelines working unchanged

## Considered Options

* Option 1: Keep the override semantics and fix the multi-component platform stripping only
* Option 2: `--platform` selects within the configured platforms and errors on unknown platforms
* Option 3: A new flag such as `--filter-platform` keeps `--platform` as the override

## Decision Outcome

Chosen option: `--platform` selects within the platforms that `hive.yml`, `image.yml` and the variant
configure, and rejects values that nothing configures. `platform.Resolve` intersects the configured list with
the requested platforms, keeps the configured order, and normalizes the values. A tag or variant whose
resolved list is empty is skipped and logged. A run where every selected tag is skipped warns and exits
successfully. A dependency must support the requested platform, or the build fails with an error that names
the dependency and the platform. `platform.Short` keeps the `linux/` prefix for multi-component platforms, so
the CI context passes values such as `linux/arm/v7` unchanged and the templates derive identifier-safe job
names and `dist/` paths from them.

The CI context still shortens `linux/amd64` to `amd64` for readability, because `platform.Normalize` maps the
short form back to the configured value.

## Pros and Cons of the Options

### Option 1: Keep the override semantics

* Good, because no user-visible behavior changes
* Good, because a user can build a platform for a quick check without touching `hive.yml`
* Bad, because a typo builds the wrong platform with no error
* Bad, because CI jobs can build platforms the project does not configure, test on them, and publish them
* Bad, because `linux/arm/v7` produced `--platform v7` in CI jobs, which skips the build

### Option 2: Select within the configured platforms (chosen)

* Good, because `hive.yml` and `image.yml` stay the single source of truth
* Good, because a typo fails with an error that lists the unsupported platforms
* Good, because the empty result is visible: each skipped tag logs, and an empty run warns
* Bad, because `ch build --platform` no longer builds a platform for a quick check without a config change

### Option 3: New `--filter-platform` flag

* Good, because no existing behavior changes
* Bad, because two flags with similar names and different semantics confuse users
* Bad, because the override behavior stays available and keeps failing builds silently

## Consequences

* `ch build --platform` errors when no project, image or variant configures the platform. Add the platform to
  the configuration or correct the value.
* The skip and warning behavior replaces the old silent override; see
  [Selecting platforms](../usage/multi-platform.md#selecting-platforms-with-platform).
* Tags and variants that do not support the requested platform are skipped per tag, so a shared image can
  serve agents with mixed runner platforms.

<!-- markdownlint-disable-file MD013 -->
