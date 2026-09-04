# Split work across shards

`ch build`, `ch sbom` and `ch test` can split the images and tags they process across a fixed number of CI jobs, so a large project can build, generate SBOMs, and test in parallel instead of one long serial job.

The main motivation is GitLab.com's SaaS limits on SBOM report artifacts: **10 MB per file, 20 MB per job**. A single build job today uploads the CycloneDX SBOM of every tag and variant of an image, so an image with many tags or variants can push the job over that budget — and the job is also slower because all of that work runs serially. Splitting one job into several `parallel:` instances, each owning a disjoint slice of the tags, keeps every job under the artifact ceiling and cuts wall-clock time on variant-heavy images.

## Shard unit

Every base tag and every variant tag is its own shard unit, addressed by the image and the exact tag name. This is deliberately fine-grained: an image with 20 variants of one base tag needs to split at the variant level, not the base-tag level, or a single job would still hold all 20 SBOMs.

One rule follows from this: a variant's Dockerfile is `FROM` its own base tag (e.g. `dotnet:8.0.200-node` is `FROM __hive__/dotnet:8.0.200`), and that dependency is invisible to ContainerHive's dependency graph because it's a same-image reference. So **a shard that owns a variant also builds that variant's base tag**, even if the base isn't assigned to it by exact match. The base may therefore be built redundantly by more than one shard — cheap, since BuildKit caches it and the resulting push is content-identical — while `ch sbom` and `ch test` still gate on exact tag ownership, so SBOM output stays strictly disjoint across shards. That disjointness is what the per-job artifact budget depends on.

## Flags

| Flag | Description |
|:-----|:------------|
| `--max-shards` | Total number of shards to split work across — sourced from `CI_NODE_TOTAL` env (default: `1`) |
| `--current-shard` | This shard's 1-based index — sourced from `CI_NODE_INDEX` env (default: `1`) |

`--max-shards`/`--current-shard` and `CI_NODE_TOTAL`/`CI_NODE_INDEX` behave as flag over env over default. GitLab sets `CI_NODE_TOTAL`/`CI_NODE_INDEX` automatically for a job declared with `parallel: N`, so sharding works with no flags at all under `parallel:`. The variable names are vendor-neutral by design — a CI provider without native parallel-job fan-out can export the same two variables from a wrapper script and get the same behavior.

## GitLab template options

The generated GitLab pipeline exposes two [template options](ci-integration.md) to control shard count without hand-editing the pipeline:

| Option | Description |
|:-------|:------------|
| `ci_build_shards` | Number of parallel instances for each build job (default: `1`) |
| `ci_test_shards` | Number of parallel instances for each test job (default: `1`) |

```yaml
# hive.yml
template_options:
  ci_build_shards: "5"
```

At the default of `1`, the rendered pipeline has no `parallel:` key at all — not `parallel: 1`, which is meaningless and renames the job to `... 1/1`. Regenerate the pipeline after changing either option:

```bash
ch template ci --provider gitlab --output .gitlab-ci.yml --image-name <your-image>
```

## Build and SBOM must run in the same job

`ch sbom` only reads local `dist/<image>/<tag>/<platform>/image.tar` — it never pulls from a registry, deliberately, since pulling a full image just to produce a comparatively small SBOM is the cost sharding exists to avoid. This means a shard's `ch build` and `ch sbom` must run in the same CI job, so the tars `ch sbom` needs are already on disk. The generated GitLab pipeline already does this (`ch build` followed by `ch sbom` in the same `.build` job template), so setting `ci_build_shards` is sufficient — no extra wiring needed. If you assemble a pipeline by hand, keep this pairing.

## Cross-image dependencies

A project with `__hive__/` references between *different* images (not the same-image base/variant case above) needs the dependency image fully built and pushed before a dependent shard starts, since a missing local tar falls back to a registry pull. The generated GitLab pipeline already stages this correctly — each build job depends on the `manifest-<image>` job of its dependencies — so this is only a concern for hand-rolled pipelines. Enabling sharding on a project with such dependencies logs a warning rather than failing, since the registry fallback is a legitimate (if slower) path.

## Zero-unit shards

More shards than units is a normal, supported configuration — for example `ci_build_shards: 10` on a project with a single image. Assignment is modulo round-robin over a canonically sorted unit list, so a single unit always lands in shard 1 and shards 2 through 10 simply own nothing. A shard that owns nothing logs `"Nothing to do for this shard"` and exits `0`, so an over-provisioned shard count never fails the pipeline.

## Cross-command alignment

`ch build`, `ch sbom` and `ch test` compute the same canonical, sorted list of shard units from the project, so shard `N` of `M` always selects the same tags in every command — `ch build --max-shards 3 --current-shard 2` and `ch sbom --max-shards 3 --current-shard 2` operate on the same slice.

## Limits

Sharding addresses the **20 MB per-job** artifact limit by splitting work across jobs. It does not help with the **10 MB per-file** limit — a single `cyclonedx.json` for one tag on one platform that is itself larger than 10 MB is not made smaller by adding more shards, since the shard unit is one tag, not a fraction of one SBOM.

`ch finalize` is never sharded. It runs once, after all shards have completed, and needs the project's complete tag set to resolve multi-arch manifests and semantic version aliases correctly.

See [Filter images and tags](filtering.md) for `image:tag` filters, which compose with sharding: filters select which images and tags are in scope at all, and sharding partitions whatever remains after filtering.
