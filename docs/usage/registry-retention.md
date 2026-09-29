# Clean up registry images with retention policies

Every pipeline run — including merge request (MR) and pull request (PR) pipelines — pushes images to the configured
registry. Without cleanup, these images accumulate indefinitely. This page explains what gets pushed and how to set up
retention policies to keep your registry tidy.

## Why images accumulate

ContainerHive pushes images on every CI run because downstream steps depend on them:

1. `ch build` pushes per-platform intermediate images so `ch test` can pull them back for validation.
2. `ch finalize` creates multi-arch manifests and pushes semantic version aliases.
3. With `--build-id`, all tags are suffixed with `-build.<id>` (e.g. `1.0.0-build.mr-42`), so each MR/PR run leaves
   permanent, distinct tags behind — even if the MR is never merged.

Because skipping these pushes would break the test and finalize steps, the recommended approach is to configure
retention policies on the registry itself to automatically clean up images after a suitable grace period.

## What gets pushed

| Image kind                 | Example tag                     | When pushed                     | Retention target                                  |
|:---------------------------|:--------------------------------|:--------------------------------|:--------------------------------------------------|
| Per-platform intermediates | `1.0.0.linux-amd64`             | Every run (overwritten per tag) | Overwrite-safe; accumulate only with `--build-id` |
| Build-ID intermediates     | `1.0.0.linux-amd64-build.mr-42` | MR/PR runs with `--build-id`    | Short-lived (days)                                |
| Build-ID manifests         | `1.0.0-build.mr-42`             | MR/PR runs with `--build-id`    | Short-lived (days)                                |
| Build-ID aliases           | `1-build.mr-42`                 | MR/PR runs with `--build-id`    | Short-lived (days)                                |
| Default branch tags        | `1.0.0`, `1.0`, `1`             | Main/default branch only        | Keep (long-lived)                                 |
| Registry cache layers      | Under `cache.ref` repository    | Every run                       | Medium-lived (weeks)                              |

To target snapshot images for cleanup without touching production tags, match tags containing `-build.` as the
infix — for example `*-build.*` — or filter by image name pattern.

## Recommended retention windows

| Category                     | Retention         | Rationale                                                       |
|:-----------------------------|:------------------|:----------------------------------------------------------------|
| Snapshot images (`-build.*`) | A few days        | MRs are typically reviewed and merged (or closed) within days   |
| Cache layers                 | A few weeks       | Older cache layers have diminishing hit rates and waste storage |
| Default branch tags          | Keep indefinitely | These are your production releases                              |

## Per-registry setup

### GitLab Container Registry

