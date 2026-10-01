---
id: 006
status: accepted
date: 2026-09-29
ticket: "268"
---

# Use registry retention policies instead of push-gating

## Context and Problem Statement

`ch build` and `ch finalize` push images to the configured registry unconditionally on every pipeline run — including
merge request (MR) and pull request (PR) pipelines. In projects with high MR churn (e.g. automated dependency-update
bots), this permanently pushes tagged, semver-aliased images into the production registry via `ch finalize`, plus
untagged per-platform intermediates via `ch build`. The result is thousands of orphaned, superseded images with no
built-in mechanism to suppress pushing for non-default branches.

Should ContainerHive add a flag or config to skip pushes on MR/PR pipelines, or should cleanup be handled via
external retention policies?

## Decision Drivers

* Pushing intermediates is required by `ch test` (which pulls them back for validation) and `ch finalize` (which
  assembles multi-arch manifests from platform-specific images)
* A `--skip-push` flag would break the standard MR test-and-finalize pipeline
* Tag namespacing via `--build-id` already isolates MR tags from production tags (`-build.<id>` suffix)
* Retention policies are available across all major container registries and operate at the registry level,
  independent of the build tool
* Users already configure registry access — adding retention policies is a natural extension of that configuration

## Considered Options

* Option 1: Add a `--skip-push` / `--dry-run` flag on `ch build` and `ch finalize`
* Option 2: Add a `push_on_branch` setting in `hive.yml` to gate pushes by CI branch
* Option 3: Add a `ci_finalize_default_branch_only` template option for generated CI pipelines
* Option 4: Document and recommend registry-level retention policies as the supported approach

## Decision Outcome

Chosen option: "Option 4 — Document retention policies", because it addresses the root cause (accumulated images)
without breaking the existing test/finalize pipeline, applies uniformly across all registries and CI providers,
and shifts lifecycle management to where it naturally belongs — the registry itself.

The `--build-id` suffix convention (`-build.<id>`) already isolates MR images from production tags, making them
ideal targets for age-based or pattern-based retention rules.

## Pros and Cons of the Options

### Option 1: `--skip-push` / `--dry-run` flag

A CLI flag to suppress pushing during build and finalize.

* Good, because simple UX — single flag
* Good, because zero registry-level configuration needed
* Bad, because `ch test` cannot pull intermediates that were not pushed — breaks the standard MR pipeline
* Bad, because `ch finalize` needs platform images in the registry to assemble manifests
* Bad, because users must manually orchestrate partial builds to avoid pushing, losing the full pipeline validation

### Option 2: `push_on_branch` in `hive.yml`

A config field that gates pushing to specific CI branches.

* Good, because declarative and version-controlled
* Good, because works across all CI providers via environment variable detection
* Bad, because same fundamental issue — `ch test`/`finalize` still need pushed intermediates
* Bad, because introduces branch-awareness into ContainerHive's core, coupling it to CI semantics
* Bad, because per-branch behavior is hard to test and document

### Option 3: `ci_finalize_default_branch_only` template option

A template option that adds branch conditions to the generated CI pipeline's finalize job.

* Good, because scoped to CI pipeline generation — no core changes
* Good, because preserves full MR pipeline validation (build + test still run)
* Bad, because only gates the finalize step, not the per-platform intermediate pushes from `ch build`
* Bad, because adds provider-specific condition logic to templates that must be maintained per CI provider
* Bad, because does not address cache accumulation

### Option 4: Document retention policies

* Good, because works with the existing push behavior — no pipeline breakage
* Good, because addresses both snapshot images and cache accumulation
* Good, because registry-native — lifecycle management belongs in the registry, not the build tool
* Good, because applies uniformly across all registries and CI providers
* Good, because no code changes — docs-only, ships immediately
* Bad, because requires user action to set up retention policies (not automatic out of the box)
* Bad, because registry-specific configuration varies — requires documentation per provider

## Links

* [Issue #268](https://github.com/ContainerHive/ContainerHive/issues/268)
* [Registry retention docs](../usage/registry-retention.md)
* [PR #269 (closed)](https://github.com/ContainerHive/ContainerHive/pull/269)

<!-- markdownlint-disable-file MD013 -->
