---
id: 005
status: accepted
date: 2026-09-09
ticket: 241
---

# Dynamically generated tags from external version sources (`tag_ranges`)

## Context and Problem Statement

Images that track upstream releases (Node.js, Python, .NET) need every tag hand-listed
in `image.yml`. That list goes stale and misses new releases; keeping it current is
manual, repetitive work. Issue #241 asks for `tag_ranges`: a declarative range that
queries an external version source (a JSON feed, an OCI registry, GitHub releases) at
discovery time and generates concrete tags plus their `versions` values. This introduces
several choices that are easy to get wrong later: a new outbound-network dependency
inside a previously offline discovery step, a new expression language for extracting
versions from arbitrary JSON, a cache with a failure policy, and a mechanism for keeping
sharded CI jobs in agreement about the resolved tag set.

## Decision Drivers

* Discovery is called from nearly every `ch` command; the feature must not make
  read-adjacent commands unexpectedly slow, flaky, or dependent on an upstream API's
  uptime.
* CI sharding runs each unit's build in a separate process; whatever those processes
  resolve must agree, or shards can overlap and leave tags unbuilt.
* The transform language must support the issue's own example verbatim
  (`$[lts != false].{...}` against `nodejs.org/dist/index.json`).
* Generated tags must flow through the existing tag pipeline (rendering, building,
  sharding, aliasing) unchanged wherever possible.

## Considered Options

* Transform language: JSONata dependency vs. a bespoke selector DSL
* Semver library for range selection: extend `internal/semantic_tags` vs. add
  `Masterminds/semver/v3`
* Registry source: hand-roll the Docker Hub v2 API vs. reuse
  `google/go-containerregistry`'s `remote.List`
* Cache failure policy: hard-fail on a cold miss vs. serve a stale entry
* Cache directory: XDG default vs. always project-relative
* CI determinism: resolve once per pipeline via network-mode differentiation
  (deferred) vs. resolve-per-invocation with a warmed, shared cache (chosen for now)
* Sharding: keep positional (`index % Max`) assignment vs. switch to hashing each unit's
  own identity

## Decision Outcome

Chosen option, per-area:

* **JSONata** (`github.com/blues/jsonata-go`) for `type: json` transforms, wrapped
  behind `internal/jsonata.EvalTransform` rather than exposed directly, so the engine can
  be swapped without touching callers. Verified against the issue's exact expression in
  `internal/tagrange/source/json_test.go`.
* **`Masterminds/semver/v3`** for range selection and filtering, kept separate from
  `internal/semantic_tags`. `internal/semantic_tags.Compare` ignores the prerelease
  suffix entirely (`1.0.0-rc1` compares equal to `1.0.0`), which is load-bearing for
  `exclude_prerelease` and "highest wins" selection; fixing that would change alias
  behavior for every existing project. `semantic_tags` keeps its own job: parsing *tag*
  strings that carry non-semver flavour suffixes (`24.11-alpine`) and prefixes.
* **`go-containerregistry`'s `remote.List`** for the `registry`/`dockerhub` source,
  reusing an existing direct dependency, inheriting credentials from the same docker
  config `ch login` already writes, and working against any OCI registry rather than
  only Docker Hub.
* **Hard-fail on a cold miss.** A fetch failure with no valid cached entry is always an
  error; the cache never serves an expired entry as a silent fallback. Predictable over
  convenient — a build that silently used week-old versions would be a worse failure
  mode than an explicit error.
* **Project-relative cache directory**, resolved as `CONTAINER_HIVE_CACHE_DIR` env (wins)
  → `hive.yml`'s `cache_dir` → the XDG cache directory. The XDG default only works for
  local development; CI's generated jobs run `ch` in a container with just
  `$RUNNER_TEMP/ch`/`$GITHUB_WORKSPACE` (GitHub) or require paths under
  `$CI_PROJECT_DIR` (GitLab) to be cacheable, so the CI templates set
  `CONTAINER_HIVE_CACHE_DIR` to a project-relative path (`.ch-cache`, cached across runs
  via `actions/cache` / GitLab's `cache:`).
* **Resolve-per-invocation with a shared, CI-cached disk cache**, not a `dist/`
  manifest, for the current iteration. `ch generate`/`ch build`/`ch lint` each resolve
  `tag_ranges` themselves; the CI-cached TTL cache (§ above) keeps this cheap and, once
  the generate job's cache is warm, keeps other jobs off the network in practice. A
  `dist/tag-ranges.json` resolved manifest with per-command network-mode
  differentiation (`ModeManifest`/`ModeCache`/`ModeResolve`/`ModeOffline`) was designed
  and remains the intended follow-up for guaranteeing (not just making likely) that
  read-only commands never touch the network — see Open Questions.
* **Hash-based shard assignment**, changed in the same effort as this feature though
  logically a prerequisite fix: `Owns` now hashes a unit's own identity
  (`fnv32a(identifier + tagName)`) instead of its position in the sorted `TagIndex`.
  Positional `index % Max` reassigns nearly the entire project to different shards when
  a single tag is added or removed — exactly what `tag_ranges` does routinely as
  upstream publishes new versions — producing shard sets that overlap in some places and
  leave gaps in others. Hashing means only the tags that actually changed can move
  shard. This is a one-time reshuffle of existing projects' shard assignment, not a
  correctness regression: coverage (each unit owned exactly once) is unchanged.

## Pros and Cons of the Options

### JSONata dependency

* Good, because it supports the issue's example verbatim, including predicate filters
  and object construction — no bespoke DSL could match this for free.
* Bad, because `blues/jsonata-go` targets jsonata-js **1.5.4**: an expression a user
  copies from a newer JSONata Exerciser (2.x) may not compile. Documented as a known
  subset in `docs/configuration/image.md` and in the wrapper's own comment.
* Bad, because `*jsonata.Expr`'s concurrency safety is undocumented — mitigated by
  compiling per evaluation rather than sharing a compiled expression across goroutines,
  and by a 5-second evaluation timeout (`jsonata-go` has no cancellation of its own).

### Resolve-per-invocation vs. a `dist/` manifest

* Resolve-per-invocation (chosen): simpler to implement and reason about; every command
  behaves the same way; the CI version cache still keeps steady-state network calls to
  roughly one fetch per source per pipeline.
* Bad, because a command that never downloads `dist/` in its generated CI job (`ch
  lint`) still performs a cache lookup — and, on a cold cache, a live fetch — rather than
  being guaranteed offline.
* `dist/` manifest (deferred): would make every post-`generate` command provably
  network-free and immune to a TTL boundary crossed mid-pipeline, at the cost of a mode
  parameter threaded through every CLI command and MCP tool call, and a decision about
  what "stale manifest" means for each of them. Left as a follow-up once real-world
  usage shows whether the simpler cache-based approach is sufficient.

### Hash-based vs. positional shard assignment

* Hash-based (chosen): adding or removing one tag only moves that tag between shards.
* Bad, because it reshuffles which CI node builds which unit for every existing project
  on first use after this change — a one-time cache-locality cost, not a correctness
  issue.

## Links

* Refined by: the deferred `dist/tag-ranges.json` manifest and CLI mode
  differentiation (§ Decision Outcome, "Resolve-per-invocation") — tracked for a
  follow-up once this iteration has real-world usage.
* Related: `internal/buildconfig_resolver`'s tag-vs-image `build_args` merge order is
  inverted relative to `versions` (tag-level values can be silently overwritten by
  image-level ones); `tag_ranges`-generated `build_args` are affected identically to
  static ones. Not fixed as part of this change — tracked as a separate, pre-existing
  issue.

<!-- markdownlint-disable-file MD013 -->