GitLab has a
built-in [cleanup policy](https://docs.gitlab.com/user/packages/container_registry/reduce_container_registry_storage/#how-the-cleanup-policy-works)
per repository. It runs on a schedule (daily or weekly) and can remove tags matching a name pattern.

**Recommended configuration:**

1. Navigate to **Settings > Packages and registries > Container Registry** in your project.
2. Enable **Cleanup policy**.
3. Set **Keep tags matching**: `^[0-9].*` (keeps version tags like `1.0.0`, `1.2.3`).
4. Set **Remove tags matching**: `.*-build\..*` (removes all build-ID suffixed tags).
5. Set **Expiration**: 3 days for `.*-build\..*` tags, keep version tags indefinitely.
6. Schedule: **weekly** is sufficient for most projects.

Alternatively, use the [GitLab API](https://docs.gitlab.com/api/container_repository_tags/) in a scheduled CI job:

```yaml
cleanup:
  stage: cleanup
  image: curlimages/curl
  rules:
    - if: $CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH
      when: always
  script:
    - |
      curl --request DELETE \
        --header "PRIVATE-TOKEN: $GITLAB_CLEANUP_TOKEN" \
        "https://gitlab.com/api/v4/projects/$CI_PROJECT_ID/\
          container_repository?tag_name_regex=.*-build\..*&expire_in=3d"
  allow_failure: true
```

!!! note "Protected and shared tags"
GitLab's cleanup policy skips tags marked as protected or shared. Ensure your MR build-ID tags are
not accidentally protected — this is rarely an issue since build-ID tags are transient.

### GitHub Container Registry (ghcr.io)

GitHub does not provide a built-in retention policy for the Container Registry. Use the
[container-retention-policy](https://github.com/snok/container-retention-policy) action in a scheduled workflow:

```yaml
name: Cleanup container images

on:
  schedule:
    - cron: "0 0 * * 0"  # weekly on Sunday

jobs:
  cleanup:
    runs-on: ubuntu-latest
    permissions:
      packages: write
      contents: read
    steps:
      - name: Delete old snapshot images
        uses: snok/container-retention-policy@v3
        with:
          image-name: ".*"
          tag-pattern: ".*-build\\..*"
          cut-off: "3d"
          keep-at-least: 0
          skip-tags: "latest,^[0-9]+\\.[0-9]+\\.[0-9]+$"
          account-type: "org"
          org-name: "your-org"
          token: ${{ secrets.DELETE_PACKAGES_TOKEN }}
```

**Setup steps:**

1. Create
   a [personal access token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens)
   (classic) with the `delete:packages` scope — or a fine-grained token with `packages: write` on the target org.
2. Store it as a repository secret named `DELETE_PACKAGES_TOKEN`.
3. Add the workflow above to `.github/workflows/cleanup.yml` and adjust `org-name` and `image-name` as needed.

!!! warning "Permissions required"
The built-in `GITHUB_TOKEN` does not have permission to delete packages. A PAT or a fine-grained token
with `packages: write` is required. The `org-name` field is only needed for organization-owned registries —
remove it for user accounts.

### Amazon Elastic Container Registry (ECR)

ECR supports [lifecycle policies](https://docs.aws.amazon.com/AmazonECR/latest/userguide/LifecyclePolicies.html)
that automatically expire images. Create a lifecycle policy JSON file matching the repository:

```json
{
  "rules": [
    {
      "rulePriority": 1,
      "description": "Expire build-ID snapshot images after 3 days",
      "selection": {
        "tagStatus": "tagged",
        "tagPatternList": [
          "*-build.*"
        ],
        "repositoryName": "your-repo",
        "countType": "sinceImagePushed",
        "countUnit": "days",
        "countNumber": 3
      },
      "action": {
        "type": "expire"
      }
    },
    {
      "rulePriority": 2,
      "description": "Keep at most 100 untagged images",
      "selection": {
        "tagStatus": "untagged",
        "repositoryName": "your-repo",
        "countType": "imageCountMoreThan",
        "countNumber": 100
      },
      "action": {
        "type": "expire"
      }
    }
  ]
}
```

Apply it with the AWS CLI:

```bash
aws ecr put-lifecycle-policy \
  --repository-name your-repo \
  --lifecycle-policy-text file://lifecycle-policy.json
```

!!! tip "Apply to all repositories"
Omit `repositoryName` from the selection to apply the rule across all repositories in the registry.

### Harbor

Harbor supports [tag retention rules](https://goharbor.io/docs/latest/administration/gc-tag-retention/)
at the project level.

1. Navigate to your project in the Harbor UI.
2. Go to **Interrogation Services > Tag Retention Rules**.
3. Create a new rule:
    - **Repository matching**: `**/*` (all repositories in the project)
    - **Tag matching**: `.*-build\..*`
    - **Retention**: 3 days
    - **Most recently pushed**: N/A (use age-based)
4. Add a second rule for cache images:
    - **Repository matching**: the cache repository (e.g. `cache/*`)
    - **Tag matching**: `**`
    - **Retention**: 3 weeks

You can also define retention rules via the [Harbor API](https://goharbor.io/docs/latest/swagger-ui/):
`POST /api/v2.0/retentions`.

### Docker Hub

Docker Hub does not provide built-in retention policies. For automated cleanup, use scheduled jobs with the
[Docker Hub API](https://docs.docker.com/docker-hub/api/#operation/RepositoryDelete):
delete tags matching your snapshot pattern on a schedule.

```bash
# Delete a specific tag
curl -X DELETE \
  -H "Authorization: Bearer $DOCKER_HUB_TOKEN" \
  "https://hub.docker.com/v2/repositories/your-org/your-repo/tags/1.0.0-build.mr-42"
```

For a ready-made solution, consider the
[docker-hub-delete-tags](https://github.com/lostutils/docker-hub-delete-tags) action or similar community tools
that support regex-based tag deletion.

!!! warning "Rate limits"
    Docker Hub API rate limits apply. For projects with many tags, batch deletions and space your API calls
    to avoid hitting limits.

### Generic OCI registries

For self-hosted OCI registries (zot, Distribution, Quay, JFrog Artifactory, etc.), use the registry's own tag
management API or a scheduled job that iterates tags and deletes those matching your snapshot pattern.

If you use BuildKit with a registry cache backend (configured via `cache.ref` in `hive.yml`), the cache repository
will also grow over time. Set up a separate, longer-lived retention policy (e.g. 3 weeks) on the cache repository
to keep recent layers while discarding stale ones.

## Links

- [Use build IDs for merge requests](build-ids.md)
- [Build caching](caching.md)
- [CI integration](ci-integration.md)
- [Issue #268 — Support skipping/gating registry pushes](https://github.com/ContainerHive/ContainerHive/issues/268)

<!-- markdownlint-disable-file MD013 -->
