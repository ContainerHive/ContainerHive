# Generate CI pipelines

ContainerHive generates CI pipeline configurations from your project definition, so you don't have to maintain pipeline
YAML by hand.

## Generating CI pipelines

Use `template ci` to generate a pipeline for your CI provider:

```bash
ch template ci --provider gitlab --output .gitlab-ci.yml
ch template ci --provider github --output .github/workflows/build.yml
```

### Supported providers

| Provider       | Flag value | Output                       |
|:---------------|:-----------|:-----------------------------|
| GitLab CI      | `gitlab`   | GitLab CI pipeline YAML      |
| GitHub Actions | `github`   | GitHub Actions workflow YAML |

### How it works

1. ContainerHive discovers your project and resolves image dependencies.
2. Images are grouped into stages based on their dependency depth — images at the same depth can build in parallel.
3. For each image, the pipeline includes jobs for building (per-platform), creating multi-arch manifests, and running
   tests.
4. If an image has a `latest_alias` configured, the highest semantic version tag is automatically retagged as the
   specified alias (e.g., `latest`, `stable`).
5. The generated pipeline references the ContainerHive container image to run all commands.
6. Before builds, Dockerfiles are linted with hadolint (enabled by default). Lint failures gate the build stage.
7. After successful builds, an HTML and JSON report is generated and published to GitHub Pages or GitLab Pages.

### Options

| Flag             | Description                                                  |
|:-----------------|:-------------------------------------------------------------|
| `--provider`     | CI provider (`gitlab` or `github`) — required                |
| `--output`       | Output file path (default: stdout)                           |
| `--template-dir` | Custom template directory to override built-in templates     |
| `--artifacts`    | Upload/download build artifacts between jobs                 |
| `--version`      | ContainerHive CLI version to use (default: current version)  |
| `--image-name`   | Container image name for the CLI (default: `containerhive/containerhive`) |

### Example with custom image

```bash
ch template ci \
  --provider github \
  --output .github/workflows/build.yml \
  --image-name ghcr.io/containerhive/containerhive \
  --version v1.0.0 \
  --artifacts
```

## Custom templates

For use cases beyond CI pipelines, the `template custom` command renders any Go template with full access to the project
context.

```bash
ch template custom --template my-template.gotpl --output output.yml
```

### Template context

Custom templates receive a `CIContext` with the following fields:

| Field       | Type   | Description                                                |
|:------------|:-------|:-----------------------------------------------------------|
| `Images`    | list   | All images with name, tags, dependencies, depth, platforms |
| `Platforms` | list   | All unique platforms across all images                     |
| `Stages`    | list   | Ordered build stages                                       |
| `Config`    | object | Registry and cache configuration                           |

Each image in `Images` provides:

| Field          | Type   | Description                            |
|:---------------|:-------|:---------------------------------------|
| `Name`         | string | Image name                             |
| `Tags`         | list   | Tag names                              |
| `Dependencies` | list   | Names of images this image depends on  |
| `Depth`        | int    | Dependency depth (0 = no dependencies) |
| `Platforms`    | list   | Target platforms                       |

### Template functions

Templates have access to [Sprig](http://masterminds.github.io/sprig/) functions plus:

| Function                  | Description                                                                             |
|:--------------------------|:----------------------------------------------------------------------------------------|
| `resolve_base(name, tag)` | Produces an internal image reference (`__hive__/name:tag`) for inter-image dependencies |
| `option(key)`             | Returns the value of a template option from `hive.yml`, or empty string if not set      |

### Example custom template

```gotpl
{{- range .Images }}
Image: {{ .Name }}
  Tags: {{ range .Tags }}{{ . }}, {{ end }}
  Platforms: {{ range .Platforms }}{{ . }}, {{ end }}
  Dependencies: {{ range .Dependencies }}{{ . }}, {{ end }}
{{ end }}
```

## Overriding built-in CI templates

You can override the built-in CI templates by providing a custom template directory:

```bash
ch template ci --provider gitlab --template-dir ./my-templates --output .gitlab-ci.yml
```

The directory should contain `.gotpl` files following the same structure as the built-in templates. Files in the custom
directory take precedence over the built-in ones.

### Built-in template structure

```
templates/<provider>/
├── pipeline.yml.gotpl   (or workflow.yml.gotpl for GitHub)
├── build-job.yml.gotpl
├── lint-job.yml.gotpl
├── manifest-job.yml.gotpl
└── test-job.yml.gotpl
```

The entrypoint template includes the job templates as partials using `{{ template "build-job.yml.gotpl" . }}`.

## The `generate` command

Before building or generating CI configs, run `generate` to discover the project and render all configurations to
`dist/`:

```bash
ch generate
```

This processes `hive.yml` and all `image.yml` definitions, renders Dockerfiles and test templates, and prepares the
`dist/` directory for subsequent `build`, `test`, and `sbom` commands.

## Report generation

By default, the generated CI pipeline includes a `report` job that runs after all build jobs complete. This job
generates:

- `dist/report.html` — Interactive HTML report with image details, tags, platforms, and SBOM data
- `dist/report.json` — Machine-readable JSON report

### GitHub Pages

The HTML report is automatically deployed to GitHub Pages. Your repository must have Pages enabled in settings with the
source set to "GitHub Actions".

### GitLab Pages

The report is uploaded as a GitLab Pages artifact and exposed as "ContainerHive Report" in the merge request widget. The
JSON report is also available as an artifact.

## Lint job

By default, the generated CI pipeline includes a `lint` job that runs after the `generate` stage and before any build
stages. This job lints all plain Dockerfiles in the project using [hadolint](https://github.com/hadolint/hadolint)
via `ch lint`.

### GitLab Code Quality report

For GitLab CI, the lint job produces a [Code Quality report](https://docs.gitlab.com/ee/ci/testing/code_quality.html)
artifact that GitLab renders inline on merge requests. The report uses the `codeclimate` output format:

```bash
ch lint --format codeclimate=gl-code-quality-report.json
```

### GitHub Actions annotations

For GitHub Actions, use the `github-actions` output format to emit annotations inline on pull request diffs:

```bash
ch lint --format github-actions
```

Lint failures still cause the job to fail and gate subsequent build jobs. Without the `--format` flag, findings only
appear in the workflow log — using `--format github-actions` makes them visible on the changed lines in the PR.

### Disabling linting

To disable the lint job entirely, set `ci_lint` to `false` in your `hive.yml`:

```yaml
template_options:
  ci_lint: false
```
