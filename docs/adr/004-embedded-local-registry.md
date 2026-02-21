---
id: 004
status: accepted
date: 2026-02-21
---

# Embedded local registry for inter-image dependency resolution

## Context and Problem Statement

ContainerHive allows images within a project to depend on each other using `__hive__/` references in Dockerfiles (e.g.
`FROM __hive__/base:1.0`). At build time these references are rewritten to point at a real registry so BuildKit can pull
the base image. In CI a production registry is available and images will be pushed there with a build-number suffix or
similar. But during local development and testing, requiring developers to configure and run an external registry just
to build images that reference each other creates unnecessary friction. How should ContainerHive resolve inter-image
dependencies locally without requiring external infrastructure?

## Decision Drivers

* Local builds must work out of the box without external services beyond BuildKit
* Developers need a fast feedback loop — spinning up infrastructure slows iteration
* The solution must be transparent: the same `__hive__/` Dockerfile syntax should work identically in local and CI
  builds
* No images should leak to a production registry during local testing
* In CI the production registry should be used directly — no extra infrastructure needed

## Considered Options

* Option 1: Embedded in-process OCI registry (zot)
* Option 2: Docker load + FROM via local Docker daemon
* Option 3: Combine dependent images into a single BuildKit solve call
* Option 4: Reference OCI layout directories directly in FROM
* Option 5: Always require a real registry, even locally

## Decision Outcome

Chosen option: "Option 1 — Embedded in-process OCI registry (zot)", because it provides a zero-configuration local
experience while keeping the same `__hive__/` rewriting mechanism used in CI. The registry starts in-process on a random
port, lives only for the duration of the build, and is discarded afterwards. In CI the same Registry interface switches
to a remote passthrough that pushes to the production registry instead.

## Pros and Cons of the Options

### Option 1: Embedded in-process OCI registry (zot)

Start an embedded zot instance on localhost during builds. Built images are pushed to it so that dependent images can
pull them via standard BuildKit FROM. The `__hive__/` prefix is rewritten to `127.0.0.1:<port>/` at build time.

* Good, because fully self-contained — no external services needed for local builds
* Good, because the registry is ephemeral, started and stopped automatically per build session
* Good, because the same `__hive__/` rewriting works for both local zot and a remote CI registry
* Good, because zot is a pure Go library, avoiding subprocess or Docker-in-Docker overhead
* Good, because nothing is pushed externally — safe for testing without polluting a production registry
* Bad, because it adds zot as a dependency, increasing binary size
* Bad, because the in-process registry must be ready before dependent builds start, adding a small startup delay

### Option 2: Docker load + FROM via local Docker daemon

After each image build, load the OCI tar into the local Docker daemon (`docker load`). Dependent images then reference
them as regular Docker images.

* Good, because it uses existing Docker infrastructure that developers likely have
* Bad, because it requires a running Docker daemon, which may not be available (e.g. rootless BuildKit setups)
* Bad, because it pollutes the local Docker image store with intermediate build artifacts
* Bad, because `docker load` is slow for large images
* Bad, because the rewriting mechanism would differ between local (Docker image names) and CI (registry references)

### Option 3: Combine dependent images into a single BuildKit solve call

Express all inter-image dependencies as a single BuildKit build graph so BuildKit resolves them internally.

* Good, because no registry or external tooling is needed at all
* Bad, because BuildKit's LLB API does not natively support referencing independently-defined Dockerfiles as stages
* Bad, because it would require fundamentally changing how ContainerHive renders and builds images
* Bad, because error reporting and caching become harder when everything is a single solve

### Option 4: Reference OCI layout directories directly in FROM

Use the built OCI tar / layout path directly in FROM instructions, bypassing a registry entirely.

* Good, because it avoids any network component
* Bad, because BuildKit does not support `FROM file:///path/to/layout` — there is no standard mechanism for this
* Bad, because it would require a custom BuildKit frontend or source provider, adding significant complexity

### Option 5: Always require a real registry, even locally

Require developers to run a registry (e.g. via Docker Compose) or point at a remote one for all builds.

* Good, because the build path is identical everywhere — no local/CI divergence
* Good, because it is simpler to implement (no embedded registry code)
* Bad, because it breaks the zero-configuration local experience — developers must manage extra infrastructure
* Bad, because pushing to a remote registry during local testing wastes bandwidth and may pollute tag namespaces
* Bad, because it slows down the feedback loop, especially with large images over the network

## Links

* [Zot registry](https://zotregistry.dev/) — OCI-native container registry used as the embedded implementation
* Relates to [ADR-001: BuildKit Integration](001-buildkit-integration.md) — the registry is needed because BuildKit
  builds each image independently
* `internal/registry/zot.go` — embedded zot implementation
* `internal/registry/remote.go` — CI remote registry passthrough
* `internal/buildkit/build_context/dockerfile.go` — `__hive__/` prefix rewriting

<!-- markdownlint-disable-file MD013 -->