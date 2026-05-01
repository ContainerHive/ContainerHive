# Pre-commit integration

ContainerHive ships [pre-commit](https://pre-commit.com) hooks that let you automate CI pipeline generation, project
verification, and other ContainerHive commands directly in your development workflow.

## Setup

Add the ContainerHive hook repository to your `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: https://github.com/ContainerHive/pre-commit
    rev: main # use tag or commit hash for your projects
    hooks:
      - id: template
      - id: verify
```

## Available hooks

### `run`

Run an arbitrary ContainerHive command. By default runs `ch` with no subcommand. Pass arguments via `args`.

```yaml
- id: run
  args: [ generate ]
```

## Example: automated CI pipeline generation

The following example can be used to keep CI pipelines in
sync with the project definition:

```yaml
repos:
  - repo: https://github.com/ContainerHive/pre-commit
    rev: main # use tag or commit hash for your projects
    hooks:
      - id: template
        args:
          - ci
          - --provider
          - github
          - --output
          - .github/workflows/main.yml
```

Every time `hive.yml` or any `image.yml` changes, pre-commit re-generates the GitHub Actions workflow so it always
reflects the current project state.
