# Use the CLI

ContainerHive's behavior can be controlled with various commands and flags.

You can get a full list by running

```bash
ch --help
```

## Global Flags

| Flag | Description |
|:-----|:------------|
| `--project`, `-p` | Project root directory |
| `--build-id` | Build ID appended to tags as `-build.<id>` |
| `--log-level` | Log level (`debug`, `info`, `warn`, `error`) — sourced from `LOG_LEVEL` env (default: `info`) |
| `--generate`, `-g` | Run `ch generate` before the command |
| `--refresh-versions` | Ignore the [`tag_ranges`](../configuration/image.md#tag_ranges) version cache TTL and refetch every external version source — sourced from `CH_REFRESH_VERSIONS` env |

See [Environment variables](../configuration/environment.md) for everything `ch` reads from the environment.

## Commands

### `generate`

Discover project and render configurations to `dist/`.

```bash
ch generate
```

This must run before `build`. It parses `hive.yml` and all image definitions.

### `build`

Build container images with BuildKit.

```bash
ch build [image:tag patterns...]
```

| Flag | Description |
|:-----|:------------|
| `--registry` | Use registry from config (auto-enabled in CI) |
| `--platform` | Target platform(s) to build (e.g. `linux/amd64`). Selects within the platforms configured in `hive.yml`/`image.yml`; tags and variants that support none of them are skipped. A platform that nothing configures is an error |
| `--max-shards` | Total number of shards to split work across — sourced from `CI_NODE_TOTAL` env (default: `1`) |
| `--current-shard` | This shard's 1-based index — sourced from `CI_NODE_INDEX` env (default: `1`) |

You can filter which images to build by passing image:tag patterns as arguments. In CI environments, the registry is auto-enabled. See [Sharding](sharding.md) for `--max-shards`/`--current-shard`.

### `test`

Run container structure tests on built images.

```bash
ch test [image:tag patterns...]
```

| Flag | Description |
|:-----|:------------|
| `--build` | Run build before tests |
| `--platform` | Target platform(s) to test (e.g. `linux/amd64`). Selects within the configured platforms like `build --platform`; also applies to `--build` |
| `--max-shards` | Total number of shards to split work across — sourced from `CI_NODE_TOTAL` env (default: `1`) |
| `--current-shard` | This shard's 1-based index — sourced from `CI_NODE_INDEX` env (default: `1`) |

See [Sharding](sharding.md) for `--max-shards`/`--current-shard`.

You can combine `--build` with `--generate` to run the full pipeline in one command:

```bash
ch test --generate --build
```

Uses [container-structure-test](https://github.com/GoogleContainerTools/container-structure-test) to validate built images against test definitions.

### `sbom`

Generate Software Bill of Materials for built images.

```bash
ch sbom [image:tag patterns...]
```

| Flag | Description |
|:-----|:------------|
| `--platform` | Target platform(s) to generate SBOMs for (e.g. `linux/amd64`). Selects within the configured platforms like `build --platform` |
| `--workers` | Number of concurrent workers for SBOM generation (default: 4) |
| `--no-cpe` | Skip CPE generation to reduce SBOM size (may reduce vulnerability match accuracy for CPE-based scanners) |
| `--max-shards` | Total number of shards to split work across — sourced from `CI_NODE_TOTAL` env (default: `1`) |
| `--current-shard` | This shard's 1-based index — sourced from `CI_NODE_INDEX` env (default: `1`) |

Generates CycloneDX JSON format SBOMs using [Syft](https://github.com/anchore/syft) as a Go library. No external tooling required. See [Sharding](sharding.md) for `--max-shards`/`--current-shard` — useful when many tags and variants push a single job's SBOM artifacts past a CI provider's size limit.

### `finalize`

Create multi-arch manifests and semantic version alias tags in the registry.

```bash
ch finalize [image:tag patterns...]
```

Creates manifests from platform-specific images and applies semantic version alias tagging. Requires a running registry.

### `verify`

Verify project structure.

```bash
ch verify
```

Validates `hive.yml` and image definitions. Useful as a quick check before building.

### `graph`

Print the dependency graph of all images and variants.

```bash
ch graph
ch graph --format dot | dot -Tsvg > graph.svg
```

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | `mermaid` | Output format: `mermaid` or `dot` |

Every image and every variant is a node, named `<image>` or `<image>/<variant>`. Tags are not nodes. An arrow points from a node to a node it depends on, which is built first. A dependency comes from a `FROM` or `parent` reference or from `depends_on`, so it does not always mean the image is built from the other one. The graph is read from the project source, so `ch generate` is not needed first, and `tag_ranges` are not expanded, so no version source is contacted. See [Dependency resolution](../how-it-works/dependency-resolution.md#visualizing-the-graph) for how references map to variants.

### `changes`

Print the images affected by changes since a base revision.

```bash
ch changes --base origin/main
git diff --no-renames --name-only origin/main... | ch changes --files-from -
```

The piped form needs the project root to be the repository root, because `git diff` prints repository-relative paths. For a project in a subdirectory, use `--base`.

| Flag | Default | Description |
|------|---------|-------------|
| `--base` | | Git revision to compare `HEAD` against, via their merge base |
| `--files-from` | | Read changed files, relative to the project root, one per line from this file (`-` for stdin) instead of git |
| `--unmatched` | `all` | Files outside every image directory: `all` rebuilds every image, `ignore` drops them |
| `--format` | `text` | Output format: `text` (one image per line) or `json` |
| `--strict` | `false` | Fail when the changed files cannot be computed from git, instead of treating every image as affected |
| `--write` | `false` | Store the result as `dist/changes.json`, which limits `build`, `test`, `finalize` and `sbom` to affected images |

Set exactly one of `--base` and `--files-from`. Like `graph`, the command reads the project source and does not expand `tag_ranges`. See [Build only changed images](change-detection.md).

### `lint`

Lint Dockerfiles in the project with [hadolint](https://github.com/hadolint/hadolint). The hadolint binary is embedded in `ch` — no separate install required.

```bash
ch lint
```

| Flag | Description |
|:-----|:------------|
| `--failure-threshold` | Lowest severity that causes a non-zero exit (`error`, `warning`, `info`, `style`, `ignore`). Overrides `lint.failure_threshold` from `hive.yml`. Defaults to `error`. |
| `--format` | Output format(s). Repeatable flag. Supported values: `terminal`, `github-actions`, `codeclimate=<path>`. Default: `terminal`. |

Findings are printed to stdout in `path:line:column level code: message` format by default. Templated Dockerfiles (files with a templating extension such as `Dockerfile.gotpl`) are **skipped** — hadolint cannot parse Go template syntax — and a warning is logged for each skipped file. Per-variant Dockerfiles are linted alongside the parent image.

Configure hadolint behaviour with a `lint:` block in `hive.yml`. See [Hive configuration](../configuration/hive.md#lint).

#### Output formats

One or more `--format` flags can be specified:

```bash
ch lint --format terminal
ch lint --format github-actions
ch lint --format codeclimate=gl-code-quality-report.json
ch lint --format terminal --format github-actions --format codeclimate=gl-code-quality-report.json
```

##### `terminal`

Colored text output to stdout. Used by default when no `--format` is given.

##### `github-actions`

Emits [GitHub Actions workflow command annotations](https://docs.github.com/en/actions/writing-workflows/workflow-commands-for-github-actions#setting-a-notice-message) to stdout, making findings appear inline on the diff in pull requests. Severities are mapped from hadolint to workflow commands as follows:

| hadolint  | GitHub Actions command |
|:----------|:-----------------------|
| `error`   | `::error`              |
| `warning` | `::warning`            |
| `info`    | `::notice`             |
| `style`   | `::notice`             |

The annotation includes the repo-relative file path, line and column numbers, the rule code as the title, and the finding message as the body. Messages are sanitized to prevent injection of false workflow commands.

Wire it into a GitHub Actions workflow step:

```yaml
- name: Lint Dockerfiles
  run: ch lint --format github-actions
```

##### `codeclimate=<path>`

Writes a [Code Climate](https://docs.gitlab.com/ee/ci/testing/code_quality.html)–compatible JSON report that GitLab CI can render inline on merge requests. Requires a file path suffix (e.g., `codeclimate=gl-code-quality-report.json`). Severities are mapped from hadolint to Code Climate as follows:

| hadolint  | Code Climate |
|:----------|:-------------|
| `error`   | `blocker`    |
| `warning` | `major`      |
| `info`    | `minor`      |
| `style`   | `info`       |

Wire it into `.gitlab-ci.yml` like any other code-quality artifact:

```yaml
lint:
  script:
    - ch lint --format codeclimate=gl-code-quality-report.json
  artifacts:
    when: always
    reports:
      codequality: gl-code-quality-report.json
```

Every parsed finding lands in the report — even ones below the failure threshold — so GitLab surfaces the full picture while the command's exit code still reflects the threshold.

### `template ci`

Generate CI pipeline configuration.

```bash
ch template ci --provider <provider> --output <path>
```

| Flag | Description |
|:-----|:------------|
| `--provider` | CI provider (`gitlab` or `github`) |
| `--output` | Output file (default: stdout) |
| `--template-dir` | Custom template directory (overrides built-in templates) |
| `--artifacts` | Upload/download build artifacts between jobs |
| `--version` | CH CLI version to use in CI templates (default: current CLI version) |
| `--image-name` | Container image name for the CH CLI (default: `ghcr.io/containerhive/containerhive` for `--provider github`, `registry.gitlab.com/containerhive/containerhive` for `--provider gitlab`) |

### `template custom`

Render custom Go templates with project context.

```bash
ch template custom --template <path.gotpl> --output <path>
```

| Flag | Description |
|:-----|:------------|
| `--template` | Path to Go template file (`.gotpl`) |
| `--output` | Output path rendered as a Go template (default: stdout). References to `.Image` render once per image, one file each |

### `wait`

Wait for infrastructure dependencies to become available.

```bash
ch wait
```

| Flag | Description |
|:-----|:------------|
| `--buildkitd` | Wait for BuildKit daemon (uses `$BUILDKIT_HOST`, default `tcp://127.0.0.1:8372`) |
| `--docker-socket` | Wait for Docker daemon (uses `$DOCKER_HOST`, default `unix:///var/run/docker.sock`) |
| `--timeout` | Maximum time to wait (default: 1m) |

### `login`

Log in to a container registry.

```bash
ch login <registry> -u <username> -p <password>
```

| Flag | Description |
|:-----|:------------|
| `--username`, `-u` | Username |
| `--password`, `-p` | Password |
| `--password-stdin` | Take the password from stdin |

### `dev buildkitd`

Manage a local BuildKit daemon container for development. See [Local development](local-development.md) for full details.

```bash
ch dev buildkitd start
ch dev buildkitd stop
ch dev buildkitd status
ch dev buildkitd logs
```

**`start` flags:**

| Flag | Description |
|:-----|:------------|
| `--image` | BuildKit image to use (`image:tag`). Defaults to the version configured in `hive.yml` `template_options` or the bundled version |
| `--port` | Host port to bind |
| `--timeout` | Maximum time to wait for buildkitd to become ready (default: 1m) |

**`stop` flags:**

| Flag | Description |
|:-----|:------------|
| `--remove` | Also remove the container after stopping |

**`logs` flags:**

| Flag | Description |
|:-----|:------------|
| `--follow`, `-f` | Follow log output |

### `report`

Generate an HTML or JSON report of container images.

```bash
ch report
```

| Flag | Description |
|:-----|:------------|
| `--output` | Output file path (default: `dist/report.html`) |
| `--json` | Output JSON instead of HTML |

Generates a report containing information about all configured images, including their build status, dependencies, and metadata.

#### Customizing the report

To customize the look of the HTML report, set `report.styleSheet` in your `hive.yml` to a path pointing at a CSS file. The contents of that file are embedded verbatim into the generated HTML via a `<style>` block. See [Report theme](../configuration/report-theme.md) for the full list of overridable variables.

#### URL parameters

The HTML report supports the following query parameters:

| Parameter | Description |
|:----------|:------------|
| `?hide-header` | Hides the top header bar. Useful when embedding the report in an iframe or dashboard. |

### `license`

Show third-party license notices.

```bash
ch license
```

### `mcp`

Start a Model Context Protocol (MCP) server for AI tool integration.

```bash
ch mcp
```

See [MCP integration](mcp.md) for full details.
