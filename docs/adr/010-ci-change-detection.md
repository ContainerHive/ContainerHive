---
id: "010"
status: accepted
date: 2026-10-30
---

# Change detection in generated CI pipelines

## Context and Problem Statement

Generated pipelines build every image on every merge or pull request, even when only one leaf image changed. ContainerHive knows the dependency graph, so it can compute the affected images: the changed images and all their dependents. How should the generated pipelines leave out the other images, given that GitHub Actions decides at run time and GitLab decides which jobs exist when it creates the pipeline?

## Decision Drivers

* Only jobs for affected images should run; on GitLab they should not even be created.
* The set of images a job builds against must match the jobs that exist. A dependent resolves its dependency to the `-build.<id>` tag only if that dependency is rebuilt in the pipeline.
* Keep the generated pipeline structure and the default branch behavior unchanged.

## Considered Options

* GitLab: static `rules:changes` per job, generated from the dependency graph
* GitLab: dynamic child pipeline rendered at run time for the affected images
* Jobs always exist and exit early for unaffected images

## Decision Outcome

Chosen option: "static `rules:changes` per job" for GitLab, and job `if:` conditions on a `generate` job output for GitHub. Both read the same rules from `ch changes`.

The `generate` job writes `dist/changes.json`. `build`, `test`, `finalize` and `sbom` only process affected images, and a dependency outside the affected set resolves to its production tag.

GitLab cannot negate globs in `rules:changes`, so a rule cannot match "any file outside the image directories". GitLab therefore runs `ch changes --unmatched ignore --strict`, which keeps the affected set within the jobs GitLab created. On GitHub, unmatched files rebuild every image, the safer default. The generated GitLab pipeline runs as branch pipelines, so the rules apply to push pipelines of non-default branches and compare to the default branch.

## Pros and Cons of the Options

### GitLab: static `rules:changes` per job

* Good, because GitLab never creates the jobs, and the pipeline keeps its shape and its MR widgets.
* Good, because the paths come from the same graph as the build order.
* Bad, because unmatched files are ignored on GitLab, unlike on GitHub.
* Bad, because `compare_to` with variables needs GitLab 17.2 or later.

### GitLab: dynamic child pipeline

* Good, because the semantics match GitHub exactly.
* Bad, because the template needs a trigger job and a second pipeline. Reports and widgets move into the child pipeline.

### Jobs always exist and exit early

* Good, because it is simple and the same on both providers.
* Bad, because every job still starts a runner and its BuildKit or Docker service.

<!-- markdownlint-disable-file MD013 -->
