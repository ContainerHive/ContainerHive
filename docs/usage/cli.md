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
| `--build-id` | Build ID appended to tags as `+<id>` |
| `--log-level` | Log level (`debug`, `info`, `warn`, `error`) — sourced from `LOG_LEVEL` env (default: `info`) |
| `--generate`, `-g` | Run `ch generate` before the command |

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
| `--platform` | Target platform(s) to build (e.g. `linux/amd64`), overrides `hive.yml` |

You can filter which images to build by passing image:tag patterns as arguments. In CI environments, the registry is auto-enabled.

### `test`

Run container structure tests on built images.

```bash
ch test [image:tag patterns...]
```

| Flag | Description |
|:-----|:------------|
| `--build` | Run build before tests |

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
| `--platform` | Target platform(s) to generate SBOMs for (e.g. `linux/amd64`), overrides `hive.yml` |
| `--workers` | Number of concurrent workers for SBOM generation (default: 4) |

Generates CycloneDX JSON format SBOMs using [Syft](https://github.com/anchore/syft) as a Go library. No external tooling required.

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
| `--image-name` | Container image name for the CH CLI (default: `containerhive/containerhive`) |

### `template custom`

Render custom Go templates with project context.

```bash
ch template custom --template <path.gotpl> --output <path>
```

| Flag | Description |
|:-----|:------------|
| `--template` | Path to Go template file (`.gotpl`) |
| `--output` | Output file (default: stdout) |

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
