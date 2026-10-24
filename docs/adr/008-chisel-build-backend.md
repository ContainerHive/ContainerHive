---
id: 008
status: accepted
date: 2026-10-25
---

# Native Chisel build backend

## Context and Problem Statement

ContainerHive builds images from Dockerfiles and, experimentally, from Nix flakes. A third source of filesystems
exists: Canonical's Chisel cuts minimal package slices from the Ubuntu archives into a rootfs. Projects that serve
distroless-style images write multi-stage Dockerfiles only to run `chisel cut` and copy the staging rootfs to
scratch. The question: how does ContainerHive support chisel natively, with reproducible inputs and a build cache
that warm builds reuse?

## Decision Drivers

* Match the shape of the existing Nix backend: one entrypoint file per image, builder options in image.yml
* Keep `ch` and the CI pipeline free of chisel dependencies
* Cache the chisel-releases input and the deb downloads between builds
* Keep the maintenance overhead low — no published builder image, no extra release surface
* Clear failure modes: non-reproducible builds must warn, ambiguous setups must fail

## Considered Options

* Option 1: Shell-like multi-stage Dockerfiles, documented but not integrated
* Option 2: Chisel built-in release fetch inside the build, cached via a cache mount
* Option 3: ContainerHive-managed git checkout of the release repo inside the build, cached via a cache mount

## Decision Outcome

Chosen option: a new experimental `chisel` build backend that combines Option 2 and 3, built on the same custom
BuildKit frontend pattern as the Nix backend:

- The entrypoint is `chisel.yaml`, a ContainerHive cut spec: the release source and the slice selection.
- The release resolves from a branch name of the canonical `chisel-releases` repository, or from a git mapping with
  a pinned ref for forks and reproducible releases. The build clones or updates the repository itself, inside
  BuildKit. Chisel's own lazy release fetch never runs.
- The build has two cached stages. Stage one installs git and ca-certificates into the builder image and extracts a
  pinned chisel release tarball into `/usr/local/bin`. Stage two cuts the rootfs with `chisel cut --arch <deb-arch>`.
  No image ships per-release: a plain base image plus content-addressed downloads keeps the overhead at zero.
- The release checkout and the deb cache persist in the `hive-chisel-cache` cache mount. `cache: false` opts out.
- The assembly order of the output filesystem is: parent rootfs, cut files, static `rootfs/` files.

Pro archives stay unsupported until someone requests them — the maintainers cannot test credentials.

## Consequences

* Good, because chisel images get caching, testing, SBOMs and pipelines without a Dockerfile.
* Good, because a pinned git ref makes deb and release inputs reproducible.
* Bad, because foreign-arch builds run under emulation, and `linux/386` has no chisel tool binary.
* Bad, because the cache mount grows with archive churn and needs manual pruning.
