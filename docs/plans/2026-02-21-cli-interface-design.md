# CLI Interface Design

## Overview

Add a CLI interface to ContainerHive using [urfave/cli](https://github.com/urfave/cli) with five top-level commands: `generate`, `build`, `test`, `sbom`, and `verify`.

## Command Structure

```
ch [global flags] <command> [command flags] [args...]

Global flags:
  -p, --project PATH    Project root (default: cwd)
  --build-id ID         Append +<id> to all tags

Commands:
  generate              Discover project and render to dist/
  build [images...]     Build images (all or filtered)
  test [images...]      Test images (all or filtered)
  sbom [images...]      Generate SBOMs (all or filtered)
  verify                Verify project structure
```

## Image Filter Parsing

Positional args after each command are parsed as `[]build.Filter`:

- `ubuntu` → match image name "ubuntu", all tags
- `ubuntu:24.04` → match image "ubuntu", tag "24.04"
- `dotnet:8.0.300-test` → match image "dotnet", tag "8.0.300-test" (variant suffix included)

Multiple args are OR'd — if any filter matches, the image/tag is included.

## Build ID Semantics

`--build-id abc123` causes the render phase to suffix all tags with `+abc123`. Tag `24.04` becomes `24.04+abc123` in `dist/`, and all downstream steps (build, test, sbom) see the suffixed tag naturally.

## hive.yml Extension

```yaml
buildkit:
  address: "tcp://127.0.0.1:8502"

cache:
  type: s3
  endpoint: "http://..."
  bucket: "buildkit-cache"
  region: "garage"
  access_key_id: "..."
  secret_access_key: "..."

registry:
  address: "registry.example.com"
```

- `build` and `test` read BuildKit/cache config from `hive.yml`
- Registry is used automatically when `CI` env var is set, or when `--registry` flag is passed locally
- `generate`, `sbom`, `verify` don't need BuildKit config

## Command Behavior

| Command    | Packages used                                                  |
|------------|----------------------------------------------------------------|
| `generate` | `discovery` → `rendering`                                      |
| `build`    | `discovery` → `rendering` → `deps` → `build` (+ `registry`)  |
| `test`     | Operates on existing OCI tars in `dist/` via `cst`             |
| `sbom`     | Operates on existing OCI tars in `dist/` via `sbom`            |
| `verify`   | `discovery` (+ future linting via TODO)                        |

`test` and `sbom` require images to be already built. They do not trigger builds. This keeps commands composable:

```sh
ch generate && ch build && ch test && ch sbom
```

## Code Organization

```
cmd/ch/
  main.go          # App setup, global flags, command registration
  generate.go      # generate command
  build.go         # build command
  test.go          # test command
  sbom.go          # sbom command
  verify.go        # verify command
  filter.go        # shared: parse positional args into []build.Filter
  config.go        # shared: read hive.yml build/cache/registry config
```

## Project Root Resolution

Default to current working directory, overridable with `-p` flag.
